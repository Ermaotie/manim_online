package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"manim-backend/internal/model"
	"manim-backend/internal/service"

	"github.com/golang-jwt/jwt/v4"
)

type AuthMiddleware struct {
	userService *service.UserService
}

func NewAuthMiddleware(userService *service.UserService) *AuthMiddleware {
	return &AuthMiddleware{
		userService: userService,
	}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, "未提供认证令牌", http.StatusUnauthorized)
			return
		}

		userID, err := m.validateToken(token)
		if err != nil {
			http.Error(w, "无效的认证令牌", http.StatusUnauthorized)
			return
		}

		user, err := m.userService.GetUserByID(r.Context(), userID)
		if err != nil {
			http.Error(w, "用户不存在", http.StatusUnauthorized)
			return
		}

		// 将用户信息添加到上下文中
		ctx := context.WithValue(r.Context(), "userID", user.ID)
		ctx = context.WithValue(ctx, "user", user)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

func extractToken(r *http.Request) string {
	// 从Authorization头中提取token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

func (m *AuthMiddleware) validateToken(tokenString string) (uint, error) {
	// 这里使用简单的JWT验证，实际项目中应该使用更安全的方案
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("manim-secret-key"), nil
	})

	if err != nil || !token.Valid {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, jwt.ErrInvalidKey
	}

	userID, ok := claims["userID"].(float64)
	if !ok {
		return 0, jwt.ErrInvalidKey
	}

	return uint(userID), nil
}

// GenerateToken 生成JWT令牌
func (m *AuthMiddleware) GenerateToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
		"exp":    jwt.NewNumericDate(jwt.TimeFunc().Add(24 * time.Hour)),
	})

	return token.SignedString([]byte("manim-secret-key"))
}

// GetUserFromContext 从上下文中获取用户信息
func GetUserFromContext(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value("user").(*model.User)
	return user, ok
}

// GetUserIDFromContext 从上下文中获取用户ID
func GetUserIDFromContext(ctx context.Context) (uint, bool) {
	userID, ok := ctx.Value("userID").(uint)
	return userID, ok
}
