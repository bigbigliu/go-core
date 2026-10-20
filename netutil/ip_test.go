package netutil

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext(remoteAddr string, headers map[string]string) *gin.Context {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	return c
}

func TestGetRemoteIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "forwarded for list",
			remoteAddr: "10.0.0.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 10.0.0.1"},
			want:       "203.0.113.1",
		},
		{
			name:       "real ip",
			remoteAddr: "10.0.0.1:1234",
			headers:    map[string]string{"X-Real-IP": "198.51.100.7"},
			want:       "198.51.100.7",
		},
		{
			name:       "ipv6 forwarded for",
			remoteAddr: "10.0.0.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "2001:db8::1"},
			want:       "2001:db8::1",
		},
		{
			name:       "remote addr fallback",
			remoteAddr: "192.0.2.1:5678",
			headers:    nil,
			want:       "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRemoteIP(newTestContext(tt.remoteAddr, tt.headers)); got != tt.want {
				t.Errorf("GetRemoteIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
