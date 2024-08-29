package qiniuOss

import (
	"bytes"
	"context"
	"errors"
	"github.com/bigbigliu/go-core/pkgs"
	"io"
	"path/filepath"

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

// QiNiuOssUpload 又拍云上传文件
type QiNiuOssUpload struct {
	AccessKey string `json:"accessKey"` // AccessKey
	SecretKey string `json:"secretKey"` // SecretKey
	Pipeline  string `json:"pipeline"`  // Pipeline
}

// UploadResourceByte 上传文件([]byte)
func (h *QiNiuOssUpload) UploadResourceByte(param *UploadResourceByteParam) (string, error) {
	key := param.SavePath + param.FileName
	mac := qbox.NewMac(h.AccessKey, h.SecretKey)

	cfg := storage.Config{
		UseHTTPS: false,
		Region:   &storage.Zone_z0,
	}

	// 强制重新执行数据处理任务
	putPolicy := storage.PutPolicy{
		Scope:               param.Bucket,
		PersistentNotifyURL: "http://49b6-69-172-67-65.ngrok.io",
		PersistentPipeline:  h.Pipeline,
	}

	upToken := putPolicy.UploadToken(mac)
	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	putExtra := storage.PutExtra{}

	err := formUploader.Put(context.Background(), &ret, upToken, key, bytes.NewReader(param.ResourceByte), int64(len(param.ResourceByte)), &putExtra)
	if err != nil {
		return "", err
	}

	if ret.Hash == "" {
		return "", errors.New("上传文件失败")
	}

	return key, nil
}

// UploadLocalFile 上传本地文件
func (h *QiNiuOssUpload) UploadLocalFile(param *UploadLocalFileParam) (string, error) {
	key := ""
	if param.FileName == "" {
		_, fileName := filepath.Split(param.LocalFilePath)
		key = param.SavePath + "/" + fileName
	} else {
		key = param.SavePath + "/" + param.FileName
	}

	mac := qbox.NewMac(h.AccessKey, h.SecretKey)

	cfg := storage.Config{
		UseHTTPS: false,
		Region:   &storage.Zone_z0,
	}

	putPolicy := storage.PutPolicy{
		Scope:               param.Bucket,
		PersistentNotifyURL: "http://49b6-69-172-67-65.ngrok.io",
		PersistentPipeline:  h.Pipeline,
	}
	upToken := putPolicy.UploadToken(mac)
	// 构建表单上传的对象
	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	putExtra := storage.PutExtra{}

	err := formUploader.PutFile(context.Background(), &ret, upToken, key, param.LocalFilePath, &putExtra)
	if err != nil {
		return "", err
	}

	if ret.Hash == "" {
		return "", errors.New("上传文件失败")
	}

	return key, nil
}

// UploadLocalFileUseResume 上传本地文件(根据文件大小自动判断普通表单上传还是分片上传)
func (h *QiNiuOssUpload) UploadLocalFileUseResume(param *UploadLocalFileParam) (string, error) {
	key := ""
	if param.FileName == "" {
		_, fileName := filepath.Split(param.LocalFilePath)
		key = param.SavePath + "/" + fileName
	} else {
		key = param.SavePath + "/" + param.FileName
	}

	mac := qbox.NewMac(h.AccessKey, h.SecretKey)

	cfg := storage.Config{
		UseHTTPS: false,
		Region:   &storage.Zone_z0,
	}

	putPolicy := storage.PutPolicy{
		Scope:               param.Bucket,
		PersistentNotifyURL: "http://49b6-69-172-67-65.ngrok.io",
		PersistentPipeline:  h.Pipeline,
	}

	upToken := putPolicy.UploadToken(mac)
	ret := storage.PutRet{}

	file_ok, err := pkgs.IsFileGreaterThan(param.LocalFilePath, 300)
	if err != nil {
		return "", err
	}
	if !file_ok {
		logger.Logger.Info("七牛云服务", zap.String("msg", "文件小于300mb, 普通表单上传"))
		// 构建表单上传的对象
		formUploader := storage.NewFormUploader(&cfg)
		putExtra := storage.PutExtra{}

		err = formUploader.PutFile(context.Background(), &ret, upToken, key, param.LocalFilePath, &putExtra)
		if err != nil {
			return "", err
		}

		if ret.Hash == "" {
			return "", errors.New("上传文件失败")
		}

		return key, nil
	}

	// 分片上传
	logger.Logger.Info("七牛云服务", zap.String("msg", "文件大于300mb, 分片上传上传"))
	resumeUploader := storage.NewResumeUploaderV2(&cfg)
	putExtra := storage.RputV2Extra{}
	err = resumeUploader.PutFile(context.Background(), &ret, upToken, key, param.LocalFilePath, &putExtra)
	if err != nil {
		return "", err
	}

	if ret.Hash == "" {
		return "", errors.New("上传文件失败")
	}

	return key, nil
}

// DownloadFile 下载文件
func (h *QiNiuOssUpload) DownloadFile(param *DownloadFileParam) ([]byte, error) {
	key := param.SavePath
	bucket := param.Bucket

	mac := qbox.NewMac(h.AccessKey, h.SecretKey)

	bm := storage.NewBucketManager(mac, &storage.Config{})

	// err 和 resp 可能同时有值，当 err 有值时，下载是失败的，此时如果 resp 也有值可以通过 resp 获取响应状态码等其他信息
	resp, err := bm.Get(bucket, key, &storage.GetObjectInput{
		DownloadDomains: []string{},
		PresignUrl:      true, // 下载 URL 是否进行签名，源站域名或者私有空间需要配置为 true
		//Range:           "bytes=2-5", // 下载文件时 HTTP 请求的 Range 请求头
	})
	if err != nil || resp == nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
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

// exampleQiNiu 使用示例
//func exampleQiNiu() {
//	var qiNiuOss IQiNiuOssUpload
//	qiNiuOss = &QiNiuOssUpload{
//		AccessKey: "",
//		SecretKey: "",
//		Pipeline:  "",
//	}
//
//	uploadResult, _ := qiNiuOss.UploadLocalFileUseResume(&UploadLocalFileParam{
//		LocalFilePath: "",
//		SavePath:      "",
//		Bucket:        "bucket",
//		FileName:      "",
//	})
//	fmt.Printf("uploadResult: ", uploadResult)
//}
