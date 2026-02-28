package middleware

import (
	"github.com/gin-gonic/gin"
	"go-redis/internal/dto"
	"go-redis/internal/pkg/database"
	"go-redis/internal/utils"
	"strconv"
	"time"
)

// 编写一个函数，比如 LoginInterceptor() gin.HandlerFunc。
func LoginInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		//1.从header中获取token
		token := c.GetHeader("authorization")

		if token == "" {
			c.JSON(401, dto.Fail("not log in"))
			c.Abort()
			return
		}

		//2.拼接token
		tokenKey := "login:token:" + token

		//3.从redis的Hash结构中查出所有的键值对(返回的是map[string]string)
		userMap, err := database.RDB.HGetAll(c, tokenKey).Result()
		if err != nil {
			c.JSON(200, dto.Fail("abnormal login info or token expired"))
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

		//在获取常规字符串字段
		userDTO.Nickname = userMap["nickname"]
		userDTO.Icon = userMap["icon"]

		//5.将保存的用户信息存入gin.context，供后续handler获取
		utils.SaveUser(c, userDTO)

		//6状态延续，刷新redis中该token的有效期(30min)
		database.RDB.Expire(c, tokenKey, 30*time.Minute)

		//7.放行
		c.Next()
	}
}
