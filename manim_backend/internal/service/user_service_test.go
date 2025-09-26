package service

import (
	"context"
	"testing"

	"manim-backend/internal/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	// 迁移表结构
	db.AutoMigrate(&model.User{})

	return db
}

func TestUserService_Register(t *testing.T) {
	db := setupTestDB()
	service := NewUserService(db)
	ctx := context.Background()

	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
	}{
		{
			name:     "注册成功",
			username: "testuser",
			email:    "test@example.com",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "用户名已存在",
			username: "testuser",
			email:    "test2@example.com",
			password: "password123",
			wantErr:  true,
		},
		{
			name:     "邮箱已存在",
			username: "testuser2",
			email:    "test@example.com",
			password: "password123",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Register(ctx, tt.username, tt.email, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
				assert.Equal(t, tt.email, user.Email)
				assert.NotEqual(t, tt.password, user.Password) // 密码应该被加密
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	db := setupTestDB()
	service := NewUserService(db)
	ctx := context.Background()

	// 先注册一个用户
	_, err := service.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{
			name:     "登录成功",
			username: "testuser",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "用户不存在",
			username: "nonexistent",
			password: "password123",
			wantErr:  true,
		},
		{
			name:     "密码错误",
			username: "testuser",
			password: "wrongpassword",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Login(ctx, tt.username, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
			}
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	db := setupTestDB()
	service := NewUserService(db)
	ctx := context.Background()

	// 先注册一个用户
	registeredUser, err := service.Register(ctx, "testuser", "test@example.com", "password123")
	assert.NoError(t, err)

	tests := []struct {
		name    string
		userID  uint
		wantErr bool
	}{
		{
			name:    "获取用户成功",
			userID:  registeredUser.ID,
			wantErr: false,
		},
		{
			name:    "用户不存在",
			userID:  999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.GetUserByID(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.userID, user.ID)
			}
		})
	}
}
