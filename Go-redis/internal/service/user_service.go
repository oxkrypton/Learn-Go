package service

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"go-redis/internal/constant"
	"go-redis/internal/dto"
	"go-redis/internal/model"
	"go-redis/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type UserService interface {
	// SendCode 生成并存储验证码到 Redis，返回生成的验证码（模拟短信）
	SendCode(ctx context.Context, phone string) (string, error)
	// Login 校验验证码 → 查找/创建用户 → 生成 Token → 存入 Redis
	Login(ctx context.Context, form dto.LoginFormDTO) (string, error)
	// LoginWithCode 保留原有方法用于查找或创建用户
	LoginWithCode(ctx context.Context, phone string) (*model.User, error)
	// QueryUserInfoById 根据用户ID查询用户详情（tb_user_info）
	QueryUserInfoById(ctx context.Context,userId uint64)(*model.UserInfo,error)

}

type userService struct {
	repo repository.UserRepository
	rdb  *redis.Client
}

func NewUserService(repo repository.UserRepository, rdb *redis.Client) UserService {
	return &userService{repo: repo, rdb: rdb}
}

func (s *userService) SendCode(ctx context.Context, phone string) (string, error) {
	//拼接redis key ，格式：login:code:{手机号}
	codeKey := constant.LoginCodeKey + phone

	//防刷限流，检查60秒内是否已经发送过验证码
	ttl, _ := s.rdb.TTL(ctx, codeKey).Result()
	// 如果 Key 还存在，且剩余时间 > 4分钟（说明距上次发送不足 60 秒）
	if ttl > 4*time.Minute {
		return "", fmt.Errorf("verification code sent too frequently")
	}

	//生成验证码
	code := generateVerifyCode()
	err := s.rdb.Set(ctx, codeKey, code, constant.LoginCodeTTL*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("failed to save verification code: %w", err)
	}
	return code, nil
}

// 登录处理
func (s *userService) Login(ctx context.Context, form dto.LoginFormDTO) (string, error) {
	//1.校验验证码
	codeKey := constant.LoginCodeKey + form.Phone
	savedCode, err := s.rdb.Get(ctx, codeKey).Result()
	if err != nil || savedCode != form.Code {
		return "", fmt.Errorf("verification code is incorrect or expired")
	}

	//2.调用Service进行登录or注册
	user, err := s.LoginWithCode(ctx, form.Phone)
	if err != nil {
		return "", err
	}

	// 3. 构建 UserDTO(脱敏，不反悔密码等信息)
	userDTO := dto.UserDTO{
		ID:       user.ID,
		Nickname: user.NickName,
		Icon:     user.Icon,
	}

	//4.踢掉旧token(单点登录)
	oldTokenKey := constant.LoginUserTokenKey + strconv.FormatUint(user.ID, 10)
	if oldToken, err := s.rdb.Get(ctx, oldTokenKey).Result(); err == nil {
		s.rdb.Del(ctx, constant.LoginTokenKey+oldToken)
	}

	//5.生成新token
	token := uuid.New().String()
	// 存入 Redis，Key 加前缀 login:token:
	tokenKey := constant.LoginTokenKey + token

	//6.将UserDTO转化为map(每个value都必须转为string)
	userMap := map[string]interface{}{
		"id":       strconv.FormatUint(userDTO.ID, 10),
		"nickname": userDTO.Nickname,
		"icon":     userDTO.Icon,
	}

	// 8. 使用 Pipeline 将多条 Redis 命令打包为一次网络往返
	pipe:=s.rdb.Pipeline()
	// 存入 Redis，使用HSet存入Map字段
	pipe.HSet(ctx, tokenKey, userMap)
	// 记录用户 ID -> Token 的映射，用于下次登录时踢掉旧 Token
	pipe.Set(ctx, oldTokenKey, token, constant.LoginTokenTTL*time.Minute)
	// hash结构本身在插入时不能带过期时间，需要单独调用expire设置
	pipe.Expire(ctx, tokenKey, constant.LoginTokenTTL*time.Minute)
	// 登录后吧验证码删除，防止重复使用
	pipe.Del(ctx, codeKey)

	// 9. 执行 Pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("fail to save login status to Redis: %w", err)
	}

	return token, nil
}

// 验证码登录核心逻辑
func (s *userService) LoginWithCode(ctx context.Context, phone string) (*model.User, error) {
	//1.查询数据库中是否存在该用户
	user, err := s.repo.QueryUserByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}

	//2.如果用户存在，直接返回
	if user != nil {
		return user, nil
	}

	newUser := &model.User{
		Phone:    phone,
		NickName: "user_" + generateRandomString(10),
	}
	err = s.repo.CreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}
	return newUser, nil
}

//QueryUserInfoById查询用户详情信息
func (s *userService) QueryUserInfoById(ctx context.Context,userId uint64)(*model.UserInfo,error){
	return s.repo.QueryUserInfoById(ctx,userId)
}

func generateRandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func generateVerifyCode() string {
	code := rand.Intn(999999) + 100000
	return strconv.Itoa(code)
}
