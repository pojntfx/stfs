package fs

import (
	"archive/tar"
	"database/sql"
	"errors"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
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
	readOps  *operations.Operations
	writeOps *operations.Operations

	metadata config.MetadataConfig

	compressionLevel string

	onHeader func(hdr *models.Header)
}

func NewFileSystem(
	readOps *operations.Operations,
	writeOps *operations.Operations,

	metadata config.MetadataConfig,

	compressionLevel string,

	onHeader func(hdr *models.Header),
) afero.Fs {
	return &FileSystem{
		readOps:  readOps,
		writeOps: writeOps,

		metadata: metadata,

		compressionLevel: compressionLevel,

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

	usr, err := user.Current()
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		return err
	}

	groups, err := usr.GroupIds()
	if err != nil {
		return err
	}

	gname := ""
	if len(groups) >= 1 {
		gname = groups[0]
	}

	hdr := &tar.Header{
		Typeflag: tar.TypeDir,

		Name: name,

		Mode:  int64(perm),
		Uid:   uid,
		Gid:   gid,
		Uname: usr.Username,
		Gname: gname,

		ModTime: time.Now(),
	}

	done := false
	if _, err := f.writeOps.Archive(
		func() (config.FileConfig, error) {
			// Exit after the first write
			if done {
				return config.FileConfig{}, io.EOF
			}
			done = true

			return config.FileConfig{
				GetFile: nil, // Not required as we never replace
				Info:    hdr.FileInfo(),
				Path:    filepath.ToSlash(name),
				Link:    filepath.ToSlash(hdr.Linkname),
			}, nil
		},
		f.compressionLevel,
		false,
	); err != nil {
		return err
	}

	return nil
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
		f.readOps,

		f.metadata,

		hdr.Name,

		path.Base(hdr.Name),
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

	return f.writeOps.Delete(name)
}

func (f *FileSystem) RemoveAll(path string) error {
	log.Println("FileSystem.RemoveAll", path)

	return f.writeOps.Delete(path)
}

func (f *FileSystem) Rename(oldname, newname string) error {
	log.Println("FileSystem.Rename", oldname, newname)

	return f.writeOps.Move(oldname, newname)
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

func (f *FileSystem) updateMetadata(hdr *tar.Header) error {
	done := false
	if _, err := f.writeOps.Update(
		func() (config.FileConfig, error) {
			// Exit after the first update
			if done {
				return config.FileConfig{}, io.EOF
			}
			done = true

			return config.FileConfig{
				GetFile: nil, // Not required as we never replace
				Info:    hdr.FileInfo(),
				Path:    filepath.ToSlash(hdr.Name),
				Link:    filepath.ToSlash(hdr.Linkname),
			}, nil
		},
		f.compressionLevel,
		false,
	); err != nil {
		return err
	}

	return nil
}

func (f *FileSystem) Chmod(name string, mode os.FileMode) error {
	log.Println("FileSystem.Chmod", name, mode)

	hdr, err := inventory.Stat(
		f.metadata,

		name,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return os.ErrNotExist
		}

		panic(err)
	}

	hdr.Mode = int64(mode)

	return f.updateMetadata(hdr)
}

func (f *FileSystem) Chown(name string, uid, gid int) error {
	log.Println("FileSystem.Chown", name, uid, gid)

	hdr, err := inventory.Stat(
		f.metadata,

		name,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return os.ErrNotExist
		}

		panic(err)
	}

	hdr.Uid = uid
	hdr.Gid = gid

	return f.updateMetadata(hdr)
}

func (f *FileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	log.Println("FileSystem.Chtimes", name, atime, mtime)

	hdr, err := inventory.Stat(
		f.metadata,

		name,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return os.ErrNotExist
		}

		panic(err)
	}

	hdr.AccessTime = atime
	hdr.ModTime = mtime

	return f.updateMetadata(hdr)
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
