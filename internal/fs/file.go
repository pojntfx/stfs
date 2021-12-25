package fs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"sync"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/spf13/afero"
)

var (
	ErrIsDirectory = errors.New("is a directory")
)

type WriteCache interface {
	io.Closer
	io.Reader
	io.Seeker
	io.Writer

	Truncate(size int64) error
	Size() (int64, error)
	Sync() error
}

type File struct {
	afero.File

	readOps  *operations.Operations
	writeOps *operations.Operations

	metadata config.MetadataConfig

	path string
	link string

	compressionLevel string
	getFileBuffer    func() (WriteCache, func() error, error)

	name string
	info os.FileInfo

	ioLock sync.Mutex

	readOpReader *ioext.CounterReadCloser
	readOpWriter io.WriteCloser

	writeBuf      WriteCache
	cleanWriteBuf func() error

	onHeader func(hdr *models.Header)
}

func NewFile(
	readOps *operations.Operations,
	writeOps *operations.Operations,

	metadata config.MetadataConfig,

	path string,
	link string,

	compressionLevel string,
	getFileBuffer func() (WriteCache, func() error, error),

	name string,
	info os.FileInfo,

	onHeader func(hdr *models.Header),
) *File {
	return &File{
		readOps:  readOps,
		writeOps: writeOps,

		metadata: metadata,

		path: path,
		link: link,

		compressionLevel: compressionLevel,
		getFileBuffer:    getFileBuffer,

		name: name,
		info: info,

		onHeader: onHeader,
	}
}

func (f *File) Name() string {
	log.Println("File.Name", f.name)

	return f.name
}

func (f *File) Stat() (os.FileInfo, error) {
	log.Println("File.Stat", f.name)

	return f.info, nil
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	log.Println("File.Readdir", f.name, count)

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
		fileInfos = append(fileInfos, NewFileInfo(hdr))
	}

	return fileInfos, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	log.Println("File.Readdirnames", f.name, n)

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

func (f *File) syncWithoutLocking() error {
	log.Println("File.syncWithoutLocking", f.name)

	if f.info.IsDir() {
		return ErrIsDirectory
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

				f.info = &FileInfo{
					name:    f.info.Name(),
					size:    size,
					mode:    f.info.Mode(),
					modTime: f.info.ModTime(),
					isDir:   f.info.IsDir(),
				}

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

func (f *File) Sync() error {
	log.Println("File.Sync", f.name)

	return f.syncWithoutLocking()
}

func (f *File) enterWriteMode() error {
	if f.readOpReader != nil || f.readOpWriter != nil {
		if err := f.closeWithoutLocking(); err != nil {
			return err
		}
	}

	return nil
}

func (f *File) Truncate(size int64) error {
	log.Println("File.Truncate", f.name, size)

	if f.info.IsDir() {
		return ErrIsDirectory
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterWriteMode(); err != nil {
		return err
	}

	if f.writeBuf == nil {
		writeBuf, cleanWriteBuf, err := f.getFileBuffer()
		if err != nil {
			return err
		}

		f.writeBuf = writeBuf
		f.cleanWriteBuf = cleanWriteBuf
	}

	return f.writeBuf.Truncate(0)
}

func (f *File) WriteString(s string) (ret int, err error) {
	log.Println("File.WriteString", f.name, s)

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterWriteMode(); err != nil {
		return -1, err
	}

	return -1, ErrNotImplemented
}

func (f *File) closeWithoutLocking() error {
	log.Println("File.closeWithoutLocking", f.name)

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

func (f *File) Close() error {
	log.Println("File.Close", f.name)

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.closeWithoutLocking()
}

func (f *File) enterReadMode(lock bool) error {
	if lock {
		f.ioLock.Lock()
		defer f.ioLock.Unlock()
	}

	if f.writeBuf != nil {
		if err := f.closeWithoutLocking(); err != nil {
			return err
		}
	}

	return nil
}

func (f *File) Read(p []byte) (n int, err error) {
	log.Println("File.Read", f.name, len(p))

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterReadMode(false); err != nil {
		return -1, err
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
	log.Println("File.ReadAt", f.name, p, off)

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	if err := f.enterReadMode(true); err != nil {
		return -1, err
	}

	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return -1, err
	}

	return f.Read(p)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	log.Println("File.Seek", f.name, offset, whence)

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterReadMode(false); err != nil {
		return -1, err
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
		return -1, ErrNotImplemented
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
		return written, io.EOF
	}

	if err != nil {
		return -1, err
	}

	return written, nil
}

func (f *File) Write(p []byte) (n int, err error) {
	log.Println("File.Write", f.name, len(p))

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterWriteMode(); err != nil {
		return -1, err
	}

	if f.writeBuf == nil {
		writeBuf, cleanWriteBuf, err := f.getFileBuffer()
		if err != nil {
			return -1, err
		}

		f.writeBuf = writeBuf
		f.cleanWriteBuf = cleanWriteBuf
	}

	n, err = f.writeBuf.Write(p)
	if err != nil {
		log.Fatal(err)

		return -1, err
	}

	return n, f.writeBuf.Sync()
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	log.Println("File.WriteAt", f.name, p, off)

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if err := f.enterWriteMode(); err != nil {
		return -1, err
	}

	return -1, ErrNotImplemented
}
