package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"manim-backend/internal/config"
	"manim-backend/internal/handler"
	"manim-backend/internal/service"
	"manim-backend/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/manim-backend.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 启动视频渲染队列工作者
	if err := ctx.VideoService.StartQueueWorkers(context.Background()); err != nil {
		log.Printf("启动队列工作者失败: %v", err)
	} else {
		log.Println("视频渲染队列工作者已启动")
	}

	// 启动定时清理服务
	cleanupService := service.NewCleanupService(ctx.VideoService)
	cleanupService.StartCleanupScheduler(context.Background())

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
