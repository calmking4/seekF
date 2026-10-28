-- 记录凭据校验失败次数，认证成功后清理对应账号与IP的失败记录。
if ARGV[1] == '0' then redis.call('DEL', KEYS[1]); return 0 end
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - tonumber(ARGV[2]))
redis.call('ZADD', KEYS[1], now, ARGV[3])
redis.call('PEXPIRE', KEYS[1], ARGV[2])
return 0
