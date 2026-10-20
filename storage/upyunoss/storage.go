package upyunoss

import (
	"context"
	"errors"
	"io"
	"strconv"

	"github.com/bigbigliu/go-core/storage"
	"github.com/upyun/go-sdk/v3/upyun"
)

// Client adapts UpYunOssUpload to the common storage.Storage interface for a fixed bucket.
type Client struct {
	uploader *UpYunOssUpload
	bucket   string
}

// NewClient creates a Storage client bound to bucket.
func NewClient(uploader *UpYunOssUpload, bucket string) (*Client, error) {
	if uploader == nil {
		return nil, errors.New("upyunoss: uploader 不能为空")
	}
	if bucket == "" {
		return nil, errors.New("upyunoss: bucket 不能为空")
	}
	return &Client{uploader: uploader, bucket: bucket}, nil
}

// Put uploads the content read from r. When size >= 0 it is sent as Content-Length.
func (c *Client) Put(_ context.Context, key string, r io.Reader, size int64) (string, error) {
	client, err := c.uploader.upYun(c.bucket)
	if err != nil {
		return "", err
	}
	headers := map[string]string{}
	if size >= 0 {
		headers["Content-Length"] = strconv.FormatInt(size, 10)
	}
	if err := client.Put(&upyun.PutObjectConfig{Path: key, Reader: r, Headers: headers}); err != nil {
		return "", err
	}
	return key, nil
}

// PutFile uploads a local file.
func (c *Client) PutFile(_ context.Context, key, localPath string) (string, error) {
	client, err := c.uploader.upYun(c.bucket)
	if err != nil {
		return "", err
	}
	if err := client.Put(&upyun.PutObjectConfig{Path: key, LocalPath: localPath}); err != nil {
		return "", err
	}
	return key, nil
}

// Delete removes an object.
func (c *Client) Delete(_ context.Context, key string) error {
	client, err := c.uploader.upYun(c.bucket)
	if err != nil {
		return err
	}
	return client.Delete(&upyun.DeleteObjectConfig{Path: key})
}

var _ storage.Storage = (*Client)(nil)
