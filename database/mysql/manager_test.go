package mysql

import (
	"errors"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

func TestManagerRegisterAndGet(t *testing.T) {
	manager := NewManager()
	db := &gorm.DB{}

	if err := manager.Register("primary", db); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got, err := manager.DB("primary"); err != nil || got != db {
		t.Fatalf("DB() = %v, %v", got, err)
	}
	if manager.Len() != 1 {
		t.Errorf("Len() = %d, want 1", manager.Len())
	}
	if names := manager.Names(); !reflect.DeepEqual(names, []string{"primary"}) {
		t.Errorf("Names() = %v", names)
	}

	if _, err := manager.DB("missing"); !errors.Is(err, ErrDBNotFound) {
		t.Errorf("DB(missing) error = %v, want ErrDBNotFound", err)
	}
	if err := manager.Register("primary", db); !errors.Is(err, ErrDBExists) {
		t.Errorf("Register(duplicate) error = %v, want ErrDBExists", err)
	}
	if err := manager.Register("", db); err == nil {
		t.Error("Register(empty name) expected error")
	}
	if err := manager.Register("nil", nil); err == nil {
		t.Error("Register(nil db) expected error")
	}
}

func TestManagerOpenValidation(t *testing.T) {
	manager := NewManager()
	if _, err := manager.Open("", &GenerateDSNParam{}); err == nil {
		t.Error("Open(empty name) expected error")
	}
	if _, err := manager.Open("primary", nil); err == nil {
		t.Error("Open(nil config) expected error")
	}
}

func TestManagerClose(t *testing.T) {
	manager := NewManager()
	// 零值 gorm.DB 无法提供底层 sql.DB，Close 会返回 ErrInvalidDB，
	// 但连接仍应从管理器中移除。
	_ = manager.Register("primary", &gorm.DB{})
	if err := manager.Close("primary"); !errors.Is(err, gorm.ErrInvalidDB) {
		t.Errorf("Close(primary) error = %v, want ErrInvalidDB", err)
	}
	if _, err := manager.DB("primary"); !errors.Is(err, ErrDBNotFound) {
		t.Errorf("DB(primary) error = %v, want ErrDBNotFound", err)
	}
	if err := manager.Close("missing"); !errors.Is(err, ErrDBNotFound) {
		t.Errorf("Close(missing) error = %v, want ErrDBNotFound", err)
	}
	if err := manager.CloseAll(); err != nil {
		t.Errorf("CloseAll() on empty manager error = %v", err)
	}
}

func TestManagerMustDBPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustDB() expected panic")
		}
	}()
	NewManager().MustDB("missing")
}
