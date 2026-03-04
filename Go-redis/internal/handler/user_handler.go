package handler

import (
	"fmt"
	"go-redis/internal/dto"
	"go-redis/internal/service"
	"go-redis/internal/utils"
	"log"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户相关的路由处理
type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
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
		c.JSON(200, dto.Fail("Invalid phone format"))
		return
	}

	//生成验证码
	code, err := h.userService.SendCode(c.Request.Context(), phone)
	if err != nil {
		c.JSON(200, dto.Fail(err.Error()))
		return
	}

	fmt.Printf("【模拟短信发送】发送短消息成功，手机号: %s, 您的验证码为: %s\n", phone, code)
	c.JSON(200, dto.Success("Verification code sent successfully"))
}

// 登录处理
func (h *UserHandler) Login(c *gin.Context) {
	//1.接受前端JSON参数
	var loginDTO dto.LoginFormDTO
	if err := c.ShouldBindJSON(&loginDTO); err != nil {
		c.JSON(200, dto.Fail("Invalid parameters format"))
		return
	}

	//2.校验手机号
	if !utils.IsValidPhone(loginDTO.Phone) {
		c.JSON(200, dto.Fail("Invalid phone format"))
		return
	}

	token, err := h.userService.Login(c.Request.Context(), loginDTO)
	if err != nil {
		log.Println("登录失败", err)
		c.JSON(200, dto.Fail(err.Error()))
		return
	}

	//将token返回前端，把token放入success的参数中
	c.JSON(200, dto.Success(token))
}

func (h *UserHandler) Me(c *gin.Context) {
	user, exists := utils.GetUser(c)
	if !exists {
		c.JSON(200, dto.Fail("用户未登录"))
		return
	}
	c.JSON(200, dto.Success(user))
}
