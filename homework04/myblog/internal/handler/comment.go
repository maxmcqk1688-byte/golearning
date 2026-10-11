package handler

import (
	"log/slog"
	"net/http"

	"blog-server/internal/auth"
	"blog-server/internal/model"
	"blog-server/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CommentHandler 处理文章评论。
type CommentHandler struct {
	db  *gorm.DB
	log *slog.Logger
}

// NewCommentHandler 创建 CommentHandler。
func NewCommentHandler(db *gorm.DB, log *slog.Logger) *CommentHandler {
	return &CommentHandler{db: db, log: log}
}

type commentRequest struct {
	Content string `json:"content" binding:"required,max=1000"`
}

// Create 对指定文章发表评论（需认证）。
func (h *CommentHandler) Create(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		response.FromError(c, auth.ErrInvalidToken)
		return
	}
	postID, ok := parseIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "无效的文章 ID")
		return
	}

	var exists int64
	if err := h.db.Model(&model.Post{}).Where("id = ?", postID).Count(&exists).Error; err != nil {
		h.log.Error("校验文章失败", "post_id", postID, "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	if exists == 0 {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "文章不存在")
		return
	}

	var req commentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数错误: "+err.Error())
		return
	}

	comment := model.Comment{Content: req.Content, UserID: uid, PostID: postID}
	if err := h.db.Create(&comment).Error; err != nil {
		h.log.Error("创建评论失败", "post_id", postID, "user_id", uid, "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	h.db.Preload("User").First(&comment, comment.ID)
	h.log.Info("评论创建成功", "comment_id", comment.ID, "post_id", postID, "user_id", uid)
	response.Created(c, comment)
}

// ListByPost 获取某篇文章的全部评论（公开）。
func (h *CommentHandler) ListByPost(c *gin.Context) {
	postID, ok := parseIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "无效的文章 ID")
		return
	}

	var comments []model.Comment
	if err := h.db.Preload("User").
		Where("post_id = ?", postID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		h.log.Error("查询评论失败", "post_id", postID, "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	response.OK(c, gin.H{"items": comments, "total": len(comments)})
}
