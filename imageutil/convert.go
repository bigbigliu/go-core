package imageutil

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
)

// EncodeImageToBase64 将图片文件编码为 Base64 字符串
func EncodeImageToBase64(imagePath string) (string, error) {
	// 读取图片文件
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	// 解码图片文件
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	// 将图片编码为 PNG 或 JPEG 格式
	var encodedData []byte
	switch format {
	case "jpeg":
		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, nil)
		if err != nil {
			return "", err
		}
		encodedData = buf.Bytes()
	case "png":
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			return "", err
		}
		encodedData = buf.Bytes()
	default:
		return "", fmt.Errorf("unsupported image format: %s", format)
	}

	// 将字节切片编码为 Base64 字符串
	base64Str := base64.StdEncoding.EncodeToString(encodedData)
	return base64Str, nil
}
