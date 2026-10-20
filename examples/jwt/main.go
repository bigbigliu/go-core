package main

import (
	"fmt"
	"os"

	"github.com/bigbigliu/go-core/auth/jwt"
)

func main() {
	coreJWT := &jwt.CoreJWT{
		Secret:  "example-secret-change-me",
		Timeout: 3600,
	}

	token, err := coreJWT.NewToken("alice")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("token: %s\n", token)

	claims, err := coreJWT.ParseToken(token)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("username: %s, expires at: %v\n", claims.Username, claims.ExpiresAt)

	// TokenVerify 还需要 Redis 中保存 token（key: coreos:<username>:accesstoken），
	// 参见 middleware 示例或 README。
}
