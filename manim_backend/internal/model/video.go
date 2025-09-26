package model

import (
	"time"

	"gorm.io/gorm"
)

type VideoStatus int

const (
	VideoStatusPending VideoStatus = iota
	VideoStatusQueued
	VideoStatusProcessing
	VideoStatusCompleted
	VideoStatusFailed
)

type Video struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	Prompt      string         `gorm:"type:text;not null" json:"prompt"`
	ManimCode   string         `gorm:"type:longtext" json:"manim_code"`
	VideoPath   string         `gorm:"size:500" json:"video_path"`
	Status      VideoStatus    `gorm:"default:0" json:"status"`
	ErrorMsg    string         `gorm:"type:text" json:"error_msg"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Video) TableName() string {
	return "videos"
}

func (s VideoStatus) String() string {
	switch s {
	case VideoStatusPending:
		return "pending"
	case VideoStatusQueued:
		return "queued"
	case VideoStatusProcessing:
		return "processing"
	case VideoStatusCompleted:
		return "completed"
	case VideoStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}
