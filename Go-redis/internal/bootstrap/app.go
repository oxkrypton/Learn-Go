package bootstrap

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"go-redis/internal/config"
	"go-redis/internal/handler"
	"go-redis/internal/pkg/database"
	"go-redis/internal/repository"
	"go-redis/internal/router"
	"go-redis/internal/service"
	"go-redis/internal/utils"
)

func Run() error {
	if err := config.InitConfig(); err != nil {
		return fmt.Errorf("load config fail: %w", err)
	}

	rdb, err := database.InitRedis(config.GlobalConfig.Redis)
	if err != nil {
		return fmt.Errorf("redis connection fail: %w", err)
	}

	db, err := database.InitMysql(config.GlobalConfig.MySQL)
	if err != nil {
		return fmt.Errorf("mysql connection fail: %w", err)
	}

	userRepo := repository.NewUserRepository(db)
	blogRepo := repository.NewBlogRepository(db)
	shopRepo := repository.NewShopRepository(db)
	voucherRepo := repository.NewVoucherRepository(db)
	bloomClient := utils.NewRedisBloomClient(rdb)

	userService := service.NewUserService(userRepo, rdb)
	blogService := service.NewBlogService(blogRepo, userRepo)
	shopService, err := service.NewShopService(shopRepo, rdb, bloomClient)
	if err != nil {
		return fmt.Errorf("init shop service fail: %w", err)
	}
	voucherService := service.NewVoucherService(voucherRepo, rdb)

	userHandler := handler.NewUserHandler(userService)
	blogHandler := handler.NewBlogHandler(blogService)
	shopHandler := handler.NewShopHandler(shopService)
	voucherHandler := handler.NewVoucherHandler(voucherService)

	shopService.SaveShopToRedis(context.Background(), 1, 30*60)

	r := gin.Default()
	router.SetupRouter(r, rdb, blogHandler, shopHandler, userHandler, voucherHandler)

	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	srv := &http.Server{Addr: addr, Handler: r}
	serverErrCh := make(chan error, 1)

	go func() {
		log.Printf("Server is running on %s ...\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serverErrCh:
		return fmt.Errorf("server start failed: %w", err)
	case <-quit:
		log.Println("Shutting down server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited.")
	return nil
}
