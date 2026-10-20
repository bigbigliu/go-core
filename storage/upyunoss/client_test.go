package upyunoss

import "testing"

func TestLocalObjectKey(t *testing.T) {
	if got := localObjectKey("images/", "a.png", "/tmp/b.png"); got != "images/a.png" {
		t.Errorf("localObjectKey() = %q", got)
	}
	if got := localObjectKey("images", "", "/tmp/a.png"); got != "images/a.png" {
		t.Errorf("localObjectKey() = %q", got)
	}
}

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(nil, "bucket"); err == nil {
		t.Error("NewClient(nil, bucket) expected error")
	}
	if _, err := NewClient(&UpYunOssUpload{}, ""); err == nil {
		t.Error("NewClient(uploader, empty bucket) expected error")
	}
}

func TestUpYunCache(t *testing.T) {
	uploader := &UpYunOssUpload{Operator: "op", Password: "pwd"}
	first, err := uploader.upYun("test-bucket")
	if err != nil {
		t.Fatal(err)
	}
	second, err := uploader.upYun("test-bucket")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("upYun() should return the cached client")
	}
	if _, err := uploader.upYun(""); err == nil {
		t.Error("upYun() expected error for empty bucket name")
	}
}
