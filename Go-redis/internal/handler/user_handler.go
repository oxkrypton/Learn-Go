package handler

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-redis/internal/dto"
	"go-redis/internal/utils"
)

// UserHandler 用户相关的路由处理
type UserHandler struct {
}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) SendCode(c *gin.Context) {
	//获取请求参数中的手机号(？phone=123)
	phone := c.Query("phone")
	if phone == "" {
		//防止前端是用表单或json中传参，做一下兼容
		phone = c.PostForm("phone")
	}

	//验证手机号格式
	if !utils.IsValidPhone(phone) {
		//返回前台统一格式，复用 dto.Fail
		c.JSON(200, dto.Fail("手机格式不正确"))
		return
	}

	//生成验证码
	code := utils.GenerateVerifyCode()

	//将验证码存入session
	session := sessions.Default(c)
	//使用带手机号前缀的key存入session，防止不同手机号冲突
	session.Set("code_"+phone, code)
	err := session.Save()
	if err != nil {
		c.JSON(200, dto.Fail("验证码发送失败"))
		return
	}

	fmt.Printf("【模拟短信发送】发送短消息成功，手机号: %s, 您的验证码为: %s\n", phone, code)

	c.JSON(200, dto.Success("验证码发送成功"))
}
