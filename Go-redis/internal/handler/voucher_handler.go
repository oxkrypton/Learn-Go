package handler

import (
	"go-redis/internal/dto"
	"go-redis/internal/service"
	"go-redis/internal/utils"
	"log"
	"net/http"

	"strconv"

	"github.com/gin-gonic/gin"
)

type VoucherHandler struct {
	svc service.VoucherService
}

// NewVoucherHandler 构造函数：注入 VoucherService
func NewVoucherHandler(svc service.VoucherService) *VoucherHandler {
	return &VoucherHandler{svc: svc}
}

func (h *VoucherHandler) AddVoucher(c *gin.Context) {
	var req dto.VoucherDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("Invalid args"))
		return
	}
	if err := h.svc.AddVoucher(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("Fail to add voucher"))
		return
	}
	c.JSON(http.StatusOK, dto.Success(req.ID))
}

// QueryVoucherList 处理 GET /voucher/list/:shopId
func (h *VoucherHandler) QueryVouchersByShopId(c *gin.Context) {
	shopIdStr := c.Param("shopId")
	shopId, err := strconv.ParseUint(shopIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("Invalid ShopId"))
		return
	}

	vouchers, err := h.svc.QueryVouchersByShopId(c.Request.Context(), shopId)
	if err != nil {
		log.Printf("[VoucherHandler] QueryVoucherList err: %v\n", err)
		c.JSON(http.StatusInternalServerError, dto.Fail("Query VoucherList fails"))
		return
	}

	c.JSON(http.StatusOK, dto.Success(vouchers))
}

// SeckillOrder 处理 POST /voucher/order/seckill
func (h *VoucherHandler) SeckillOrder(c *gin.Context) {
	//从URL中获取 voucherId
	voucherIdStr := c.Param("id")
	voucherId, err := strconv.ParseUint(voucherIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("invalid voucher id"))
		return
	}

	// 从 gin.Context 获取当前登录用户
	user, ok := utils.GetUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.Fail("User not login"))
		return
	}

	// 调用 service 层执行秒杀下单
	orderId, err := h.svc.SeckillVoucher(c.Request.Context(), voucherId, user.ID)
	if err != nil {
		log.Printf("[VoucherHandler] SeckillOrder err: %v\n", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(err.Error()))
		return
	}

	//返回订单ID
	c.JSON(http.StatusOK, dto.Success(orderId))
}
