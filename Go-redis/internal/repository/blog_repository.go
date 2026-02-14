package repository

import (
	"context"
	"go-redis/internal/model"

	"gorm.io/gorm"
)

// BlogRepository 定义探店笔记数据接口
type BlogRepository interface {
	// QueryBlogById 根据ID查询笔记
	QueryBlogById(ctx context.Context, id uint64) (*model.Blog, error)
	// QueryHotBlogs 分页查询热门笔记 (按点赞数降序)
	QueryHotBlogs(ctx context.Context, current int, size int) ([]model.Blog, error)
	// CreateBlog 发布笔记
	CreateBlog(ctx context.Context, blog *model.Blog) error
	// UpdateBlog 更新笔记 (如点赞数更新)
	UpdateBlog(ctx context.Context, blog *model.Blog) error
}

type blogRepository struct {
	db *gorm.DB
}

func NewBlogRepository(db *gorm.DB) BlogRepository {
	return &blogRepository{db: db}
}

func (r *blogRepository) QueryBlogById(ctx context.Context, id uint64) (*model.Blog, error) {
	var blog model.Blog
	err := r.db.WithContext(ctx).First(&blog, id).Error
	if err != nil {
		return nil, err
	}
	return &blog, nil
}

func (r *blogRepository) QueryHotBlogs(ctx context.Context, current int, size int) ([]model.Blog, error) {
	var blogs []model.Blog
	offset := (current - 1) * size
	// 对应 tb_blog 表，按 liked 降序
	err := r.db.WithContext(ctx).
		Order("liked DESC").
		Limit(size).
		Offset(offset).
		Find(&blogs).Error
	return blogs, err
}

func (r *blogRepository) CreateBlog(ctx context.Context, blog *model.Blog) error {
	return r.db.WithContext(ctx).Create(blog).Error
}

func (r *blogRepository) UpdateBlog(ctx context.Context, blog *model.Blog) error {
	// 只更新非零值字段
	return r.db.WithContext(ctx).Model(blog).Updates(blog).Error
}