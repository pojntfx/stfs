package fs

import (
	"archive/tar"
	"context"
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
	"github.com/pojntfx/stfs/pkg/encryption"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/pojntfx/stfs/pkg/logging"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/recovery"
	"github.com/pojntfx/stfs/pkg/signature"
	"github.com/spf13/afero"
)

type STFS struct {
	readOps  *operations.Operations
	writeOps *operations.Operations

	metadata config.MetadataConfig

	compressionLevel string
	getFileBuffer    func() (cache.WriteCache, func() error, error)
	readOnly         bool

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
	readOnly bool,

	onHeader func(hdr *config.Header),
	log logging.StructuredLogger,
) *STFS {
	return &STFS{
		readOps:  readOps,
		writeOps: writeOps,

		metadata: metadata,

		compressionLevel: compressionLevel,
		getFileBuffer:    getFileBuffer,
		readOnly:         readOnly,

		onHeader: onHeader,
		log:      log,
	}
}

func (f *STFS) Name() string {
	f.log.Debug("FileSystem.Name", map[string]interface{}{
		"name": config.FileSystemNameSTFS,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return config.FileSystemNameSTFS
}

func (f *STFS) Create(name string) (afero.File, error) {
	f.log.Debug("FileSystem.Name", map[string]interface{}{
		"name": name,
	})

	if f.readOnly {
		return nil, os.ErrPermission
	}

	return f.OpenFile(name, os.O_CREATE|os.O_RDWR, 0666)
}

func (f *STFS) mknodeWithoutLocking(dir bool, name string, perm os.FileMode, overwrite bool, linkname string, initializing bool) error {
	f.log.Trace("FileSystem.mknodeWithoutLocking", map[string]interface{}{
		"name": name,
		"perm": perm,
	})

	if f.readOnly {
		return os.ErrPermission
	}

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
		initializing,
	); err != nil {
		return err
	}

	return nil
}

func (f *STFS) Initialize(rootProposal string, rootPerm os.FileMode) (root string, err error) {
	f.log.Debug("FileSystem.InitializeIfEmpty", map[string]interface{}{
		"rootProposal": rootProposal,
		"rootPerm":     rootPerm,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	existingRoot, err := f.metadata.Metadata.GetRootPath(context.Background())
	if err == config.ErrNoRootDirectory {
		drive, err := f.readOps.GetBackend().GetDrive()

		mkdirRoot := func() (string, error) {
			if err := f.readOps.GetBackend().CloseDrive(); err != nil {
				return "", err
			}

			if f.readOnly {
				return "", os.ErrPermission
			}

			if err := f.mknodeWithoutLocking(true, rootProposal, rootPerm, true, "", true); err != nil {
				return "", err
			}

			// Ensure that the new root path is being used
			return f.metadata.Metadata.GetRootPath(context.Background())
		}

		if err != nil {
			return mkdirRoot()
		}

		if err := recovery.Index(
			config.DriveReaderConfig{
				Drive:          drive.Drive,
				DriveIsRegular: drive.DriveIsRegular,
			},
			drive,
			f.readOps.GetMetadata(),
			f.readOps.GetPipes(),
			f.readOps.GetCrypto(),

			0,
			0,
			true,
			true,
			0,

			func(hdr *tar.Header, i int) error {
				return encryption.DecryptHeader(hdr, f.readOps.GetPipes().Encryption, f.readOps.GetCrypto().Identity)
			},
			func(hdr *tar.Header, isRegular bool) error {
				return signature.VerifyHeader(hdr, isRegular, f.readOps.GetPipes().Signature, f.readOps.GetCrypto().Recipient)
			},

			f.onHeader,
		); err != nil {
			return mkdirRoot()
		}

		// Ensure that the new root path is being used
		return f.metadata.Metadata.GetRootPath(context.Background())
	} else if err != nil {
		return "", err
	}

	return existingRoot, nil
}

func (f *STFS) Mkdir(name string, perm os.FileMode) error {
	f.log.Debug("FileSystem.Mkdir", map[string]interface{}{
		"name": name,
		"perm": perm,
	})

	if f.readOnly {
		return os.ErrPermission
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.mknodeWithoutLocking(true, name, perm, false, "", false)
}

func (f *STFS) MkdirAll(path string, perm os.FileMode) error {
	f.log.Debug("FileSystem.MkdirAll", map[string]interface{}{
		"path": path,
		"perm": perm,
	})

	if f.readOnly {
		return os.ErrPermission
	}

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

		if err := f.mknodeWithoutLocking(true, currentPath, perm, false, "", false); err != nil {
			return err
		}
	}

	return nil
}

func (f *STFS) Open(name string) (afero.File, error) {
	f.log.Debug("FileSystem.Open", map[string]interface{}{
		"name": name,
	})

	return f.OpenFile(name, os.O_RDONLY, 0)
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
	if f.readOnly {
		if (flag&O_ACCMODE) == os.O_RDONLY || (flag&O_ACCMODE) == os.O_RDWR {
			flags.Read = true
		}
	} else {
		if (flag & O_ACCMODE) == os.O_RDONLY {
			flags.Read = true
		} else if (flag & O_ACCMODE) == os.O_WRONLY {
			flags.Write = true
		} else if (flag & O_ACCMODE) == os.O_RDWR {
			flags.Read = true
			flags.Write = true
		}

		if flag&os.O_APPEND != 0 {
			flags.Append = true
		}

		if flag&os.O_TRUNC != 0 {
			flags.Truncate = true
		}
	}

	hdr, err := inventory.Stat(
		f.metadata,

		name,
		false,

		f.onHeader,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			if !f.readOnly && flag&os.O_CREATE != 0 && flag&os.O_EXCL == 0 {
				if err := f.mknodeWithoutLocking(false, name, perm, false, "", false); err != nil {
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

	if f.readOnly {
		return os.ErrPermission
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.writeOps.Delete(name)
}

func (f *STFS) RemoveAll(path string) error {
	f.log.Debug("FileSystem.RemoveAll", map[string]interface{}{
		"path": path,
	})

	if f.readOnly {
		return os.ErrPermission
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.writeOps.Delete(path)
}

func (f *STFS) Rename(oldname, newname string) error {
	f.log.Debug("FileSystem.Rename", map[string]interface{}{
		"oldname": oldname,
		"newname": newname,
	})

	if f.readOnly {
		return os.ErrPermission
	}

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
	if f.readOnly {
		return os.ErrPermission
	}

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

	if f.readOnly {
		return os.ErrPermission
	}

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

	if f.readOnly {
		return os.ErrPermission
	}

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

	if f.readOnly {
		return os.ErrPermission
	}

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

func (f *STFS) lstatIfPossibleWithoutLocking(name string) (os.FileInfo, bool, error) {
	f.log.Debug("FileSystem.lstatIfPossibleWithoutLocking", map[string]interface{}{
		"name": name,
	})

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

func (f *STFS) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	f.log.Debug("FileSystem.LstatIfPossible", map[string]interface{}{
		"name": name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.lstatIfPossibleWithoutLocking(name)
}

func (f *STFS) SymlinkIfPossible(oldname, newname string) error {
	f.log.Debug("FileSystem.SymlinkIfPossible", map[string]interface{}{
		"oldname": oldname,
		"newname": newname,
	})

	if f.readOnly {
		return os.ErrPermission
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.mknodeWithoutLocking(false, oldname, os.ModePerm, false, newname, false)
}

func (f *STFS) ReadlinkIfPossible(name string) (string, error) {
	f.log.Debug("FileSystem.ReadlinkIfPossible", map[string]interface{}{
		"name": name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	info, _, err := f.lstatIfPossibleWithoutLocking(name)
	if err != nil {
		return "", err
	}

	return info.Name(), nil
}
