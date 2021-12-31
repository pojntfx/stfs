package cache

import "io"

type WriteCache interface {
	io.Closer
	io.Reader
	io.Seeker
	io.Writer

	Truncate(size int64) error
	Size() (int64, error)
	Sync() error
}
