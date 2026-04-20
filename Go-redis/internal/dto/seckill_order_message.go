package dto

type SeckillOrderMessage struct {
	OrderID   int64  `json:"orderId"`
	UserID    uint64 `json:"userId"`
	VoucherID uint64 `json:"voucherId"`
}
