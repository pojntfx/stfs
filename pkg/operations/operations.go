package operations

import (
	"sync"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
)

type Operations struct {
	getWriter   func() (config.DriveWriterConfig, error)
	closeWriter func() error

	getReader   func() (config.DriveReaderConfig, error)
	closeReader func() error

	getDrive   func() (config.DriveConfig, error)
	closeDrive func() error

	metadataPersister *persisters.MetadataPersister

	pipes  config.PipeConfig
	crypto config.CryptoConfig

	recordSize int

	onHeader func(hdr *models.Header)

	diskOperationLock sync.Mutex
}

func NewOperations(
	getWriter func() (config.DriveWriterConfig, error),
	closeWriter func() error,

	getReader func() (config.DriveReaderConfig, error),
	closeReader func() error,

	getDrive func() (config.DriveConfig, error),
	closeDrive func() error,

	metadataPersister *persisters.MetadataPersister,

	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,

	onHeader func(hdr *models.Header),
) *Operations {
	return &Operations{
		getWriter:   getWriter,
		closeWriter: closeWriter,

		getReader:   getReader,
		closeReader: closeReader,

		getDrive:   getDrive,
		closeDrive: closeDrive,

		metadataPersister: metadataPersister,

		pipes:  pipes,
		crypto: crypto,

		recordSize: recordSize,

		onHeader: onHeader,
	}
}
