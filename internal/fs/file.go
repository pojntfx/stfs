package fs

import (
	"log"
	"os"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/spf13/afero"
)

type File struct {
	afero.File

	metadata config.MetadataConfig

	path string

	name string
	info os.FileInfo

	onHeader func(hdr *models.Header)
}

func NewFile(
	metadata config.MetadataConfig,

	path string,

	name string,
	stat os.FileInfo,

	onHeader func(hdr *models.Header),
) *File {
	return &File{
		metadata: metadata,

		path: path,

		name: name,
		info: stat,

		onHeader: onHeader,
	}
}

func (f File) Name() string {
	log.Println("File.Name", f.name)

	return f.name
}

func (f File) Stat() (os.FileInfo, error) {
	log.Println("File.Stat", f.name)

	return f.info, nil
}

func (f File) Readdir(count int) ([]os.FileInfo, error) {
	log.Println("File.Readdir", f.name, count)

	hdrs, err := inventory.List(
		f.metadata,

		f.path,

		f.onHeader,
	)
	if err != nil {
		return nil, err
	}

	fileInfos := []os.FileInfo{}
	for _, hdr := range hdrs {
		// TODO: Handle count; only return all if count = -1
		fileInfos = append(fileInfos, NewFileInfo(hdr))
	}

	return fileInfos, nil
}

func (f File) Readdirnames(n int) ([]string, error) {
	log.Println("File.Readdirnames", f.name, n)

	return nil, ErrNotImplemented
}

func (f File) Sync() error {
	log.Println("File.Sync", f.name)

	return ErrNotImplemented
}

func (f File) Truncate(size int64) error {
	log.Println("File.Truncate", f.name, size)

	return ErrNotImplemented
}

func (f File) WriteString(s string) (ret int, err error) {
	log.Println("File.WriteString", f.name, s)

	return -1, ErrNotImplemented
}

func (f File) Close() error {
	log.Println("File.Close", f.name)

	return ErrNotImplemented
}

func (f File) Read(p []byte) (n int, err error) {
	log.Println("File.Read", f.name, p)

	return -1, ErrNotImplemented
}

func (f File) ReadAt(p []byte, off int64) (n int, err error) {
	log.Println("File.ReadAt", f.name, p, off)

	return -1, ErrNotImplemented
}

func (f File) Seek(offset int64, whence int) (int64, error) {
	log.Println("File.Seek", f.name, offset, whence)

	return -1, ErrNotImplemented
}

func (f File) Write(p []byte) (n int, err error) {
	log.Println("File.Write", f.name, p)

	return -1, ErrNotImplemented
}

func (f File) WriteAt(p []byte, off int64) (n int, err error) {
	log.Println("File.WriteAt", f.name, p, off)

	return -1, ErrNotImplemented
}
