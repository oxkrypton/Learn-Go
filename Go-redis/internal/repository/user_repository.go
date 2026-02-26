package repository

import (
	"context"
	"errors"
	"go-redis/internal/model"

	"gorm.io/gorm"
)

// UserRepository 定义用户及用户详情的数据访问接口
type UserRepository interface {
	// QueryUserByPhone 根据手机号查询用户 (对应 tb_user)
	QueryUserByPhone(ctx context.Context, phone string) (*model.User, error)
	// QueryUserById 根据ID查询用户 (对应 tb_user)
	QueryUserById(ctx context.Context, id uint64) (*model.User, error)
	// CreateUser 创建新用户 (对应 tb_user)
	CreateUser(ctx context.Context, user *model.User) error
	// QueryUserInfoById 查询用户详情 (对应 tb_user_info)
	QueryUserInfoById(ctx context.Context, userId uint64) (*model.UserInfo, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造函数
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// QueryUserByPhone 根据手机号查询用户
func (r *userRepository) QueryUserByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没找到不报错，返回 nil
		}
		return nil, err
	}
	return &user, nil
}

// QueryUserById 根据ID查询用户
func (r *userRepository) QueryUserById(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// CreateUser 创建新用户
func (r *userRepository) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// QueryUserInfoById 查询用户详情
func (r *userRepository) QueryUserInfoById(ctx context.Context, userId uint64) (*model.UserInfo, error) {
	var userInfo model.UserInfo
	// 对应 tb_user_info 表，主键为 user_id
	err := r.db.WithContext(ctx).Where("user_id = ?", userId).First(&userInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &userInfo, nil
}