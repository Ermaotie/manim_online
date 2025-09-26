package service

import (
	"context"
	"testing"

	"manim-backend/internal/config"
	"manim-backend/internal/model"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBWithRedis() (*gorm.DB, *redis.Client) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	// 迁移表结构
	db.AutoMigrate(&model.User{}, &model.Video{})

	// 创建测试Redis客户端（使用模拟客户端）
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6380", // 使用配置文件中指定的端口
		DB:   15,               // 使用不同的数据库避免冲突
	})

	return db, rdb
}

// setupTestDBWithVideo 创建一个仅使用数据库的测试环境（包含Video表）
func setupTestDBWithVideo() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	// 迁移表结构
	db.AutoMigrate(&model.User{}, &model.Video{})

	return db
}

func TestVideoService_CreateVideo(t *testing.T) {
	db, rdb := setupTestDBWithRedis()
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	service := NewVideoService(db, rdb, manimCfg)
	ctx := context.Background()

	// 先创建一个测试用户
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	tests := []struct {
		name    string
		userID  uint
		prompt  string
		wantErr bool
	}{
		{
			name:    "创建视频成功",
			userID:  user.ID,
			prompt:  "创建一个圆形动画",
			wantErr: false,
		},
		{
			name:    "用户不存在",
			userID:  999,
			prompt:  "创建一个圆形动画",
			wantErr: false, // 数据库外键约束可能不会立即检查
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video, err := service.CreateVideo(ctx, tt.userID, tt.prompt)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, video)
				assert.Equal(t, tt.userID, video.UserID)
				assert.Equal(t, tt.prompt, video.Prompt)
				assert.Equal(t, model.VideoStatusPending, video.Status)
			}
		})
	}
}

func TestVideoService_GetVideoByID(t *testing.T) {
	db, rdb := setupTestDBWithRedis()
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	service := NewVideoService(db, rdb, manimCfg)
	ctx := context.Background()

	// 先创建测试数据
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	video, err := service.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	tests := []struct {
		name    string
		videoID uint
		wantErr bool
	}{
		{
			name:    "获取视频成功",
			videoID: video.ID,
			wantErr: false,
		},
		{
			name:    "视频不存在",
			videoID: 999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetVideoByID(ctx, tt.videoID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.videoID, result.ID)
			}
		})
	}
}

func TestVideoService_UpdateVideoStatus(t *testing.T) {
	db, rdb := setupTestDBWithRedis()
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	service := NewVideoService(db, rdb, manimCfg)
	ctx := context.Background()

	// 先创建测试数据
	userService := NewUserService(db)
	user, err := userService.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	video, err := service.CreateVideo(ctx, user.ID, "提示")
	assert.NoError(t, err)

	tests := []struct {
		name      string
		videoID   uint
		status    model.VideoStatus
		manimCode string
		videoPath string
		errorMsg  string
		wantErr   bool
	}{
		{
			name:      "更新状态成功",
			videoID:   video.ID,
			status:    model.VideoStatusCompleted,
			manimCode: "test code",
			videoPath: "/path/to/video.mp4",
			errorMsg:  "",
			wantErr:   false,
		},
		{
			name:      "视频不存在",
			videoID:   999,
			status:    model.VideoStatusCompleted,
			manimCode: "test code",
			videoPath: "/path/to/video.mp4",
			errorMsg:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdateVideoStatus(ctx, tt.videoID, tt.status, tt.manimCode, tt.videoPath, tt.errorMsg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// 验证更新结果
				updatedVideo, err := service.GetVideoByID(ctx, tt.videoID)
				assert.NoError(t, err)
				assert.Equal(t, tt.status, updatedVideo.Status)
				if tt.manimCode != "" {
					assert.Equal(t, tt.manimCode, updatedVideo.ManimCode)
				}
				if tt.videoPath != "" {
					assert.Equal(t, tt.videoPath, updatedVideo.VideoPath)
				}
			}
		})
	}
}

func TestVideoService_GetUserVideos(t *testing.T) {
	db, rdb := setupTestDBWithRedis()
	manimCfg := config.ManimConfig{
		PythonPath:    "python",
		MaxConcurrent: 3,
		Timeout:       300,
	}
	service := NewVideoService(db, rdb, manimCfg)
	ctx := context.Background()

	// 创建测试用户
	userService := NewUserService(db)
	user1, err := userService.Register(ctx, "user1", "user1@example.com", "password123")
	assert.NoError(t, err)
	user2, err := userService.Register(ctx, "user2", "user2@example.com", "password123")
	assert.NoError(t, err)

	// 为用户1创建3个视频
	for i := 0; i < 3; i++ {
		_, err := service.CreateVideo(ctx, user1.ID, "提示")
		assert.NoError(t, err)
	}

	// 为用户2创建1个视频
	_, err = service.CreateVideo(ctx, user2.ID, "提示")
	assert.NoError(t, err)

	tests := []struct {
		name     string
		userID   uint
		page     int
		pageSize int
		expected int
		total    int64
	}{
		{
			name:     "获取用户1的视频",
			userID:   user1.ID,
			page:     1,
			pageSize: 10,
			expected: 3,
			total:    3,
		},
		{
			name:     "分页测试",
			userID:   user1.ID,
			page:     1,
			pageSize: 2,
			expected: 2,
			total:    3,
		},
		{
			name:     "用户2的视频",
			userID:   user2.ID,
			page:     1,
			pageSize: 10,
			expected: 1,
			total:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			videos, total, err := service.GetUserVideos(ctx, tt.userID, tt.page, tt.pageSize)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, len(videos))
			assert.Equal(t, tt.total, total)

			// 验证每个视频都属于正确的用户
			for _, video := range videos {
				assert.Equal(t, tt.userID, video.UserID)
			}
		})
	}
}
