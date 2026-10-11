// 接口集成测试：内存 SQLite + httptest，直接驱动完整路由，无需启动服务。
package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"blog-server/internal/auth"
	"blog-server/internal/config"
	"blog-server/internal/database"
	"blog-server/internal/handler"
	"blog-server/internal/router"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var dbCounter int64

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// newTestEnv 为每个用例构建独立的内存库与完整路由。
func newTestEnv(t *testing.T) (http.Handler, *gorm.DB) {
	t.Helper()
	n := atomic.AddInt64(&dbCounter, 1)
	dsn := fmt.Sprintf("file:testdb%d?mode=memory&cache=shared", n)

	db, err := database.Open(&config.Config{DBDriver: "sqlite", DBDSN: dsn})
	if err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwtMgr := auth.NewManager("integration-test-secret", time.Hour)
	engine := router.New(log, jwtMgr,
		handler.NewUserHandler(db, jwtMgr, log),
		handler.NewPostHandler(db, log),
		handler.NewCommentHandler(db, log),
	)
	return engine, db
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// do 发送请求并解析统一响应信封。
func do(t *testing.T, h http.Handler, method, path, token string, body any) (int, envelope, string) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		rdr = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var env envelope
	raw := w.Body.String()
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应不是合法 JSON (%s %s): %s", method, path, raw)
	}
	return w.Code, env, raw
}

func mustCode(t *testing.T, desc string, got, want int, raw string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: 期望状态码 %d, 实际 %d, 响应: %s", desc, want, got, raw)
	}
}

func registerUser(t *testing.T, h http.Handler, username, password, email string) int {
	code, _, raw := do(t, h, http.MethodPost, "/api/v1/users/register", "", map[string]string{
		"username": username, "password": password, "email": email,
	})
	mustCode(t, "注册 "+username, code, http.StatusCreated, raw)
	return code
}

func loginUser(t *testing.T, h http.Handler, username, password string) string {
	code, env, raw := do(t, h, http.MethodPost, "/api/v1/users/login", "", map[string]string{
		"username": username, "password": password,
	})
	mustCode(t, "登录 "+username, code, http.StatusOK, raw)

	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("解析登录响应失败: %v (%s)", err, raw)
	}
	if data.Token == "" {
		t.Fatalf("登录响应缺少 token: %s", raw)
	}
	return data.Token
}

// createUser 注册并登录一个用户，返回令牌。
func createUser(t *testing.T, h http.Handler, username, password, email string) string {
	registerUser(t, h, username, password, email)
	return loginUser(t, h, username, password)
}

func TestHealthz(t *testing.T) {
	h, _ := newTestEnv(t)
	code, _, raw := do(t, h, http.MethodGet, "/healthz", "", nil)
	mustCode(t, "健康检查", code, http.StatusOK, raw)
}

func TestRegisterAndLogin(t *testing.T) {
	h, _ := newTestEnv(t)

	// 注册成功，且响应不泄漏密码字段。
	_, _, raw := do(t, h, http.MethodPost, "/api/v1/users/register", "", map[string]string{
		"username": "alice", "password": "Passw0rd!", "email": "alice@example.com",
	})
	if strings.Contains(raw, "password") {
		t.Errorf("注册响应不应包含 password 字段: %s", raw)
	}

	// 用户名重复 → 409；邮箱重复 → 409。
	code, _, raw := do(t, h, http.MethodPost, "/api/v1/users/register", "", map[string]string{
		"username": "alice", "password": "Passw0rd!", "email": "other@example.com",
	})
	mustCode(t, "重复用户名", code, http.StatusConflict, raw)

	code, _, raw = do(t, h, http.MethodPost, "/api/v1/users/register", "", map[string]string{
		"username": "another", "password": "Passw0rd!", "email": "alice@example.com",
	})
	mustCode(t, "重复邮箱", code, http.StatusConflict, raw)

	// 参数校验：密码过短 / 邮箱非法 / 缺少字段 → 400。
	for _, bad := range []map[string]string{
		{"username": "shorty", "password": "123", "email": "shorty@example.com"},
		{"username": "validname", "password": "Passw0rd!", "email": "not-an-email"},
		{"username": "validname", "password": "Passw0rd!"},
	} {
		code, _, raw = do(t, h, http.MethodPost, "/api/v1/users/register", "", bad)
		mustCode(t, fmt.Sprintf("非法注册 %v", bad), code, http.StatusBadRequest, raw)
	}

	// 登录：成功 / 密码错误 / 用户不存在。
	registerUser(t, h, "bob", "Passw0rd2!", "bob@example.com")
	if token := loginUser(t, h, "alice", "Passw0rd!"); token == "" {
		t.Fatal("alice 登录应返回令牌")
	}

	code, _, raw = do(t, h, http.MethodPost, "/api/v1/users/login", "", map[string]string{
		"username": "alice", "password": "wrong-password",
	})
	mustCode(t, "密码错误", code, http.StatusUnauthorized, raw)

	code, _, raw = do(t, h, http.MethodPost, "/api/v1/users/login", "", map[string]string{
		"username": "nobody", "password": "whatever123",
	})
	mustCode(t, "用户不存在", code, http.StatusUnauthorized, raw)
}

func TestAuthMiddlewareAndMe(t *testing.T) {
	h, _ := newTestEnv(t)
	token := createUser(t, h, "alice", "Passw0rd!", "alice@example.com")

	// 无令牌 → 401 缺少认证令牌。
	code, env, raw := do(t, h, http.MethodGet, "/api/v1/users/me", "", nil)
	mustCode(t, "无令牌访问 me", code, http.StatusUnauthorized, raw)
	if env.Code != 40100 {
		t.Errorf("业务码应为 40100, 实际 %d", env.Code)
	}

	// 格式错误的 Authorization 头 → 401。
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Basic abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	mustCode(t, "Basic 认证头", w.Code, http.StatusUnauthorized, w.Body.String())

	// 伪造令牌 → 401。
	code, _, raw = do(t, h, http.MethodGet, "/api/v1/users/me", "not.a.valid.token", nil)
	mustCode(t, "伪造令牌", code, http.StatusUnauthorized, raw)

	// 有效令牌 → 200，返回本人信息且不含密码。
	code, env, raw = do(t, h, http.MethodGet, "/api/v1/users/me", token, nil)
	mustCode(t, "me 带令牌", code, http.StatusOK, raw)
	var user struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.Unmarshal(env.Data, &user); err != nil {
		t.Fatalf("解析 me 响应失败: %v (%s)", err, raw)
	}
	if user.Username != "alice" || user.Email != "alice@example.com" {
		t.Errorf("me 返回数据异常: %+v", user)
	}
	if strings.Contains(raw, "password") {
		t.Errorf("me 响应不应包含 password 字段: %s", raw)
	}
}

func TestPostCRUD(t *testing.T) {
	h, _ := newTestEnv(t)
	alice := createUser(t, h, "alice", "Passw0rd!", "alice@example.com")

	// 未认证创建 → 401；参数缺失 → 400。
	code, _, raw := do(t, h, http.MethodPost, "/api/v1/posts", "", map[string]string{
		"title": "x", "content": "y",
	})
	mustCode(t, "未认证创建文章", code, http.StatusUnauthorized, raw)

	code, _, raw = do(t, h, http.MethodPost, "/api/v1/posts", alice, map[string]string{"title": ""})
	mustCode(t, "标题为空", code, http.StatusBadRequest, raw)

	// 创建两篇文章。
	code, env, raw := do(t, h, http.MethodPost, "/api/v1/posts", alice, map[string]string{
		"title": "Go 入门", "content": "Gin 与 GORM 的实践。",
	})
	mustCode(t, "创建文章 1", code, http.StatusCreated, raw)
	var p1 struct {
		ID     uint `json:"id"`
		UserID uint `json:"user_id"`
	}
	if err := json.Unmarshal(env.Data, &p1); err != nil {
		t.Fatalf("解析创建响应失败: %v (%s)", err, raw)
	}
	if p1.ID == 0 || p1.UserID == 0 {
		t.Fatalf("创建响应缺少 id/user_id: %s", raw)
	}

	do(t, h, http.MethodPost, "/api/v1/posts", alice, map[string]string{
		"title": "数据库笔记", "content": "SQLite 使用心得。",
	})

	// 列表：分页元数据正确。
	code, env, raw = do(t, h, http.MethodGet, "/api/v1/posts?page=1&page_size=1", "", nil)
	mustCode(t, "文章列表", code, http.StatusOK, raw)
	var list struct {
		Items    []json.RawMessage `json:"items"`
		Total    int64             `json:"total"`
		Page     int               `json:"page"`
		PageSize int               `json:"page_size"`
	}
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("解析列表响应失败: %v (%s)", err, raw)
	}
	if list.Total != 2 || len(list.Items) != 1 || list.Page != 1 || list.PageSize != 1 {
		t.Errorf("分页元数据异常: total=%d items=%d page=%d size=%d", list.Total, len(list.Items), list.Page, list.PageSize)
	}

	// page_size 超上限时收敛为默认值 10。
	_, env, raw = do(t, h, http.MethodGet, "/api/v1/posts?page_size=999", "", nil)
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("解析列表响应失败: %v (%s)", err, raw)
	}
	if list.PageSize != 10 {
		t.Errorf("page_size 超上限应回退为 10, 实际 %d (%s)", list.PageSize, raw)
	}

	// 关键字与作者过滤。
	_, env, raw = do(t, h, http.MethodGet, "/api/v1/posts?keyword=数据库", "", nil)
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("解析搜索结果失败: %v (%s)", err, raw)
	}
	if list.Total != 1 || len(list.Items) != 1 {
		t.Errorf("关键字搜索应命中 1 篇, 实际 total=%d (%s)", list.Total, raw)
	}

	_, env, raw = do(t, h, http.MethodGet, "/api/v1/posts?author=alice", "", nil)
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("解析作者过滤结果失败: %v (%s)", err, raw)
	}
	if list.Total != 2 {
		t.Errorf("author=alice 应有 2 篇, 实际 %d (%s)", list.Total, raw)
	}

	// 详情：预加载作者。
	code, env, raw = do(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", p1.ID), "", nil)
	mustCode(t, "文章详情", code, http.StatusOK, raw)
	var detail struct {
		Title  string `json:"title"`
		Author struct {
			Username string `json:"username"`
		} `json:"author"`
	}
	if err := json.Unmarshal(env.Data, &detail); err != nil {
		t.Fatalf("解析详情失败: %v (%s)", err, raw)
	}
	if detail.Title != "Go 入门" || detail.Author.Username != "alice" {
		t.Errorf("详情数据异常: %+v", detail)
	}

	// 不存在 → 404；ID 非法 → 400。
	code, _, raw = do(t, h, http.MethodGet, "/api/v1/posts/999999", "", nil)
	mustCode(t, "不存在的文章", code, http.StatusNotFound, raw)
	code, _, raw = do(t, h, http.MethodGet, "/api/v1/posts/abc", "", nil)
	mustCode(t, "非法文章 ID", code, http.StatusBadRequest, raw)

	// 作者更新 → 200，内容确实变化。
	code, _, raw = do(t, h, http.MethodPut, fmt.Sprintf("/api/v1/posts/%d", p1.ID), alice, map[string]string{
		"title": "Go 入门（第二版）", "content": "修订后的正文。",
	})
	mustCode(t, "作者更新", code, http.StatusOK, raw)

	_, env, raw = do(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", p1.ID), "", nil)
	if err := json.Unmarshal(env.Data, &detail); err != nil {
		t.Fatalf("解析更新后详情失败: %v (%s)", err, raw)
	}
	if detail.Title != "Go 入门（第二版）" {
		t.Errorf("更新未生效: %+v (%s)", detail, raw)
	}

	// 作者删除 → 200，再查 → 404（软删除后不可见）。
	code, _, raw = do(t, h, http.MethodDelete, fmt.Sprintf("/api/v1/posts/%d", p1.ID), alice, nil)
	mustCode(t, "作者删除", code, http.StatusOK, raw)
	code, _, raw = do(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", p1.ID), "", nil)
	mustCode(t, "删除后详情", code, http.StatusNotFound, raw)
}

func TestPostAuthorization(t *testing.T) {
	h, _ := newTestEnv(t)
	alice := createUser(t, h, "alice", "Passw0rd!", "alice@example.com")
	bob := createUser(t, h, "bob", "Passw0rd2!", "bob@example.com")

	code, env, raw := do(t, h, http.MethodPost, "/api/v1/posts", alice, map[string]string{
		"title": "Alice 的文章", "content": "只有作者能改。",
	})
	mustCode(t, "创建文章", code, http.StatusCreated, raw)
	var created struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &created); err != nil {
		t.Fatalf("解析创建响应失败: %v (%s)", err, raw)
	}

	// 非作者更新/删除 → 403，业务码 40300。
	code, env, raw = do(t, h, http.MethodPut, fmt.Sprintf("/api/v1/posts/%d", created.ID), bob, map[string]string{
		"title": "劫持", "content": "劫持",
	})
	mustCode(t, "非作者更新", code, http.StatusForbidden, raw)
	if env.Code != 40300 {
		t.Errorf("越权业务码应为 40300, 实际 %d", env.Code)
	}

	code, _, raw = do(t, h, http.MethodDelete, fmt.Sprintf("/api/v1/posts/%d", created.ID), bob, nil)
	mustCode(t, "非作者删除", code, http.StatusForbidden, raw)

	// 越权操作后原文未被篡改。
	_, env, raw = do(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", created.ID), "", nil)
	var detail struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(env.Data, &detail); err != nil {
		t.Fatalf("解析详情失败: %v (%s)", err, raw)
	}
	if detail.Title != "Alice 的文章" {
		t.Errorf("越权请求不应生效, 标题变成了 %q", detail.Title)
	}
}

func TestCommentFlow(t *testing.T) {
	h, _ := newTestEnv(t)
	alice := createUser(t, h, "alice", "Passw0rd!", "alice@example.com")
	bob := createUser(t, h, "bob", "Passw0rd2!", "bob@example.com")

	_, env, raw := do(t, h, http.MethodPost, "/api/v1/posts", alice, map[string]string{
		"title": "求评论", "content": "正文。",
	})
	var post struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &post); err != nil {
		t.Fatalf("解析文章失败: %v (%s)", err, raw)
	}

	// 认证用户评论 → 201。
	code, _, raw := do(t, h, http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), bob, map[string]string{
		"content": "写得不错！",
	})
	mustCode(t, "发表评论", code, http.StatusCreated, raw)

	// 未认证 → 401；空内容 → 400；文章不存在 → 404。
	code, _, raw = do(t, h, http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), "", map[string]string{
		"content": "hi",
	})
	mustCode(t, "未认证评论", code, http.StatusUnauthorized, raw)

	code, _, raw = do(t, h, http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), bob, map[string]string{
		"content": "",
	})
	mustCode(t, "空评论内容", code, http.StatusBadRequest, raw)

	code, _, raw = do(t, h, http.MethodPost, "/api/v1/posts/999999/comments", bob, map[string]string{
		"content": "hello",
	})
	mustCode(t, "评论不存在的文章", code, http.StatusNotFound, raw)

	// 评论列表（公开）→ 1 条，且带评论者信息。
	code, env, raw = do(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), "", nil)
	mustCode(t, "评论列表", code, http.StatusOK, raw)
	var list struct {
		Items []struct {
			Content string `json:"content"`
			User    struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("解析评论列表失败: %v (%s)", err, raw)
	}
	if list.Total != 1 || len(list.Items) != 1 {
		t.Fatalf("应有 1 条评论, 实际 %+v (%s)", list, raw)
	}
	if list.Items[0].Content != "写得不错！" || list.Items[0].User.Username != "bob" {
		t.Errorf("评论数据异常: %+v", list.Items[0])
	}

	// 文章详情应内嵌评论。
	_, env, raw = do(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", post.ID), "", nil)
	var detail struct {
		Comments []json.RawMessage `json:"comments"`
	}
	if err := json.Unmarshal(env.Data, &detail); err != nil {
		t.Fatalf("解析文章详情失败: %v (%s)", err, raw)
	}
	if len(detail.Comments) != 1 {
		t.Errorf("文章详情应内嵌 1 条评论, 实际 %d (%s)", len(detail.Comments), raw)
	}
}
