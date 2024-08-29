package pkgs

import (
	"os"
)

// IsFileGreaterThan 判断文件是否大于多少MB
func IsFileGreaterThan(filePath string, size int64) (bool, error) {
	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	// 获取文件大小
	fileSize := fileInfo.Size() / (1024 * 1024) // 转换为MB

	if fileSize > size {
		return true, nil
	} else {
		return false, nil
	}
}

// SaveToFile 保存字节数据到指定文件
func SaveToFile(fileName string, data []byte) error {
	err := os.WriteFile(fileName, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
