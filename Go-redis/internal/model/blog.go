package model

import (
	"time"
)

// Blog 对应 tb_blog 表：探店笔记
type Blog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ShopID     uint64    `gorm:"column:shop_id;not null" json:"shop_id"`
	UserID     uint64    `gorm:"column:user_id;not null" json:"user_id"`
	Title      string    `gorm:"column:title;not null;type:varchar(255)" json:"title"`
	Images     string    `gorm:"column:images;not null;type:varchar(2048)" json:"images"` // 存储多张图片地址，以逗号分隔
	Content    string    `gorm:"column:content;not null;type:varchar(2048)" json:"content"`
	Liked      uint32    `gorm:"column:liked;default:0" json:"liked"`
	Comments   uint32    `gorm:"column:comments" json:"comments"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

func (Blog) TableName() string {
	return "tb_blog"
}

// BlogComments 对应 tb_blog_comments 表：笔记评论
type BlogComments struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID     uint64    `gorm:"column:user_id;not null" json:"user_id"`
	BlogID     uint64    `gorm:"column:blog_id;not null" json:"blog_id"`
	ParentID   uint64    `gorm:"column:parent_id;not null;comment:一级评论id，0代表一级评论" json:"parent_id"`
	AnswerID   uint64    `gorm:"column:answer_id;not null;comment:回复的评论id" json:"answer_id"`
	Content    string    `gorm:"column:content;not null;type:varchar(255)" json:"content"`
	Liked      uint32    `gorm:"column:liked" json:"liked"`
	Status     uint8     `gorm:"column:status;comment:0正常 1举报 2禁止查看" json:"status"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

func (BlogComments) TableName() string {
	return "tb_blog_comments"
}
