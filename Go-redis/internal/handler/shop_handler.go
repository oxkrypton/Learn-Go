package handler

import (
	"log"
	"net/http"

	"go-redis/internal/dto"
	"go-redis/internal/service"

	"github.com/gin-gonic/gin"
)

type ShopHandler struct {
	svc service.ShopService
}

// NewShopHandler 构造函数：注入 ShopService
func NewShopHandler(svc service.ShopService) *ShopHandler {
	return &ShopHandler{svc: svc}
}

// QueryShopTypeList 处理 GET /shop-type/list 请求
func (h *ShopHandler) QueryShopTypeList(c *gin.Context) {
	// 核心逻辑 1：调用 Service 获取所有商品分类
	types, err := h.svc.QueryShopTypeList(c.Request.Context())
	if err != nil {
		log.Printf("[ShopHandler] QueryShopTypeList err: %v\n", err)
		c.JSON(http.StatusOK, dto.Fail("查询商铺类型分类失败"))
		return
	}

	// 核心逻辑 2：成功响应前端所需的 List 数据
	c.JSON(http.StatusOK, dto.Success(types))
}
