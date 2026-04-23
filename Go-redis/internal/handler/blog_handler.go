package handler

import (
	"log"
	"net/http"
	"strconv"

	"go-redis/internal/dto"
	"go-redis/internal/pkg/ginx"
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
		log.Printf("[BlogHandler] QueryHotBlogs err: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail("internal server error"))
		return
	}

	// 核心逻辑 3：响应成功结果，使用统一定义的 dto.Result 格式
	// (当前响应结构为 {"success":true, "data":[...]}，失败时使用 errMsg)
	c.JSON(http.StatusOK, dto.Success(blogs))
}

// QueryMyBlogs 处理 GET /blog/of/me?current=1 请求
// 该路由已被 LoginInterceptor 保护，到达此处时用户一定已登录
func (h *BlogHandler) QueryMyBlogs(c *gin.Context) {
	//1.从context获取当前用户(中间件已保证存在)
	userDTO, _ := ginx.GetUser(c)

	//2.解析分页参数，默认第一页
	currentStr := c.DefaultQuery("current", "1")
	current, err := strconv.Atoi(currentStr)
	if err != nil || current < 1 {
		current = 1
	}

	// 3. 调用 Service 查询当前用户的笔记
	blogs, err := h.svc.QueryMyBlogs(c.Request.Context(), userDTO.ID, current)
	if err != nil {
		log.Printf("[BlogHandler] QueryMyBlogs err: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail("internal server error"))
		return
	}

	//4.返回笔记列表
	c.JSON(http.StatusOK, dto.Success(blogs))
}
