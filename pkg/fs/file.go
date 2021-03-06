package fs

import (
	"bytes"
	"database/sql"
	"io"
	"io/fs"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/pathext"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/pojntfx/stfs/pkg/logging"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/spf13/afero"
)

type FileFlags struct {
	Read  bool
	Write bool

	Append   bool
	Truncate bool
}

type File struct {
	afero.File

	readOps  *operations.Operations
	writeOps *operations.Operations

	metadata config.MetadataConfig

	path  string
	link  string
	flags *FileFlags

	compressionLevel string
	getFileBuffer    func() (cache.WriteCache, func() error, error)

	name string
	info *FileInfo

	ioLock *sync.Mutex

	readOpReader *ioext.CounterReadCloser
	readOpWriter io.WriteCloser

	writeBuf      cache.WriteCache
	cleanWriteBuf func() error

	onHeader func(hdr *config.Header)
	log      logging.StructuredLogger
}

func NewFile(
	readOps *operations.Operations,
	writeOps *operations.Operations,

	metadata config.MetadataConfig,

	path string,
	link string,
	flags *FileFlags,

	compressionLevel string,
	getFileBuffer func() (cache.WriteCache, func() error, error),
	ioLock *sync.Mutex,

	name string,
	info *FileInfo,

	onHeader func(hdr *config.Header),
	log logging.StructuredLogger,
) *File {
	return &File{
		readOps:  readOps,
		writeOps: writeOps,

		metadata: metadata,

		path:  path,
		link:  link,
		flags: flags,

		compressionLevel: compressionLevel,
		getFileBuffer:    getFileBuffer,
		ioLock:           ioLock,

		name: name,
		info: info,

		onHeader: onHeader,
		log:      log,
	}
}

func (f *File) syncWithoutLocking() error {
	f.log.Trace("File.syncWithoutLocking", map[string]interface{}{
		"name": f.name,
	})

	if f.info.IsDir() {
		return config.ErrIsDirectory
	}

	if f.writeBuf != nil {
		done := false
		if _, err := f.writeOps.Update(
			func() (config.FileConfig, error) {
				// Exit after the first write
				if done {
					return config.FileConfig{}, io.EOF
				}
				done = true

				size, err := f.writeBuf.Size()
				if err != nil {
					return config.FileConfig{}, err
				}

				// Some OSes like i.e. Windows don't support numeric GIDs and UIDs, so use 0 instead
				gid := 0
				uid := 0
				modTime := f.info.ModTime()
				accessTime := f.info.ModTime()
				changeTime := f.info.ModTime()
				sys, ok := f.info.Sys().(*Stat)
				if ok {
					gid = int(sys.Gid)
					uid = int(sys.Uid)
					accessTime = time.Unix(0, sys.Atim.Nano())
					changeTime = time.Unix(0, sys.Ctim.Nano())
				}

				f.info = NewFileInfo(
					f.info.Name(),
					size,
					f.info.Mode(),
					modTime,
					accessTime,
					changeTime,
					gid,
					uid,
					f.info.IsDir(),
					f.log,
				)

				return config.FileConfig{
					GetFile: func() (io.ReadSeekCloser, error) {
						if _, err := f.writeBuf.Seek(0, io.SeekStart); err != nil {
							return nil, err
						}

						return f.writeBuf, nil
					},
					Info: f.info,
					Path: f.path,
					Link: f.link,
				}, nil
			},
			f.compressionLevel,
			true,
			true,
		); err != nil {
			return err
		}
	}

	return nil
}

func (f *File) closeWithoutLocking() error {
	f.log.Trace("File.closeWithoutLocking", map[string]interface{}{
		"name": f.name,
	})

	if f.readOpReader != nil {
		if err := f.readOpReader.Close(); err != nil {
			return err
		}
	}

	if f.readOpWriter != nil {
		if err := f.readOpWriter.Close(); err != nil {
			return err
		}
	}

	if f.writeBuf != nil {
		// No need to close write buffer, the `update` operation closes it itself
		if err := f.syncWithoutLocking(); err != nil {
			return err
		}

		if err := f.cleanWriteBuf(); err != nil {
			return err
		}
	}

	f.readOpReader = nil
	f.readOpWriter = nil
	f.writeBuf = nil

	return nil
}

func (f *File) enterWriteMode() error {
	f.log.Trace("File.enterWriteMode", map[string]interface{}{
		"name": f.name,
	})

	if f.readOpReader != nil || f.readOpWriter != nil {
		if err := f.closeWithoutLocking(); err != nil {
			return err
		}
	}

	if f.writeBuf == nil {
		exists := false
		existingFile, err := inventory.Stat(
			f.metadata,

			f.path,
			false,

			f.onHeader,
		)
		if err == nil {
			if existingFile.Size != 0 {
				exists = true
			}
		} else {
			if err != sql.ErrNoRows {
				return err
			}
		}

		// Create new buffer
		writeBuf, cleanWriteBuf, err := f.getFileBuffer()
		if err != nil {
			return err
		}

		f.writeBuf = writeBuf
		f.cleanWriteBuf = cleanWriteBuf

		// Read existing file into buffer
		if exists {
			if err := f.readOps.Restore(
				func(path string, mode fs.FileMode) (io.WriteCloser, error) {
					// Don't close the file here, we want to re-use it!
					return ioext.AddCloseNopToWriter(f.writeBuf), nil
				},
				func(path string, mode fs.FileMode) error {
					// Not necessary; can't read on a directory
					return nil
				},

				f.path,
				"",
				true,
			); err != nil {
				return err
			}
		}

		if f.flags.Truncate {
			if err := f.writeBuf.Truncate(0); err != nil {
				return err
			}
		}

		if !f.flags.Append {
			if _, err := f.writeBuf.Seek(0, io.SeekStart); err != nil {
				return err
			}
		}
	}

	return nil
}

func (f *File) seekWithoutLocking(offset int64, whence int) (int64, error) {
	f.log.Trace("File.seekWithoutLocking", map[string]interface{}{
		"name":   f.name,
		"offset": offset,
		"whence": whence,
	})

	if f.info.IsDir() {
		// Noop
		return 0, nil
	}

	if f.writeBuf != nil {
		return f.writeBuf.Seek(offset, whence)
	}

	dst := int64(0)
	switch whence {
	case io.SeekStart:
		dst = offset
	case io.SeekCurrent:
		curr := 0
		if f.readOpReader != nil {
			curr = f.readOpReader.BytesRead
		}
		dst = int64(curr) + offset
	case io.SeekEnd:
		dst = f.info.Size() - offset
	default:
		return -1, config.ErrNotImplemented
	}

	if f.readOpReader == nil || f.readOpWriter == nil || dst < int64(f.readOpReader.BytesRead) { // We have to re-open as we can't seek backwards
		_ = f.closeWithoutLocking() // Ignore errors here as it might not be opened

		r, writer := io.Pipe()
		reader := &ioext.CounterReadCloser{
			Reader:    r,
			BytesRead: 0,
		}

		go func() {
			if err := f.readOps.Restore(
				func(path string, mode fs.FileMode) (io.WriteCloser, error) {
					return writer, nil
				},
				func(path string, mode fs.FileMode) error {
					// Not necessary; can't read on a directory
					return nil
				},

				f.path,
				"",
				true,
			); err != nil {
				if err == io.ErrClosedPipe {
					return
				}

				// TODO: Handle error
				panic(err)
			}
		}()

		f.readOpReader = reader
		f.readOpWriter = writer
	}

	written, err := io.CopyN(io.Discard, f.readOpReader, dst-int64(f.readOpReader.BytesRead))
	if err == io.EOF {
		// Noop
		switch whence {
		case io.SeekStart:
			return offset, nil
		case io.SeekCurrent:
			return int64(f.readOpReader.BytesRead) + offset, nil
		case io.SeekEnd:
			return int64(f.info.Size()) - offset, nil
		default:
			return -1, config.ErrNotImplemented
		}
	}

	if err != nil {
		return -1, err
	}

	return written, nil
}

// Inventory
func (f *File) Name() string {
	f.log.Trace("File.Name", map[string]interface{}{
		"name": f.name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if f.link != "" {
		if pathext.IsRoot(f.link, false) {
			return ""
		}

		return f.link
	}

	if pathext.IsRoot(f.path, false) {
		return ""
	}

	return f.path
}

func (f *File) Stat() (os.FileInfo, error) {
	f.log.Trace("File.Stat", map[string]interface{}{
		"name": f.name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if f.writeBuf != nil {
		size, err := f.writeBuf.Size()
		if err != nil {
			return nil, err
		}

		f.info.size = size
	}

	if f.link != "" {
		info := f.info

		info.name = path.Base(f.link)

		return info, nil
	}

	return f.info, nil
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	f.log.Trace("File.Readdir", map[string]interface{}{
		"name":  f.name,
		"count": count,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if !f.info.IsDir() {
		return []os.FileInfo{}, config.ErrIsFile
	}
	hdrs, err := inventory.List(
		f.metadata,

		f.path,
		count,

		f.onHeader,
	)
	if err != nil {
		return nil, err
	}

	fileInfos := []os.FileInfo{}
	for _, hdr := range hdrs {
		fileInfos = append(fileInfos, NewFileInfoFromTarHeader(hdr, f.log))
	}

	return fileInfos, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	f.log.Trace("File.Readdirnames", map[string]interface{}{
		"name": f.name,
		"n":    n,
	})

	if !f.info.IsDir() {
		return []string{}, config.ErrIsFile
	}

	dirs, err := f.Readdir(n)
	if err != nil {
		return []string{}, err
	}

	names := []string{}
	for _, dir := range dirs {
		names = append(names, dir.Name())
	}

	return names, err
}

// Read operations
func (f *File) Read(p []byte) (n int, err error) {
	f.log.Trace("File.Read", map[string]interface{}{
		"name": f.name,
		"p":    len(p),
	})

	if !f.flags.Read {
		return -1, os.ErrPermission
	}

	if len(p) <= 0 {
		return 0, nil
	}

	if f.info.IsDir() {
		return -1, config.ErrIsDirectory
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if f.writeBuf != nil {
		return f.writeBuf.Read(p)
	}

	if f.readOpReader == nil || f.readOpWriter == nil {
		r, writer := io.Pipe()
		reader := &ioext.CounterReadCloser{
			Reader:    r,
			BytesRead: 0,
		}

		go func() {
			if err := f.readOps.Restore(
				func(path string, mode fs.FileMode) (io.WriteCloser, error) {
					return writer, nil
				},
				func(path string, mode fs.FileMode) error {
					// Not necessary; can't read on a directory
					return nil
				},

				f.path,
				"",
				true,
			); err != nil {
				if err == io.ErrClosedPipe {
					return
				}

				// TODO: Handle error
				panic(err)
			}
		}()

		f.readOpReader = reader
		f.readOpWriter = writer
	}

	w := &bytes.Buffer{}
	_, err = io.CopyN(w, f.readOpReader, int64(len(p)))
	if err == io.EOF {
		return copy(p, w.Bytes()), io.EOF
	}

	if err != nil {
		return -1, err
	}

	return copy(p, w.Bytes()), nil
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	f.log.Trace("File.ReadAt", map[string]interface{}{
		"name": f.name,
		"p":    len(p),
		"off":  off,
	})

	if !f.flags.Read {
		return -1, os.ErrPermission
	}

	if len(p) <= 0 {
		return 0, nil
	}

	if f.info.IsDir() {
		return -1, config.ErrIsDirectory
	}

	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return -1, err
	}

	return f.Read(p)
}

// Read/write operations
func (f *File) Seek(offset int64, whence int) (int64, error) {
	f.log.Trace("File.Seek", map[string]interface{}{
		"name":   f.name,
		"offset": offset,
		"whence": whence,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.seekWithoutLocking(offset, whence)
}

// Write operations
func (f *File) Write(p []byte) (n int, err error) {
	f.log.Trace("File.Write", map[string]interface{}{
		"name": f.name,
		"p":    len(p),
	})

	if f.info.IsDir() {
		return -1, config.ErrIsDirectory
	}

	if !f.flags.Write {
		return -1, os.ErrPermission
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterWriteMode(); err != nil {
		return -1, err
	}

	n, err = f.writeBuf.Write(p)
	if err != nil {
		return -1, err
	}

	return n, nil
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	f.log.Trace("File.WriteAt", map[string]interface{}{
		"name": f.name,
		"p":    len(p),
		"off":  off,
	})

	if f.info.IsDir() {
		return -1, config.ErrIsDirectory
	}

	if !f.flags.Write {
		return -1, os.ErrPermission
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterWriteMode(); err != nil {
		return -1, err
	}

	if _, err := f.seekWithoutLocking(off, io.SeekStart); err != nil {
		return -1, err
	}

	return f.writeBuf.Write(p)
}

func (f *File) WriteString(s string) (ret int, err error) {
	f.log.Trace("File.WriteString", map[string]interface{}{
		"name": f.name,
		"s":    len(s),
	})

	if f.info.IsDir() {
		return -1, config.ErrIsDirectory
	}

	if !f.flags.Write {
		return -1, os.ErrPermission
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterWriteMode(); err != nil {
		return -1, err
	}

	return f.writeBuf.Write([]byte(s))
}

func (f *File) Truncate(size int64) error {
	f.log.Trace("File.Truncate", map[string]interface{}{
		"name": f.name,
		"size": size,
	})

	if f.info.IsDir() {
		return config.ErrIsDirectory
	}

	if !f.flags.Write {
		return os.ErrPermission
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterWriteMode(); err != nil {
		return err
	}

	oldSize, err := f.writeBuf.Size()
	if err != nil {
		return err
	}

	if size > oldSize {
		if err := f.writeBuf.Truncate(0); err != nil {
			return err
		}

		for i := int64(0); i < size; i++ {
			if _, err := f.writeBuf.Write(make([]byte, 1)); err != nil {
				return err
			}
		}

		return nil
	}

	if err := f.writeBuf.Truncate(size); err != nil {
		return err
	}

	return nil
}

// Cleanup operations
func (f *File) Sync() error {
	f.log.Trace("File.Sync", map[string]interface{}{
		"name": f.name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.syncWithoutLocking()
}

func (f *File) Close() error {
	f.log.Debug("File.Close", map[string]interface{}{
		"name": f.name,
	})

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.closeWithoutLocking()
}
