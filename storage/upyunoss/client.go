package upyunoss

import (
	"errors"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/bigbigliu/go-core/fileutil"
	"github.com/upyun/go-sdk/v3/upyun"
)

// IUpYunOssUpload 又拍云上传文件
type IUpYunOssUpload interface {
	// UploadLocalFile 上传本地文件(form表单上传)
	UploadLocalFile(param *UploadLocalFileParam) (string, error)
	// UploadLocalFileUseResume 上传本地文件((form表单上传或者分片上传))
	UploadLocalFileUseResume(param *UploadLocalFileParam) (string, error)
	// GetInfo 获取文件信息
	GetInfo(param *GetInfoParam) (*FileInfo, error)
}

// UpYunOssUpload 又拍云上传文件
type UpYunOssUpload struct {
	Operator string `json:"operator"` // Operator
	Password string `json:"password"` // Password
	Secret   string `json:"secret"`   // Secret

	mu      sync.Mutex
	clients map[string]*upyun.UpYun
}

func (h *UpYunOssUpload) upYun(bucket string) (*upyun.UpYun, error) {
	if bucket == "" {
		return nil, errors.New("bucket 不能为空")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients == nil {
		h.clients = make(map[string]*upyun.UpYun)
	}
	if client, ok := h.clients[bucket]; ok {
		return client, nil
	}
	client := upyun.NewUpYun(&upyun.UpYunConfig{
		Bucket:   bucket,
		Operator: h.Operator,
		Password: h.Password,
	})
	h.clients[bucket] = client
	return client, nil
}

// UploadLocalFile 上传本地文件(form表单上传)
func (h *UpYunOssUpload) UploadLocalFile(param *UploadLocalFileParam) (string, error) {
	upNew, err := h.upYun(param.Bucket)
	if err != nil {
		return "", err
	}

	objectPath := localObjectKey(param.SavePath, param.FileName, param.LocalFilePath)
	uploadParam := &upyun.PutObjectConfig{
		Path:      objectPath,
		LocalPath: param.LocalFilePath,
	}
	if err := upNew.Put(uploadParam); err != nil {
		return "", err
	}

	return uploadParam.Path, nil
}

// UploadLocalFileUseResume 上传本地文件((form表单上传或者分片上传))
func (h *UpYunOssUpload) UploadLocalFileUseResume(param *UploadLocalFileParam) (string, error) {
	upNew, err := h.upYun(param.Bucket)
	if err != nil {
		return "", err
	}

	objectPath := localObjectKey(param.SavePath, param.FileName, param.LocalFilePath)

	// 文件 > 300 mb 就分片上传
	fileLarge, err := fileutil.IsFileGreaterThan(param.LocalFilePath, 300)
	if err != nil {
		return "", err
	}

	if !fileLarge {
		uploadParam := &upyun.PutObjectConfig{
			Path:      objectPath,
			LocalPath: param.LocalFilePath,
		}

		err = upNew.Put(uploadParam)
		if err != nil {
			return "", err
		}
		return uploadParam.Path, nil
	}

	// 开始分片上传
	// 断点续传 文件大于 10M 才会分片
	//uploadParam := &upyun.MemoryRecorder{}
	// 若设置为 nil，则为正常的分片上传
	upNew.SetRecorder(nil)
	err = upNew.Put(&upyun.PutObjectConfig{
		Path:            objectPath,
		LocalPath:       param.LocalFilePath,
		UseResumeUpload: true,
	})
	if err != nil {
		return "", err
	}

	return objectPath, nil
}

// GetInfo 获取文件信息
func (h *UpYunOssUpload) GetInfo(param *GetInfoParam) (*FileInfo, error) {
	upNew, err := h.upYun(param.Bucket)
	if err != nil {
		return nil, err
	}

	dataInfo, err := upNew.GetInfo(param.Path)
	if err != nil {
		return nil, err
	}

	if dataInfo == nil {
		return nil, errors.New("获取文件信息失败")
	}

	result := &FileInfo{
		Name:        dataInfo.Name,
		Size:        dataInfo.Size,
		ContentType: dataInfo.ContentType,
		IsDir:       dataInfo.IsDir,
		IsEmptyDir:  dataInfo.IsEmptyDir,
		MD5:         dataInfo.MD5,
		Time:        dataInfo.Time,
		Meta:        dataInfo.Meta,
		ImgType:     dataInfo.ImgType,
		ImgWidth:    dataInfo.ImgWidth,
		ImgHeight:   dataInfo.ImgHeight,
		ImgFrames:   dataInfo.ImgFrames,
	}
	return result, nil
}

// UploadLocalFileParam 上传本地文件请求参数
type UploadLocalFileParam struct {
	Bucket        string // Bucket
	SavePath      string // SavePath 云存储中的保存目录
	LocalFilePath string // LocalFilePath 本地文件路径
	FileName      string // FileName 文件名(非必填参数)
}

// GetInfoParam 获取文件信息请求参数
type GetInfoParam struct {
	Path   string // Path oss存储目录
	Bucket string // Bucket
}

// FileInfo 参数返回
type FileInfo struct {
	Name        string
	Size        int64
	ContentType string
	IsDir       bool
	IsEmptyDir  bool
	MD5         string
	Time        time.Time
	Meta        map[string]string
	ImgType     string
	ImgWidth    int64
	ImgHeight   int64
	ImgFrames   int64
}

// 参数注释
//type PutObjectConfig struct {
//	Path              string            // 云存储中的路径
//	LocalPath         string            // 待上传文件在本地文件系统中的路径
//	Reader            io.Reader         // 待上传的内容
//	Headers           map[string]string // 额外的 HTTP 请求头
//	UseMD5            bool              // 是否需要 MD5 校验
//	UseResumeUpload   bool              // 是否使用断点续传
//	AppendContent     bool              // 是否需要追加文件内容
//	ResumePartSize    int64             // 断点续传块大小
//	MaxResumePutTries int               // 断点续传最大重试次数
//}

func localObjectKey(savePath, fileName, localPath string) string {
	if fileName == "" {
		fileName = filepath.Base(localPath)
	}
	return path.Join(savePath, fileName)
}
