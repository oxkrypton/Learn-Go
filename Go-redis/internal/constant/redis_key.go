package constant

const(
	//验证码Key前缀：login:code:{phone}
	LoginCodeKey="login:code:"
	//Token—>用户信息HashKey前缀：login:token:{token}
	LoginTokenKey="login:token:"
	//用户ID->Token映射Key前缀：login:user:token:{userId}
	LoginUserTokenKey="login:user:token:"
	// Token 有效期 (分钟)
    LoginTokenTTL = 30
    // 验证码有效期 (分钟)
    LoginCodeTTL = 5
)