package fs

import (
	"archive/tar"
	"database/sql"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	ifs "github.com/pojntfx/stfs/internal/fs"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/pojntfx/stfs/pkg/logging"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/spf13/afero"
)

type STFS struct {
	readOps  *operations.Operations
	writeOps *operations.Operations

	metadata config.MetadataConfig

	compressionLevel           string
	getFileBuffer              func() (cache.WriteCache, func() error, error)
	ignoreReadWritePermissions bool

	ioLock sync.Mutex

	onHeader func(hdr *config.Header)
	log      logging.StructuredLogger
}

func NewSTFS(
	readOps *operations.Operations,
	writeOps *operations.Operations,

	metadata config.MetadataConfig,

	compressionLevel string,
	getFileBuffer func() (cache.WriteCache, func() error, error),
	ignorePermissionFlags bool,

	onHeader func(hdr *config.Header),
	log logging.StructuredLogger,
) *STFS {
	return &STFS{
		readOps:  readOps,
		writeOps: writeOps,

		metadata: metadata,

		compressionLevel:           compressionLevel,
		getFileBuffer:              getFileBuffer,
		ignoreReadWritePermissions: ignorePermissionFlags,

		onHeader: onHeader,
		log:      log,
	}
}

func (f *STFS) Name() string {
	f.log.Debug("FileSystem.Name", map[string]interface{}{
		"name": config.FileSystemNameSTFS,
	})

	return config.FileSystemNameSTFS
}

func (f *STFS) Create(name string) (afero.File, error) {
	f.log.Debug("FileSystem.Name", map[string]interface{}{
		"name": name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return os.OpenFile(name, os.O_CREATE, 0666)
}

func (f *STFS) mknode(dir bool, name string, perm os.FileMode, overwrite bool, linkname string) error {
	f.log.Trace("FileSystem.mknode", map[string]interface{}{
		"name": name,
		"perm": perm,
	})

	usr, err := user.Current()
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		// Some OSes like i.e. Windows don't support numeric UIDs, so use 0 instead
		uid = 0
	}

	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		// Some OSes like i.e. Windows don't support numeric GIDs, so use 0 instead
		gid = 0
	}

	groups, err := usr.GroupIds()
	if err != nil {
		return err
	}

	gname := ""
	if len(groups) >= 1 {
		gname = groups[0]
	}

	typeflag := tar.TypeReg
	if dir {
		typeflag = tar.TypeDir
	} else if linkname != "" {
		typeflag = tar.TypeSymlink
	}

	hdr := &tar.Header{
		Typeflag: byte(typeflag),

		Name:     name,
		Linkname: linkname,

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
		overwrite,
	); err != nil {
		return err
	}

	return nil
}

func (f *STFS) MkdirRoot(name string, perm os.FileMode) error {
	f.log.Debug("FileSystem.MkdirRoot", map[string]interface{}{
		"name": name,
		"perm": perm,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.mknode(true, name, perm, true, "")
}

func (f *STFS) Mkdir(name string, perm os.FileMode) error {
	f.log.Debug("FileSystem.Mkdir", map[string]interface{}{
		"name": name,
		"perm": perm,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.mknode(true, name, perm, false, "")
}

func (f *STFS) MkdirAll(path string, perm os.FileMode) error {
	f.log.Debug("FileSystem.MkdirAll", map[string]interface{}{
		"path": path,
		"perm": perm,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	parts := filepath.SplitList(path)
	currentPath := ""

	for _, part := range parts {
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = filepath.Join(currentPath, part)
		}

		if err := f.mknode(true, currentPath, perm, false, ""); err != nil {
			return err
		}
	}

	return nil
}

func (f *STFS) Open(name string) (afero.File, error) {
	f.log.Debug("FileSystem.Open", map[string]interface{}{
		"name": name,
	})

	return f.OpenFile(name, os.O_RDWR, os.ModePerm)
}

func (f *STFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	f.log.Debug("FileSystem.OpenFile", map[string]interface{}{
		"name": name,
		"flag": flag,
		"perm": perm,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	flags := &ifs.FileFlags{}
	if flag&os.O_RDONLY != 0 {
		flags.Read = true
	}

	if flag&os.O_WRONLY != 0 {
		flags.Write = true
	}

	if flag&os.O_RDWR != 0 {
		flags.Read = true
		flags.Write = true
	}

	if f.ignoreReadWritePermissions {
		flags.Read = true
		flags.Write = true
	}

	if flag&os.O_APPEND != 0 {
		flags.Append = true
	}

	if flag&os.O_TRUNC != 0 {
		flags.Truncate = true
	}

	hdr, err := inventory.Stat(
		f.metadata,

		name,
		false,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			if flag&os.O_CREATE != 0 && flag&os.O_EXCL == 0 {
				if err := f.mknode(false, name, perm, false, ""); err != nil {
					return nil, err
				}

				hdr, err = inventory.Stat(
					f.metadata,

					name,
					false,

					f.onHeader,
				)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, os.ErrNotExist
			}

		} else {
			return nil, err
		}
	}

	return ifs.NewFile(
		f.readOps,
		f.writeOps,

		f.metadata,

		hdr.Name,
		hdr.Linkname,
		flags,

		f.compressionLevel,
		f.getFileBuffer,
		&f.ioLock,

		path.Base(hdr.Name),
		ifs.NewFileInfoFromTarHeader(hdr, f.log),

		f.onHeader,
		f.log,
	), nil
}

func (f *STFS) Remove(name string) error {
	f.log.Debug("FileSystem.Remove", map[string]interface{}{
		"name": name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.writeOps.Delete(name)
}

func (f *STFS) RemoveAll(path string) error {
	f.log.Debug("FileSystem.RemoveAll", map[string]interface{}{
		"path": path,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.writeOps.Delete(path)
}

func (f *STFS) Rename(oldname, newname string) error {
	f.log.Debug("FileSystem.Rename", map[string]interface{}{
		"oldname": oldname,
		"newname": newname,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.writeOps.Move(oldname, newname)
}

func (f *STFS) Stat(name string) (os.FileInfo, error) {
	f.log.Debug("FileSystem.Stat", map[string]interface{}{
		"name": name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	hdr, err := inventory.Stat(
		f.metadata,

		name,
		false,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	return ifs.NewFileInfoFromTarHeader(hdr, f.log), nil
}

func (f *STFS) updateMetadata(hdr *tar.Header) error {
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
		false,
	); err != nil {
		return err
	}

	return nil
}

func (f *STFS) Chmod(name string, mode os.FileMode) error {
	f.log.Debug("FileSystem.Chmod", map[string]interface{}{
		"name": mode,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	hdr, err := inventory.Stat(
		f.metadata,

		name,
		false,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return os.ErrNotExist
		}

		return err
	}

	hdr.Mode = int64(mode)

	return f.updateMetadata(hdr)
}

func (f *STFS) Chown(name string, uid, gid int) error {
	f.log.Debug("FileSystem.Chown", map[string]interface{}{
		"name": name,
		"uid":  uid,
		"gid":  gid,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	hdr, err := inventory.Stat(
		f.metadata,

		name,
		false,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return os.ErrNotExist
		}

		return err
	}

	hdr.Uid = uid
	hdr.Gid = gid

	return f.updateMetadata(hdr)
}

func (f *STFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	f.log.Debug("FileSystem.Chtimes", map[string]interface{}{
		"name":  name,
		"atime": atime,
		"mtime": mtime,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	hdr, err := inventory.Stat(
		f.metadata,

		name,
		false,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return os.ErrNotExist
		}

		return err
	}

	hdr.AccessTime = atime
	hdr.ModTime = mtime

	return f.updateMetadata(hdr)
}

func (f *STFS) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	f.log.Debug("FileSystem.LstatIfPossible", map[string]interface{}{
		"name": name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	hdr, err := inventory.Stat(
		f.metadata,

		name,
		true,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, true, os.ErrNotExist
		}

		return nil, true, err
	}

	return ifs.NewFileInfoFromTarHeader(hdr, f.log), true, nil
}

func (f *STFS) SymlinkIfPossible(oldname, newname string) error {
	f.log.Debug("FileSystem.SymlinkIfPossible", map[string]interface{}{
		"oldname": oldname,
		"newname": newname,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.mknode(false, oldname, os.ModePerm, false, newname)
}

func (f *STFS) ReadlinkIfPossible(name string) (string, error) {
	f.log.Debug("FileSystem.ReadlinkIfPossible", map[string]interface{}{
		"name": name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return "", config.ErrNotImplemented
}
