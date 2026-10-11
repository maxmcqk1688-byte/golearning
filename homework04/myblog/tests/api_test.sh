#!/usr/bin/env bash
# 博客后端接口冒烟测试。
#
# 用法：
#   BASE_URL=http://127.0.0.1:8080 JWT_SECRET=your_secret ./tests/api_test.sh
#
# 前置条件：服务已启动，且本机安装 curl 与 jq。
set -uo pipefail

BASE="${BASE_URL:-http://127.0.0.1:8080}/api/v1"
PASS=0
FAIL=0

command -v jq >/dev/null || { echo "需要安装 jq"; exit 1; }

# check <描述> <期望状态码> <curl 参数...>
check() {
  local desc="$1" expect="$2"
  shift 2
  local body code
  body=$(curl -s -w '\n%{http_code}' "$@")
  code=$(echo "$body" | tail -n1)
  body=$(echo "$body" | sed '$d')
  if [[ "$code" == "$expect" ]]; then
    printf 'PASS  %-46s %s\n' "$desc" "$code"
    PASS=$((PASS + 1))
  else
    printf 'FAIL  %-46s 期望 %s 实际 %s\n%s\n' "$desc" "$expect" "$code" "$body"
    FAIL=$((FAIL + 1))
  fi
  LAST_BODY="$body"
}

SUFFIX=$(date +%s)
U1="alice$SUFFIX"
U2="bob$SUFFIX"
E1="alice$SUFFIX@example.com"
E2="bob$SUFFIX@example.com"
P1='Passw0rd!'
P2='Passw0rd2!'

echo "== 健康检查 =="
check "GET /healthz" 200 "${BASE%/api/v1}/healthz"

echo
echo "== 用户注册 / 登录 =="
check "注册新用户 $U1" 201 -X POST "$BASE/users/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$U1\",\"password\":\"$P1\",\"email\":\"$E1\"}"
check "重复用户名注册" 409 -X POST "$BASE/users/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$U1\",\"password\":\"$P1\",\"email\":\"other-$SUFFIX@example.com\"}"
check "注册第二个用户 $U2" 201 -X POST "$BASE/users/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$U2\",\"password\":\"$P2\",\"email\":\"$E2\"}"
check "参数非法（密码过短）" 400 -X POST "$BASE/users/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"shorty$SUFFIX\",\"password\":\"123\",\"email\":\"shorty$SUFFIX@example.com\"}"

check "登录成功" 200 -X POST "$BASE/users/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$U1\",\"password\":\"$P1\"}"
TOKEN=$(echo "$LAST_BODY" | jq -r '.data.token')
AUTH1="Authorization: Bearer $TOKEN"

check "登录密码错误" 401 -X POST "$BASE/users/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$U1\",\"password\":\"wrong-password\"}"
check "登录用户不存在" 401 -X POST "$BASE/users/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"nobody$SUFFIX\",\"password\":\"whatever123\"}"
check "携带令牌获取个人信息" 200 "$BASE/users/me" -H "$AUTH1"
check "无令牌访问受保护接口" 401 "$BASE/users/me"
check "伪造令牌访问受保护接口" 401 "$BASE/users/me" -H 'Authorization: Bearer not.a.valid.token'

echo
echo "== 文章 CRUD =="
check "创建文章" 201 -X POST "$BASE/posts" -H "$AUTH1" \
  -H 'Content-Type: application/json' \
  -d '{"title":"我的第一篇文章","content":"这是正文内容，用于接口测试。"}'
PID1=$(echo "$LAST_BODY" | jq -r '.data.id')
check "未认证创建文章" 401 -X POST "$BASE/posts" \
  -H 'Content-Type: application/json' -d '{"title":"x","content":"y"}'
check "创建文章参数缺失" 400 -X POST "$BASE/posts" -H "$AUTH1" \
  -H 'Content-Type: application/json' -d '{"title":""}'

check "文章列表（公开）" 200 "$BASE/posts?page=1&page_size=5"
check "关键字搜索文章" 200 "$BASE/posts?keyword=第一篇"
check "文章详情（公开）" 200 "$BASE/posts/$PID1"
check "不存在的文章" 404 "$BASE/posts/999999"
check "无效文章 ID" 400 "$BASE/posts/abc"

check "作者更新文章" 200 -X PUT "$BASE/posts/$PID1" -H "$AUTH1" \
  -H 'Content-Type: application/json' \
  -d '{"title":"更新后的标题","content":"更新后的正文。"}'

check "$U2 登录" 200 -X POST "$BASE/users/login" \
  -H 'Content-Type: application/json' -d "{\"username\":\"$U2\",\"password\":\"$P2\"}"
TOKEN2=$(echo "$LAST_BODY" | jq -r '.data.token')
AUTH2="Authorization: Bearer $TOKEN2"

check "非作者更新文章" 403 -X PUT "$BASE/posts/$PID1" -H "$AUTH2" \
  -H 'Content-Type: application/json' -d '{"title":"劫持","content":"劫持"}'
check "非作者删除文章" 403 -X DELETE "$BASE/posts/$PID1" -H "$AUTH2"

echo
echo "== 评论 =="
check "发表评论" 201 -X POST "$BASE/posts/$PID1/comments" -H "$AUTH2" \
  -H 'Content-Type: application/json' -d '{"content":"写得不错！"}'
check "未认证评论" 401 -X POST "$BASE/posts/$PID1/comments" \
  -H 'Content-Type: application/json' -d '{"content":"hi"}'
check "评论内容为空" 400 -X POST "$BASE/posts/$PID1/comments" -H "$AUTH2" \
  -H 'Content-Type: application/json' -d '{"content":""}'
check "对不存在文章评论" 404 -X POST "$BASE/posts/999999/comments" -H "$AUTH2" \
  -H 'Content-Type: application/json' -d '{"content":"hello"}'
check "评论列表（公开）" 200 "$BASE/posts/$PID1/comments"

echo
echo "== 删除 =="
check "作者删除文章" 200 -X DELETE "$BASE/posts/$PID1" -H "$AUTH1"
check "删除后访问已删文章" 404 "$BASE/posts/$PID1"

echo
echo "================ 结果: 通过 $PASS, 失败 $FAIL ================"
[[ "$FAIL" -eq 0 ]]
