package handler

import (
	"go-redis/internal/dto"
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
