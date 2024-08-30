package aliyunOss

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// IAliyunOssUpload 阿里云oss上传方法
type IAliyunOssUpload interface {
	// UploadResourceByte 上传Byte数组
	UploadResourceByte(param *UploadResourceByteReq) (path string, err error)
	// UploadLocalFile 上传本地文件
	UploadLocalFile(param *UploadLocalFileReq) (path string, err error)
	// DeleteFile 删除文件
	DeleteFile(param *DeleteFileParam) error
}

// AliyunOss ...
type AliyunOssUpload struct {
	AccessKeyId     string `json:"AccessKeyId"`     // AccessKeyId
	AccessKeySecret string `json:"AccessKeySecret"` // AccessKeySecret
}

// UploadResourceByte 上传Byte数组
func (h *AliyunOssUpload) UploadResourceByte(param *UploadResourceByteReq) (path string, err error) {
	resourceByte := param.ResourceByte

	client, err := oss.New("https://oss-cn-hangzhou.aliyuncs.com",
		h.AccessKeyId,
		h.AccessKeySecret,
		oss.HTTPClient(&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}))
	if err != nil {
		return "", err
	}

	bucket, err := client.Bucket(param.Bucket)
	if err != nil {
		return "", err
	}

	storagePath := param.SavePath + param.FileName
	err = bucket.PutObject(storagePath, bytes.NewReader(resourceByte))
	if err != nil {
		return "", err
	}

	return storagePath, nil
}

// UploadLocalFile 上传本地文件
func (h *AliyunOssUpload) UploadLocalFile(param *UploadLocalFileReq) (path string, err error) {
	client, err := oss.New("https://oss-cn-hangzhou.aliyuncs.com",
		h.AccessKeyId,
		h.AccessKeySecret,
		oss.HTTPClient(&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}))
	if err != nil {
		return "", err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client.HTTPClient.Transport = tr

	bucket, err := client.Bucket(param.Bucket)
	if err != nil {
		return "", err
	}

	// param.SavePath 表示删除OSS文件时需要指定包含文件后缀，不包含Bucket名称在内的完整路径，例如exampledir/exampleobject.txt。
	err = bucket.PutObjectFromFile(param.SavePath, param.LocalPath)
	if err != nil {
		return "", err
	}

	return param.SavePath, nil
}

// DeleteFile 删除文件
func (h *AliyunOssUpload) DeleteFile(param *DeleteFileParam) error {
	client, err := oss.New("https://oss-cn-hangzhou.aliyuncs.com",
		h.AccessKeyId,
		h.AccessKeySecret,
		oss.HTTPClient(&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}))
	if err != nil {
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client.HTTPClient.Transport = tr

	bucket, err := client.Bucket(param.Bucket)
	if err != nil {
		return err
	}

	// 删除单个文件。
	// param.SavePath 表示删除OSS文件时需要指定包含文件后缀，不包含Bucket名称在内的完整路径，例如exampledir/exampleobject.txt。
	// 如需删除文件夹，请将param.SavePath设置为对应的文件夹名称。如果文件夹非空，则需要将文件夹下的所有object删除后才能删除该文件夹。
	err = bucket.DeleteObject(param.SavePath)
	if err != nil {
		return err
	}

	return nil
}

// UploadLocalFileUseResume 分片上传文件
func (h *AliyunOssUpload) UploadLocalFileUseResume(param *UploadLocalFileReq) (path string, err error) {
	client, err := oss.New("https://oss-cn-hangzhou.aliyuncs.com",
		h.AccessKeyId,
		h.AccessKeySecret,
		oss.HTTPClient(&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}))
	if err != nil {
		return "", err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client.HTTPClient.Transport = tr

	bucket, err := client.Bucket(param.Bucket)
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
	defer fd.Close()

	// 指定过期时间。
	expires := time.Date(2099, time.January, 10, 23, 0, 0, 0, time.UTC)

	// 如果需要在初始化分片时设置请求头，请参考以下示例代码。
	options := []oss.Option{
		oss.MetadataDirective(oss.MetaReplace),
		oss.Expires(expires),
		// 指定该Object被下载时的网页缓存行为。
		// oss.CacheControl("no-cache"),
		// 指定该Object被下载时的名称。
		// oss.ContentDisposition("attachment;filename=FileName.txt"),        ,
		// 指定对返回的Key进行编码，目前支持URL编码。
		// oss.EncodingType("url"),
		// 指定Object的存储类型。
		// oss.ObjectStorageClass(oss.StorageStandard),
	}
	objectName := param.SavePath + param.FileName
	// 步骤1：初始化一个分片上传事件。
	imur, err := bucket.InitiateMultipartUpload(objectName, options...)
	if err != nil {
		return "", err
	}

	// 步骤2：上传分片。
	var parts []oss.UploadPart
	for _, chunk := range chunks {
		fd.Seek(chunk.Offset, os.SEEK_SET)
		// 调用UploadPart方法上传每个分片。
		part, err := bucket.UploadPart(imur, fd, chunk.Size, chunk.Number)
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}

	// 指定Object的读写权限为私有，默认为继承Bucket的读写权限。
	objectAcl := oss.ObjectACL(oss.ACLPrivate)
	// 步骤3：完成分片上传。
	cmur, err := bucket.CompleteMultipartUpload(imur, parts, objectAcl)
	if err != nil {
		return "", err
	}
	return cmur.Key, nil
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
	FileName  string `json:"fileName"`   // FileName 文件名
	SavePath  string `json:"save_path"`  // SavePath oss保存的目录
	Bucket    string `json:"bucket"`     // Bucket
}

// AliyunOssConf 阿里云oss配置
type AliyunOssConf struct {
	AccessKeyId     string `json:"AccessKeyId"`     // AccessKeyId
	AccessKeySecret string `json:"AccessKeySecret"` // AccessKeySecret
}

// DeleteFileParam 删除文件请求参数
type DeleteFileParam struct {
	Bucket   string `json:"bucket"`    // Bucket
	SavePath string `json:"save_path"` // SavePath oss保存的目录
}
