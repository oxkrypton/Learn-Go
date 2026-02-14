package repository

import (
	"context"
	"go-redis/internal/model"

	"gorm.io/gorm"
)

// FollowRepository 定义关注相关数据接口
type FollowRepository interface {
	// IsFollow 查询是否关注了某用户
	IsFollow(ctx context.Context, userId uint64, followUserId uint64) (bool, error)
	// Follow 关注用户
	Follow(ctx context.Context, follow *model.Follow) error
	// Unfollow 取消关注
	Unfollow(ctx context.Context, userId uint64, followUserId uint64) error
	// QueryFollowsByUserId 查询用户关注列表 (用于 Feed 流推送)
	QueryFollowsByUserId(ctx context.Context, userId uint64) ([]model.Follow, error)
}

type followRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

// IsFollow 查询是否关注
func (r *followRepository) IsFollow(ctx context.Context, userId uint64, followUserId uint64) (bool, error) {
	var count int64
	// 对应 tb_follow 表
	err := r.db.WithContext(ctx).
		Model(&model.Follow{}).
		Where("user_id = ? AND follow_user_id = ?", userId, followUserId).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Follow 关注用户
func (r *followRepository) Follow(ctx context.Context, follow *model.Follow) error {
	return r.db.WithContext(ctx).Create(follow).Error
}

// Unfollow 取消关注
func (r *followRepository) Unfollow(ctx context.Context, userId uint64, followUserId uint64) error {
	// 根据双主键逻辑删除
	return r.db.WithContext(ctx).
		Where("user_id = ? AND follow_user_id = ?", userId, followUserId).
		Delete(&model.Follow{}).Error
}

// QueryFollowsByUserId 查询某人关注了谁
func (r *followRepository) QueryFollowsByUserId(ctx context.Context, userId uint64) ([]model.Follow, error) {
	var follows []model.Follow
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Find(&follows).Error
	return follows, err
}