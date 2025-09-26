package handler

import (
	"net/http"
	"strconv"

	"manim-backend/internal/middleware"
	"manim-backend/internal/svc"
	"manim-backend/internal/types"
)

type VideoHandler struct {
	ctx *svc.ServiceContext
}

func NewVideoHandler(ctx *svc.ServiceContext) *VideoHandler {
	return &VideoHandler{ctx: ctx}
}

// CreateVideo 创建视频任务
func (h *VideoHandler) CreateVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, types.ErrorResponse{Error: "用户未认证"})
		return
	}

	var req types.CreateVideoRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "请求参数错误"})
		return
	}

	if req.Prompt == "" {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "描述不能为空"})
		return
	}

	var manimCode string
	var err error

	// 如果前端提供了代码，使用前端代码；否则重新生成
	if req.Code != "" {
		manimCode = req.Code
		// 验证前端提供的代码
		isValid, msg := h.ctx.AIService.ValidateManimCode(r.Context(), manimCode)
		if !isValid {
			WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "提供的代码无效: " + msg})
			return
		}
	} else {
		// 生成Manim代码
		manimCode, err = h.ctx.AIService.GenerateManimCode(r.Context(), req.Prompt)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "代码生成失败: " + err.Error()})
			return
		}

		// 验证生成的代码
		isValid, msg := h.ctx.AIService.ValidateManimCode(r.Context(), manimCode)
		if !isValid {
			WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "生成的代码无效: " + msg})
			return
		}
	}

	// 创建视频记录
	video, err := h.ctx.VideoService.CreateVideo(r.Context(), userID, req.Prompt)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "创建视频失败: " + err.Error()})
		return
	}

	// 更新视频记录，添加生成的代码
	err = h.ctx.VideoService.UpdateVideoStatus(r.Context(), video.ID, video.Status, manimCode, "", "")
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "更新视频失败: " + err.Error()})
		return
	}

	// 将视频添加到渲染队列
	err = h.ctx.VideoService.AddToQueue(r.Context(), video.ID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "添加到队列失败: " + err.Error()})
		return
	}

	// 异步处理视频生成
	go func() {
		h.ctx.ManimService.GenerateVideo(r.Context(), video.ID, manimCode)
	}()

	WriteJSON(w, http.StatusOK, types.VideoResponse{
		ID:        video.ID,
		Prompt:    video.Prompt,
		ManimCode: manimCode,
		Status:    video.Status.String(),
		CreatedAt: video.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: video.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// GetVideo 获取视频详情
func (h *VideoHandler) GetVideo(w http.ResponseWriter, r *http.Request) {
	// 优先从查询参数获取ID，如果没有则从路径参数获取
	videoID := r.URL.Query().Get("id")
	if videoID == "" {
		videoID = r.PathValue("id")
	}

	if videoID == "" {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "视频ID不能为空"})
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, types.ErrorResponse{Error: "用户未认证"})
		return
	}

	// 将字符串ID转换为uint
	videoIDUint, err := strconv.ParseUint(videoID, 10, 32)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "无效的视频ID"})
		return
	}

	video, err := h.ctx.VideoService.GetVideoByID(r.Context(), uint(videoIDUint))
	if err != nil {
		WriteJSON(w, http.StatusNotFound, types.ErrorResponse{Error: "视频不存在"})
		return
	}

	// 检查视频是否属于当前用户
	if video.UserID != userID {
		WriteJSON(w, http.StatusForbidden, types.ErrorResponse{Error: "无权访问此视频"})
		return
	}

	// 将相对路径转换为完整URL
	var videoURL string
	if video.VideoPath != "" {
		videoURL = "/" + video.VideoPath
	}

	WriteJSON(w, http.StatusOK, types.VideoResponse{
		ID:        video.ID,
		Prompt:    video.Prompt,
		ManimCode: video.ManimCode,
		VideoPath: videoURL,
		Status:    video.Status.String(),
		ErrorMsg:  video.ErrorMsg,
		CreatedAt: video.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: video.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// ListVideos 获取用户视频列表
func (h *VideoHandler) ListVideos(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, types.ErrorResponse{Error: "用户未认证"})
		return
	}

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	pageSize := 10

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	videos, total, err := h.ctx.VideoService.GetUserVideos(r.Context(), userID, page, pageSize)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "获取视频列表失败: " + err.Error()})
		return
	}

	var videoResponses []types.VideoResponse
	for _, video := range videos {
		// 将相对路径转换为完整URL
		var videoURL string
		if video.VideoPath != "" {
			videoURL = "/" + video.VideoPath
		}

		videoResponses = append(videoResponses, types.VideoResponse{
			ID:        video.ID,
			Prompt:    video.Prompt,
			VideoPath: videoURL,
			Status:    video.Status.String(),
			ErrorMsg:  video.ErrorMsg,
			CreatedAt: video.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: video.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	WriteJSON(w, http.StatusOK, types.VideoListResponse{
		Videos:   videoResponses,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// DeleteVideo 删除视频
func (h *VideoHandler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("id")
	if videoID == "" {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "视频ID不能为空"})
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, types.ErrorResponse{Error: "用户未认证"})
		return
	}

	// 将字符串ID转换为uint
	videoIDUint, err := strconv.ParseUint(videoID, 10, 32)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "无效的视频ID"})
		return
	}

	video, err := h.ctx.VideoService.GetVideoByID(r.Context(), uint(videoIDUint))
	if err != nil {
		WriteJSON(w, http.StatusNotFound, types.ErrorResponse{Error: "视频不存在"})
		return
	}

	if video.UserID != userID {
		WriteJSON(w, http.StatusForbidden, types.ErrorResponse{Error: "无权删除此视频"})
		return
	}

	if err := h.ctx.VideoService.DeleteVideo(r.Context(), uint(videoIDUint)); err != nil {
		WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "删除视频失败: " + err.Error()})
		return
	}

	WriteJSON(w, http.StatusOK, types.SuccessResponse{
		Message: "删除成功",
	})
}

// GenerateCode 生成Manim代码
func (h *VideoHandler) GenerateCode(w http.ResponseWriter, r *http.Request) {
	var req types.GenerateCodeRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "请求参数错误"})
		return
	}

	if req.Prompt == "" {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "描述不能为空"})
		return
	}

	// 生成Manim代码
	manimCode, err := h.ctx.AIService.GenerateManimCode(r.Context(), req.Prompt)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "代码生成失败: " + err.Error()})
		return
	}

	WriteJSON(w, http.StatusOK, types.GenerateCodeResponse{
		Code:    manimCode,
		IsValid: true,
		Message: "代码生成成功",
	})
}

// ValidateCode 验证Manim代码
func (h *VideoHandler) ValidateCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := ParseJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "请求参数错误"})
		return
	}

	if req.Code == "" {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "代码不能为空"})
		return
	}

	// 验证代码
	isValid, msg := h.ctx.AIService.ValidateManimCode(r.Context(), req.Code)

	WriteJSON(w, http.StatusOK, types.GenerateCodeResponse{
		Code:    req.Code,
		IsValid: isValid,
		Message: msg,
	})
}

// CheckAPIHealth 检测AI API健康状态
func (h *VideoHandler) CheckAPIHealth(w http.ResponseWriter, r *http.Request) {
	// 首先验证配置
	configValid, configErrors := h.ctx.AIService.ValidateConfig(h.ctx.Config.OpenAI)

	if !configValid {
		WriteJSON(w, http.StatusBadRequest, types.APIHealthResponse{
			Status:    "error",
			Message:   "配置验证失败",
			Details:   configErrors,
			IsHealthy: false,
		})
		return
	}

	// 然后测试API连接
	isHealthy, healthMsg := h.ctx.AIService.CheckAPIHealth(r.Context())

	response := types.APIHealthResponse{
		Status:    "success",
		Message:   healthMsg,
		IsHealthy: isHealthy,
	}

	if !isHealthy {
		response.Status = "error"
		WriteJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	WriteJSON(w, http.StatusOK, response)
}
