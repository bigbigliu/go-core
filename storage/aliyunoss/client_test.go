package aliyunoss

import "testing"

func TestObjectKey(t *testing.T) {
	tests := []struct {
		savePath string
		fileName string
		want     string
	}{
		{"images", "a.png", "images/a.png"},
		{"images/", "a.png", "images/a.png"},
		{"/images", "a.png", "images/a.png"},
		{"", "a.png", "a.png"},
	}
	for _, tt := range tests {
		if got := objectKey(tt.savePath, tt.fileName); got != tt.want {
			t.Errorf("objectKey(%q, %q) = %q, want %q", tt.savePath, tt.fileName, got, tt.want)
		}
	}
}

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(nil, "bucket"); err == nil {
		t.Error("NewClient(nil, bucket) expected error")
	}
	if _, err := NewClient(&AliyunOssUpload{}, ""); err == nil {
		t.Error("NewClient(uploader, empty bucket) expected error")
	}
}

func TestBucketCache(t *testing.T) {
	uploader := &AliyunOssUpload{AccessKeyId: "ak", AccessKeySecret: "sk"}
	first, err := uploader.bucket("test-bucket")
	if err != nil {
		t.Fatal(err)
	}
	second, err := uploader.bucket("test-bucket")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("bucket() should return the cached bucket")
	}
	if _, err := uploader.bucket(""); err == nil {
		t.Error("bucket() expected error for empty bucket name")
	}
}
