package handler

import (
	"fmt"
	"go-redis/internal/dto"
	"go-redis/internal/service"
	"go-redis/internal/utils"
	"log"
	"net/http"
	"strconv"

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

// QueryUserInfo 处理 GET /user/info/:id 请求
// 该路由需要登录认证，查询指定用户的详细信息（tb_user_info）
func (h *UserHandler) QueryUserInfo(c *gin.Context) {
	//1.解析路径参数中的用户id
	idStr := c.Param("id")
	userId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail("UserID agrs wrong"))
		return
	}

	//2.调用service查询用户详情
	userInfo, err := h.userService.QueryUserInfoById(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail("Query UserInfo fail"))
		return
	}

	//3.返回结果（如果 userInfo 为 nil，前端会收到 data:null，页面已处理此情况）
	c.JSON(http.StatusOK, dto.Success(userInfo))
}
