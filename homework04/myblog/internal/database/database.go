// Package database 负责建立数据库连接并完成模型自动迁移。
package database

import (
	"fmt"
	stdlog "log"
	"os"

	"blog-server/internal/config"
	"blog-server/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open 根据配置打开数据库连接并自动迁移表结构。
func Open(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.DBDriver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DBDSN)
	case "mysql":
		dialector = mysql.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.DBDriver)
	}

	// 业务层已通过 slog 记录错误，这里只保留 GORM 自身的 error 级日志，
	// 并忽略 ErrRecordNotFound（404 属正常业务分支，不应刷 SQL 噪音）。
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.New(
			stdlog.New(os.Stdout, "\r\n", stdlog.LstdFlags),
			logger.Config{
				LogLevel:                  logger.Error,
				IgnoreRecordNotFoundError: true,
			},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Post{}, &model.Comment{}); err != nil {
		return nil, fmt.Errorf("执行数据库迁移失败: %w", err)
	}

	return db, nil
}
