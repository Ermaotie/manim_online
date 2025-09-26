package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"manim-backend/internal/config"
	"manim-backend/internal/model"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type VideoQueueService struct {
	db        *gorm.DB
	rdb       *redis.Client
	manimCfg  config.ManimConfig
	manimSvc  *ManimService
	queueName string
	workers   int
	mu        sync.Mutex
	isRunning bool
}

func NewVideoQueueService(db *gorm.DB, rdb *redis.Client, manimCfg config.ManimConfig, manimSvc *ManimService) *VideoQueueService {
	return &VideoQueueService{
		db:        db,
		rdb:       rdb,
		manimCfg:  manimCfg,
		manimSvc:  manimSvc,
		queueName: "video_render_queue",
		workers:   manimCfg.MaxConcurrent,
		isRunning: false,
	}
}

// AddToQueue 添加视频到渲染队列
func (s *VideoQueueService) AddToQueue(ctx context.Context, videoID uint) error {
	// 检查视频是否存在
	var video model.Video
	if err := s.db.WithContext(ctx).First(&video, videoID).Error; err != nil {
		return fmt.Errorf("视频不存在: %v", err)
	}

	// 检查视频是否已经在队列中
	exists, err := s.rdb.ZScore(ctx, s.queueName, fmt.Sprintf("%d", videoID)).Result()
	if err == nil && exists > 0 {
		return fmt.Errorf("视频已在队列中")
	}

	// 添加到Redis有序集合，使用时间戳作为分数
	score := float64(time.Now().Unix())
	if err := s.rdb.ZAdd(ctx, s.queueName, &redis.Z{
		Score:  score,
		Member: fmt.Sprintf("%d", videoID),
	}).Err(); err != nil {
		return fmt.Errorf("添加到队列失败: %v", err)
	}

	// 更新视频状态为排队中
	if err := s.db.WithContext(ctx).Model(&video).Update("status", model.VideoStatusQueued).Error; err != nil {
		return fmt.Errorf("更新视频状态失败: %v", err)
	}

	log.Printf("视频 %d 已添加到渲染队列", videoID)
	return nil
}

// StartWorkers 启动队列工作者
func (s *VideoQueueService) StartWorkers(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("队列工作者已在运行")
	}
	s.isRunning = true
	s.mu.Unlock()

	log.Printf("启动 %d 个视频渲染工作者", s.workers)

	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go s.worker(ctx, i, &wg)
	}

	// 等待所有工作者完成
	go func() {
		wg.Wait()
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		log.Println("所有视频渲染工作者已停止")
	}()

	return nil
}

// StopWorkers 停止队列工作者
func (s *VideoQueueService) StopWorkers() {
	s.mu.Lock()
	s.isRunning = false
	s.mu.Unlock()
}

// worker 队列工作者
func (s *VideoQueueService) worker(ctx context.Context, id int, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Printf("视频渲染工作者 %d 启动", id)

	for {
		// 检查是否应该停止
		s.mu.Lock()
		if !s.isRunning {
			s.mu.Unlock()
			break
		}
		s.mu.Unlock()

		// 从队列中获取任务
		videoID, err := s.getNextTask(ctx)
		if err != nil {
			if err != redis.Nil {
				log.Printf("工作者 %d 获取任务失败: %v", id, err)
			}
			time.Sleep(5 * time.Second) // 队列为空，等待5秒
			continue
		}

		log.Printf("工作者 %d 开始处理视频 %d", id, videoID)
		s.processVideo(ctx, videoID, id)
	}

	log.Printf("视频渲染工作者 %d 停止", id)
}

// getNextTask 获取下一个任务
func (s *VideoQueueService) getNextTask(ctx context.Context) (uint, error) {
	// 使用阻塞弹出操作获取最早的任务
	result, err := s.rdb.ZPopMin(ctx, s.queueName, 1).Result()
	if err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, redis.Nil
	}

	var videoID uint
	_, err = fmt.Sscanf(result[0].Member.(string), "%d", &videoID)
	if err != nil {
		return 0, fmt.Errorf("解析视频ID失败: %v", err)
	}

	return videoID, nil
}

// processVideo 处理视频渲染
func (s *VideoQueueService) processVideo(ctx context.Context, videoID uint, workerID int) {
	// 获取视频信息
	var video model.Video
	if err := s.db.WithContext(ctx).First(&video, videoID).Error; err != nil {
		log.Printf("工作者 %d 获取视频 %d 信息失败: %v", workerID, videoID, err)
		return
	}

	// 检查是否有Manim代码
	if video.ManimCode == "" {
		log.Printf("工作者 %d 视频 %d 没有Manim代码，无法渲染", workerID, videoID)
		// 更新视频状态为失败
		s.db.WithContext(ctx).Model(&video).Update("status", model.VideoStatusFailed)
		s.db.WithContext(ctx).Model(&video).Update("error_msg", "没有Manim代码")
		return
	}

	log.Printf("工作者 %d 开始渲染视频 %d", workerID, videoID)

	// 调用Manim服务进行实际渲染
	err := s.manimSvc.GenerateVideo(ctx, videoID, video.ManimCode)
	if err != nil {
		log.Printf("工作者 %d 渲染视频 %d 失败: %v", workerID, videoID, err)
		// 更新视频状态为失败
		s.db.WithContext(ctx).Model(&video).Update("status", model.VideoStatusFailed)
		s.db.WithContext(ctx).Model(&video).Update("error_msg", err.Error())
		return
	}

	log.Printf("工作者 %d 完成视频 %d 渲染", workerID, videoID)
}

// GetQueueStatus 获取队列状态
func (s *VideoQueueService) GetQueueStatus(ctx context.Context) (int64, []uint, error) {
	// 获取队列长度
	count, err := s.rdb.ZCard(ctx, s.queueName).Result()
	if err != nil {
		return 0, nil, err
	}

	// 获取队列中的视频ID
	result, err := s.rdb.ZRange(ctx, s.queueName, 0, -1).Result()
	if err != nil {
		return count, nil, err
	}

	var videoIDs []uint
	for _, member := range result {
		var videoID uint
		_, err := fmt.Sscanf(member, "%d", &videoID)
		if err == nil {
			videoIDs = append(videoIDs, videoID)
		}
	}

	return count, videoIDs, nil
}

// RemoveFromQueue 从队列中移除视频
func (s *VideoQueueService) RemoveFromQueue(ctx context.Context, videoID uint) error {
	removed, err := s.rdb.ZRem(ctx, s.queueName, fmt.Sprintf("%d", videoID)).Result()
	if err != nil {
		return err
	}

	if removed == 0 {
		return fmt.Errorf("视频不在队列中")
	}

	log.Printf("视频 %d 已从队列中移除", videoID)
	return nil
}
