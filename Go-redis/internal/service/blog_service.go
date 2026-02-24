package service

import (
	"context"
	"go-redis/internal/model"
	"go-redis/internal/repository"
)

// BlogService 探店笔记业务逻辑接口
type BlogService interface {
	// QueryHotBlogs 分页查询热门笔记，并封装返回
	QueryHotBlogs(ctx context.Context, current int) ([]model.Blog, error)
}

type blogService struct {
	repo     repository.BlogRepository
	userRepo repository.UserRepository // 引入 userRepo 是因为可能需要关联查询博主信息
}

// NewBlogService 构造函数：注入所需的 Repo
func NewBlogService(repo repository.BlogRepository, userRepo repository.UserRepository) BlogService {
	return &blogService{
		repo:     repo,
		userRepo: userRepo,
	}
}

func (s *blogService) QueryHotBlogs(ctx context.Context, current int) ([]model.Blog, error) {
	// 核心逻辑 1：每页默认查 10 条热门数据
	size := 10
	
	// 核心逻辑 2：调用 Repository 层获取当前页的 Blog 数据
	blogs, err := s.repo.QueryHotBlogs(ctx, current, size)
	if err != nil {
		return nil, err
	}

	// 核心逻辑 3：此处可以根据业务需要，通过 userRepo.QueryUserById(blog.UserID) 查询并映射博主姓名和头像
	// (当前示例简化，直接返回数据库查询到的 Blog 列表)
	
	return blogs, nil
}
