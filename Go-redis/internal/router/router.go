package router

import (
	"go-redis/internal/handler"
	"go-redis/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 统一管理所有路由组的注册
func SetupRouter(r *gin.Engine, blogHandler *handler.BlogHandler, shopHandler *handler.ShopHandler, userHandler *handler.UserHandler) {

	// 1. 商铺分类模块路由组处理 (对应前端 /api/shop-type/xxx )
	shopGroup := r.Group("/shop-type")
	{
		// GET /shop-type/list
		shopGroup.GET("/list", shopHandler.QueryShopTypeList)
	}

	// 2. 探店笔记模块路由组处理 (对应前端 /api/blog/xxx )
	blogGroup := r.Group("/blog")
	{
		// GET /blog/hot
		blogGroup.GET("/hot", blogHandler.QueryHotBlogs)
	}

	// 3. 用户登陆模块路由处理 (对应前端 /api/user/xxx )
	userGroup := r.Group("/user")
	{
		userGroup.POST("/code", userHandler.SendCode)
		userGroup.POST("/login", userHandler.Login)
	}

		authGroup := r.Group("/user")
	authGroup.Use(middleware.LoginInterceptor())
	{
		authGroup.GET("/me", userHandler.Me)
	}

}
