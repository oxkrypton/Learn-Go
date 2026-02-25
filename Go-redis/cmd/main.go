package main

import (
	"fmt"
	"log"

	"go-redis/internal/config"
	"go-redis/internal/handler"
	"go-redis/internal/pkg/database"
	"go-redis/internal/repository"
	"go-redis/internal/router"
	"go-redis/internal/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化配置
	err := config.InitConfig()
	if err != nil {
		panic(fmt.Sprintf("load config fail: %v", err))
	}

	// 2. 初始化 Redis
	err = database.InitRedis()
	if err != nil {
		panic(fmt.Sprintf("redis connection fail: %v", err))
	}

	// 3. 初始化 MySQL (GORM 会自动建立连接池)
	if err := database.InitMysql(); err != nil {
		panic(fmt.Sprintf("mysql connection fail: %v", err))
	}

	// ----------- 核心逻辑：层级组装 / 依赖注入 (DI) -----------

	// 层级 A: Repository 获取数据库实例
	userRepo := repository.NewUserRepository(database.DB)
	blogRepo := repository.NewBlogRepository(database.DB)
	shopRepo := repository.NewShopRepository(database.DB)

	// 层级 B: Service 注入 Repository
	blogService := service.NewBlogService(blogRepo, userRepo)
	shopService := service.NewShopService(shopRepo)
	userService := service.NewUserService(userRepo)

	// 层级 C: Handler 注入 Service
	blogHandler := handler.NewBlogHandler(blogService)
	shopHandler := handler.NewShopHandler(shopService)
	userHandler := handler.NewUserHandler(userService)

	// ----------- 引擎与路由初始化 -----------

	r := gin.Default()

	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("redis-session", store))

	// 核心逻辑：统一挂载业务 API 路由
	router.SetupRouter(r, blogHandler, shopHandler, userHandler)

	// 启动并监听配置的端口
	log.Println("Server is running on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server start failed: ", err)
	}
}
