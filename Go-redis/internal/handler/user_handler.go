package handler

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-redis/internal/dto"
	"go-redis/internal/service"
	"go-redis/internal/utils"
	"log"
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

func (h *UserHandler) Login(c *gin.Context) {
	//1.接受前端JSON参数
	var loginDTO dto.LoginFormDTO
	if err := c.ShouldBindJSON(&loginDTO); err != nil {
		c.JSON(200, dto.Fail("参数格式错误"))
		return
	}

	//2.校验手机号
	if !utils.IsValidPhone(loginDTO.Phone) {
		c.JSON(200, dto.Fail("手机格式不正确"))
		return
	}

	//3.校验验证码
	session := sessions.Default(c)
	savedCode := session.Get("code_" + loginDTO.Code)

	if savedCode == nil {
		c.JSON(200, dto.Fail("验证码错误或已过期"))
		return
	}

	//4.一致调用Service进行登录or注册
	user, err := h.userService.LoginWithCode(c, loginDTO.Phone)
	if err != nil {
		log.Println("登录失败：", err)
		c.JSON(200, dto.Fail("系统异常，登录失败"))
		return
	}

	//5.登录成功，将用户信息存入session(脱敏，不反悔密码等信息)
	userDTO := dto.UserDTO{
		ID:       user.ID,
		Nickname: user.NickName,
		Icon:     user.Icon,
	}

	//6.将UserDTO存入session，表示用户已登录
	session.Set("user", userDTO)
	if err := session.Save(); err != nil {
		c.JSON(200, dto.Fail("登录状态保存失败"))
		return
	}

	//登录后吧验证码删除，防止重复使用
	session.Delete("code_" + loginDTO.Phone)
	session.Save()

	//7.返回登录成功信息
	c.JSON(200, dto.Success("登录成功"))
}
