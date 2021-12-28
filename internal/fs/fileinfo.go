package fs

import (
	"archive/tar"
	"io/fs"
	"os"
	"time"

	"github.com/pojntfx/stfs/internal/logging"
)

type FileInfo struct {
	os.FileInfo

	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool

	log *logging.JSONLogger
}

func NewFileInfo(
	hdr *tar.Header,

	log *logging.JSONLogger,
) *FileInfo {
	return &FileInfo{
		name:    hdr.FileInfo().Name(),
		size:    hdr.FileInfo().Size(),
		mode:    hdr.FileInfo().Mode(),
		modTime: hdr.FileInfo().ModTime(),
		isDir:   hdr.FileInfo().IsDir(),

		log: log,
	}
}

func (f *FileInfo) Name() string {
	f.log.Trace("FileInfo.Name", map[string]interface{}{
		"name": f.name,
	})

	return f.name
}

func (f *FileInfo) Size() int64 {
	f.log.Trace("FileInfo.Size", map[string]interface{}{
		"name": f.name,
	})

	return f.size
}

func (f *FileInfo) Mode() os.FileMode {
	f.log.Trace("FileInfo.Mode", map[string]interface{}{
		"name": f.name,
	})

	return f.mode
}

func (f *FileInfo) ModTime() time.Time {
	f.log.Trace("FileInfo.ModTime", map[string]interface{}{
		"name": f.name,
	})

	return f.modTime
}

func (f *FileInfo) IsDir() bool {
	f.log.Trace("FileInfo.IsDir", map[string]interface{}{
		"name": f.name,
	})

	return f.isDir
}

func (f *FileInfo) Sys() interface{} {
	f.log.Trace("FileInfo.Sys", map[string]interface{}{
		"name": f.name,
	})

	return nil
}
