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

type File struct {
	afero.File

	ops *operations.Operations

	metadata config.MetadataConfig

	path string

	name string
	info os.FileInfo

	ioLock sync.Mutex
	reader *ioext.CounterReadCloser
	writer io.WriteCloser

	onHeader func(hdr *models.Header)
}

func NewFile(
	ops *operations.Operations,

	metadata config.MetadataConfig,

	path string,

	name string,
	info os.FileInfo,

	onHeader func(hdr *models.Header),
) *File {
	return &File{
		ops: ops,

		metadata: metadata,

		path: path,

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

func (f *File) Sync() error {
	log.Println("File.Sync", f.name)

	if f.info.IsDir() {
		return ErrIsDirectory
	}

	return ErrNotImplemented
}

func (f *File) Truncate(size int64) error {
	log.Println("File.Truncate", f.name, size)

	if f.info.IsDir() {
		return ErrIsDirectory
	}

	return ErrNotImplemented
}

func (f *File) WriteString(s string) (ret int, err error) {
	log.Println("File.WriteString", f.name, s)

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	return -1, ErrNotImplemented
}

func (f *File) closeWithoutLocking() error {
	log.Println("File.closeWithoutLocking", f.name)

	if f.reader != nil {
		if err := f.reader.Close(); err != nil {
			return err
		}
	}

	if f.writer != nil {
		if err := f.writer.Close(); err != nil {
			return err
		}
	}

	f.reader = nil
	f.writer = nil

	return nil
}

func (f *File) Close() error {
	log.Println("File.Close", f.name)

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	return f.closeWithoutLocking()
}

func (f *File) Read(p []byte) (n int, err error) {
	log.Println("File.Read", f.name, len(p))

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	f.ioLock.Lock()
	defer f.ioLock.Unlock()

	if f.reader == nil || f.writer == nil {
		r, writer := io.Pipe()
		reader := &ioext.CounterReadCloser{
			Reader:    r,
			BytesRead: 0,
		}

		go func() {
			if err := f.ops.Restore(
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

		f.reader = reader
		f.writer = writer
	}

	w := &bytes.Buffer{}
	_, err = io.CopyN(w, f.reader, int64(len(p)))
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

	dst := int64(0)
	switch whence {
	case io.SeekStart:
		dst = offset
	case io.SeekCurrent:
		curr := 0
		if f.reader != nil {
			curr = f.reader.BytesRead
		}
		dst = int64(curr) + offset
	case io.SeekEnd:
		dst = f.info.Size() - offset
	default:
		return -1, ErrNotImplemented
	}

	if f.reader == nil || f.writer == nil || dst < int64(f.reader.BytesRead) { // We have to re-open as we can't seek backwards
		_ = f.closeWithoutLocking() // Ignore errors here as it might not be opened

		r, writer := io.Pipe()
		reader := &ioext.CounterReadCloser{
			Reader:    r,
			BytesRead: 0,
		}

		go func() {
			if err := f.ops.Restore(
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

		f.reader = reader
		f.writer = writer
	}

	written, err := io.CopyN(io.Discard, f.reader, dst-int64(f.reader.BytesRead))
	if err == io.EOF {
		return written, io.EOF
	}

	if err != nil {
		return -1, err
	}

	return written, nil
}

func (f *File) Write(p []byte) (n int, err error) {
	log.Println("File.Write", f.name, p)

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	return -1, ErrNotImplemented
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	log.Println("File.WriteAt", f.name, p, off)

	if f.info.IsDir() {
		return -1, ErrIsDirectory
	}

	return -1, ErrNotImplemented
}
