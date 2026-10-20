package jwt

import (
	"testing"

	goJwt "github.com/golang-jwt/jwt/v5"
)

func TestNewTokenAndParseToken(t *testing.T) {
	coreJWT := &CoreJWT{Secret: "test-secret", Timeout: 60}

	token, err := coreJWT.NewToken("alice")
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}

	claims, err := coreJWT.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.Username != "alice" {
		t.Errorf("username = %q, want alice", claims.Username)
	}

	wrong := &CoreJWT{Secret: "other-secret", Timeout: 60}
	if _, err := wrong.ParseToken(token); err == nil {
		t.Error("ParseToken() expected error with wrong secret")
	}
}

func TestParseTokenRejectsNoneAlgorithm(t *testing.T) {
	coreJWT := &CoreJWT{Secret: "test-secret", Timeout: 60}

	token := goJwt.NewWithClaims(goJwt.SigningMethodNone, Claims{Username: "alice"})
	signed, err := token.SignedString(goJwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := coreJWT.ParseToken(signed); err == nil {
		t.Error("ParseToken() must reject alg=none")
	}
}
