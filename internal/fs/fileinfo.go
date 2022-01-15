package fs

import (
	"archive/tar"
	"io/fs"
	"os"
	"time"

	"github.com/pojntfx/stfs/pkg/logging"
)

type FileInfo struct {
	os.FileInfo

	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	gid     int
	uid     int
	isDir   bool

	log logging.StructuredLogger
}

func NewFileInfo(
	name string,
	size int64,
	mode fs.FileMode,
	modTime time.Time,
	gid int,
	uid int,
	isDir bool,

	log logging.StructuredLogger,
) *FileInfo {
	return &FileInfo{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: modTime,
		gid:     gid,
		uid:     uid,
		isDir:   isDir,

		log: log,
	}
}

func NewFileInfoFromTarHeader(
	hdr *tar.Header,

	log logging.StructuredLogger,
) *FileInfo {
	return &FileInfo{
		name:    hdr.FileInfo().Name(),
		size:    hdr.FileInfo().Size(),
		mode:    hdr.FileInfo().Mode(),
		modTime: hdr.FileInfo().ModTime(),
		gid:     hdr.Gid,
		uid:     hdr.Uid,
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

	return &Stat{
		Uid: uint32(f.uid),
		Gid: uint32(f.gid),
	}
}
