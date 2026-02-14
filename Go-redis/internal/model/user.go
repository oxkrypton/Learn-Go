package model

import "time"

// User 对应数据库 tb_user 表
type User struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;column:id;comment:主键" json:"id"`
	City       string    `gorm:"column:city;default:'';comment:城市名称" json:"city"`
	Introduce  string    `gorm:"column:introduce;comment:个人介绍" json:"introduce"`
	Fans       uint32    `gorm:"column:fans;default:0;comment:粉丝数量" json:"fans"`
	Followee   uint32    `gorm:"column:followee;default:0;comment:关注的人的数量" json:"followee"`
	Gender     uint8     `gorm:"column:gender;default:0;comment:性别，0：男，1：女" json:"gender"`
	Birthday   time.Time `gorm:"column:birthday;comment:生日" json:"birthday"`
	Credits    uint32    `gorm:"column:credits;default:0;comment:积分" json:"credits"`
	Level      uint8     `gorm:"column:level;default:0;comment:会员级别" json:"level"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;comment:创建时间" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime;comment:更新时间" json:"update_time"`
}

func (User) TableName() string {
	return "tb_user"
}

// UserInfo 对应数据库 tb_user_info 表
type UserInfo struct {
	UserID     uint64    `gorm:"primaryKey;column:user_id;comment:主键，用户id" json:"user_id"`
	City       string    `gorm:"column:city;default:'';comment:城市名称" json:"city"`
	Introduce  string    `gorm:"column:introduce;comment:个人介绍" json:"introduce"`
	Fans       uint32    `gorm:"column:fans;default:0;comment:粉丝数量" json:"fans"`
	Followee   uint32    `gorm:"column:followee;default:0;comment:关注的人的数量" json:"followee"`
	Gender     uint8     `gorm:"column:gender;default:0;comment:性别，0：男，1：女" json:"gender"`
	Birthday   time.Time `gorm:"column:birthday;comment:生日" json:"birthday"`
	Credits    uint32    `gorm:"column:credits;default:0;comment:积分" json:"credits"`
	Level      uint8     `gorm:"column:level;default:0;comment:会员级别" json:"level"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;comment:创建时间" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime;comment:更新时间" json:"update_time"`
}

func (UserInfo) TableName() string {
	return "tb_user_info"
}
