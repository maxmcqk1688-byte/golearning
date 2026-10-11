# 个人博客系统后端（Go + Gin + GORM）

基于 Go 语言、Gin 框架与 GORM 实现的个人博客系统后端，提供用户认证（JWT）、文章 CRUD 与评论功能，默认使用 SQLite 存储，可一键切换 MySQL。

## 技术栈

| 组件 | 选型 | 说明 |
| --- | --- | --- |
| Web 框架 | `github.com/gin-gonic/gin` v1.12 | 路由与参数绑定校验 |
| ORM | `gorm.io/gorm` v1.31 | 模型定义、自动迁移、预加载 |
| 数据库驱动 | `github.com/glebarez/sqlite`（纯 Go，无需 cgo）/ `gorm.io/driver/mysql` | 通过 `DB_DRIVER` 切换 |
| 认证 | `github.com/golang-jwt/jwt/v5` + `golang.org/x/crypto/bcrypt` | HS256 签名，密码 bcrypt 加密 |
| 日志 | 标准库 `log/slog` | 结构化文本日志，含请求访问日志 |
| CORS | `github.com/gin-contrib/cors` | 便于前端联调 |

## 环境要求

- Go **1.22+**（开发环境为 go1.26.5）
- SQLite 场景无需任何外部服务；MySQL 场景需要可访问的 MySQL 8 实例
- 接口冒烟测试需要 `curl` 与 `jq`

## 快速开始

```bash
# 1. 安装依赖
go mod tidy

# 2. 设置 JWT 密钥（必填，未设置服务拒绝启动）
export JWT_SECRET="please-change-this-secret"

# 3. 启动（默认 SQLite，数据文件 blog.db 自动创建并迁移表结构）
go run ./cmd/server
# 或
make run
```

服务默认监听 `:8080`，启动后访问 `GET /healthz` 应返回 `{"status":"ok"}`。

### 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER_PORT` | `:8080` | HTTP 监听地址 |
| `DB_DRIVER` | `sqlite` | `sqlite` 或 `mysql` |
| `DB_DSN` | `blog.db` | sqlite 为文件路径；mysql 为 DSN |
| `JWT_SECRET` | 无（**必填**） | JWT 签名密钥 |
| `JWT_EXPIRE` | `24h` | 令牌有效期，支持 `30m`/`168h` 等写法 |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |
| `GIN_MODE` | `debug` | 生产环境建议 `release` |

### 使用 MySQL

```bash
export DB_DRIVER=mysql
export DB_DSN="user:pass@tcp(127.0.0.1:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"
export JWT_SECRET="please-change-this-secret"
go run ./cmd/server
```

表结构由 GORM `AutoMigrate` 自动创建，无需手工执行 SQL。

## 项目结构

```
.
├── cmd/server/main.go              # 入口：装配配置/日志/DB/路由，优雅停机
└── internal/
    ├── config/config.go            # 环境变量配置加载与校验
    ├── logger/logger.go            # slog 日志初始化
    ├── database/database.go        # 数据库连接与 AutoMigrate
    ├── model/model.go              # User / Post / Comment 模型
    ├── auth/auth.go                # bcrypt 哈希 + JWT 签发与校验
    ├── middleware/middleware.go     # Auth、访问日志、panic Recovery
    ├── response/response.go        # 统一响应结构与错误码映射
    ├── handler/                    # user.go / post.go / comment.go / helper.go
    └── router/router.go            # 路由注册与 CORS
```

## 数据库设计

**users**

| 字段 | 类型 | 约束 |
| --- | --- | --- |
| id | INTEGER | 主键 |
| username | VARCHAR(64) | 唯一索引，非空 |
| password | VARCHAR(128) | 非空，bcrypt 哈希，JSON 序列化时隐藏 |
| email | VARCHAR(128) | 唯一索引，非空 |
| created_at / updated_at / deleted_at | DATETIME | 时间戳与软删除标记 |

**posts**：`id`、`title`(非空)、`content`(text 非空)、`user_id`(索引，关联 users.id)、时间戳、软删除；与 `User` 一对多、与 `Comment` 一对多。

**comments**：`id`、`content`(text 非空)、`user_id`(索引)、`post_id`(索引)、`created_at`。

## API 说明

统一响应结构：

```json
{ "code": 0, "message": "success", "data": {} }
```

业务码：`0` 成功、`40000` 参数错误、`40100` 未认证、`40300` 无权限、`40400` 资源不存在、`40900` 冲突、`50000` 服务端错误。HTTP 状态码与业务码同步返回（400/401/403/404/409/500）。

受保护接口需携带请求头：`Authorization: Bearer <token>`。

| 方法 | 路径 | 认证 | 说明 |
| --- | --- | --- | --- |
| GET | `/healthz` | 否 | 健康检查 |
| POST | `/api/v1/users/register` | 否 | 注册（username 3–64、password ≥6、合法 email） |
| POST | `/api/v1/users/login` | 否 | 登录，返回 JWT |
| GET | `/api/v1/users/me` | 是 | 当前用户信息 |
| GET | `/api/v1/posts` | 否 | 文章列表，支持 `page`/`page_size`/`keyword`/`author` |
| POST | `/api/v1/posts` | 是 | 创建文章 |
| GET | `/api/v1/posts/:id` | 否 | 文章详情（预加载作者与评论） |
| PUT | `/api/v1/posts/:id` | 是（作者） | 更新文章 |
| DELETE | `/api/v1/posts/:id` | 是（作者） | 删除文章（软删除） |
| POST | `/api/v1/posts/:id/comments` | 是 | 发表评论 |
| GET | `/api/v1/posts/:id/comments` | 否 | 某篇文章的评论列表 |

### 请求示例

注册：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/users/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"Passw0rd!","email":"alice@example.com"}'
```

登录并取令牌：

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/users/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"Passw0rd!"}' | jq -r '.data.token')
```

创建文章：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/posts \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"我的第一篇文章","content":"这是正文内容。"}'
```

评论文章（`1` 为文章 ID）：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/posts/1/comments \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"content":"写得不错！"}'
```

## 测试

Go 单元 + 集成测试（内存 SQLite，无需启动服务）：

```bash
go test ./... -count=1     # 或 make gotest
```

覆盖内容：

| 包 | 用例 |
| --- | --- |
| `internal/auth` | bcrypt 哈希/校验/盐值随机；JWT 签发-解析往返、错误密钥、过期令牌、`alg=none` 算法混淆、乱码令牌（语句覆盖率 93.8%） |
| `internal/config` | 默认值、`JWT_SECRET` 缺失报错、环境变量覆盖、`JWT_EXPIRE` 多格式解析（覆盖率 100%） |
| `internal/database` | AutoMigrate 建表与唯一索引、非法驱动、非法 DSN |
| `internal/middleware` | panic 注入（主动 `panic` 与运行时 nil map 写入）→ 统一 500 信封；日志保留完整调用栈并指向 panic 触发点，且堆栈与 panic 原文均不泄漏进响应体；`Logger` 叠加时以 ERROR 级别记录 `status=500` |
| `internal/handler` | 经完整路由的端到端用例：注册/登录/重复冲突/参数校验、JWT 中间件（无令牌/伪造令牌/`/users/me`）、文章 CRUD 全流程、分页与关键字/作者过滤、非作者 403 且数据未被篡改、评论创建与列表、软删除后 404、响应不泄漏密码 |

共 22 个 Go 用例。`Recovery`/`Auth` 中间件语句覆盖率 100%，`Logger` 92.9%。

> 覆盖率数字随 `-coverpkg` 范围与执行包集合变化，跨轮次不宜直接比较；如需稳定口径，固定使用
> `go test ./... -coverpkg=./... -coverprofile=cover.out` 后看 `go tool cover -func=cover.out | tail -1`。
> 未覆盖部分集中在 `cmd/server/main.go` 与 `internal/logger.New` 等装配代码（仅在进程启动时执行）。

接口冒烟脚本 `tests/api_test.sh` 覆盖同样的公开面，共 30 个用例：

```bash
go run ./cmd/server &          # 先启动服务
BASE_URL=http://127.0.0.1:8080 ./tests/api_test.sh
```

实测结果（go1.26.5 / darwin-arm64 / SQLite）：**通过 30，失败 0**。

| 分组 | 用例与期望状态码 |
| --- | --- |
| 健康检查 | `GET /healthz` → 200 |
| 用户 | 注册 → 201；重复用户名 → 409；密码过短 → 400；登录成功 → 200；密码错误 → 401；用户不存在 → 401；`/users/me` 带令牌 → 200；无令牌 → 401；伪造令牌 → 401 |
| 文章 | 创建 → 201；未认证创建 → 401；参数缺失 → 400；列表 → 200；关键字搜索 → 200；详情 → 200；不存在 → 404；ID 非法 → 400；作者更新 → 200；非作者更新 → 403；非作者删除 → 403 |
| 评论 | 发表 → 201；未认证 → 401；内容为空 → 400；文章不存在 → 404；评论列表 → 200 |
| 删除 | 作者删除 → 200；删除后再查 → 404 |

如需 Postman，可按上表在集合中依次录入：登录请求的响应 `data.token` 通过 Tests 脚本写入环境变量，供受保护接口的 `Authorization: Bearer {{token}}` 引用。

## 实现要点

- **密码安全**：注册时 `bcrypt.GenerateFromPassword`（DefaultCost）加密，登录时 `CompareHashAndPassword` 校验；`Password` 字段 JSON tag 为 `-`，任何响应都不会泄漏哈希。
- **JWT**：HS256 签名，载荷含 `id`、`username`、`iat`、`exp`、`iss`；解析时用 `jwt.WithValidMethods` 锁定算法，防算法混淆；密钥未配置则启动失败。
- **授权**：更新/删除前先查记录再比对 `post.UserID` 与令牌中的 `uid`，不匹配返回 403；不存在返回 404，避免通过状态码探测资源是否存在。
- **错误处理**：`response.FromError` 集中把 `gorm.ErrRecordNotFound`、认证错误映射为对应 HTTP 状态码；未知错误统一 500 并记录日志，不向客户端暴露内部细节。
- **日志**：`Recovery` 捕获 panic 并记录；`Logger` 输出方法、路径、状态码、耗时、客户端 IP，5xx 记为 error、4xx 记为 warn。
- **其他**：文章列表分页并对 `page_size` 做上限（50）保护；软删除历史数据；`ReadHeaderTimeout` 防慢速攻击；进程收到 SIGINT/SIGTERM 后 5 秒内优雅退出。
