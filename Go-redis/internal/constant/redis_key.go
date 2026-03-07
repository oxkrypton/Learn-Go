package constant

const (
	//验证码Key前缀：login:code:{phone}
	LoginCodeKey = "login:code:"
	//Token—>用户信息HashKey前缀：login:token:{token}
	LoginTokenKey = "login:token:"
	//用户ID->Token映射Key前缀：login:user:token:{userId}
	LoginUserTokenKey = "login:user:token:"
	// Token 有效期 (分钟)
	LoginTokenTTL = 30
	// 验证码有效期 (分钟)
	LoginCodeTTL = 5
	//商铺列表缓存Key前缀：cache:shop-type:list
	CacheShopTypeListKey = "cache:shop-type:list"
	//商铺列表缓存过期时间
	CacheShopTypeListTTL = 30
	// 商铺缓存Key前缀：cache:shop:{id}
	CacheShopKey = "cache:shop:"
	//商铺缓存过期时间
	CacheShopTTL = 30
)
