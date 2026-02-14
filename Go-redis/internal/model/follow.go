package model

import "time"

// Follow 对应 tb_follow 表：用户关注关系
type Follow struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID       uint64    `gorm:"column:user_id;not null;comment:用户id" json:"user_id"`
	FollowUserID uint64    `gorm:"column:follow_user_id;not null;comment:关联的用户id" json:"follow_user_id"`
	CreateTime   time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

func (Follow) TableName() string {
	return "tb_follow"
}
