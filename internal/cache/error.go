package cache

import "errors"

var (
	ErrFileSystemCacheTypeUnsupported = errors.New("file system cache type unsupported")
	ErrFileSystemCacheTypeUnknown     = errors.New("file system cache type unknown")

	ErrWriteCacheTypeUnsupported = errors.New("write cache type unsupported")
	ErrWriteCacheTypeUnknown     = errors.New("write cache type unknown")
)
