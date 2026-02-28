package middleware

import (
	"encoding/json"
	"time"
	"go-redis/internal/dto"
	"go-redis/internal/pkg/database"
	"go-redis/internal/utils"
	"github.com/gin-gonic/gin"
)

// 编写一个函数，比如 LoginInterceptor() gin.HandlerFunc。
func LoginInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		//1.从header中获取token
		token:=c.GetHeader("authorization")

		if token==""{
			c.JSON(401,dto.Fail("not log in"))
			c.Abort()
			return
		}

		//2.拼接token
		tokenKey:="login:token:"+token

		//3.从redis获取token
		userBytes,err:=database.RDB.Get(c,tokenKey).Bytes()
		if err!=nil{
			c.JSON(200,dto.Fail("abnormal login info or token expired"))
			c.Abort()
			return
		}

		//4.将token的JSON字符串反序列化为userDTO
		var userDTO dto.UserDTO
		if err:=json.Unmarshal(userBytes,&userDTO);err!=nil{
			c.JSON(200,dto.Fail("server internal error"))
			c.Abort()
			return
		}

		//5.将保存的用户信息存入gin.context，供后续handler获取
		utils.SaveUser(c,userDTO)

		//6状态延续，刷新redis中该token的有效期
		database.RDB.Expire(c,tokenKey,30*time.Minute)

		//7.放行
		c.Next()
	}
}

