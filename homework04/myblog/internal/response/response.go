// Package response 提供统一的 HTTP 响应格式与错误码定义。
package response

import (
	"blog-server/internal/auth"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 业务错误码，与 HTTP 状态码解耦，便于客户端做细分处理。
const (
	CodeOK           = 0
	CodeBadRequest   = 40000
	CodeUnauthorized = 40100
	CodeForbidden    = 40300
	CodeNotFound     = 40400
	CodeConflict     = 40900
	CodeInternal     = 50000
)

// Body 是所有接口的统一响应结构。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 返回成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: CodeOK, Message: "success", Data: data})
}

// Created 返回创建成功响应。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{Code: CodeOK, Message: "created", Data: data})
}

// Fail 返回带业务码的错误响应。
func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, Body{Code: code, Message: msg})
}

// FromError 将底层错误（尤其是 GORM 与认证错误）映射为合适的 HTTP 响应。
func FromError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		Fail(c, http.StatusNotFound, CodeNotFound, "资源不存在")
	case errors.Is(err, auth.ErrInvalidCredentials):
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "用户名或密码错误")
	case errors.Is(err, auth.ErrMissingToken):
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "缺少认证令牌")
	case errors.Is(err, auth.ErrInvalidToken):
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "认证令牌无效或已过期")
	case errors.Is(err, auth.ErrNotAuthor):
		Fail(c, http.StatusForbidden, CodeForbidden, "无权操作该资源")
	default:
		Fail(c, http.StatusInternalServerError, CodeInternal, "服务器内部错误")
	}
}
