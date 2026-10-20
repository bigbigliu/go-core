package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/bigbigliu/go-core/httpx"
)

func main() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
		})
	}))
	defer server.Close()

	client := httpx.New(
		httpx.WithTimeout(2*time.Second),
		httpx.WithRetry(1),
		httpx.WithUserAgent("httpx-example/1.0"),
	)

	ctx := context.Background()
	body, status, err := client.Get(ctx, server.URL+"/hello", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("GET status=%d body=%s\n", status, body)

	body, status, err = client.PostJSON(ctx, server.URL+"/users", nil, map[string]string{"name": "alice"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("POST status=%d body=%s\n", status, body)
}
