// Package middleware 提供 Gin 中间件：JWT 认证、访问日志与错误兜底。
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"blog-server/internal/auth"
	"blog-server/internal/response"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserIDKey   = "userID"
	ctxUsernameKey = "username"
)

// Auth 校验 Authorization: Bearer <token> 头，并把用户信息写入上下文。
func Auth(mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.FromError(c, auth.ErrMissingToken)
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			response.FromError(c, auth.ErrInvalidToken)
			return
		}

		claims, err := mgr.ParseToken(strings.TrimSpace(parts[1]))
		if err != nil {
			response.FromError(c, err)
			return
		}

		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxUsernameKey, claims.Username)
		c.Next()
	}
}

// CurrentUserID 从上下文取出已认证用户 ID。
func CurrentUserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}

// Logger 记录每个请求的方法、路径、状态码与耗时。
func Logger(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
			"ip", c.ClientIP(),
		}
		if query != "" {
			attrs = append(attrs, "query", query)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case c.Writer.Status() >= http.StatusInternalServerError:
			l.Error("request", attrs...)
		case c.Writer.Status() >= http.StatusBadRequest:
			l.Warn("request", attrs...)
		default:
			l.Info("request", attrs...)
		}
	}
}

// Recovery 捕获 handler panic，记录堆栈并返回统一的 500 响应。
//
// gin 内置的 CustomRecoveryWithWriter 在 writer 为 nil 时不会输出任何堆栈，
// 因此这里在回调内自行采集：回调运行于 panic 所在 goroutine 的 defer 中，
// 栈帧尚未展开，debug.Stack 能拿到完整的 panic 调用链。
// 堆栈只进日志，绝不写入响应体。
func Recovery(l *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		l.Error("panic recovered",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"error", recovered,
			"stack", string(debug.Stack()),
		)
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
	})
}
