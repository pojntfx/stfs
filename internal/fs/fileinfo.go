package fs

import (
	"log"
	"os"
	"time"
)

type FileInfo struct {
	os.FileInfo

	name    string
	size    int64
	mode    int64
	modTime time.Time
	isDir   bool
}

func NewFileInfo(
	name string,
	size int64,
	mode int64,
	modTime time.Time,
	isDir bool,
) *FileInfo {
	return &FileInfo{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: modTime,
		isDir:   isDir,
	}
}

func (f FileInfo) Name() string {
	log.Println("FileInfo.Name", f.name)

	return f.name
}

func (f FileInfo) Size() int64 {
	log.Println("FileInfo.Size", f.name)

	return f.size
}

func (f FileInfo) Mode() os.FileMode {
	log.Println("FileInfo.Mode", f.name)

	return os.FileMode(f.mode)
}

func (f FileInfo) ModTime() time.Time {
	log.Println("FileInfo.ModTime", f.name)

	return f.modTime
}

func (f FileInfo) IsDir() bool {
	log.Println("FileInfo.IsDir", f.name)

	return f.isDir
}

func (f FileInfo) Sys() interface{} {
	log.Println("FileInfo.Sys", f.name)

	return nil
}
