package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPRequestGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.Header.Get("X-Test") != "1" {
			t.Errorf("missing header X-Test")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	body, status, err := HTTPRequest(server.URL, http.MethodGet, "", map[string]string{"X-Test": "1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || string(body) != "ok" {
		t.Fatalf("status=%d body=%q", status, body)
	}
}

func TestClientPostJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if payload["name"] != "alice" {
			t.Errorf("payload = %v", payload)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := New()
	body, status, err := client.PostJSON(context.Background(), server.URL, nil, map[string]interface{}{"name": "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("status=%d body=%s", status, body)
	}
}

func TestHTTPRequestWithCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if _, _, err := HTTPRequestWithContext(ctx, server.URL, http.MethodGet, "", nil, nil); err == nil {
		t.Fatal("expected context deadline error")
	}
}

func TestStructToQueryParams(t *testing.T) {
	type params struct {
		Name string `queryparam:"name"`
		Age  int    `queryparam:"age"`
		Skip string `queryparam:"-"`
	}

	query, err := StructToQueryParams(params{Name: "alice", Age: 0, Skip: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if query != "name=alice" {
		t.Errorf("query = %q, want %q", query, "name=alice")
	}
}

func TestPostForm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if r.PostForm.Get("key") != "value" {
			t.Errorf("form = %v", r.PostForm)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := New()
	body, status, err := client.PostForm(context.Background(), server.URL, nil, url.Values{"key": []string{"value"}})
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || string(body) != "ok" {
		t.Fatalf("status=%d body=%q", status, body)
	}
}

func TestFormMapBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if r.PostForm.Get("key") != "value" {
			t.Errorf("form = %v", r.PostForm)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := New()
	_, status, err := client.Do(context.Background(), http.MethodPost, server.URL,
		"application/x-www-form-urlencoded", nil, map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
}

func TestResponseTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer server.Close()

	client := New(WithMaxResponseSize(10))
	if _, _, err := client.Get(context.Background(), server.URL, nil); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("error = %v, want ErrResponseTooLarge", err)
	}
}

func TestRetryOnServerError(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := New(WithRetry(2), WithRetryWait(time.Millisecond))
	body, status, err := client.Get(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || string(body) != "ok" {
		t.Fatalf("status=%d body=%q", status, body)
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestDefaultUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != DefaultUserAgent {
			t.Errorf("User-Agent = %q, want %q", got, DefaultUserAgent)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	if _, _, err := New().Get(context.Background(), server.URL, nil); err != nil {
		t.Fatal(err)
	}
}
