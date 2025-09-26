package service

import (
	"context"
	"log"
	"time"

	"manim-backend/internal/config"
	"manim-backend/internal/model"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type VideoService struct {
	db       *gorm.DB
	rdb      *redis.Client
	queueSvc *VideoQueueService
	manimCfg config.ManimConfig
}

func NewVideoService(db *gorm.DB, rdb *redis.Client, manimCfg config.ManimConfig) *VideoService {
	// 先创建VideoService实例
	videoService := &VideoService{
		db:       db,
		rdb:      rdb,
		manimCfg: manimCfg,
	}

	// 创建ManimService实例
	manimSvc := NewManimService(manimCfg, videoService)

	// 创建队列服务
	queueSvc := NewVideoQueueService(db, rdb, manimCfg, manimSvc)
	videoService.queueSvc = queueSvc

	return videoService
}

// NewVideoServiceWithManim 创建VideoService并传入已有的ManimService
func NewVideoServiceWithManim(db *gorm.DB, rdb *redis.Client, manimCfg config.ManimConfig, manimService *ManimService) *VideoService {
	// 创建VideoService实例
	videoService := &VideoService{
		db:       db,
		rdb:      rdb,
		manimCfg: manimCfg,
	}

	// 创建队列服务，使用传入的ManimService
	queueSvc := NewVideoQueueService(db, rdb, manimCfg, manimService)
	videoService.queueSvc = queueSvc

	return videoService
}

// CreateVideo 创建视频任务
func (s *VideoService) CreateVideo(ctx context.Context, userID uint, prompt string) (*model.Video, error) {
	// 为数据库操作创建带超时的上下文（30秒超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	video := &model.Video{
		UserID: userID,
		Prompt: prompt,
		Status: model.VideoStatusPending,
	}

	if err := s.db.WithContext(timeoutCtx).Create(video).Error; err != nil {
		return nil, err
	}

	return video, nil
}

// GetVideoByID 根据ID获取视频
func (s *VideoService) GetVideoByID(ctx context.Context, id uint) (*model.Video, error) {
	// 为数据库操作创建带超时的上下文（30秒超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var video model.Video
	if err := s.db.WithContext(timeoutCtx).First(&video, id).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

// GetUserVideos 获取用户的所有视频
func (s *VideoService) GetUserVideos(ctx context.Context, userID uint, page, pageSize int) ([]model.Video, int64, error) {
	var videos []model.Video
	var total int64

	// 为数据库查询创建带超时的上下文（30秒超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 获取总数
	err := s.db.WithContext(timeoutCtx).Model(&model.Video{}).Where("user_id = ?", userID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	err = s.db.WithContext(timeoutCtx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&videos).Error

	return videos, total, err
}

// UpdateVideoStatus 更新视频状态
func (s *VideoService) UpdateVideoStatus(ctx context.Context, id uint, status model.VideoStatus, manimCode, videoPath, errorMsg string) error {
	// 为数据库操作创建带超时的上下文（30秒超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 先检查视频是否存在
	var video model.Video
	if err := s.db.WithContext(timeoutCtx).First(&video, id).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if manimCode != "" {
		updates["manim_code"] = manimCode
	}
	if videoPath != "" {
		updates["video_path"] = videoPath
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}

	return s.db.WithContext(timeoutCtx).Model(&model.Video{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteVideo 删除视频
func (s *VideoService) DeleteVideo(ctx context.Context, id uint) error {
	// 为数据库操作创建带超时的上下文（30秒超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 先从队列中移除视频
	err := s.queueSvc.RemoveFromQueue(ctx, id)
	if err != nil {
		// 如果视频不在队列中，忽略错误
		log.Printf("从队列中移除视频 %d 失败: %v", id, err)
	}

	return s.db.WithContext(timeoutCtx).Delete(&model.Video{}, id).Error
}

// AddToQueue 添加视频到渲染队列
func (s *VideoService) AddToQueue(ctx context.Context, videoID uint) error {
	return s.queueSvc.AddToQueue(ctx, videoID)
}

// StartQueueWorkers 启动队列工作者
func (s *VideoService) StartQueueWorkers(ctx context.Context) error {
	return s.queueSvc.StartWorkers(ctx)
}

// StopQueueWorkers 停止队列工作者
func (s *VideoService) StopQueueWorkers() {
	s.queueSvc.StopWorkers()
}

// GetQueueStatus 获取队列状态
func (s *VideoService) GetQueueStatus(ctx context.Context) (int64, []uint, error) {
	return s.queueSvc.GetQueueStatus(ctx)
}

// CleanOldVideos 清理过期视频（避免长时间堆积）
func (s *VideoService) CleanOldVideos(ctx context.Context, days int) error {
	// 为数据库操作创建带超时的上下文（30秒超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 计算过期时间
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 先获取要删除的视频ID列表
	var videoIDs []uint
	if err := s.db.WithContext(timeoutCtx).Model(&model.Video{}).
		Where("created_at < ?", cutoffTime).
		Pluck("id", &videoIDs).Error; err != nil {
		return err
	}

	// 从队列中移除这些视频
	for _, videoID := range videoIDs {
		err := s.queueSvc.RemoveFromQueue(ctx, videoID)
		if err != nil {
			log.Printf("从队列中移除视频 %d 失败: %v", videoID, err)
		}
	}

	// 删除过期视频
	result := s.db.WithContext(timeoutCtx).Where("created_at < ?", cutoffTime).Delete(&model.Video{})
	if result.Error != nil {
		return result.Error
	}

	log.Printf("清理了 %d 个过期视频", result.RowsAffected)
	return nil
}

// GetProcessingVideos 获取正在处理的视频
func (s *VideoService) GetProcessingVideos(ctx context.Context) ([]model.Video, error) {
	var videos []model.Video
	err := s.db.WithContext(ctx).
		Where("status IN ?", []model.VideoStatus{model.VideoStatusPending, model.VideoStatusProcessing}).
		Order("created_at ASC").
		Find(&videos).Error
	return videos, err
}
