package cache

import "errors"

var (
	ErrCacheTypeUnsupported = errors.New("cache type unsupported")
	ErrCacheTypeUnknown     = errors.New("cache type unknown")
)
