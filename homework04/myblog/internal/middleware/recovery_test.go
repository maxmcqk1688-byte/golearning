// Recovery 中间件的 panic 注入测试：验证兜底响应与日志记录。
package middleware_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/internal/middleware"
	"blog-server/internal/response"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// newEngine 构建挂载 Recovery（可选叠加访问日志）与 panic 路由的引擎。
// 第二个返回值为日志缓冲区，用于断言 panic 与访问记录是否落盘。
func newEngine(t *testing.T, withLogger bool, panicFn gin.HandlerFunc) (*gin.Engine, *bytes.Buffer) {
	t.Helper()

	buf := &bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(buf, nil))

	// Recovery 传入 nil writer，gin 自身不输出堆栈；
	// 堆栈由中间件回调内 debug.Stack 采集后写入下面的 slog。
	e := gin.New()
	if withLogger {
		e.Use(middleware.Logger(log))
	}
	e.Use(middleware.Recovery(log))
	e.GET("/boom", panicFn)

	return e, buf
}

func get(t *testing.T, e http.Handler, path string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

// TestRecoveryExplicitPanic 业务代码主动 panic 时应返回统一 500，且不泄漏内部信息。
func TestRecoveryExplicitPanic(t *testing.T) {
	sentinel := errors.New("database connection leaked secret")
	e, buf := newEngine(t, false, func(c *gin.Context) { panic(sentinel) })

	code, body := get(t, e, "/boom")
	if code != http.StatusInternalServerError {
		t.Fatalf("期望 500, 实际 %d: %s", code, body)
	}

	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("响应应为合法 JSON: %v (%s)", err, body)
	}
	if env.Code != response.CodeInternal || env.Message != "服务器内部错误" {
		t.Errorf("响应体异常: %+v", env)
	}
	if strings.Contains(body, "database connection leaked secret") {
		t.Errorf("响应不应包含 panic 原文: %s", body)
	}
	// 堆栈只进日志，不得出现在响应体中。
	for _, leak := range []string{"goroutine ", "runtime/debug.Stack", ".go:"} {
		if strings.Contains(body, leak) {
			t.Errorf("响应不应包含堆栈信息 %q: %s", leak, body)
		}
	}

	logged := buf.String()
	for _, want := range []string{"panic recovered", "/boom", "GET", "database connection leaked secret"} {
		if !strings.Contains(logged, want) {
			t.Errorf("日志应记录 %q, 实际: %s", want, logged)
		}
	}
	// 堆栈应包含 panic 触发点，便于线上定位（不断言行号，避免文件变动时脆断）。
	for _, want := range []string{"stack=", "TestRecoveryExplicitPanic.func1", "gin-gonic/gin"} {
		if !strings.Contains(logged, want) {
			t.Errorf("日志堆栈应包含 %q, 实际: %s", want, logged)
		}
	}
}

// TestRecoveryRuntimeError 运行时恐慌（空 map 写入）同样应被捕获。
func TestRecoveryRuntimeError(t *testing.T) {
	e, buf := newEngine(t, false, func(c *gin.Context) {
		var m map[string]string
		m["key"] = "value" // assignment to entry in nil map
	})

	code, body := get(t, e, "/boom")
	if code != http.StatusInternalServerError {
		t.Fatalf("期望 500, 实际 %d: %s", code, body)
	}
	if !strings.Contains(body, `"code":50000`) {
		t.Errorf("响应应为统一错误信封: %s", body)
	}
	if !strings.Contains(buf.String(), "nil map") {
		t.Errorf("日志应记录运行时错误: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "TestRecoveryRuntimeError") {
		t.Errorf("日志堆栈应指向 panic 触发点: %s", buf.String())
	}
}

// TestRecoveryWithLogger 组合访问日志时，panic 请求应先落 panic 明细，
// 再以 ERROR 级别记录一条 status=500 的访问日志。
func TestRecoveryWithLogger(t *testing.T) {
	e, buf := newEngine(t, true, func(c *gin.Context) { panic("boom with logger") })

	code, _ := get(t, e, "/boom")
	if code != http.StatusInternalServerError {
		t.Fatalf("期望 500, 实际 %d", code)
	}

	logged := buf.String()
	if !strings.Contains(logged, "panic recovered") {
		t.Errorf("应记录 panic 明细: %s", logged)
	}
	if !strings.Contains(logged, "level=ERROR") {
		t.Errorf("访问日志应以 ERROR 级别记录: %s", logged)
	}
	if !strings.Contains(logged, "status=500") {
		t.Errorf("访问日志应包含最终状态码 500: %s", logged)
	}
}
