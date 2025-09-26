package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"manim-backend/internal/middleware"
	"manim-backend/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	userHandler := NewUserHandler(serverCtx)
	videoHandler := NewVideoHandler(serverCtx)

	// 公开路由（无需认证）
	server.AddRoutes([]rest.Route{
		{
			Method:  "POST",
			Path:    "/api/auth/register",
			Handler: userHandler.Register,
		},
		{
			Method:  "POST",
			Path:    "/api/auth/login",
			Handler: userHandler.Login,
		},
	})

	// 需要认证的路由
	server.AddRoutes([]rest.Route{
		{
			Method:  "GET",
			Path:    "/api/user/profile",
			Handler: serverCtx.Auth.Handle(userHandler.GetProfile),
		},
		{
			Method:  "POST",
			Path:    "/api/videos",
			Handler: serverCtx.Auth.Handle(videoHandler.CreateVideo),
		},
		{
			Method:  "GET",
			Path:    "/api/videos",
			Handler: serverCtx.Auth.Handle(videoHandler.ListVideos),
		},
		{
			Method:  "GET",
			Path:    "/api/videos/detail",
			Handler: serverCtx.Auth.Handle(videoHandler.GetVideo),
		},
		{
			Method:  "DELETE",
			Path:    "/api/videos",
			Handler: serverCtx.Auth.Handle(videoHandler.DeleteVideo),
		},
		{
			Method:  "POST",
			Path:    "/api/ai/generate",
			Handler: serverCtx.Auth.Handle(videoHandler.GenerateCode),
		},
		{
			Method:  "POST",
			Path:    "/api/ai/validate",
			Handler: serverCtx.Auth.Handle(videoHandler.ValidateCode),
		},
		{
			Method:  "GET",
			Path:    "/api/ai/health",
			Handler: serverCtx.Auth.Handle(videoHandler.CheckAPIHealth),
		},
	})

	// 静态文件服务 - 用于提供生成的视频文件
	server.AddRoute(rest.Route{
		Method:  "GET",
		Path:    "/downloadvideo/:id/:file",
		Handler: middleware.WithCORS(serveVideoFile),
	})
}

// serveVideoFile 提供视频文件服务
func serveVideoFile(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头部，允许跨域访问
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 从URL路径参数获取用户ID和文件名
	userID := r.PathValue("id")
	file := r.PathValue("file")
	
	// 如果PathValue获取为空，尝试从URL路径中解析
	if userID == "" || file == "" {
		// 从URL路径中直接解析参数
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 3 && pathParts[0] == "downloadvideo" {
			userID = pathParts[1]
			file = pathParts[2]
		}
	}

	// 调试信息：打印获取到的参数
	fmt.Printf("获取到的参数 - userID: %s, file: %s\n", userID, file)

	// 检查文件是否存在
	filePath := "videos/" + userID + "/" + file
	fmt.Printf("构建的文件路径: %s\n", filePath)
	
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("文件不存在: %s\n", filePath)
		http.NotFound(w, r)
		return
	}
	
	fmt.Printf("文件存在，准备提供文件: %s\n", filePath)

	// 设置正确的Content-Type头部
	w.Header().Set("Content-Type", "video/mp4")

	// 设置缓存控制头部，避免浏览器缓存问题
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// 设置额外的安全头部，避免ORB错误
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")

	// 提供文件
	http.ServeFile(w, r, filePath)
}
