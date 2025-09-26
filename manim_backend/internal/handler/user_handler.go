package handler

import (
	"encoding/json"
	"net/http"

	"manim-backend/internal/middleware"
	"manim-backend/internal/svc"
	"manim-backend/internal/types"
)

type UserHandler struct {
	ctx *svc.ServiceContext
}

func NewUserHandler(ctx *svc.ServiceContext) *UserHandler {
	return &UserHandler{ctx: ctx}
}

// WriteJSON 辅助函数用于写入JSON响应
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// ParseJSON 辅助函数用于解析JSON请求
func ParseJSON(r *http.Request, data interface{}) error {
	return json.NewDecoder(r.Body).Decode(data)
}

// Register 用户注册
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req types.RegisterRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "请求参数错误"})
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "用户名、邮箱和密码不能为空"})
		return
	}

	user, err := h.ctx.UserService.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
		return
	}

	token, err := h.ctx.Auth.GenerateToken(user.ID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "令牌生成失败"})
		return
	}

	WriteJSON(w, http.StatusOK, types.AuthResponse{
		Token: token,
		User: types.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
	})
}

// Login 用户登录
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req types.LoginRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "请求参数错误"})
		return
	}

	if req.Username == "" || req.Password == "" {
		WriteJSON(w, http.StatusBadRequest, types.ErrorResponse{Error: "用户名和密码不能为空"})
		return
	}

	user, err := h.ctx.UserService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		WriteJSON(w, http.StatusUnauthorized, types.ErrorResponse{Error: err.Error()})
		return
	}

	token, err := h.ctx.Auth.GenerateToken(user.ID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, types.ErrorResponse{Error: "令牌生成失败"})
		return
	}

	WriteJSON(w, http.StatusOK, types.AuthResponse{
		Token: token,
		User: types.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
	})
}

// GetProfile 获取用户信息
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, types.ErrorResponse{Error: "用户未认证"})
		return
	}

	WriteJSON(w, http.StatusOK, types.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	})
}
