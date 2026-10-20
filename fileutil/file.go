package fileutil

import (
	"os"
)

// IsFileGreaterThan 判断文件是否大于 size MB
func IsFileGreaterThan(filePath string, size int64) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}
	return fileInfo.Size() > size*1024*1024, nil
}

// SaveToFile 保存字节数据到指定文件
func SaveToFile(fileName string, data []byte) error {
	return os.WriteFile(fileName, data, 0o644)
}
