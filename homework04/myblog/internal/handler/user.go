// Package handler 实现各业务接口，负责参数校验、调用领域逻辑并输出统一响应。
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"blog-server/internal/auth"
	"blog-server/internal/model"
	"blog-server/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserHandler 处理用户注册、登录与信息查询。
type UserHandler struct {
	db  *gorm.DB
	jwt *auth.Manager
	log *slog.Logger
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(db *gorm.DB, jwt *auth.Manager, log *slog.Logger) *UserHandler {
	return &UserHandler{db: db, jwt: jwt, log: log}
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Email    string `json:"email" binding:"required,email"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register 用户注册：密码经 bcrypt 加密后入库。
func (h *UserHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数错误: "+err.Error())
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		h.log.Error("密码加密失败", "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	user := model.User{
		Username: strings.TrimSpace(req.Username),
		Password: hashed,
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
	}
	if err := h.db.Create(&user).Error; err != nil {
		if isDuplicate(err) {
			response.Fail(c, http.StatusConflict, response.CodeConflict, "用户名或邮箱已被注册")
			return
		}
		h.log.Error("创建用户失败", "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	h.log.Info("用户注册成功", "user_id", user.ID, "username", user.Username)
	response.Created(c, user)
}

// Login 用户登录：校验密码后签发 JWT。
func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数错误: "+err.Error())
		return
	}

	var user model.User
	if err := h.db.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FromError(c, auth.ErrInvalidCredentials)
			return
		}
		h.log.Error("查询用户失败", "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	if !auth.VerifyPassword(user.Password, req.Password) {
		response.FromError(c, auth.ErrInvalidCredentials)
		return
	}

	token, err := h.jwt.GenerateToken(user.ID, user.Username)
	if err != nil {
		h.log.Error("签发令牌失败", "error", err)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}

	h.log.Info("用户登录成功", "user_id", user.ID, "username", user.Username)
	response.OK(c, gin.H{"token": token, "user": user})
}

// Me 返回当前登录用户信息。
func (h *UserHandler) Me(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		response.FromError(c, auth.ErrInvalidToken)
		return
	}
	var user model.User
	if err := h.db.First(&user, uid).Error; err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, user)
}

func isDuplicate(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "constraint failed")
}
