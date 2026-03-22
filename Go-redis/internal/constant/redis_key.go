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
	//商铺缓存Key前缀：cache:shop:{id}
	CacheShopKey = "cache:shop:"
	//商铺缓存过期时间
	CacheShopTTL = 30
	//空值缓存过期时间
	CacheNilTTL = 3
	//布隆过滤器Key
	BloomFilterShopIdsKey = "bloom:shop:ids"
	// 布隆过滤器默认误判率
	BloomFilterErrorRate = 0.01
	// 布隆过滤器默认预估容量
	BloomFilterCapacity = 100000
	//商铺互斥锁Key前缀：lock:shop:{id}
	LockShopKey = "lock:shop"
	//商铺互斥锁过期时间（秒）
	LockShopTTL = 10
	// 热点商铺缓存Key前缀（逻辑过期）：cache:shop:hot:{id}
	CacheHotShopKey = "cache:shop:hot:"
)
