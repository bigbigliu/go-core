package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFileGreaterThan(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "data.bin")

	if err := os.WriteFile(filename, make([]byte, 2*1024*1024), 0o600); err != nil {
		t.Fatal(err)
	}

	greater, err := IsFileGreaterThan(filename, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !greater {
		t.Error("expected 2MB file to be greater than 1MB")
	}

	greater, err = IsFileGreaterThan(filename, 2)
	if err != nil {
		t.Fatal(err)
	}
	if greater {
		t.Error("expected 2MB file not to be greater than 2MB")
	}
}
