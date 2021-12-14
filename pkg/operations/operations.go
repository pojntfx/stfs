package operations

import (
	"sync"

	"github.com/pojntfx/stfs/pkg/config"
)

type Operations struct {
	getWriter   func() (config.DriveWriterConfig, error)
	closeWriter func() error

	getReader   func() (config.DriveReaderConfig, error)
	closeReader func() error

	getDrive   func() (config.DriveConfig, error)
	closeDrive func() error

	writeLock sync.Mutex
}

func NewOperations(
	getWriter func() (config.DriveWriterConfig, error),
	closeWriter func() error,

	getReader func() (config.DriveReaderConfig, error),
	closeReader func() error,

	getDrive func() (config.DriveConfig, error),
	closeDrive func() error,
) *Operations {
	return &Operations{
		getWriter:   getWriter,
		closeWriter: closeWriter,

		getReader:   getReader,
		closeReader: closeReader,

		getDrive:   getDrive,
		closeDrive: closeDrive,
	}
}
