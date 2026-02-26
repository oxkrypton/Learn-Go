package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-redis/internal/dto"
	"go-redis/internal/utils"
)

// 编写一个函数，比如 LoginInterceptor() gin.HandlerFunc。
func LoginInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		//获取session
		session := sessions.Default(c)
		
		//获取session中的user信息
		userObj := session.Get("user")
		
		//判断是否存在
		if userObj == nil {
			//不存在返回错误
			c.JSON(401, dto.Fail("not log in"))
			c.Abort()
			return
		}
		
		//存在
		userDTO, ok := userObj.(dto.UserDTO)
		if !ok {
			c.JSON(401, dto.Fail("abnormal login info"))
			c.Abort()
			return
		}
		
		//保存到gin.Context中
		utils.SaveUser(c, userDTO)

		//放行，继续执行后面对handler
		c.Next()
	}
}

