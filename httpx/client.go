package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

const (
	// DefaultUserAgent 默认 User-Agent
	DefaultUserAgent = "go-core/httpx"
	// DefaultMaxResponseSize 默认最大响应体大小（10 MiB）
	DefaultMaxResponseSize int64 = 10 << 20
)

// ErrResponseTooLarge 响应体超过限制时返回
var ErrResponseTooLarge = errors.New("httpx: response body exceeds limit")

// Client 可配置的 HTTP 客户端，复用底层连接池并支持重试。
type Client struct {
	httpClient      *http.Client
	userAgent       string
	maxResponseSize int64
	maxRetries      int
	retryWait       time.Duration
}

// Option 客户端配置项
type Option func(*Client)

// New 创建 HTTP 客户端。
// 默认超时 30 秒、响应体上限 10 MiB、User-Agent 为 go-core/httpx、不重试。
func New(opts ...Option) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: newTransport(),
		},
		userAgent:       DefaultUserAgent,
		maxResponseSize: DefaultMaxResponseSize,
		retryWait:       100 * time.Millisecond,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

// WithTimeout 设置请求超时时间。
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.httpClient.Timeout = d
		}
	}
}

// WithHTTPClient 使用自定义的 http.Client。
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithUserAgent 设置默认 User-Agent。
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithMaxResponseSize 设置响应体最大字节数，<=0 表示不限制。
func WithMaxResponseSize(size int64) Option {
	return func(c *Client) {
		c.maxResponseSize = size
	}
}

// WithRetry 设置幂等请求（GET/HEAD/OPTIONS/PUT/DELETE）的最大重试次数。
func WithRetry(n int) Option {
	return func(c *Client) {
		if n < 0 {
			n = 0
		}
		c.maxRetries = n
	}
}

// WithRetryWait 设置重试间隔。
func WithRetryWait(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.retryWait = d
		}
	}
}

// WithInsecureSkipVerify 跳过 TLS 证书校验。仅用于自签名证书的测试环境，
// 生产环境不要使用。
func WithInsecureSkipVerify() Option {
	return func(c *Client) {
		transport, ok := c.httpClient.Transport.(*http.Transport)
		if !ok || transport == nil {
			transport = newTransport()
		} else {
			transport = transport.Clone()
		}
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec // 显式选择跳过校验
		c.httpClient.Transport = transport
	}
}

// HTTPClient 返回底层 http.Client，便于高级用法。
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// Do 发起请求并返回响应体与状态码。5xx/429 对幂等请求会按配置重试。
func (c *Client) Do(ctx context.Context, method, urlStr, contentType string, headers map[string]string, body any) ([]byte, int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if method == "" {
		method = http.MethodGet
	}

	payload, err := encodeRequestBody(contentType, body)
	if err != nil {
		return nil, 0, err
	}

	idempotent := isIdempotent(method)
	attempts := c.maxRetries + 1
	var lastErr error
	var lastStatus int
	var lastBody []byte

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return lastBody, lastStatus, ctx.Err()
			case <-time.After(c.retryWait):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, urlStr, bytes.NewReader(payload))
		if err != nil {
			return nil, 0, err
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		if contentType != "" && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", contentType)
		}
		if c.userAgent != "" && req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", c.userAgent)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if idempotent && attempt+1 < attempts && ctx.Err() == nil {
				continue
			}
			return nil, 0, err
		}

		data, readErr := readLimited(resp.Body, c.maxResponseSize)
		closeErr := resp.Body.Close()
		lastStatus, lastBody = resp.StatusCode, data
		if readErr != nil {
			return data, resp.StatusCode, readErr
		}
		if closeErr != nil {
			return data, resp.StatusCode, closeErr
		}
		if shouldRetryStatus(resp.StatusCode) && idempotent && attempt+1 < attempts {
			lastErr = fmt.Errorf("httpx: server returned status %d", resp.StatusCode)
			continue
		}
		return data, resp.StatusCode, nil
	}

	if lastErr == nil {
		lastErr = errors.New("httpx: request failed")
	}
	return lastBody, lastStatus, lastErr
}

// Get 发起 GET 请求。
func (c *Client) Get(ctx context.Context, urlStr string, headers map[string]string) ([]byte, int, error) {
	return c.Do(ctx, http.MethodGet, urlStr, "", headers, nil)
}

// PostJSON 发起 JSON POST 请求。
func (c *Client) PostJSON(ctx context.Context, urlStr string, headers map[string]string, body any) ([]byte, int, error) {
	return c.Do(ctx, http.MethodPost, urlStr, "application/json", headers, body)
}

// PostForm 发起表单 POST 请求。
func (c *Client) PostForm(ctx context.Context, urlStr string, headers map[string]string, form url.Values) ([]byte, int, error) {
	return c.Do(ctx, http.MethodPost, urlStr, "application/x-www-form-urlencoded", headers, form)
}

// StructToQueryParams 将结构体字段绑定到 URL 参数。
// 字段可通过 `queryparam` tag 指定参数名，tag 为 "-" 时忽略该字段。
func (c *Client) StructToQueryParams(input any) (string, error) {
	return StructToQueryParams(input)
}

// StructToQueryParams 将结构体字段绑定到 URL 参数。
func StructToQueryParams(input any) (string, error) {
	values := url.Values{}
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", errors.New("httpx: input must not be nil")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", errors.New("httpx: input must be a struct or a pointer to a struct")
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Tag.Get("queryparam")
		if fieldName == "-" {
			continue
		}
		if fieldName == "" {
			fieldName = field.Name
		}
		value := v.Field(i)
		if !value.CanInterface() {
			continue
		}
		// 保持原有行为：int 类型为 0 时不参与查询
		if value.Kind() == reflect.Int && value.Int() == 0 {
			continue
		}
		values.Add(fieldName, strings.TrimSpace(fmt.Sprintf("%v", value.Interface())))
	}
	return values.Encode(), nil
}

// HTTPRequest 使用默认客户端发起请求。
func HTTPRequest(urlStr, method, contentType string, headers map[string]string, body any) ([]byte, int, error) {
	return defaultClient.Do(context.Background(), method, urlStr, contentType, headers, body)
}

// HTTPRequestWithContext 使用默认客户端发起请求，支持 context。
func HTTPRequestWithContext(ctx context.Context, urlStr, method, contentType string, headers map[string]string, body any) ([]byte, int, error) {
	return defaultClient.Do(ctx, method, urlStr, contentType, headers, body)
}

var defaultClient = New()

func newTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
}

func encodeRequestBody(contentType string, body any) ([]byte, error) {
	switch v := body.(type) {
	case nil:
		return nil, nil
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	case io.Reader:
		return io.ReadAll(v)
	case url.Values:
		return []byte(v.Encode()), nil
	}

	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		return encodeFormBody(body)
	}
	return json.Marshal(body)
}

func encodeFormBody(body any) ([]byte, error) {
	values := url.Values{}
	switch v := body.(type) {
	case map[string]string:
		for key, value := range v {
			values.Set(key, value)
		}
	case map[string][]string:
		for key, val := range v {
			values[key] = val
		}
	default:
		return nil, fmt.Errorf("httpx: unsupported form body type %T", body)
	}
	return []byte(values.Encode()), nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(r)
	}
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrResponseTooLarge
	}
	return data, nil
}

func isIdempotent(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func shouldRetryStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
