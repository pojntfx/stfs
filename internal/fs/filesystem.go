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

func (S *FileSystem) Name() string {
	log.Println("FileSystem.Name")

	return "STFS"
}

func (S *FileSystem) Create(name string) (afero.File, error) {
	log.Println("FileSystem.Name", name)

	panic(ErrNotImplemented)
}

func (S *FileSystem) Mkdir(name string, perm os.FileMode) error {
	log.Println("FileSystem.Mkdir", name, perm)

	panic(ErrNotImplemented)
}

func (S *FileSystem) MkdirAll(path string, perm os.FileMode) error {
	log.Println("FileSystem.MkdirAll", path, perm)

	panic(ErrNotImplemented)
}

func (s *FileSystem) Open(name string) (afero.File, error) {
	log.Println("FileSystem.Open", name)

	hdr, err := inventory.Stat(
		s.metadata,

		name,

		s.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, os.ErrNotExist
		}

		panic(err)
	}

	return NewFile(
		filepath.Base(hdr.Name),
		NewFileInfo(hdr),
	), nil
}

func (S *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	log.Println("FileSystem.OpenFile", name, flag, perm)

	panic(ErrNotImplemented)
}

func (S *FileSystem) Remove(name string) error {
	log.Println("FileSystem.Remove", name)

	panic(ErrNotImplemented)
}

func (S *FileSystem) RemoveAll(path string) error {
	log.Println("FileSystem.RemoveAll", path)

	panic(ErrNotImplemented)
}

func (S *FileSystem) Rename(oldname, newname string) error {
	log.Println("FileSystem.Rename", oldname, newname)

	panic(ErrNotImplemented)
}

func (S *FileSystem) Stat(name string) (os.FileInfo, error) {
	log.Println("FileSystem.Stat", name)

	panic(ErrNotImplemented)
}

func (S *FileSystem) Chmod(name string, mode os.FileMode) error {
	log.Println("FileSystem.Chmod", name, mode)

	panic(ErrNotImplemented)
}

func (S *FileSystem) Chown(name string, uid, gid int) error {
	log.Println("FileSystem.Chown", name, uid, gid)

	panic(ErrNotImplemented)
}

func (S *FileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	log.Println("FileSystem.Chtimes", name, atime, mtime)

	panic(ErrNotImplemented)
}

func (S *FileSystem) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	log.Println("FileSystem.LstatIfPossible", name)

	panic(ErrNotImplemented)
}

func (S *FileSystem) SymlinkIfPossible(oldname, newname string) error {
	log.Println("FileSystem.SymlinkIfPossible", oldname, newname)

	panic(ErrNotImplemented)
}

func (S *FileSystem) ReadlinkIfPossible(name string) (string, error) {
	log.Println("FileSystem.ReadlinkIfPossible", name)

	panic(ErrNotImplemented)
}
