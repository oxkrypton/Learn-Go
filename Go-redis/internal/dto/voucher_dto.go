package dto

import "time"

// VoucherDTO 聚合了普通券 + 秒杀券的展示信息
type VoucherDTO struct {
    ID          uint64     `json:"id"`
    ShopID      uint64     `json:"shopId"`
    Title       string     `json:"title"`
    SubTitle    string     `json:"subTitle"`
    Rules       string     `json:"rules"`
    PayValue    int64      `json:"payValue"`
    ActualValue int64      `json:"actualValue"`
    Type        uint8      `json:"type"`
    Status      uint8      `json:"status"`
    // 以下为秒杀券专属字段，普通券时为零值
    Stock       int32      `json:"stock"`
    BeginTime   *time.Time `json:"beginTime,omitempty"`
    EndTime     *time.Time `json:"endTime,omitempty"`
}
