package dto

type Result struct {
	Success  bool        `json:"success"`
	ErrorMsg string      `json:"errorMsg,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Total    int64       `json:"total,omitempty"`
}

type UserDTO struct {
	ID       uint64 `json:"id"`
	Nickname string `json:"nick_name"`
	Icon     string `json:"icon"`
}

func Success(data interface{}) *Result {
	return &Result{
		Success: true,
		Data:    data,
	}
}

func Fail(errorMsg string) *Result {
	return &Result{
		Success:  false,
		ErrorMsg: errorMsg,
	}
}
