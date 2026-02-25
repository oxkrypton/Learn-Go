package utils

import (
	"math/rand"
	"regexp"
	"strconv"
	"time"
)

// 检验手机格式是否正确
func IsValidPhone(phone string) bool {
	regex := `^1[3-9]\d{9}$`
	match, _ := regexp.MatchString(regex, phone)
	return match
}

// 生成六位验证码
func GenerateVerifyCode() string {
	rand.Seed(time.Now().UnixNano())
	code := rand.Intn(999999)
	return strconv.Itoa(code)
}
