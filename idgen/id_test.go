package idgen

import (
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestGenUniversalIdConcurrent(t *testing.T) {
	// 设置并发测试的协程数
	numGoroutines := 10

	// 使用 WaitGroup 等待所有协程完成
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 用于存储生成的 UUID
	ids := make(map[string]struct{})
	var mutex sync.Mutex // 用于保护 ids 的互斥锁

	// 启动并发协程生成 UUID
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			uuid := GenUniversalID()

			// 通过互斥锁保护对 ids 的并发访问
			mutex.Lock()
			defer mutex.Unlock()

			// 判断是否有重复的 UUID
			if _, exists := ids[uuid]; exists {
				t.Errorf("发现重复的 UUID: %s", uuid)
			}

			// 打印生成的 UUID
			//t.Log("uuid: ", uuid)

			// 将 UUID 加入到 ids 中
			ids[uuid] = struct{}{}
		}()
	}

	// 等待所有协程完成
	wg.Wait()
}

func TestGenUniversalIDMonotonic(t *testing.T) {
	previous := int64(-1)
	for i := 0; i < 1000; i++ {
		id, err := strconv.ParseInt(GenUniversalID(), 10, 64)
		if err != nil {
			t.Fatalf("ParseInt(%q) error = %v", GenUniversalID(), err)
		}
		if id <= previous {
			t.Fatalf("id %d is not greater than previous %d", id, previous)
		}
		previous = id
	}
}

func TestNewGeneratorValidation(t *testing.T) {
	if _, err := NewGenerator(-1); !errors.Is(err, ErrInvalidNodeID) {
		t.Errorf("NewGenerator(-1) error = %v, want ErrInvalidNodeID", err)
	}
	if _, err := NewGenerator(NodeMax + 1); !errors.Is(err, ErrInvalidNodeID) {
		t.Errorf("NewGenerator(NodeMax+1) error = %v, want ErrInvalidNodeID", err)
	}
	if _, err := NewGenerator(0); err != nil {
		t.Errorf("NewGenerator(0) error = %v", err)
	}
	if _, err := NewGenerator(NodeMax); err != nil {
		t.Errorf("NewGenerator(NodeMax) error = %v", err)
	}
}

func TestGeneratorMonotonic(t *testing.T) {
	generator, err := NewGenerator(7)
	if err != nil {
		t.Fatal(err)
	}
	previous := int64(-1)
	for i := 0; i < 1000; i++ {
		id := generator.NextIDInt64()
		if id <= previous {
			t.Fatalf("id %d is not greater than previous %d", id, previous)
		}
		previous = id
	}
}

func TestSetDefaultNodeID(t *testing.T) {
	old := defaultGenerator.Load()
	defer defaultGenerator.Store(old)

	if err := SetDefaultNodeID(NodeMax + 1); !errors.Is(err, ErrInvalidNodeID) {
		t.Errorf("SetDefaultNodeID() error = %v, want ErrInvalidNodeID", err)
	}
	if err := SetDefaultNodeID(42); err != nil {
		t.Fatalf("SetDefaultNodeID(42) error = %v", err)
	}
	if got := defaultGenerator.Load().NextIDInt64() & NodeMax; got != 42 {
		t.Errorf("node id = %d, want 42", got)
	}
}

func TestUUIDv7(t *testing.T) {
	value, err := UUIDv7()
	if err != nil {
		t.Fatalf("UUIDv7() error = %v", err)
	}
	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if id.Version() != 7 {
		t.Errorf("version = %d, want 7", id.Version())
	}
}

func TestUUIDv4(t *testing.T) {
	id, err := uuid.Parse(UUIDv4())
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if id.Version() != 4 {
		t.Errorf("version = %d, want 4", id.Version())
	}
}
