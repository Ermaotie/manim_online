package svc

import (
	"manim-backend/internal/config"
	"manim-backend/internal/middleware"
	"manim-backend/internal/model"
	"manim-backend/internal/service"
)

type ServiceContext struct {
	Config       config.Config
	Auth         *middleware.AuthMiddleware
	UserService  *service.UserService
	VideoService *service.VideoService
	AIService    *service.AIService
	ManimService *service.ManimService
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := model.InitDB(c.MySQL)

	// 初始化Redis连接
	redisClient := model.InitRedis(c.Redis)

	// 初始化服务
	userService := service.NewUserService(db)

	// 先创建ManimService（不依赖VideoService）
	manimService := service.NewManimService(c.Manim, nil)

	// 创建VideoService，传入ManimService
	videoService := service.NewVideoServiceWithManim(db, redisClient, c.Manim, manimService)

	// 设置ManimService的VideoService依赖
	manimService.SetVideoService(videoService)

	aiService := service.NewAIService(c.OpenAI)

	// 初始化中间件
	auth := middleware.NewAuthMiddleware(userService)

	return &ServiceContext{
		Config:       c,
		Auth:         auth,
		UserService:  userService,
		VideoService: videoService,
		AIService:    aiService,
		ManimService: manimService,
	}
}
