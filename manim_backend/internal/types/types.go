package types

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type CreateVideoRequest struct {
	Prompt string `json:"prompt"`
	Code   string `json:"code,omitempty"`
}

type VideoResponse struct {
	ID        uint   `json:"id"`
	Prompt    string `json:"prompt"`
	ManimCode string `json:"manim_code,omitempty"`
	VideoPath string `json:"video_path,omitempty"`
	Status    string `json:"status"`
	ErrorMsg  string `json:"error_msg,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type VideoListResponse struct {
	Videos   []VideoResponse `json:"videos"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

type GenerateCodeRequest struct {
	Prompt string `json:"prompt"`
}

type GenerateCodeResponse struct {
	Code    string `json:"code"`
	IsValid bool   `json:"is_valid"`
	Message string `json:"message,omitempty"`
}

type QueueStatusResponse struct {
	Current int `json:"current"`
	Max     int `json:"max"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type APIHealthResponse struct {
	Status    string   `json:"status"`
	Message   string   `json:"message"`
	Details   []string `json:"details,omitempty"`
	IsHealthy bool     `json:"is_healthy"`
}
