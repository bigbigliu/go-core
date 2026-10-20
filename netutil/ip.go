package netutil

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRemoteIP 获取客户端 IP。
//
// 注意：X-Real-IP 和 X-Forwarded-For 可被客户端伪造，只有在服务位于可信代理之后
// 并且代理会覆盖这些请求头时才应依赖它们。更推荐配置 gin 的 SetTrustedProxies 后
// 直接使用 c.ClientIP()。
func GetRemoteIP(c *gin.Context) string {
	if ip := firstValidIP(c.Request.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}
	if ip := firstValidIP(c.Request.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return host
	}
	return c.ClientIP()
}

func firstValidIP(header string) string {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if host, _, err := net.SplitHostPort(part); err == nil {
			part = host
		}
		if net.ParseIP(part) != nil {
			return part
		}
	}
	return ""
}
