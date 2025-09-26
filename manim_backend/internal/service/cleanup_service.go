package service

import (
	"context"
	"log"
	"time"
)

type CleanupService struct {
	videoService *VideoService
}

func NewCleanupService(videoService *VideoService) *CleanupService {
	return &CleanupService{
		videoService: videoService,
	}
}

// StartCleanupScheduler 启动定时清理任务
func (s *CleanupService) StartCleanupScheduler(ctx context.Context) {
	go s.scheduleCleanup(ctx)
}

// scheduleCleanup 定时清理调度器
func (s *CleanupService) scheduleCleanup(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // 每天执行一次
	defer ticker.Stop()

	// 立即执行一次清理
	s.cleanupOldVideos(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("清理调度器已停止")
			return
		case <-ticker.C:
			s.cleanupOldVideos(ctx)
		}
	}
}

// cleanupOldVideos 清理过期视频
func (s *CleanupService) cleanupOldVideos(ctx context.Context) {
	log.Println("开始清理过期视频...")

	// 清理30天前的视频
	err := s.videoService.CleanOldVideos(ctx, 30)
	if err != nil {
		log.Printf("清理过期视频失败: %v", err)
	} else {
		log.Println("过期视频清理完成")
	}
}

// CleanupNow 立即执行清理
func (s *CleanupService) CleanupNow(ctx context.Context) error {
	return s.videoService.CleanOldVideos(ctx, 30)
}
