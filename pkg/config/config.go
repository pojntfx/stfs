package config

import (
	"context"
	"io"
	"io/fs"
	"os"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
)

type DriveReaderConfig struct {
	Drive          io.ReadSeeker
	DriveIsRegular bool
}

type DriveWriterConfig struct {
	Drive          io.Writer
	DriveIsRegular bool
}

type DriveConfig struct {
	Drive          *os.File
	DriveIsRegular bool
}

type BackendConfig struct {
	GetWriter   func() (DriveWriterConfig, error)
	CloseWriter func() error

	GetReader   func() (DriveReaderConfig, error)
	CloseReader func() error

	GetDrive   func() (DriveConfig, error)
	CloseDrive func() error
}

type MetadataPersister interface {
	UpsertHeader(ctx context.Context, dbhdr *models.Header) error
	UpdateHeaderMetadata(ctx context.Context, dbhdr *models.Header) error
	MoveHeader(ctx context.Context, oldName string, newName string, lastknownrecord, lastknownblock int64) error
	GetHeaders(ctx context.Context) (models.HeaderSlice, error)
	GetHeader(ctx context.Context, name string) (*models.Header, error)
	GetHeaderChildren(ctx context.Context, name string) (models.HeaderSlice, error)
	GetRootPath(ctx context.Context) (string, error)
	GetHeaderDirectChildren(ctx context.Context, name string, limit int) (models.HeaderSlice, error)
	DeleteHeader(ctx context.Context, name string, lastknownrecord, lastknownblock int64) (*models.Header, error)
	GetLastIndexedRecordAndBlock(ctx context.Context, recordSize int) (int64, int64, error)
	PurgeAllHeaders(ctx context.Context) error
}

type MetadataConfig struct {
	Metadata MetadataPersister
}

type PipeConfig struct {
	Compression string
	Encryption  string
	Signature   string
	RecordSize  int
}

type CryptoConfig struct {
	Recipient interface{}
	Identity  interface{}
	Password  string
}

type PasswordConfig struct {
	Password string
}

type FileConfig struct {
	GetFile func() (io.ReadSeekCloser, error)
	Info    fs.FileInfo
	Path    string
	Link    string
}
