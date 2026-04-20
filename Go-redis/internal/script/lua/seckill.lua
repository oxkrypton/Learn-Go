local stockKey = KEYS[1]
local orderKey = KEYS[2]
local userId = ARGV[1]

-- 判断库存是否充足，不足返回 1
local stock = tonumber(redis.call('GET', stockKey))
if not stock or stock <= 0 then
    return 1
end

-- 若用户已下单，返回 2
if redis.call('SISMEMBER', orderKey, userId) == 1 then
    return 2
end

-- 通过校验后，原子扣库存并记录一人一单
redis.call('INCRBY', stockKey, -1)
redis.call('SADD', orderKey, userId)

-- 抢购资格成功，返回 0
return 0
