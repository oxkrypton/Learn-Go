package router

import (
	"go-redis/internal/handler"
	"go-redis/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// SetupRouter 统一管理所有路由组的注册
func SetupRouter(r *gin.Engine, rdb *redis.Client,
	blogHandler *handler.BlogHandler,
	shopHandler *handler.ShopHandler,
	userHandler *handler.UserHandler,
	voucherHandler *handler.VoucherHandler,
) {

	// ==================== 1. 商铺模块 ====================
	// 1.1 商铺分类 - 公开路由 (对应前端 /api/shop-type/xxx)
	shopTypeGroup := r.Group("/shop-type")
	{
		shopTypeGroup.GET("/list", shopHandler.QueryShopTypeList) // GET /shop-type/list
	}
	// 1.2 商铺查询 - 公开路由 (对应前端 /api/shop/xxx)
	shopGroup := r.Group("/shop")
	{
		shopGroup.GET("/of/type", shopHandler.QueryShopsByType) // GET /shop/of/type?typeId=1&current=1
		shopGroup.GET("/hot/:id",shopHandler.QueryHotShopById)  // GET /shop/hot/:id
		shopGroup.GET("/:id", shopHandler.QueryShopById)        // GET /shop/:id
	}
	// 1.3 商铺更新 - 认证路由 (需要登录，更新时会删除缓存)
	shopAuthGroup := r.Group("/shop")
	shopAuthGroup.Use(middleware.LoginInterceptor(rdb))
	{
		shopAuthGroup.POST("/create", shopHandler.CreateShop)
		shopAuthGroup.PUT("", shopHandler.UpdateShop) // PUT /shop
	}

	// ==================== 2. 探店笔记模块 ====================
	// 2.1 笔记 - 公开路由 (对应前端 /api/blog/xxx)
	blogGroup := r.Group("/blog")
	{
		blogGroup.GET("/hot", blogHandler.QueryHotBlogs) // GET /blog/hot?current=1
	}
	// 2.2 笔记 - 认证路由 (需要登录)
	blogAuthGroup := r.Group("/blog")
	blogAuthGroup.Use(middleware.LoginInterceptor(rdb))
	{
		blogAuthGroup.GET("/of/me", blogHandler.QueryMyBlogs) // GET /blog/of/me
	}
	// ==================== 3. 用户模块 ====================
	// 3.1 用户 - 公开路由 (对应前端 /api/user/xxx)
	userGroup := r.Group("/user")
	{
		userGroup.POST("/code", userHandler.SendCode) // POST /user/code?phone=xxx
		userGroup.POST("/login", userHandler.Login)   // POST /user/login
	}
	// 3.2 用户 - 认证路由 (需要登录)
	userAuthGroup := r.Group("/user")
	userAuthGroup.Use(middleware.LoginInterceptor(rdb))
	{
		userAuthGroup.GET("/me", userHandler.Me)                  // GET /user/me
		userAuthGroup.GET("/info/:id", userHandler.QueryUserInfo) // GET /user/info/:id
	}

	// ==================== 优惠券模块 ====================
	voucherGroup := r.Group("/voucher")
	{
		voucherGroup.GET("/list/:shopId", voucherHandler.QueryVouchersByShopId)
	}

}
