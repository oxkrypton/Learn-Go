package handler

import (
	"fmt"
	"go-redis/internal/dto"
	"go-redis/internal/pkg/database"
	"go-redis/internal/service"
	"go-redis/internal/utils"
	"log"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	code := utils.GenerateVerifyCode()

	//将验证码存入session
	session := sessions.Default(c)
	//使用带手机号前缀的key存入session，防止不同手机号冲突
	session.Set("code_"+phone, code)
	err := session.Save()
	if err != nil {
		c.JSON(200, dto.Fail("Failed to send verification code"))
		return
	}

	fmt.Printf("【模拟短信发送】发送短消息成功，手机号: %s, 您的验证码为: %s\n", phone, code)

	c.JSON(200, dto.Success("Verification code sent successfully"))
}

//登录处理
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

	//3.校验验证码
	session := sessions.Default(c)
	savedCode := session.Get("code_" + loginDTO.Phone)

	if savedCode == nil || fmt.Sprintf("%v", savedCode) != loginDTO.Code {
		c.JSON(200, dto.Fail("Verification code is incorrect or expired"))
		return
	}

	//4.一致调用Service进行登录or注册
	user, err := h.userService.LoginWithCode(c, loginDTO.Phone)
	if err != nil {
		log.Println("登录失败：", err)
		c.JSON(200, dto.Fail("System exception, login failed"))
		return
	}

	//5.登录成功，将用户信息存入session(脱敏，不反悔密码等信息)
	userDTO := dto.UserDTO{
		ID:       user.ID,
		Nickname: user.NickName,
		Icon:     user.Icon,
	}

	//6.使用uuid生成随机token，作为登录凭证
	token := uuid.New().String()

	//7.将UserDTO转化为map(每个value都必须转为string)
	userMap := map[string]interface{}{
		"id":       strconv.FormatUint(userDTO.ID, 10), //将uint64转位string
		"nickname": userDTO.Nickname,
		"icon":     userDTO.Icon,
	}

	// 8. 存入 Redis，Key 加前缀 login:token:
	tokenKey := "login:token:" + token

	//9.使用HMSet批量存入Map字段
	err = database.RDB.HMSet(c, tokenKey, userMap).Err()
	if err != nil {
		c.JSON(200, dto.Fail("Fail to save login status to Redis"))
		return
	}

	//10.hash结构本身在插入时不能带过期时间，需要单独调用expire设置
	database.RDB.Expire(c, tokenKey, 30*time.Minute)

	//登录后吧验证码删除，防止重复使用
	session.Delete("code_" + loginDTO.Phone)
	session.Save()

	//11.将token返回前端，把token放入success的参数中
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
