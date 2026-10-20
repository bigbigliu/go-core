package aliyunoss

import (
	"context"
	"errors"
	"io"

	corestorage "github.com/bigbigliu/go-core/storage"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Client adapts AliyunOssUpload to the common storage.Storage interface for a fixed bucket.
type Client struct {
	uploader *AliyunOssUpload
	bucket   string
}

// NewClient creates a Storage client bound to bucket.
func NewClient(uploader *AliyunOssUpload, bucket string) (*Client, error) {
	if uploader == nil {
		return nil, errors.New("aliyunoss: uploader 不能为空")
	}
	if bucket == "" {
		return nil, errors.New("aliyunoss: bucket 不能为空")
	}
	return &Client{uploader: uploader, bucket: bucket}, nil
}

// Put uploads the content read from r. When size >= 0 it is sent as Content-Length.
func (c *Client) Put(_ context.Context, key string, r io.Reader, size int64) (string, error) {
	bucket, err := c.uploader.bucket(c.bucket)
	if err != nil {
		return "", err
	}
	key = trimKey(key)
	options := make([]oss.Option, 0, 1)
	if size >= 0 {
		options = append(options, oss.ContentLength(size))
	}
	if err := bucket.PutObject(key, r, options...); err != nil {
		return "", err
	}
	return key, nil
}

// PutFile uploads a local file.
func (c *Client) PutFile(_ context.Context, key, localPath string) (string, error) {
	bucket, err := c.uploader.bucket(c.bucket)
	if err != nil {
		return "", err
	}
	key = trimKey(key)
	if err := bucket.PutObjectFromFile(key, localPath); err != nil {
		return "", err
	}
	return key, nil
}

// Delete removes an object.
func (c *Client) Delete(_ context.Context, key string) error {
	bucket, err := c.uploader.bucket(c.bucket)
	if err != nil {
		return err
	}
	return bucket.DeleteObject(trimKey(key))
}

var _ corestorage.Storage = (*Client)(nil)
