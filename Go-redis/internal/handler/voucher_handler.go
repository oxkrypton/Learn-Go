package handler

import (
	"go-redis/internal/dto"
	"go-redis/internal/pkg/bizerr"
	"go-redis/internal/pkg/ginx"
	"go-redis/internal/service"
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
		c.JSON(http.StatusOK, dto.Fail("invalid parameters"))
		return
	}
	if err := h.svc.AddVoucher(c.Request.Context(), &req); err != nil {
		log.Printf("[VoucherHandler] AddVoucher err: %v", err)
		if bizerr.Is(err) {
			c.JSON(http.StatusOK, dto.Fail(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Fail("internal server error"))
		return
	}
	c.JSON(http.StatusOK, dto.Success(req.ID))
}

// QueryVoucherList 处理 GET /voucher/list/:shopId
func (h *VoucherHandler) QueryVouchersByShopId(c *gin.Context) {
	shopIDStr := c.Param("shopId")
	shopID, err := strconv.ParseUint(shopIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail("invalid shop id"))
		return
	}

	vouchers, err := h.svc.QueryVouchersByShopId(c.Request.Context(), shopID)
	if err != nil {
		log.Printf("[VoucherHandler] QueryVouchersByShopId err: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail("internal server error"))
		return
	}

	c.JSON(http.StatusOK, dto.Success(vouchers))
}

// SeckillOrder 处理 POST /voucher/order/seckill
func (h *VoucherHandler) SeckillOrder(c *gin.Context) {
	// 从URL中获取 voucherId
	voucherIDStr := c.Param("id")
	voucherID, err := strconv.ParseUint(voucherIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail("invalid voucher id"))
		return
	}

	// 从 gin.Context 获取当前登录用户
	user, ok := ginx.GetUser(c)
	if !ok {
		c.JSON(http.StatusOK, dto.Fail("user not logged in"))
		return
	}

	// 调用 service 层执行秒杀下单
	orderID, err := h.svc.SeckillVoucher(c.Request.Context(), voucherID, user.ID)
	if err != nil {
		log.Printf("[VoucherHandler] SeckillOrder err: %v", err)
		if bizerr.Is(err) {
			c.JSON(http.StatusOK, dto.Fail(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Fail("internal server error"))
		return
	}

	// 返回订单ID
	c.JSON(http.StatusOK, dto.Success(orderID))
}
