package validator

import "regexp"

// IsValidPhone 检验手机格式是否正确
func IsValidPhone(phone string) bool {
	regex := `^1[3-9]\d{9}$`
	match, _ := regexp.MatchString(regex, phone)
	return match
}
