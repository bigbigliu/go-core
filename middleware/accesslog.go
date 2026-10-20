package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bigbigliu/go-core/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// maxLoggedBody 请求/响应日志中最多记录的 body 字节数，避免大响应占用内存
const maxLoggedBody = 64 * 1024

// responseBodyWriter 是一个自定义的 ResponseWriter，用于捕获响应体。
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 重写 Write 方法，将响应体写入缓冲区（超出上限后截断）
func (w responseBodyWriter) Write(b []byte) (int, error) {
	if remaining := maxLoggedBody - w.body.Len(); remaining > 0 {
		if len(b) > remaining {
			w.body.Write(b[:remaining])
		} else {
			w.body.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

// GinLogger gin 日志请求中间件
// skipPaths: 需要跳过的完全匹配路径列表
// skipKeywords: 需要跳过的包含关键字列表
func GinLogger(skipPaths []string, skipKeywords []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentPath := c.Request.URL.Path

		// 检查完全匹配的路径
		for _, skipPath := range skipPaths {
			if currentPath == skipPath {
				c.Next()
				return
			}
		}

		// 检查包含关键字的路径
		for _, keyword := range skipKeywords {
			if strings.Contains(currentPath, keyword) {
				c.Next()
				return
			}
		}

		if logger.Logger == nil {
			c.Next()
			return
		}

		var bodyBytes []byte
		contentType := c.Request.Header.Get("Content-Type")
		isJSON := strings.HasPrefix(contentType, "application/json")
		if isJSON {
			// 限制读取长度，避免超大请求体占用内存；剩余部分仍可被 handler 读取
			var err error
			bodyBytes, err = io.ReadAll(io.LimitReader(c.Request.Body, maxLoggedBody+1))
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to read request body.")
				return
			}
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes), c.Request.Body))
		}

		// 从 Gin 上下文中获取 Request ID
		requestIDStr, _ := c.Get(logger.RequestIDKey)
		requestID, _ := requestIDStr.(string)

		startTime := time.Now()

		// 创建 responseBodyWriter 替换原始的 ResponseWriter
		rbw := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = rbw

		c.Next()

		elapsedTime := time.Since(startTime)

		fields := []zap.Field{
			zap.String(logger.RequestIDKey, requestID),
			zap.String("X-Response-Time", fmt.Sprintf("%.2fms", float64(elapsedTime.Milliseconds()))),
			zap.String("http-method", c.Request.Method),
			zap.String("http-path", c.Request.URL.Path),
			zap.String("http-query-param", c.Request.URL.Query().Encode()),
			zap.Int("http-status", c.Writer.Status()),
			zap.String("http-remote-ip", c.ClientIP()),
			zap.String("http-response-body", truncate(rbw.body.String())),
			zap.String("handler", c.HandlerName()),
		}

		// 如果不是表单文件上传，记录 request-body
		if isJSON {
			fields = append(fields, zap.String("http-request-body", truncate(string(bodyBytes))))
		}

		// 记录错误信息
		if len(c.Errors) > 0 {
			errMessages := make([]string, 0, len(c.Errors))
			for _, e := range c.Errors {
				errMessages = append(errMessages, e.Error())
			}
			fields = append(fields, zap.Strings("errors", errMessages))
		}

		logger.Logger.Info("Request Handled", fields...)
	}
}

func truncate(s string) string {
	if len(s) <= maxLoggedBody {
		return s
	}
	return s[:maxLoggedBody] + "...(truncated)"
}
