package model

import (
	"time"
)

// Voucher 对应 tb_voucher 表：普通代金券
type Voucher struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ShopID      uint64    `gorm:"column:shop_id" json:"shopId"`
	Title       string    `gorm:"column:title;not null" json:"title"`
	SubTitle    string    `gorm:"column:sub_title" json:"subTitle"`
	Rules       string    `gorm:"column:rules;type:varchar(1024)" json:"rules"`
	PayValue    int64     `gorm:"column:pay_value;not null;comment:支付金额(分)" json:"payValue"`
	ActualValue int64     `gorm:"column:actual_value;not null;comment:抵扣金额(分)" json:"actualValue"`
	Type        uint8     `gorm:"column:type;default:0;comment:0普通券 1秒杀券" json:"type"`
	Status      uint8     `gorm:"column:status;default:1;comment:1上架 2下架 3过期" json:"status"`
	CreateTime  time.Time `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime  time.Time `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
}

func (Voucher) TableName() string {
	return "tb_voucher"
}

// SeckillVoucher 对应 tb_seckill_voucher 表：秒杀券信息
type SeckillVoucher struct {
	VoucherID  uint64    `gorm:"primaryKey;column:voucher_id" json:"voucherId"`
	Stock      int32     `gorm:"column:stock;not null" json:"stock"`
	BeginTime  time.Time `gorm:"column:begin_time;not null" json:"beginTime"`
	EndTime    time.Time `gorm:"column:end_time;not null" json:"endTime"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
}

func (SeckillVoucher) TableName() string {
	return "tb_seckill_voucher"
}

// VoucherOrder 对应 tb_voucher_order 表：代金券订单
type VoucherOrder struct {
	ID         int64      `gorm:"primaryKey;column:id;autoIncrement:false" json:"id"` // 通常分布式ID不自增
	UserID     uint64     `gorm:"column:user_id;not null" json:"userId"`
	VoucherID  uint64     `gorm:"column:voucher_id;not null" json:"voucherId"`
	PayType    uint8      `gorm:"column:pay_type;default:1;comment:1余额 2支付宝 3微信" json:"payType"`
	Status     uint8      `gorm:"column:status;default:1;comment:1未支付 2已支付 3已核销..." json:"status"`
	CreateTime time.Time  `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	PayTime    *time.Time `gorm:"column:pay_time" json:"payTime"`
	UseTime    *time.Time `gorm:"column:use_time" json:"useTime"`
	RefundTime *time.Time `gorm:"column:refund_time" json:"refundTime"`
	UpdateTime time.Time  `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
}

func (VoucherOrder) TableName() string {
	return "tb_voucher_order"
}
