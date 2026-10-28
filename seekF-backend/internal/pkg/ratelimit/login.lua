-- 登录与验证码校验限流：检查账号和IP的滑动窗口请求数及凭据失败次数。
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local windows = {60000, 60000, tonumber(ARGV[3])}
local limits = {tonumber(ARGV[1]), tonumber(ARGV[2]), tonumber(ARGV[4])}
local wait = 0
for i = 1, 3 do
 redis.call('ZREMRANGEBYSCORE', KEYS[i], '-inf', now - windows[i])
 if redis.call('ZCARD', KEYS[i]) >= limits[i] then
 local first = redis.call('ZRANGE', KEYS[i], 0, 0, 'WITHSCORES')
 wait = math.max(wait, tonumber(first[2]) + windows[i] - now)
 end
end
if wait > 0 then return wait end
for i = 1, 2 do
 redis.call('ZADD', KEYS[i], now, ARGV[5])
 redis.call('PEXPIRE', KEYS[i], windows[i])
end
return 0
