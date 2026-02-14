package model

import "time"

// Shop 对应 tb_shop 表：商铺核心数据
type Shop struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name       string    `gorm:"column:name;not null;type:varchar(128)" json:"name"`
	TypeID     uint64    `gorm:"column:type_id;not null" json:"type_id"`
	Images     string    `gorm:"column:images;not null;type:varchar(1024)" json:"images"`
	Area       string    `gorm:"column:area;type:varchar(128)" json:"area"`
	Address    string    `gorm:"column:address;not null;type:varchar(255)" json:"address"`
	X          float64   `gorm:"column:x;not null" json:"x"` // 经度
	Y          float64   `gorm:"column:y;not null" json:"y"` // 纬度
	AvgPrice   uint64    `gorm:"column:avg_price" json:"avg_price"`
	Sold       uint32    `gorm:"column:sold;not null" json:"sold"`
	Comments   uint32    `gorm:"column:comments;not null" json:"comments"`
	Score      uint8     `gorm:"column:score;not null;comment:评分x10保存" json:"score"`
	OpenHours  string    `gorm:"column:open_hours;type:varchar(32)" json:"open_hours"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

func (Shop) TableName() string {
	return "tb_shop"
}

// ShopType 对应 tb_shop_type 表：商铺类目（如美食、KTV等）
type ShopType struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name       string    `gorm:"column:name;type:varchar(32)" json:"name"`
	Icon       string    `gorm:"column:icon;type:varchar(255)" json:"icon"`
	Sort       uint8     `gorm:"column:sort" json:"sort"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

func (ShopType) TableName() string {
	return "tb_shop_type"
}
