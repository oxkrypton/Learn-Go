package middleware

import (
	"go-redis/internal/constant"
	"go-redis/internal/dto"
	"go-redis/internal/pkg/ginx"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// LoginInterceptor 接收 redis.Client 作为参数
func LoginInterceptor(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1.从header中获取token
		token := c.GetHeader("authorization")
		if token == "" {
			c.JSON(200, dto.Fail("user not logged in"))
			c.Abort()
			return
		}

		//2.拼接token
		tokenKey := constant.LoginTokenKey + token

		//3.从redis的Hash结构中查出所有的键值对(返回的是map[string]string)
		userMap, err := rdb.HGetAll(c, tokenKey).Result()
		if err != nil {
			log.Printf("[LoginInterceptor] query login state err: %v", err)
			c.JSON(500, dto.Fail("internal server error"))
			c.Abort()
			return
		}
		if len(userMap) == 0 {
			c.JSON(200, dto.Fail("login expired"))
			c.Abort()
			return
		}

		//4.将map中的字段，手动转换并装配回真正的UserDTO结构
		var userDTO dto.UserDTO

		//先获取id字段，并转换为uint64
		if idStr, exists := userMap["id"]; exists {
			id, _ := strconv.ParseUint(idStr, 10, 64)
			userDTO.ID = id
		}

		//再获取常规字符串字段
		userDTO.Nickname = userMap["nickname"]
		userDTO.Icon = userMap["icon"]

		//5.将保存的用户信息存入gin.context，供后续handler获取
		ginx.SaveUser(c, userDTO)

		//6状态延续，刷新redis中该token的有效期(30min)
		rdb.Expire(c, tokenKey, constant.LoginTokenTTL*time.Minute)

		//7.放行
		c.Next()
	}
}
