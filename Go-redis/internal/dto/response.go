package dto

// Result 统一 HTTP 响应信封
type Result struct {
	Success  bool        `json:"success"`
	ErrorMsg string      `json:"errMsg,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Total    int64       `json:"total,omitempty"`
}

// UserDTO脱敏后的用户信息（用于token存储和应用）
type UserDTO struct {
	ID       uint64 `json:"id"`
	Nickname string `json:"nickName"`
	Icon     string `json:"icon"`
}

func Success(data interface{}) *Result {
	return &Result{Success: true, Data: data}
}

func Fail(errorMsg string) *Result {
	return &Result{Success: false, ErrorMsg: errorMsg}
}
