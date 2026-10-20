package aliyunoss

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

const defaultEndpoint = "https://oss-cn-hangzhou.aliyuncs.com"

// IAliyunOssUpload 阿里云oss上传方法
type IAliyunOssUpload interface {
	// UploadResourceByte 上传Byte数组
	UploadResourceByte(param *UploadResourceByteReq) (path string, err error)
	// UploadLocalFile 上传本地文件
	UploadLocalFile(param *UploadLocalFileReq) (path string, err error)
	// DeleteFile 删除文件
	DeleteFile(param *DeleteFileParam) error
}

// AliyunOssUpload 阿里云 OSS 上传实现
type AliyunOssUpload struct {
	AccessKeyId     string `json:"AccessKeyId"`     // AccessKeyId
	AccessKeySecret string `json:"AccessKeySecret"` // AccessKeySecret
	Endpoint        string `json:"Endpoint"`        // Endpoint 为空时使用杭州默认地址

	mu      sync.Mutex
	client  *oss.Client
	buckets map[string]*oss.Bucket
}

// UploadResourceByte 上传Byte数组
func (h *AliyunOssUpload) UploadResourceByte(param *UploadResourceByteReq) (string, error) {
	bucket, err := h.bucket(param.Bucket)
	if err != nil {
		return "", err
	}

	storagePath := objectKey(param.SavePath, param.FileName)
	if err := bucket.PutObject(storagePath, bytes.NewReader(param.ResourceByte),
		oss.ContentLength(int64(len(param.ResourceByte)))); err != nil {
		return "", err
	}
	return storagePath, nil
}

// UploadLocalFile 上传本地文件
func (h *AliyunOssUpload) UploadLocalFile(param *UploadLocalFileReq) (string, error) {
	bucket, err := h.bucket(param.Bucket)
	if err != nil {
		return "", err
	}

	storagePath := objectKey(param.SavePath, param.FileName)
	if param.FileName == "" {
		storagePath = objectKey(param.SavePath, filepath.Base(param.LocalPath))
	}
	if err := bucket.PutObjectFromFile(storagePath, param.LocalPath); err != nil {
		return "", err
	}
	return storagePath, nil
}

// DeleteFile 删除文件
func (h *AliyunOssUpload) DeleteFile(param *DeleteFileParam) error {
	bucket, err := h.bucket(param.Bucket)
	if err != nil {
		return err
	}
	return bucket.DeleteObject(trimKey(param.SavePath))
}

// UploadLocalFileUseResume 分片上传文件
func (h *AliyunOssUpload) UploadLocalFileUseResume(param *UploadLocalFileReq) (string, error) {
	bucket, err := h.bucket(param.Bucket)
	if err != nil {
		return "", err
	}

	// 将本地文件分片，且分片数量指定为10
	chunks, err := oss.SplitFileByPartNum(param.LocalPath, 10)
	if err != nil {
		return "", err
	}
	fd, err := os.Open(param.LocalPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = fd.Close() }()

	// 指定过期时间。
	expires := time.Date(2099, time.January, 10, 23, 0, 0, 0, time.UTC)

	options := []oss.Option{
		oss.MetadataDirective(oss.MetaReplace),
		oss.Expires(expires),
	}
	objectName := objectKey(param.SavePath, param.FileName)
	if param.FileName == "" {
		objectName = objectKey(param.SavePath, filepath.Base(param.LocalPath))
	}

	// 步骤1：初始化一个分片上传事件。
	imur, err := bucket.InitiateMultipartUpload(objectName, options...)
	if err != nil {
		return "", err
	}

	// 步骤2：上传分片。
	parts := make([]oss.UploadPart, 0, len(chunks))
	for _, chunk := range chunks {
		if _, err := fd.Seek(chunk.Offset, io.SeekStart); err != nil {
			return "", err
		}
		part, err := bucket.UploadPart(imur, fd, chunk.Size, chunk.Number)
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}

	// 步骤3：完成分片上传。
	cmur, err := bucket.CompleteMultipartUpload(imur, parts, oss.ObjectACL(oss.ACLPrivate))
	if err != nil {
		return "", err
	}
	return cmur.Key, nil
}

func (h *AliyunOssUpload) ossClient() (*oss.Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.client != nil {
		return h.client, nil
	}
	endpoint := h.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	client, err := oss.New(endpoint, h.AccessKeyId, h.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("创建 OSS 客户端失败: %w", err)
	}
	h.client = client
	return client, nil
}

func (h *AliyunOssUpload) bucket(name string) (*oss.Bucket, error) {
	if name == "" {
		return nil, errors.New("bucket 不能为空")
	}

	client, err := h.ossClient()
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.buckets == nil {
		h.buckets = make(map[string]*oss.Bucket)
	}
	if bucket, ok := h.buckets[name]; ok {
		return bucket, nil
	}
	bucket, err := client.Bucket(name)
	if err != nil {
		return nil, fmt.Errorf("获取 OSS Bucket 失败: %w", err)
	}
	h.buckets[name] = bucket
	return bucket, nil
}

func objectKey(savePath, fileName string) string {
	return trimKey(path.Join(savePath, fileName))
}

func trimKey(key string) string {
	return strings.TrimPrefix(strings.TrimSpace(key), "/")
}

// UploadResourceByteReq ...
type UploadResourceByteReq struct {
	ResourceByte []byte `json:"resource_byte"` // ResourceByte 文件[]byte
	FileName     string `json:"file_name"`     // FileName 文件名
	SavePath     string `json:"save_path"`     // SavePath oss保存的目录
	Bucket       string `json:"bucket"`        // Bucket
}

// UploadLocalFileReq 上传本地文件
type UploadLocalFileReq struct {
	LocalPath string `json:"local_path"` // LocalPath 本地文件路径
	FileName  string `json:"file_name"`  // FileName 文件名
	SavePath  string `json:"save_path"`  // SavePath oss保存的目录
	Bucket    string `json:"bucket"`     // Bucket
}

// AliyunOssConf 阿里云oss配置
type AliyunOssConf struct {
	AccessKeyId     string `json:"AccessKeyId"`     // AccessKeyId
	AccessKeySecret string `json:"AccessKeySecret"` // AccessKeySecret
	Endpoint        string `json:"Endpoint"`        // Endpoint 为空时使用杭州默认地址
}

// DeleteFileParam 删除文件请求参数
type DeleteFileParam struct {
	Bucket   string `json:"bucket"`    // Bucket
	SavePath string `json:"save_path"` // SavePath oss保存的目录
}
