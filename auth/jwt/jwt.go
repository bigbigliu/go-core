package jwt

import (
	"errors"
	"time"

	goJwt "github.com/golang-jwt/jwt/v5"
)

// ICoreJwt JWT 接口
type ICoreJwt interface {
	NewToken(username string) (string, error)
	ParseToken(token string) (*Claims, error)
}

// Claims 自定义 claims
type Claims struct {
	Username string `json:"username"`
	goJwt.RegisteredClaims
}

// CoreJWT JWT 实现
type CoreJWT struct {
	Secret  string `json:"secret"`  // jwt密钥
	Timeout int    `json:"timeout"` // jwt过期时间(秒)
}

// NewToken 生成新token
func (c *CoreJWT) NewToken(username string) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(time.Duration(c.Timeout) * time.Second)

	claims := Claims{
		Username: username,
		RegisteredClaims: goJwt.RegisteredClaims{
			Issuer:    "core_os",                        // 发签人
			ExpiresAt: goJwt.NewNumericDate(expireTime), // 过期时间
			NotBefore: goJwt.NewNumericDate(nowTime),    // 生效时间
			IssuedAt:  goJwt.NewNumericDate(nowTime),    // 签发时间
		},
	}

	t := goJwt.NewWithClaims(goJwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(c.Secret))
}

// ParseToken 解析token，并限制只接受 HS256 签名。
func (c *CoreJWT) ParseToken(token string) (*Claims, error) {
	claims := &Claims{}
	tokenClaims, err := goJwt.ParseWithClaims(token, claims, func(token *goJwt.Token) (interface{}, error) {
		return []byte(c.Secret), nil
	}, goJwt.WithValidMethods([]string{goJwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if tokenClaims == nil || !tokenClaims.Valid {
		return nil, errors.New("token 无效")
	}
	return claims, nil
}
