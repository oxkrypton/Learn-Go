package service

import (
	"context"
	"go-redis/internal/model"
	"go-redis/internal/repository"
	"math/rand"
	"time"
)

type UserService interface {
	LoginWithCode(ctx context.Context, phone string) (*model.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

// 验证码登录核心逻辑
func (s *userService) LoginWithCode(ctx context.Context, phone string) (*model.User, error) {
	//1.查询数据库中是否存在该用户
	user, err := s.repo.QueryUserByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}

	//2.如果用户存在，直接返回
	if user != nil {
		return user, nil
	}

	newUser := &model.User{
		Phone:    phone,
		NickName: "user_" + generateRandomString(10),
	}
	err = s.repo.CreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}
	return newUser, nil
}

func generateRandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
