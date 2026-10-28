-- AI请求令牌桶限流：按时间补充令牌，每次放行消耗一个，令牌不足时返回等待时间。
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local rate = tonumber(ARGV[1]) / 60000
local capacity = tonumber(ARGV[2])
local state = redis.call('HMGET', KEYS[1], 'tokens', 'at')
local tokens = capacity
if state[1] then tokens = math.min(capacity, tonumber(state[1]) + math.max(0, now - tonumber(state[2])) * rate) end
local wait = 0
if tokens >= 1 then tokens = tokens - 1 else wait = math.ceil((1 - tokens) / rate) end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'at', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(capacity / rate) + 1000)
return wait
