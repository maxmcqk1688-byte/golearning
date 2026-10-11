// Package config 负责加载服务运行所需的环境配置。
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 聚合了服务的全部可配置项，均支持通过环境变量覆盖默认值。
type Config struct {
	// ServerPort HTTP 监听端口，例如 ":8080"。
	ServerPort string
	// DBDriver 数据库驱动，支持 sqlite 与 mysql。
	DBDriver string
	// DBDSN 数据库连接串。sqlite 为文件路径，mysql 为 DSN。
	DBDSN string
	// JWTSecret 签发与校验 JWT 的密钥，生产环境务必替换。
	JWTSecret string
	// JWTExpire JWT 有效期。
	JWTExpire time.Duration
	// LogLevel 日志级别：debug / info / warn / error。
	LogLevel string
	// GinMode Gin 运行模式：debug / release / test。
	GinMode string
}

// Load 从环境变量读取配置，未设置时使用适合本地开发的默认值。
func Load() (*Config, error) {
	cfg := &Config{
		ServerPort: getenv("SERVER_PORT", ":8080"),
		DBDriver:   getenv("DB_DRIVER", "sqlite"),
		DBDSN:      getenv("DB_DSN", "blog.db"),
		JWTSecret:  getenv("JWT_SECRET", ""),
		JWTExpire:  getDuration("JWT_EXPIRE", 24*time.Hour),
		LogLevel:   getenv("LOG_LEVEL", "info"),
		GinMode:    getenv("GIN_MODE", "debug"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("配置项 JWT_SECRET 不能为空，请通过环境变量设置密钥")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	// 兼容只填数字（按小时）的写法。
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if h, err := strconv.Atoi(v); err == nil {
			return time.Duration(h) * time.Hour
		}
	}
	return fallback
}
