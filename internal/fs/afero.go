package fs

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/afero"
)

var (
	ErrNotImplemented = errors.New("not implemented")
)

type STFS struct{}

func NewSTFS() afero.Fs {
	return &STFS{}
}

func (STFS) Name() string { return "STFS" }

func (STFS) Create(name string) (afero.File, error) {
	panic(ErrNotImplemented)

	// f, e := os.Create(name)
	// if f == nil {
	// 	return nil, e
	// }
	// return f, e
}

func (STFS) Mkdir(name string, perm os.FileMode) error {
	panic(ErrNotImplemented)

	// return os.Mkdir(name, perm)
}

func (STFS) MkdirAll(path string, perm os.FileMode) error {
	panic(ErrNotImplemented)

	// return os.MkdirAll(path, perm)
}

func (STFS) Open(name string) (afero.File, error) {
	panic(ErrNotImplemented)

	// f, e := os.Open(name)
	// if f == nil {
	// 	return nil, e
	// }
	// return f, e
}

func (STFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	panic(ErrNotImplemented)

	// f, e := os.OpenFile(name, flag, perm)
	// if f == nil {
	// 	return nil, e
	// }
	// return f, e
}

func (STFS) Remove(name string) error {
	panic(ErrNotImplemented)

	// return os.Remove(name)
}

func (STFS) RemoveAll(path string) error {
	panic(ErrNotImplemented)

	// return os.RemoveAll(path)
}

func (STFS) Rename(oldname, newname string) error {
	panic(ErrNotImplemented)

	// return os.Rename(oldname, newname)
}

func (STFS) Stat(name string) (os.FileInfo, error) {
	panic(ErrNotImplemented)

	// return os.Stat(name)
}

func (STFS) Chmod(name string, mode os.FileMode) error {
	panic(ErrNotImplemented)

	// return os.Chmod(name, mode)
}

func (STFS) Chown(name string, uid, gid int) error {
	panic(ErrNotImplemented)

	// return os.Chown(name, uid, gid)
}

func (STFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	panic(ErrNotImplemented)

	// return os.Chtimes(name, atime, mtime)
}

func (STFS) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	panic(ErrNotImplemented)

	// fi, err := os.Lstat(name)
	// return fi, true, err
}

func (STFS) SymlinkIfPossible(oldname, newname string) error {
	panic(ErrNotImplemented)

	// return os.Symlink(oldname, newname)
}

func (STFS) ReadlinkIfPossible(name string) (string, error) {
	panic(ErrNotImplemented)

	// return os.Readlink(name)
}
