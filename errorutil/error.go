package errorutil

import (
	"errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// IsNoRowFoundError gorm 'record not found' 错误处理
func IsNoRowFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsRedisNilError redis 'redis: nil' 错误处理
func IsRedisNilError(err error) bool {
	return errors.Is(err, redis.Nil)
}
