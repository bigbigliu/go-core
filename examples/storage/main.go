package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bigbigliu/go-core/storage"
	"github.com/bigbigliu/go-core/storage/aliyunoss"
)

func main() {
	accessKey := os.Getenv("ALIYUN_OSS_ACCESS_KEY_ID")
	accessSecret := os.Getenv("ALIYUN_OSS_ACCESS_KEY_SECRET")
	bucketName := os.Getenv("ALIYUN_OSS_BUCKET")
	if accessKey == "" || accessSecret == "" || bucketName == "" {
		fmt.Println("set ALIYUN_OSS_ACCESS_KEY_ID, ALIYUN_OSS_ACCESS_KEY_SECRET and ALIYUN_OSS_BUCKET to run this example")
		fmt.Println("optional: ALIYUN_OSS_ENDPOINT (default https://oss-cn-hangzhou.aliyuncs.com)")
		return
	}

	uploader := &aliyunoss.AliyunOssUpload{
		AccessKeyId:     accessKey,
		AccessKeySecret: accessSecret,
		Endpoint:        os.Getenv("ALIYUN_OSS_ENDPOINT"),
	}
	var store storage.Storage
	store, err := aliyunoss.NewClient(uploader, bucketName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	key, err := store.Put(ctx, "examples/hello.txt", strings.NewReader("hello go-core"), int64(len("hello go-core")))
	if err != nil {
		fmt.Fprintln(os.Stderr, "put:", err)
		os.Exit(1)
	}
	fmt.Printf("uploaded: %s\n", key)

	if err := store.Delete(ctx, key); err != nil {
		fmt.Fprintln(os.Stderr, "delete:", err)
		os.Exit(1)
	}
	fmt.Printf("deleted: %s\n", key)
}
