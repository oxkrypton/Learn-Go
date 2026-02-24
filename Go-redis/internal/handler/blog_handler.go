package handler

import (
	"log"
	"net/http"
	"strconv"

	"go-redis/internal/dto"
	"go-redis/internal/service"

	"github.com/gin-gonic/gin"
)

type BlogHandler struct {
	svc service.BlogService
}

// NewBlogHandler 构造函数：注入 BlogService
func NewBlogHandler(svc service.BlogService) *BlogHandler {
	return &BlogHandler{svc: svc}
}

// QueryHotBlogs 处理 GET /blog/hot?current=1 请求
func (h *BlogHandler) QueryHotBlogs(c *gin.Context) {
	// 核心逻辑 1：解析 Query 参数 ?current=x，默认值为 1
	currentStr := c.DefaultQuery("current", "1")
	current, err := strconv.Atoi(currentStr)
	if err != nil || current < 1 {
		current = 1
	}

	// 核心逻辑 2：调用 Service 层业务逻辑
	blogs, err := h.svc.QueryHotBlogs(c.Request.Context(), current)
	if err != nil {
		log.Printf("[BlogHandler] QueryHotBlogs err: %v\n", err)
		c.JSON(http.StatusOK, dto.Fail("查询热门笔记失败"))
		return
	}

	// 核心逻辑 3：响应成功结果，使用统一定义的 dto.Result 格式 
	// (匹配前端的 {"code":200, "data": [...], "msg": "success"})
	c.JSON(http.StatusOK, dto.Success(blogs))
}
