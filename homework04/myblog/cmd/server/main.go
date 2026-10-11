// blog-server 个人博客系统后端入口。
//
// 环境变量：
//
//	SERVER_PORT  监听端口，默认 :8080
//	DB_DRIVER    sqlite（默认，纯 Go 驱动无需 cgo）或 mysql
//	DB_DSN       数据库连接串，sqlite 默认为 blog.db
//	JWT_SECRET   JWT 签名密钥（必填）
//	JWT_EXPIRE   令牌有效期，默认 24h
//	LOG_LEVEL    日志级别 debug/info/warn/error，默认 info
//	GIN_MODE     Gin 运行模式，默认 debug
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"blog-server/internal/auth"
	"blog-server/internal/config"
	"blog-server/internal/database"
	"blog-server/internal/handler"
	"blog-server/internal/logger"
	"blog-server/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	gin.SetMode(cfg.GinMode)

	db, err := database.Open(cfg)
	if err != nil {
		log.Error("初始化数据库失败", "error", err)
		os.Exit(1)
	}
	log.Info("数据库连接成功", "driver", cfg.DBDriver)

	jwtMgr := auth.NewManager(cfg.JWTSecret, cfg.JWTExpire)
	uh := handler.NewUserHandler(db, jwtMgr, log)
	ph := handler.NewPostHandler(db, log)
	ch := handler.NewCommentHandler(db, log)

	engine := router.New(log, jwtMgr, uh, ph, ch)
	srv := &http.Server{
		Addr:              cfg.ServerPort,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("HTTP 服务启动", "addr", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP 服务异常退出", "error", err)
			os.Exit(1)
		}
	}()

	// 优雅停机：等待 SIGINT/SIGTERM 后最多 5 秒收尾。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("服务关闭失败", "error", err)
	}
	log.Info("服务已退出")
}
