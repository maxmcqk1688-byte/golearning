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

// PostHandler 处理文章 CRUD。
type PostHandler struct {
	db  *gorm.DB
	log *slog.Logger
}

// NewPostHandler 创建 PostHandler。
func NewPostHandler(db *gorm.DB, log *slog.Logger) *PostHandler {
	return &PostHandler{db: db, log: log}
}

type postRequest struct {
	Title   string `json:"title" binding:"required,max=200"`
	Content string `json:"content" binding:"required"`
}

// Create 创建文章（需认证）。
func (h *PostHandler) Create(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		response.FromError(c, auth.ErrInvalidToken)
		return
	}

	var req postRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数错误: "+err.Error())
		return
	}

	post := model.Post{Title: req.Title, Content: req.Content, UserID: uid}
	if err := h.db.Create(&post).Error; err != nil {
		h.log.Error("创建文章失败", "user_id", uid, "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	h.db.Preload("User").First(&post, post.ID)
	h.log.Info("文章创建成功", "post_id", post.ID, "user_id", uid)
	response.Created(c, post)
}

// List 分页获取文章列表，支持按标题/内容关键字搜索。
func (h *PostHandler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)

	query := h.db.Model(&model.Post{})
	if kw := c.Query("keyword"); kw != "" {
		like := "%" + kw + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	if author := c.Query("author"); author != "" {
		query = query.Where("user_id IN (?)", h.db.Model(&model.User{}).Select("ID").Where("username = ?", author))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.log.Error("统计文章失败", "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	var posts []model.Post
	if err := query.Preload("User").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		h.log.Error("查询文章列表失败", "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	response.OK(c, gin.H{"items": posts, "total": total, "page": page, "page_size": pageSize})
}

// Get 获取单篇文章详情（公开），附带评论列表。
func (h *PostHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "无效的文章 ID")
		return
	}

	var post model.Post
	if err := h.db.Preload("User").Preload("Comments.User").First(&post, id).Error; err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, post)
}

// Update 更新文章，仅作者本人可操作。
func (h *PostHandler) Update(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		response.FromError(c, auth.ErrInvalidToken)
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "无效的文章 ID")
		return
	}

	var post model.Post
	if err := h.db.First(&post, id).Error; err != nil {
		response.FromError(c, err)
		return
	}
	if post.UserID != uid {
		response.FromError(c, auth.ErrNotAuthor)
		return
	}

	var req postRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.db.Model(&post).Updates(model.Post{Title: req.Title, Content: req.Content}).Error; err != nil {
		h.log.Error("更新文章失败", "post_id", id, "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	h.db.Preload("User").First(&post, post.ID)
	h.log.Info("文章更新成功", "post_id", post.ID, "user_id", uid)
	response.OK(c, post)
}

// Delete 删除文章（软删除），仅作者本人可操作。
func (h *PostHandler) Delete(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		response.FromError(c, auth.ErrInvalidToken)
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "无效的文章 ID")
		return
	}

	var post model.Post
	if err := h.db.First(&post, id).Error; err != nil {
		response.FromError(c, err)
		return
	}
	if post.UserID != uid {
		response.FromError(c, auth.ErrNotAuthor)
		return
	}

	if err := h.db.Delete(&post).Error; err != nil {
		h.log.Error("删除文章失败", "post_id", id, "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	h.log.Info("文章删除成功", "post_id", id, "user_id", uid)
	response.OK(c, gin.H{"message": "文章已删除"})
}
