package handler

import (
	"strconv"

	"blog-server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// currentUserID 读取认证中间件写入的用户 ID。
func currentUserID(c *gin.Context) (uint, bool) {
	return middleware.CurrentUserID(c)
}

// parseIDParam 解析路径中的数字 ID 参数。
func parseIDParam(c *gin.Context, name string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || v == 0 {
		return 0, false
	}
	return uint(v), true
}

// parsePagination 解析 page / page_size 查询参数并给出安全默认值。
func parsePagination(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	return page, pageSize
}
