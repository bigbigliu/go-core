// Package storage defines a provider-agnostic object storage interface.
//
// Provider implementations live in subpackages (aliyunoss, qiniuoss, upyunoss)
// and expose a Client that implements Storage.
package storage

import (
	"context"
	"errors"
	"io"
)

// ErrSizeRequired is returned when a provider needs the object size for a
// streaming upload but the caller passed a negative size.
var ErrSizeRequired = errors.New("storage: size is required for this upload")

// Storage is the common object storage interface. Keys are object paths
// relative to the configured bucket and must not start with "/".
type Storage interface {
	// Put uploads the content read from r. size is the number of bytes in r,
	// or -1 when unknown. Providers that require a size return ErrSizeRequired
	// for -1.
	Put(ctx context.Context, key string, r io.Reader, size int64) (string, error)
	// PutFile uploads a local file.
	PutFile(ctx context.Context, key, localPath string) (string, error)
	// Delete removes an object.
	Delete(ctx context.Context, key string) error
}
