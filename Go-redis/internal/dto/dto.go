package dto

type Result struct {
	Code int         `json:"code"` //状态码
	Msg  string      `json:"msg"`  //提示信息
	Data interface{} `json:"data"` //数据载体
}

type UserDTO struct {
	ID       uint64 `json:"id"`
	Nickname string `json:"nick_name"`
	Icon     string `json:"icon"`
}

func Success(data interface{}) *Result {
	return &Result{
		Code: 200,
		Msg:  "success",
		Data: data,
	}
}

func Fail(msg string) *Result {
	return &Result{
		Code: 500,
		Msg:  msg,
		Data: nil,
	}
}
