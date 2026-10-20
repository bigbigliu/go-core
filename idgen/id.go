// Package idgen provides Snowflake-style numeric IDs and UUID helpers.
package idgen

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Snowflake ID 结构（毫秒时间戳 + 12 位序列号 + 10 位节点号），共 63 位，十进制输出。
// 时间戳基于 2024-01-01 UTC，可保证同一 Generator 内单调递增。
const (
	// EpochMillis 自定义时间戳起点：2024-01-01 00:00:00 UTC
	EpochMillis    int64 = 1704067200000
	NodeBits             = 10
	SequenceBits         = 12
	NodeMax        int64 = 1<<NodeBits - 1
	sequenceMask   int64 = 1<<SequenceBits - 1
	timestampShift       = NodeBits + SequenceBits
)

// ErrInvalidNodeID 节点 ID 超出 0..NodeMax 范围
var ErrInvalidNodeID = errors.New("idgen: invalid node id")

// Generator Snowflake ID 生成器。不同进程/实例必须使用不同的 NodeID。
type Generator struct {
	mu            sync.Mutex
	nodeID        int64
	lastTimestamp int64
	sequence      int64
}

// NewGenerator 创建指定节点 ID 的生成器。
// 分布式部署时，每个实例应分配唯一且固定的 nodeID（0..1023）。
func NewGenerator(nodeID int64) (*Generator, error) {
	if nodeID < 0 || nodeID > NodeMax {
		return nil, fmt.Errorf("%w: must be between 0 and %d", ErrInvalidNodeID, NodeMax)
	}
	return &Generator{nodeID: nodeID, lastTimestamp: -1}, nil
}

var defaultGenerator atomic.Pointer[Generator]

func init() {
	defaultGenerator.Store(newRandomGenerator())
}

// SetDefaultNodeID 设置默认生成器的节点 ID。
// 适用于多实例部署时通过配置/环境变量指定固定节点号。
func SetDefaultNodeID(nodeID int64) error {
	generator, err := NewGenerator(nodeID)
	if err != nil {
		return err
	}
	defaultGenerator.Store(generator)
	return nil
}

// GenUniversalID 使用默认生成器生成全局唯一的十进制 ID，同一进程内按时间递增。
func GenUniversalID() string {
	return defaultGenerator.Load().NextID()
}

// GenUniversalId 兼容旧命名。
//
// Deprecated: 使用 GenUniversalID。
func GenUniversalId() string {
	return GenUniversalID()
}

// NextID 生成十进制 ID。
func (g *Generator) NextID() string {
	return strconv.FormatInt(g.NextIDInt64(), 10)
}

// NextIDInt64 生成 int64 类型的 ID。
func (g *Generator) NextIDInt64() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli() - EpochMillis
	if now < 0 {
		now = 0
	}

	// 时钟回拨时沿用上一个时间戳，并消耗序列号
	if now < g.lastTimestamp {
		now = g.lastTimestamp
	}

	if now == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & sequenceMask
		if g.sequence == 0 {
			// 当前毫秒序列号用尽，等待时钟前进（避免 CPU 忙等）
			for {
				time.Sleep(time.Millisecond)
				next := time.Now().UnixMilli() - EpochMillis
				if next > g.lastTimestamp {
					now = next
					break
				}
			}
		}
	} else {
		g.sequence = 0
	}
	g.lastTimestamp = now

	return (now << timestampShift) | (g.sequence << NodeBits) | g.nodeID
}

// UUIDv7 生成时间有序的 UUID v7。适合不希望维护节点 ID 的分布式场景。
func UUIDv7() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// UUIDv4 生成随机 UUID v4。
func UUIDv4() string {
	return uuid.NewString()
}

func newRandomGenerator() *Generator {
	var buf [2]byte
	var nodeID int64
	if _, err := rand.Read(buf[:]); err == nil {
		nodeID = int64(binary.BigEndian.Uint16(buf[:])) & NodeMax
	}
	generator, _ := NewGenerator(nodeID)
	return generator
}
