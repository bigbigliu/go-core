package web_middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bigbigliu/go-core/pkgs"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const redisKeyPrefix = "ip_requests:"

// IPFilterWithRedisMiddleware 是一个基于 Redis 的滑动窗口限流中间件。
// 实现原理：
//   - 每次请求时，将当前毫秒时间戳作为 score 和 member 添加到 Redis 的 ZSET（有序集合）中。
//   - 先移除 ZSET 中窗口外（即当前时间减去 timeWindow 之前）的所有请求记录。
//   - 统计 ZSET 当前元素数量，即为当前时间窗口内的请求数。
//   - 若请求数超过 maxRequestsPerIP，则拒绝请求，否则允许通过。
//   - 所有 Redis 操作通过 Lua 脚本一次性原子完成，提升性能和一致性。
//   - 支持自定义超限响应消息，并在响应头中返回剩余可用请求数（X-RateLimit-Remaining）。
//
// 参数说明：
//
//	redisClient      Redis 客户端实例，用于存储和操作限流数据。
//	maxRequestsPerIP 单个 IP 在 timeWindow 时间窗口内允许的最大请求数。
//	timeWindow       滑动时间窗口长度，例如 1 秒、1 分钟等。
//	customMsg        （可选）自定义超限时的响应消息，未传递则使用默认提示。
func IPFilterWithRedisMiddleware(redisClient *redis.Client, maxRequestsPerIP int, timeWindow time.Duration, customMsg ...string) gin.HandlerFunc {
	// Lua脚本：原子性移除过期、添加当前、计数、设置过期
	luaScript := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local windowStart = tonumber(ARGV[2])
		local expireSec = tonumber(ARGV[3])
		redis.call('ZREMRANGEBYSCORE', key, 0, windowStart)
		redis.call('ZADD', key, now, now)
		local count = redis.call('ZCARD', key)
		redis.call('EXPIRE', key, expireSec)
		return count
	`
	return func(c *gin.Context) {
		ip := pkgs.GetRemoteIP(c)
		zsetKey := redisKeyPrefix + ip
		now := time.Now().UnixNano() / int64(time.Millisecond)
		windowStart := now - timeWindow.Milliseconds()
		expireSec := int(timeWindow.Seconds() * 2)

		// 执行Lua脚本
		res, err := redisClient.Eval(c, luaScript, []string{zsetKey},
			now, windowStart, expireSec).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": "-1",
				"msg":  "Internal server error",
			})
			c.Abort()
			return
		}

		count, _ := res.(int64)
		remaining := maxRequestsPerIP - int(count)
		if remaining < 0 {
			remaining = 0
		}
		// 设置剩余请求数到响应头
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if int(count) > maxRequestsPerIP {
			msg := "Too many requests"
			if len(customMsg) > 0 {
				msg = customMsg[0]
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code": "-1",
				"msg":  msg,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
