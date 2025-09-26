package handler

import (
	"net/http"

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
		Method: "GET",
		Path:   "/videos/:file",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			// 从URL路径参数获取文件名
			file := r.URL.Path[len("/videos/"):]
			http.ServeFile(w, r, "videos/"+file)
		},
	})
}
