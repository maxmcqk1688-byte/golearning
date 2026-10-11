// Package router 组装 Gin 引擎并注册全部路由。
package router

import (
	"log/slog"
	"net/http"
	"time"

	"blog-server/internal/auth"
	"blog-server/internal/handler"
	"blog-server/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// New 构建带中间件与业务路由的 Gin 引擎。
func New(log *slog.Logger, jwtMgr *auth.Manager, uh *handler.UserHandler, ph *handler.PostHandler, ch *handler.CommentHandler) *gin.Engine {
	r := gin.New()
	r.Use(
		middleware.Recovery(log),
		middleware.Logger(log),
		cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			MaxAge:           12 * time.Hour,
			AllowCredentials: false,
		}),
	)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		// 公开接口
		api.POST("/users/register", uh.Register)
		api.POST("/users/login", uh.Login)
		api.GET("/posts", ph.List)
		api.GET("/posts/:id", ph.Get)
		api.GET("/posts/:id/comments", ch.ListByPost)

		// 需认证接口
		authed := api.Group("", middleware.Auth(jwtMgr))
		{
			authed.GET("/users/me", uh.Me)
			authed.POST("/posts", ph.Create)
			authed.PUT("/posts/:id", ph.Update)
			authed.DELETE("/posts/:id", ph.Delete)
			authed.POST("/posts/:id/comments", ch.Create)
		}
	}

	return r
}
