if redis.call('HEXISTS', KEYS[1],ARGV[1]) == 0 then
    return -1
end

local counter = redis.call('HINCRBY', KEYS[1], ARGV[1], -1)
if counter > 0 then
    redis.call('PEXPIRE', KEYS[1], ARGV[2])
    return 1
end

redis.call('DEL', KEYS[1])
return 0
