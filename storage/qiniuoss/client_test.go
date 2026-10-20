package qiniuoss

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bigbigliu/go-core/storage"
)

func TestObjectKey(t *testing.T) {
	if got := objectKey("images/", "a.png"); got != "images/a.png" {
		t.Errorf("objectKey() = %q", got)
	}
	if got := localObjectKey("images", "", "/tmp/a.png"); got != "images/a.png" {
		t.Errorf("localObjectKey() = %q", got)
	}
	if got := localObjectKey("", "", "/tmp/a.png"); got != "a.png" {
		t.Errorf("localObjectKey() = %q", got)
	}
}

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(nil, "bucket"); err == nil {
		t.Error("NewClient(nil, bucket) expected error")
	}
	if _, err := NewClient(&QiNiuOssUpload{}, ""); err == nil {
		t.Error("NewClient(uploader, empty bucket) expected error")
	}
}

func TestPutRequiresSize(t *testing.T) {
	client, err := NewClient(&QiNiuOssUpload{}, "bucket")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Put(context.Background(), "a.txt", strings.NewReader("x"), -1); !errors.Is(err, storage.ErrSizeRequired) {
		t.Errorf("Put() error = %v, want ErrSizeRequired", err)
	}
}
