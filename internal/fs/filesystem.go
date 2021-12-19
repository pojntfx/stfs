package fs

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/spf13/afero"
)

var (
	ErrNotImplemented = errors.New("not implemented")
)

type FileSystem struct {
	ops *operations.Operations

	metadata config.MetadataConfig

	onHeader func(hdr *models.Header)
}

func NewFileSystem(
	ops *operations.Operations,

	metadata config.MetadataConfig,

	onHeader func(hdr *models.Header),
) afero.Fs {
	return &FileSystem{
		ops: ops,

		metadata: metadata,

		onHeader: onHeader,
	}
}

func (f *FileSystem) Name() string {
	log.Println("FileSystem.Name")

	return "STFS"
}

func (f *FileSystem) Create(name string) (afero.File, error) {
	log.Println("FileSystem.Name", name)

	panic(ErrNotImplemented)
}

func (f *FileSystem) Mkdir(name string, perm os.FileMode) error {
	log.Println("FileSystem.Mkdir", name, perm)

	panic(ErrNotImplemented)
}

func (f *FileSystem) MkdirAll(path string, perm os.FileMode) error {
	log.Println("FileSystem.MkdirAll", path, perm)

	panic(ErrNotImplemented)
}

func (f *FileSystem) Open(name string) (afero.File, error) {
	log.Println("FileSystem.Open", name)

	hdr, err := inventory.Stat(
		f.metadata,

		name,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, os.ErrNotExist
		}

		panic(err)
	}

	return NewFile(
		f.ops,

		f.metadata,

		hdr.Name,

		filepath.Base(hdr.Name),
		NewFileInfo(hdr),

		f.onHeader,
	), nil
}

func (f *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	log.Println("FileSystem.OpenFile", name, flag, perm)

	if flag != 0 {
		// TODO: Implement update and write
		panic(ErrNotImplemented)
	}

	// TODO: Implement `perm` support

	return f.Open(name)
}

func (f *FileSystem) Remove(name string) error {
	log.Println("FileSystem.Remove", name)

	panic(ErrNotImplemented)
}

func (f *FileSystem) RemoveAll(path string) error {
	log.Println("FileSystem.RemoveAll", path)

	panic(ErrNotImplemented)
}

func (f *FileSystem) Rename(oldname, newname string) error {
	log.Println("FileSystem.Rename", oldname, newname)

	panic(ErrNotImplemented)
}

func (f *FileSystem) Stat(name string) (os.FileInfo, error) {
	log.Println("FileSystem.Stat", name)

	hdr, err := inventory.Stat(
		f.metadata,

		name,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, os.ErrNotExist
		}

		panic(err)
	}

	return NewFileInfo(hdr), nil
}

func (f *FileSystem) Chmod(name string, mode os.FileMode) error {
	log.Println("FileSystem.Chmod", name, mode)

	panic(ErrNotImplemented)
}

func (f *FileSystem) Chown(name string, uid, gid int) error {
	log.Println("FileSystem.Chown", name, uid, gid)

	panic(ErrNotImplemented)
}

func (f *FileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	log.Println("FileSystem.Chtimes", name, atime, mtime)

	panic(ErrNotImplemented)
}

func (f *FileSystem) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	log.Println("FileSystem.LstatIfPossible", name)

	panic(ErrNotImplemented)
}

func (f *FileSystem) SymlinkIfPossible(oldname, newname string) error {
	log.Println("FileSystem.SymlinkIfPossible", oldname, newname)

	panic(ErrNotImplemented)
}

func (f *FileSystem) ReadlinkIfPossible(name string) (string, error) {
	log.Println("FileSystem.ReadlinkIfPossible", name)

	panic(ErrNotImplemented)
}
