package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/bigbigliu/go-core/netutil"
	"github.com/bigbigliu/go-core/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "ip_requests:"

var rateLimitSeq atomic.Uint64

// ipRateLimitScript 原子地清理过期记录、写入当前请求并统计窗口内请求数。
// KEYS[1]: zset key
// ARGV[1]: 当前时间戳(score)
// ARGV[2]: 窗口起始时间戳
// ARGV[3]: key 过期时间(秒)
// ARGV[4]: 当前请求的唯一 member
var ipRateLimitScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local windowStart = tonumber(ARGV[2])
local expireSec = tonumber(ARGV[3])
local member = ARGV[4]
redis.call('ZREMRANGEBYSCORE', key, 0, windowStart)
redis.call('ZADD', key, now, member)
local count = redis.call('ZCARD', key)
redis.call('EXPIRE', key, expireSec)
return count
`)

// IPFilterWithRedisMiddleware 是一个基于 Redis 的滑动窗口限流中间件。
// 实现原理：
//   - 每次请求时，将当前毫秒时间戳作为 score、唯一值作为 member 添加到 Redis 的 ZSET。
//   - 先移除 ZSET 中窗口外的请求记录，再统计当前窗口内的请求数。
//   - 若请求数超过 maxRequestsPerIP，则拒绝请求。
//   - 所有 Redis 操作通过 Lua 脚本一次性原子完成。
//   - 响应头 X-RateLimit-Remaining 返回剩余可用请求数。
//
// 参数说明：
//
//	redisClient      Redis 客户端实例，用于存储和操作限流数据。
//	maxRequestsPerIP 单个 IP 在 timeWindow 时间窗口内允许的最大请求数。
//	timeWindow       滑动时间窗口长度，例如 1 秒、1 分钟等。
//	customMsg        （可选）自定义超限时的响应消息。
func IPFilterWithRedisMiddleware(redisClient *redis.Client, maxRequestsPerIP int, timeWindow time.Duration, customMsg ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redisClient == nil {
			c.JSON(http.StatusInternalServerError, response.FailMsg("redis未初始化"))
			c.Abort()
			return
		}
		if timeWindow <= 0 {
			timeWindow = time.Second
		}

		ip := netutil.GetRemoteIP(c)
		zsetKey := redisKeyPrefix + ip
		now := time.Now().UnixMilli()
		windowStart := now - timeWindow.Milliseconds()
		expireSec := int(timeWindow.Seconds() * 2)
		if expireSec < 1 {
			expireSec = 1
		}
		// 同一毫秒内的多个请求必须使用不同 member，否则会被 ZADD 覆盖导致计数偏小
		member := fmt.Sprintf("%d-%d", now, rateLimitSeq.Add(1))

		res, err := ipRateLimitScript.Run(c.Request.Context(), redisClient, []string{zsetKey},
			now, windowStart, expireSec, member).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, response.FailMsg("Internal server error"))
			c.Abort()
			return
		}

		count, ok := res.(int64)
		if !ok {
			count = 0
		}
		remaining := maxRequestsPerIP - int(count)
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if int(count) > maxRequestsPerIP {
			msg := "Too many requests"
			if len(customMsg) > 0 {
				msg = customMsg[0]
			}
			c.JSON(http.StatusTooManyRequests, response.FailMsg(msg))
			c.Abort()
			return
		}

		c.Next()
	}
}
