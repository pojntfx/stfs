package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mattetti/filebuffer"
	"github.com/pojntfx/stfs/internal/fs"
	"github.com/spf13/afero"
)

type fileWithSize struct {
	afero.File
}

func (f fileWithSize) Size() (int64, error) {
	info, err := f.Stat()
	if err != nil {
		return -1, err
	}

	return info.Size(), nil
}

type filebufferWithSize struct {
	*filebuffer.Buffer
}

func (f filebufferWithSize) Size() (int64, error) {
	return int64(f.Buff.Len()), nil
}

func (f filebufferWithSize) Sync() error {
	// No need to sync a in-memory buffer
	return nil
}

func (f filebufferWithSize) Truncate(size int64) error {
	f.Buff.Truncate(int(size))

	return nil
}

func NewCacheWrite(
	root string,
	cacheType string,
) (cache fs.WriteCache, cleanup func() error, err error) {
	switch cacheType {
	case WriteCacheTypeMemory:
		buff := &filebufferWithSize{filebuffer.New([]byte{})}

		return buff, func() error {
			buff = nil

			return nil
		}, nil
	case WriteCacheTypeFile:
		tmpdir := filepath.Join(root, "io")

		if err := os.MkdirAll(tmpdir, os.ModePerm); err != nil {
			return nil, nil, err
		}

		f, err := ioutil.TempFile(tmpdir, "*")
		if err != nil {
			return nil, nil, err
		}

		return fileWithSize{f}, func() error {
			return os.Remove(filepath.Join(tmpdir, f.Name()))
		}, nil
	default:
		return nil, nil, ErrWriteCacheTypeUnsupported
	}
}
