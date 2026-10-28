-- 验证码发送限流：检查账号发送冷却时间，以及账号和IP的固定窗口发送次数。
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local wait = math.max(0, redis.call('PTTL', KEYS[1]))
local windows = {86400000, 60000, 86400000}
local counts = {}
local epochs = {}
for i = 1, 3 do
 local epoch = math.floor(now / windows[i])
 local row = redis.call('HMGET', KEYS[i + 1], 'epoch', 'count')
 local count = 0
 if tonumber(row[1]) == epoch then count = tonumber(row[2]) or 0 end
 counts[i] = count
 epochs[i] = epoch
 if count >= tonumber(ARGV[i + 1]) then wait = math.max(wait, (epoch + 1) * windows[i] - now) end
end
if wait > 0 then return wait end
redis.call('SET', KEYS[1], '1', 'PX', ARGV[1])
for i = 1, 3 do
 redis.call('HSET', KEYS[i + 1], 'epoch', epochs[i], 'count', counts[i] + 1)
 redis.call('PEXPIRE', KEYS[i + 1], (epochs[i] + 1) * windows[i] - now)
end
return 0
