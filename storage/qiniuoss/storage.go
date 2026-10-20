package qiniuoss

import (
	"context"
	"errors"
	"io"

	"github.com/bigbigliu/go-core/storage"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	qiniustorage "github.com/qiniu/go-sdk/v7/storage"
)

// Client adapts QiNiuOssUpload to the common storage.Storage interface for a fixed bucket.
type Client struct {
	uploader *QiNiuOssUpload
	bucket   string
}

// NewClient creates a Storage client bound to bucket.
func NewClient(uploader *QiNiuOssUpload, bucket string) (*Client, error) {
	if uploader == nil {
		return nil, errors.New("qiniuoss: uploader 不能为空")
	}
	if bucket == "" {
		return nil, errors.New("qiniuoss: bucket 不能为空")
	}
	return &Client{uploader: uploader, bucket: bucket}, nil
}

// Put uploads the content read from r.
func (c *Client) Put(ctx context.Context, key string, r io.Reader, size int64) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if size < 0 {
		return "", storage.ErrSizeRequired
	}

	mac := qbox.NewMac(c.uploader.AccessKey, c.uploader.SecretKey)
	cfg := c.uploader.storageConfig()
	upToken := c.uploader.uploadToken(c.bucket, mac)
	uploader := qiniustorage.NewFormUploader(&cfg)
	ret := qiniustorage.PutRet{}
	if err := uploader.Put(ctx, &ret, upToken, key, r, size, &qiniustorage.PutExtra{}); err != nil {
		return "", err
	}
	if ret.Hash == "" {
		return "", errors.New("上传文件失败")
	}
	return key, nil
}

// PutFile uploads a local file.
func (c *Client) PutFile(ctx context.Context, key, localPath string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	mac := qbox.NewMac(c.uploader.AccessKey, c.uploader.SecretKey)
	cfg := c.uploader.storageConfig()
	upToken := c.uploader.uploadToken(c.bucket, mac)
	uploader := qiniustorage.NewFormUploader(&cfg)
	ret := qiniustorage.PutRet{}
	if err := uploader.PutFile(ctx, &ret, upToken, key, localPath, &qiniustorage.PutExtra{}); err != nil {
		return "", err
	}
	if ret.Hash == "" {
		return "", errors.New("上传文件失败")
	}
	return key, nil
}

// Delete removes an object.
func (c *Client) Delete(_ context.Context, key string) error {
	mac := qbox.NewMac(c.uploader.AccessKey, c.uploader.SecretKey)
	cfg := c.uploader.storageConfig()
	return qiniustorage.NewBucketManager(mac, &cfg).Delete(c.bucket, key)
}

var _ storage.Storage = (*Client)(nil)
