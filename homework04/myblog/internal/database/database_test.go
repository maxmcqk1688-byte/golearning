package database

import (
	"fmt"
	"sync/atomic"
	"testing"

	"blog-server/internal/config"
	"blog-server/internal/model"
)

var counter int64

func TestOpenMigratesTables(t *testing.T) {
	n := atomic.AddInt64(&counter, 1)
	db, err := Open(&config.Config{
		DBDriver: "sqlite",
		DBDSN:    fmt.Sprintf("file:dbtest%d?mode=memory&cache=shared", n),
	})
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}

	if !db.Migrator().HasTable(&model.User{}) {
		t.Error("users 表未创建")
	}
	if !db.Migrator().HasTable(&model.Post{}) {
		t.Error("posts 表未创建")
	}
	if !db.Migrator().HasTable(&model.Comment{}) {
		t.Error("comments 表未创建")
	}
	if !db.Migrator().HasIndex(&model.User{}, "idx_users_username") {
		t.Error("users.username 唯一索引未创建")
	}
}

func TestOpenUnsupportedDriver(t *testing.T) {
	if _, err := Open(&config.Config{DBDriver: "postgres", DBDSN: "..."}); err == nil {
		t.Error("不支持的驱动应返回错误")
	}
}

func TestOpenInvalidDSN(t *testing.T) {
	// 指向不存在目录的 sqlite 路径，Open 应返回连接错误而非 panic。
	if _, err := Open(&config.Config{DBDriver: "sqlite", DBDSN: "/nonexistent-dir-xyz/blog.db"}); err == nil {
		t.Error("非法 DSN 应返回错误")
	}
}
