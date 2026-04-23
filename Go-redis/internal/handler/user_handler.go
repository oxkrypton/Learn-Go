package handler

import (
	"go-redis/internal/dto"
	"go-redis/internal/pkg/bizerr"
	"go-redis/internal/pkg/ginx"
	"go-redis/internal/pkg/validator"
	"go-redis/internal/service"
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
	if !validator.IsValidPhone(phone) {
		c.JSON(http.StatusOK, dto.Fail("invalid phone format"))
		return
	}

	//生成验证码
	code, err := h.userService.SendCode(c.Request.Context(), phone)
	if err != nil {
		log.Printf("[UserHandler] SendCode err: %v", err)
		if bizerr.Is(err) {
			c.JSON(http.StatusOK, dto.Fail(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Fail("internal server error"))
		return
	}

	log.Printf("[UserHandler] verification code sent, phone=%s code=%s", phone, code)
	c.JSON(http.StatusOK, dto.Success("verification code sent successfully"))
}

func (h *UserHandler) Login(c *gin.Context) {
	//1.接受前端JSON参数
	var loginForm dto.LoginFormDTO
	if err := c.ShouldBindJSON(&loginForm); err != nil {
		c.JSON(http.StatusOK, dto.Fail("invalid parameters"))
		return
	}

	//2.校验手机号
	if !validator.IsValidPhone(loginForm.Phone) {
		c.JSON(http.StatusOK, dto.Fail("invalid phone format"))
		return
	}

	token, err := h.userService.Login(c.Request.Context(), loginForm)
	if err != nil {
		log.Printf("[UserHandler] Login err: %v", err)
		if bizerr.Is(err) {
			c.JSON(http.StatusOK, dto.Fail(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Fail("internal server error"))
		return
	}

	c.JSON(http.StatusOK, dto.Success(token))
}

func (h *UserHandler) Me(c *gin.Context) {
	user, exists := ginx.GetUser(c)
	if !exists {
		c.JSON(http.StatusOK, dto.Fail("user not logged in"))
		return
	}
	c.JSON(http.StatusOK, dto.Success(user))
}

// QueryUserInfo 处理 GET /user/info/:id 请求
// 该路由需要登录认证，查询指定用户的详细信息（tb_user_info）
func (h *UserHandler) QueryUserInfo(c *gin.Context) {
	//1.解析路径参数中的用户id
	idStr := c.Param("id")
	userID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail("invalid user id"))
		return
	}

	//2.调用service查询用户详情
	userInfo, err := h.userService.QueryUserInfoById(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[UserHandler] QueryUserInfo err: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail("internal server error"))
		return
	}

	//3.返回结果（如果 userInfo 为 nil，前端会收到 data:null，页面已处理此情况）
	c.JSON(http.StatusOK, dto.Success(userInfo))
}
