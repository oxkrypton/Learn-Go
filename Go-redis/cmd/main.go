package main

import (
	"fmt"
	"go-redis/internal/config"
	"go-redis/internal/pkg/database"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	//初始化配置
	err := config.InitConfig()
	if err != nil {
		panic(fmt.Sprintf("load config fail: %v", err))
	}

	//初始化redis
	err = database.InitRedis()
	if err != nil {
		panic(fmt.Sprintf("redis connection fail: %v", err))
	}

	//初始化mysql
	if err := database.InitMysql(); err != nil {
		panic(fmt.Sprintf("mysql connection fail: %v", err))
	}

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {

		database.RDB.Set(c, "string_key", "Hello Redis", 0)
		strVal, _ := database.RDB.Get(c, "string_key").Result()

		database.RDB.HSet(c, "hash_key", "name", "krypton", "age", 21)
		hashVal, _ := database.RDB.HGetAll(c, "hash_key").Result()

		database.RDB.RPush(c, "list_key", "item1", "item2", "item3")
		listVal, _ := database.RDB.LRange(c, "list_key", 0, -1).Result()

		database.RDB.SAdd(c, "set_key", "member1", "member2", "member1")
		setVal, _ := database.RDB.SMembers(c, "set_key").Result()

		database.RDB.ZAdd(c, "zset_key", redis.Z{Score: 100, Member: "user1"}, redis.Z{Score: 90, Member: "user2"})
		zsetVal, _ := database.RDB.ZRevRange(c, "zset_key", 0, -1).Result()

		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "pong",
			"data": gin.H{
				"string": strVal,
				"hash":   hashVal,
				"list":   listVal,
				"set":    setVal,
				"zset":   zsetVal,
			},
		})
	})
	r.Run(":8080")
}
