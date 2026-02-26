package utils

import (
	"github.com/gin-gonic/gin"
	"go-redis/internal/dto"
)

// 定义一个常量作为 key，例如 const UserKey = "user"。
const UserKey = "user"

// 保存用户：SaveUser(c *gin.Context, user dto.UserDTO)，内部调用 c.Set。
func SaveUser(c *gin.Context, user dto.UserDTO) {
	c.Set(UserKey, user)
}

// 获取用户：GetUser(c *gin.Context) (dto.UserDTO, bool)，内部调用 c.Get，并将其强转断言为 dto.UserDTO。
func GetUser(c *gin.Context) (dto.UserDTO, bool) {
	value, exists := c.Get(UserKey)
	if !exists {
		return dto.UserDTO{}, false
	}
	user, ok := value.(dto.UserDTO)
	return user, ok
}
