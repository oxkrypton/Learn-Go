package handler

import (
	"log"
	"net/http"
	"strconv"

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
		c.JSON(http.StatusOK, dto.Fail("query shop_types fails"))
		return
	}

	// 核心逻辑 2：成功响应前端所需的 List 数据
	c.JSON(http.StatusOK, dto.Success(types))
}

// QueryShopsByType 处理 GET /shop/of/type?typeId=1&current=1 请求
// 该路由为公开接口，无需登录
func (h *ShopHandler) QueryShopsByType(c *gin.Context) {
	// 1. 解析 typeId 参数
	typeIdStr := c.Query("typeId")
	typeId, err := strconv.ParseUint(typeIdStr, 10, 64)
	if err != nil || typeId < 1 {
		c.JSON(http.StatusOK, dto.Fail("Shoptype args wrong"))
		return
	}

	//2.解析分页参数，默认第一页
	currentStr := c.DefaultQuery("current", "1")
	current, err := strconv.Atoi(currentStr)
	if err != nil || current < 1 {
		current = 1
	}

	//3.调用Service按类型分页查询
	shops, err := h.svc.QueryShopsByType(c.Request.Context(), typeId, current)
	if err != nil {
		log.Printf("[ShopHandler] QueryShopsByType err: %v\n", err)
		c.JSON(http.StatusOK, dto.Fail("Query ShopeTypeList fail"))
		return
	}

	// 4. 返回商铺列表
	c.JSON(http.StatusOK, dto.Success(shops))
}

func (h *ShopHandler) QueryShopById(c *gin.Context) {
	// 1. 从 URL 路径参数获取 id：c.Param("id")
	idStr := c.Param("id")

	// 2. 转换为 uint64：strconv.ParseUint(idStr, 10, 64)
	//    - 转换失败 → c.JSON(400, dto.Fail("商铺id不合法"))，return
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, dto.Fail("Invalid ShopId"))
		return
	}

	// 3. 调用 Service：h.svc.QueryShopById(ctx, id)
	//    - err != nil → c.JSON(500, dto.Fail("查询失败"))，return
	shop, err := h.svc.QueryShopById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("Query ShopById fails"))
		return
	}

	// 4. 判断 shop 是否为 nil
	//    - nil → c.JSON(404, dto.Fail("商铺不存在"))，return（这里是关键的404）
	if shop == nil {
		c.JSON(http.StatusNotFound, dto.Fail("Shop Not Found"))
		return
	}

	// 5. 成功 → c.JSON(200, dto.Success(shop))
	c.JSON(http.StatusOK, dto.Success(shop))
}
