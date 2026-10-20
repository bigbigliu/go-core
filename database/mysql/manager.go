package mysql

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"gorm.io/gorm"
)

var (
	// ErrDBNotFound 指定的数据库名称不存在
	ErrDBNotFound = errors.New("mysql: database not found")
	// ErrDBExists 数据库名称已注册
	ErrDBExists = errors.New("mysql: database already exists")
)

// Manager 管理多个命名数据库连接，适用于分库或读写分离场景。
// 连接数量不受限制，可以按业务需要注册任意多个命名连接。
// Manager 本身并发安全，底层 *gorm.DB 也可并发使用。
type Manager struct {
	mu  sync.RWMutex
	dbs map[string]*gorm.DB
}

// DefaultManager 默认的多数据库管理器，可直接使用。
var DefaultManager = NewManager()

// NewManager 创建多数据库管理器。
func NewManager() *Manager {
	return &Manager{dbs: make(map[string]*gorm.DB)}
}

// Open 创建并注册一个命名数据库连接。
// 名称已存在时返回 ErrDBExists，且不会覆盖已有连接。
func (m *Manager) Open(name string, cfg *GenerateDSNParam) (*gorm.DB, error) {
	if name == "" {
		return nil, errors.New("mysql: name 不能为空")
	}
	if cfg == nil {
		return nil, errors.New("mysql: config 不能为空")
	}

	db, err := cfg.Open()
	if err != nil {
		return nil, err
	}
	if err := m.Register(name, db); err != nil {
		_ = closeGormDB(db)
		return nil, err
	}
	return db, nil
}

// Register 注册一个已创建的数据库连接，主要用于测试或自定义连接池。
func (m *Manager) Register(name string, db *gorm.DB) error {
	if name == "" {
		return errors.New("mysql: name 不能为空")
	}
	if db == nil {
		return errors.New("mysql: db 不能为空")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.dbs[name]; ok {
		return fmt.Errorf("%w: %s", ErrDBExists, name)
	}
	m.dbs[name] = db
	return nil
}

// DB 获取命名数据库连接。
func (m *Manager) DB(name string) (*gorm.DB, error) {
	m.mu.RLock()
	db, ok := m.dbs[name]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrDBNotFound, name)
	}
	return db, nil
}

// MustDB 获取命名数据库连接，不存在时 panic。
func (m *Manager) MustDB(name string) *gorm.DB {
	db, err := m.DB(name)
	if err != nil {
		panic(err)
	}
	return db
}

// Names 返回所有已注册的数据库名称（按字典序排序）。
func (m *Manager) Names() []string {
	m.mu.RLock()
	names := make([]string, 0, len(m.dbs))
	for name := range m.dbs {
		names = append(names, name)
	}
	m.mu.RUnlock()
	sort.Strings(names)
	return names
}

// Len 返回已注册的数据库数量。
func (m *Manager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.dbs)
}

// Close 关闭并移除指定数据库连接。
func (m *Manager) Close(name string) error {
	m.mu.Lock()
	db, ok := m.dbs[name]
	if ok {
		delete(m.dbs, name)
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrDBNotFound, name)
	}
	return closeGormDB(db)
}

// CloseAll 关闭并移除所有数据库连接。
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	dbs := m.dbs
	m.dbs = make(map[string]*gorm.DB)
	m.mu.Unlock()

	var errs []error
	for _, db := range dbs {
		if err := closeGormDB(db); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func closeGormDB(db *gorm.DB) error {
	if db == nil || db.Config == nil {
		return gorm.ErrInvalidDB
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
