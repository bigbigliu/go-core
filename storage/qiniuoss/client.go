package qiniuoss

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"path/filepath"

	"github.com/bigbigliu/go-core/fileutil"
	"github.com/bigbigliu/go-core/logger"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"go.uber.org/zap"
)

// IQiNiuOssUpload 七牛云上传文件
type IQiNiuOssUpload interface {
	// UploadResourceByte 上传文件([]byte)
	UploadResourceByte(param *UploadResourceByteParam) (string, error)
	// UploadLocalFile 上传本地文件
	UploadLocalFile(param *UploadLocalFileParam) (string, error)
	// UploadLocalFileUseResume 上传本地文件(根据文件大小自动判断普通表单上传还是分片上传)
	UploadLocalFileUseResume(param *UploadLocalFileParam) (string, error)
	// DownloadFile 下载文件
	DownloadFile(param *DownloadFileParam) ([]byte, error)
}

// QiNiuOssUpload 七牛云上传文件
type QiNiuOssUpload struct {
	AccessKey    string        `json:"accessKey"`     // AccessKey
	SecretKey    string        `json:"secretKey"`     // SecretKey
	Pipeline     string        `json:"pipeline"`      // Pipeline
	Zone         *storage.Zone `json:"-"`             // Zone 为空时使用华东 Zone_z0
	NotifyURL    string        `json:"notify_url"`    // NotifyURL 持久化处理回调地址，可为空
	DisableHTTPS bool          `json:"disable_https"` // DisableHTTPS 为 true 时使用 HTTP
	MaxFormSize  int64         `json:"-"`             // MaxFormSize 超过该大小(MB)使用分片上传，默认 300
}

// UploadResourceByte 上传文件([]byte)
func (h *QiNiuOssUpload) UploadResourceByte(param *UploadResourceByteParam) (string, error) {
	key := objectKey(param.SavePath, param.FileName)
	mac := qbox.NewMac(h.AccessKey, h.SecretKey)

	cfg := h.storageConfig()
	upToken := h.uploadToken(param.Bucket, mac)

	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	putExtra := storage.PutExtra{}

	if err := formUploader.Put(context.Background(), &ret, upToken, key, bytes.NewReader(param.ResourceByte), int64(len(param.ResourceByte)), &putExtra); err != nil {
		return "", err
	}
	if ret.Hash == "" {
		return "", errors.New("上传文件失败")
	}
	return key, nil
}

// UploadLocalFile 上传本地文件
func (h *QiNiuOssUpload) UploadLocalFile(param *UploadLocalFileParam) (string, error) {
	key := localObjectKey(param.SavePath, param.FileName, param.LocalFilePath)

	mac := qbox.NewMac(h.AccessKey, h.SecretKey)
	cfg := h.storageConfig()
	upToken := h.uploadToken(param.Bucket, mac)

	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	putExtra := storage.PutExtra{}

	if err := formUploader.PutFile(context.Background(), &ret, upToken, key, param.LocalFilePath, &putExtra); err != nil {
		return "", err
	}
	if ret.Hash == "" {
		return "", errors.New("上传文件失败")
	}
	return key, nil
}

// UploadLocalFileUseResume 上传本地文件(根据文件大小自动判断普通表单上传还是分片上传)
func (h *QiNiuOssUpload) UploadLocalFileUseResume(param *UploadLocalFileParam) (string, error) {
	key := localObjectKey(param.SavePath, param.FileName, param.LocalFilePath)

	mac := qbox.NewMac(h.AccessKey, h.SecretKey)
	cfg := h.storageConfig()
	upToken := h.uploadToken(param.Bucket, mac)
	ret := storage.PutRet{}

	maxFormSize := h.MaxFormSize
	if maxFormSize <= 0 {
		maxFormSize = 300
	}
	fileLarge, err := fileutil.IsFileGreaterThan(param.LocalFilePath, maxFormSize)
	if err != nil {
		return "", err
	}

	if !fileLarge {
		logInfo("文件小于阈值, 普通表单上传", zap.Int64("threshold_mb", maxFormSize))
		formUploader := storage.NewFormUploader(&cfg)
		putExtra := storage.PutExtra{}
		if err := formUploader.PutFile(context.Background(), &ret, upToken, key, param.LocalFilePath, &putExtra); err != nil {
			return "", err
		}
		if ret.Hash == "" {
			return "", errors.New("上传文件失败")
		}
		return key, nil
	}

	logInfo("文件大于阈值, 分片上传", zap.Int64("threshold_mb", maxFormSize))
	resumeUploader := storage.NewResumeUploaderV2(&cfg)
	putExtra := storage.RputV2Extra{}
	if err := resumeUploader.PutFile(context.Background(), &ret, upToken, key, param.LocalFilePath, &putExtra); err != nil {
		return "", err
	}
	if ret.Hash == "" {
		return "", errors.New("上传文件失败")
	}
	return key, nil
}

// DownloadFile 下载文件
func (h *QiNiuOssUpload) DownloadFile(param *DownloadFileParam) ([]byte, error) {
	mac := qbox.NewMac(h.AccessKey, h.SecretKey)
	bm := storage.NewBucketManager(mac, &storage.Config{})

	resp, err := bm.Get(param.Bucket, param.SavePath, &storage.GetObjectInput{
		PresignUrl: true, // 源站域名或者私有空间需要配置为 true
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("下载文件失败")
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(resp.Body)
}

func (h *QiNiuOssUpload) storageConfig() storage.Config {
	zone := h.Zone
	if zone == nil {
		zone = &storage.Zone_z0
	}
	return storage.Config{
		UseHTTPS: !h.DisableHTTPS,
		Region:   zone,
	}
}

func (h *QiNiuOssUpload) uploadToken(bucket string, mac *qbox.Mac) string {
	policy := h.putPolicy(bucket)
	return policy.UploadToken(mac)
}

func (h *QiNiuOssUpload) putPolicy(bucket string) storage.PutPolicy {
	return storage.PutPolicy{
		Scope:               bucket,
		PersistentNotifyURL: h.NotifyURL,
		PersistentPipeline:  h.Pipeline,
	}
}

func objectKey(savePath, fileName string) string {
	return path.Join(savePath, fileName)
}

func localObjectKey(savePath, fileName, localPath string) string {
	if fileName == "" {
		fileName = filepath.Base(localPath)
	}
	return objectKey(savePath, fileName)
}

func logInfo(msg string, fields ...zap.Field) {
	if logger.Logger != nil {
		logger.Logger.Info(msg, fields...)
	}
}

// DownloadFileParam 下载文件请求参数
type DownloadFileParam struct {
	SavePath string `json:"save_path"` // SavePath 保存目录
	Bucket   string `json:"bucket"`    // Bucket 存放空间
}

// UploadResourceByteParam 上传文件([]byte)请求参数
type UploadResourceByteParam struct {
	ResourceByte []byte `json:"resource_byte"` // ResourceByte 文件[]byte流
	FileName     string `json:"fileName"`      // FileName 文件名(非必填参数)
	SavePath     string `json:"save_path"`     // SavePath 保存目录
	Bucket       string `json:"bucket"`        // Bucket 存放空间
}

// UploadLocalFileParam 上传本地文件请求参数
type UploadLocalFileParam struct {
	LocalFilePath string `json:"local_file_path" example:"ltest"` // LocalFilePath 本地文件路径(最前面不要带 / )
	SavePath      string `json:"save_path"`                       // SavePath 保存目录
	Bucket        string `json:"bucket"`                          // Bucket 存放空间
	FileName      string `json:"file_name"`                       // FileName 文件名(非必填参数)
}
