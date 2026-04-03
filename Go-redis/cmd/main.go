package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-redis/internal/config"
	"go-redis/internal/handler"
	"go-redis/internal/pkg/database"
	"go-redis/internal/repository"
	"go-redis/internal/router"
	"go-redis/internal/service"
	"go-redis/internal/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化配置
	if err := config.InitConfig(); err != nil {
		panic(fmt.Sprintf("load config fail: %v", err))
	}

	// 2. 初始化 Redis
	rdb, err := database.InitRedis(config.GlobalConfig.Redis)
	if err != nil {
		panic(fmt.Sprintf("redis connection fail: %v", err))
	}

	// 3. 初始化 MySQL (GORM 会自动建立连接池)
	db, err := database.InitMysql(config.GlobalConfig.MySQL)
	if err != nil {
		panic(fmt.Sprintf("mysql connection fail: %v", err))
	}

	// ----------- 核心逻辑：层级组装 / 依赖注入 (DI) -----------

	// 层级 A: Repository 获取数据库实例
	userRepo := repository.NewUserRepository(db)
	blogRepo := repository.NewBlogRepository(db)
	shopRepo := repository.NewShopRepository(db)
	voucherRepo := repository.NewVoucherRepository(db)
	bloomClient := utils.NewRedisBloomClient(rdb)

	// 层级 B: Service 注入 Repository
	userService := service.NewUserService(userRepo, rdb)
	blogService := service.NewBlogService(blogRepo, userRepo)
	shopService, err := service.NewShopService(shopRepo, rdb, bloomClient)
	if err != nil {
		panic(fmt.Sprintf("init shop service fail:%v", err))
	}
	voucherService := service.NewVoucherService(voucherRepo, rdb)

	// 层级 C: Handler 注入 Service
	userHandler := handler.NewUserHandler(userService)
	blogHandler := handler.NewBlogHandler(blogService)
	shopHandler := handler.NewShopHandler(shopService)
	voucherHandler := handler.NewVoucherHandler(voucherService)

	// 假设 id=1 的商铺是热点
	shopService.SaveShopToRedis(context.Background(), 1, 30*60) // 逻辑过期时间 30 分钟

	// ----------- 引擎与路由初始化 -----------

	r := gin.Default()
	router.SetupRouter(r, rdb, blogHandler, shopHandler, userHandler, voucherHandler)

	// ----------- 优雅启动与关闭 -----------
	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("Server is running on %s ...\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server start failed: ", err)
		}
	}()

	//监听系统信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exited.")
}
