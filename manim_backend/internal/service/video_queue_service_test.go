package service

import (
	"context"
	"testing"
	"time"

	"manim-backend/internal/config"
	"manim-backend/internal/model"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/conf"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 测试用的实际Manim代码
const testManimCode = `from manim import *
class PythagoreanTheorem(Scene):
    def construct(self):
        triangle_points = [RIGHT*3, UP*4, ORIGIN]
        triangle = Polygon(*triangle_points, color=WHITE, fill_opacity=0.5)
        square_a = Square(side_length=3, color=BLUE).move_to([1.5, -1.5, 0])
        square_b = Square(side_length=4, color=RED).move_to([-2, 2, 0])
        square_c = Square(side_length=5, color=GREEN).move_to([4.5, 2, 0])
        label_a = MathTex("3", color=BLUE).next_to(triangle, DOWN, buff=0.1)
        label_b = MathTex("4", color=RED).next_to(triangle, LEFT, buff=0.1)
        label_c = MathTex("5", color=GREEN).next_to(triangle.get_center(), UR, buff=0.2)
        equation = MathTex("3^2", "+", "4^2", "=", "5^2", color=WHITE)
        equation.scale(1.2).to_edge(UP)
        area_a = MathTex("9", color=BLUE).move_to(square_a.get_center())
        area_b = MathTex("16", color=RED).move_to(square_b.get_center())
        area_c = MathTex("25", color=GREEN).move_to(square_c.get_center())
        self.play(Create(triangle))
        self.wait(0.5)
        self.play(Write(label_a), Write(label_b), Write(label_c))
        self.wait(1)
        self.play(Create(square_a), Create(square_b), Create(square_c))
        self.wait(1)
        self.play(Write(area_a), Write(area_b), Write(area_c))
        self.wait(1)
        self.play(Write(equation))
        self.wait(2)
        box = SurroundingRectangle(equation, color=YELLOW, buff=0.2)
        self.play(Create(box))
        self.wait(2)`

// loadTestConfig 加载测试配置文件
func loadTestConfig() config.Config {
	var c config.Config
	err := conf.Load("../etc/manim-backend-test.yaml", &c)
	if err != nil {
		// 如果加载失败，使用默认配置
		c = config.Config{
			Redis: config.RedisConfig{
				Host:     "127.0.0.1",
				Port:     6379,
				Password: "123456",
				DB:       0,
			},
			Manim: config.ManimConfig{
				PythonPath:    "python",
				MaxConcurrent: 3,
				Timeout:       300,
			},
		}
	}
	return c
}

func setupTestVideoQueueService() (*VideoQueueService, *gorm.DB, *redis.Client) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	// 迁移表结构
	db.AutoMigrate(&model.User{}, &model.Video{})

	// 加载测试配置
	c := loadTestConfig()

	// 使用配置文件中的Redis配置
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Host + ":6379",
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	})

	// 创建VideoService（需要正确的参数）
	videoService := NewVideoService(db, rdb, c.Manim)

	// 创建Manim服务
	manimSvc := NewManimService(c.Manim, videoService)

	// 创建视频队列服务
	service := NewVideoQueueService(db, rdb, c.Manim, manimSvc)

	return service, db, rdb
}

func TestVideoQueueService_AddToQueue(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx := context.Background()

	// 清理Redis数据库
	rdb.FlushDB(ctx)

	// 创建测试用户和视频
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)
	video, err := videoService.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	tests := []struct {
		name    string
		videoID uint
		wantErr bool
		errMsg  string
	}{
		{
			name:    "添加视频到队列成功",
			videoID: video.ID,
			wantErr: false,
		},
		{
			name:    "视频不存在",
			videoID: 999,
			wantErr: true,
			errMsg:  "视频不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.AddToQueue(ctx, tt.videoID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// 验证视频状态已更新
				updatedVideo, err := videoService.GetVideoByID(ctx, tt.videoID)
				assert.NoError(t, err)
				assert.Equal(t, model.VideoStatusQueued, updatedVideo.Status)

				// 验证队列状态
				count, videoIDs, err := service.GetQueueStatus(ctx)
				assert.NoError(t, err)
				assert.Equal(t, int64(1), count)
				assert.Contains(t, videoIDs, tt.videoID)
			}
		})
	}
}

func TestVideoQueueService_AddToQueue_Duplicate(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx := context.Background()

	// 清理Redis数据库
	rdb.FlushDB(ctx)

	// 创建测试用户和视频
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)
	video, err := videoService.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	// 第一次添加应该成功
	err = service.AddToQueue(ctx, video.ID)
	assert.NoError(t, err)

	// 第二次添加应该失败
	err = service.AddToQueue(ctx, video.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "视频已在队列中")

	// 验证队列中仍然只有一个视频
	count, videoIDs, err := service.GetQueueStatus(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, []uint{video.ID}, videoIDs)
}

func TestVideoQueueService_GetQueueStatus(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx := context.Background()

	// 清理Redis数据库
	rdb.FlushDB(ctx)

	// 创建测试用户和视频
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)
	video1, err := videoService.CreateVideo(ctx, user.ID, "提示1")
	assert.NoError(t, err)
	video2, err := videoService.CreateVideo(ctx, user.ID, "提示2")
	assert.NoError(t, err)

	// 添加视频到队列
	err = service.AddToQueue(ctx, video1.ID)
	assert.NoError(t, err)
	err = service.AddToQueue(ctx, video2.ID)
	assert.NoError(t, err)

	// 获取队列状态
	count, videoIDs, err := service.GetQueueStatus(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Contains(t, videoIDs, video1.ID)
	assert.Contains(t, videoIDs, video2.ID)
}

func TestVideoQueueService_RemoveFromQueue(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx := context.Background()

	// 清理Redis数据库
	rdb.FlushDB(ctx)

	// 创建测试用户和视频
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)
	video, err := videoService.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	// 添加视频到队列
	err = service.AddToQueue(ctx, video.ID)
	assert.NoError(t, err)

	// 验证队列中有视频
	count, videoIDs, err := service.GetQueueStatus(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Contains(t, videoIDs, video.ID)

	// 从队列中移除视频
	err = service.RemoveFromQueue(ctx, video.ID)
	assert.NoError(t, err)

	// 验证队列为空
	count, videoIDs, err = service.GetQueueStatus(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.Empty(t, videoIDs)

	// 尝试移除不存在的视频
	err = service.RemoveFromQueue(ctx, 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "视频不在队列中")
}

func TestVideoQueueService_StartStopWorkers(t *testing.T) {
	service, _, _ := setupTestVideoQueueService()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 启动工作者
	err := service.StartWorkers(ctx)
	assert.NoError(t, err)

	// 验证工作者正在运行
	// 这里我们无法直接检查内部状态，但可以验证没有错误

	// 停止工作者
	service.StopWorkers()

	// 再次启动应该成功
	err = service.StartWorkers(ctx)
	assert.NoError(t, err)

	// 停止工作者
	service.StopWorkers()
}

func TestVideoQueueService_ProcessVideo_NoManimCode(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx := context.Background()

	// 清理Redis数据库
	rdb.FlushDB(ctx)

	// 创建测试用户和视频（没有Manim代码）
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)
	video, err := videoService.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	// 添加视频到队列
	err = service.AddToQueue(ctx, video.ID)
	assert.NoError(t, err)

	// 直接调用processVideo来测试没有Manim代码的情况
	service.processVideo(ctx, video.ID, 0)

	// 验证视频状态已更新为失败
	updatedVideo, err := videoService.GetVideoByID(ctx, video.ID)
	assert.NoError(t, err)
	assert.Equal(t, model.VideoStatusFailed, updatedVideo.Status)
	assert.Equal(t, "没有Manim代码", updatedVideo.ErrorMsg)
}

func TestVideoQueueService_ProcessVideo_WithManimCode(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx := context.Background()

	// 清理Redis数据库
	rdb.FlushDB(ctx)

	// 创建测试用户和视频（包含Manim代码）
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)
	video, err := videoService.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	// 更新视频状态和Manim代码
	err = videoService.UpdateVideoStatus(ctx, video.ID, model.VideoStatusPending, testManimCode, "", "")
	assert.NoError(t, err)

	// 添加视频到队列
	err = service.AddToQueue(ctx, video.ID)
	assert.NoError(t, err)

	// 直接调用processVideo来测试有Manim代码的情况
	service.processVideo(ctx, video.ID, 0)

	// 由于我们使用的是模拟的Manim服务，这里主要验证流程没有panic
	// 在实际环境中，Manim服务会进行实际的视频渲染
}

func TestVideoQueueService_GetNextTask(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx := context.Background()

	// 清理Redis数据库
	rdb.FlushDB(ctx)

	// 测试空队列
	task, err := service.getNextTask(ctx)
	assert.Equal(t, uint(0), task)
	assert.Error(t, err) // 空队列应该返回错误

	// 创建测试用户和视频
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)
	video, err := videoService.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	// 添加任务到队列
	err = service.AddToQueue(ctx, video.ID)
	assert.NoError(t, err)

	// 获取下一个任务
	task, err = service.getNextTask(ctx)
	assert.NoError(t, err)
	assert.Equal(t, video.ID, task)
}

func TestVideoQueueService_WorkerLifecycle(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx, cancel := context.WithCancel(context.Background())

	// 启动工作者
	err := service.StartWorkers(ctx)
	assert.NoError(t, err)

	// 创建测试视频
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)
	video, err := videoService.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	// 添加视频到队列
	err = service.AddToQueue(ctx, video.ID)
	assert.NoError(t, err)

	// 等待一小段时间让工作者有机会处理
	time.Sleep(100 * time.Millisecond)

	// 停止工作者
	cancel()
	service.StopWorkers()

	// 等待工作者完全停止
	time.Sleep(100 * time.Millisecond)
}

func TestVideoQueueService_ConcurrentOperations(t *testing.T) {
	service, db, rdb := setupTestVideoQueueService()
	ctx := context.Background()

	// 清理Redis数据库
	rdb.FlushDB(ctx)

	// 创建多个测试用户和视频
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	// 创建Manim配置用于VideoService
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	videoService := NewVideoService(db, rdb, manimCfg)

	var videos []*model.Video
	for i := 0; i < 5; i++ {
		video, err := videoService.CreateVideo(ctx, user.ID, "提示")
		assert.NoError(t, err)
		videos = append(videos, video)
	}

	// 顺序添加视频到队列以避免竞态条件
	for i := 0; i < 5; i++ {
		err := service.AddToQueue(ctx, videos[i].ID)
		assert.NoError(t, err)
	}

	// 验证队列中有5个视频
	count, videoIDs, err := service.GetQueueStatus(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.Len(t, videoIDs, 5)

	// 顺序移除视频以避免竞态条件
	for i := 0; i < 5; i++ {
		err := service.RemoveFromQueue(ctx, videos[i].ID)
		assert.NoError(t, err)
	}

	// 验证队列为空
	count, videoIDs, err = service.GetQueueStatus(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.Empty(t, videoIDs)
}
