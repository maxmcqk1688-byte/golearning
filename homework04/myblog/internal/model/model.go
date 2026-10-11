// Package model 定义博客系统的领域模型与请求/响应结构。
package model

import (
	"time"

	"gorm.io/gorm"
)

// User 博客用户。
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex;not null;size:64" json:"username" binding:"required,min=3,max=64,alphanum"`
	Password  string         `gorm:"not null;size:128" json:"-"` // 序列化时隐藏密码
	Email     string         `gorm:"uniqueIndex;not null;size:128" json:"email" binding:"required,email"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Post 博客文章。
type Post struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"not null;size:200" json:"title"`
	Content   string         `gorm:"not null;type:text" json:"content"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	User      User           `json:"author,omitempty"`
	Comments  []Comment      `json:"comments,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Comment 文章评论。
type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Content   string    `gorm:"not null;type:text" json:"content"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      User      `json:"user,omitempty"`
	PostID    uint      `gorm:"not null;index" json:"post_id"`
	Post      Post      `json:"post,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定评论表名，避免 GORM 默认复数化产生歧义。
func (Comment) TableName() string { return "comments" }
