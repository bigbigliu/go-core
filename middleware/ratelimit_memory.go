package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bigbigliu/go-core/netutil"
	"github.com/bigbigliu/go-core/response"
	"github.com/gin-gonic/gin"
)

// ipCounter 用于记录单个 IP 在固定时间窗口内的请求计数
type ipCounter struct {
	count       int
	windowStart time.Time
}

// MemoryLimiter 基于内存的固定窗口限流器。
// 使用 NewMemoryLimiter 创建，进程退出或不再使用时调用 Close 释放清理 goroutine。
type MemoryLimiter struct {
	maxRequests int
	window      time.Duration

	mu       sync.Mutex
	counters map[string]*ipCounter

	stop      chan struct{}
	closeOnce sync.Once
}

// NewMemoryLimiter 创建内存限流器并启动过期数据清理 goroutine。
func NewMemoryLimiter(maxRequestsPerIP int, timeWindow time.Duration) *MemoryLimiter {
	if timeWindow <= 0 {
		timeWindow = time.Second
	}
	limiter := &MemoryLimiter{
		maxRequests: maxRequestsPerIP,
		window:      timeWindow,
		counters:    make(map[string]*ipCounter),
		stop:        make(chan struct{}),
	}
	go limiter.cleanupLoop()
	return limiter
}

// Close 停止清理 goroutine，可重复调用。
func (l *MemoryLimiter) Close() {
	l.closeOnce.Do(func() {
		close(l.stop)
	})
}

// Middleware 返回限流中间件。
func (l *MemoryLimiter) Middleware() gin.HandlerFunc {
	return l.handle
}

// IPFilterMiddleware 中间件用于过滤IP并限制请求频率（固定窗口）。
// 内部创建一个长期运行的 MemoryLimiter；需要显式释放时请使用 NewMemoryLimiter。
func IPFilterMiddleware(maxRequestsPerIP int, timeWindow time.Duration) gin.HandlerFunc {
	return NewMemoryLimiter(maxRequestsPerIP, timeWindow).Middleware()
}

func (l *MemoryLimiter) handle(c *gin.Context) {
	ip := netutil.GetRemoteIP(c)
	now := time.Now()

	l.mu.Lock()
	counter, exists := l.counters[ip]
	if !exists || now.Sub(counter.windowStart) >= l.window {
		counter = &ipCounter{windowStart: now}
		l.counters[ip] = counter
	}
	counter.count++
	count := counter.count
	l.mu.Unlock()

	remaining := l.maxRequests - count
	if remaining < 0 {
		remaining = 0
	}
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

	if count > l.maxRequests {
		c.JSON(http.StatusTooManyRequests, response.FailMsg("Too many requests"))
		c.Abort()
		return
	}

	c.Next()
}

func (l *MemoryLimiter) cleanupLoop() {
	interval := l.window
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case now := <-ticker.C:
			l.mu.Lock()
			for ip, counter := range l.counters {
				if now.Sub(counter.windowStart) >= l.window {
					delete(l.counters, ip)
				}
			}
			l.mu.Unlock()
		}
	}
}
