package main

import (
	"go-redis/internal/config"
	"go-redis/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	err := config.InitConfig()
	if err != nil {
		panic("load config fail :" + err.Error())
	}

	err = repository.InitRedis()
	if err != nil {
		panic("Redis connection fail :" + err.Error())
	}

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		repository.RDB.Set(c, "test_kry", "Hello Redis", 0)
		val, _ := repository.RDB.Get(c, "test_key").Result()

		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "pong",
			"data": val,
		})
	})
	r.Run(":8080")
}
