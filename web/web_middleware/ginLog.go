package web_middleware

import (
	"bytes"
	"fmt"
	"github.com/bigbigliu/go-core/logger"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// responseBodyWriter 是一个自定义的 ResponseWriter，用于捕获响应体。
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// GinLogger gin 日志请求中间件
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		sugarLogger := logger.Logger

		c.Set("zapLogger", sugarLogger)
		var bodyBytes []byte
		var err error

		// 检查请求的 Content-Type 是否为 multipart/form-data
		contentType := c.Request.Header.Get("Content-Type")
		if contentType == "application/json" {
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to read request body.")
				return
			}

			// 重新设置请求体，以便后续处理程序可以访问
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 从 Gin 上下文中获取 Request ID
		requestID, exists := c.Get(logger.RequestIDKey)
		var requestIDStr string
		if exists {
			requestIDStr, _ = requestID.(string)
		}

		startTime := time.Now()

		// 创建 responseBodyWriter 替换原始的 ResponseWriter
		rbw := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = rbw

		c.Next()

		endTime := time.Now()
		elapsedTime := endTime.Sub(startTime)
		responseTimeMs := float64(elapsedTime.Milliseconds())

		// 请求结束后记录请求信息
		fields := []zap.Field{
			zap.String(logger.RequestIDKey, requestIDStr),
			zap.String("X-Response-Time", fmt.Sprintf("%.2fms", responseTimeMs)),
			zap.String("http-method", c.Request.Method),
			zap.String("http-path", c.Request.URL.Path),
			zap.String("http-query-param", c.Request.URL.Query().Encode()),
			zap.Int("http-status", c.Writer.Status()),
			zap.String("http-remote-ip", c.ClientIP()),
			zap.String("http-response-body", rbw.body.String()),
		}

		// 如果不是表单文件上传，记录 request-body
		if contentType == "application/json" {
			fields = append(fields, zap.Any("http-request-body", string(bodyBytes)))
		}

		// 记录错误信息
		if len(c.Errors) > 0 {
			errMessages := []string{}
			for _, e := range c.Errors {
				errMessages = append(errMessages, e.Error())
			}
			fields = append(fields, zap.Strings("errors", errMessages))
		}

		// 记录HandlerName
		fields = append(fields, zap.String("handler", c.HandlerName()))

		sugarLogger.Info("Request Handled", fields...)
	}
}
