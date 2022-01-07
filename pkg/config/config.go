package config

import (
	"context"
	"io"
	"io/fs"
	"time"
)

type DriveReaderConfig struct {
	Drive          io.ReadSeeker
	DriveIsRegular bool
}

type DriveWriterConfig struct {
	Drive          io.Writer
	DriveIsRegular bool
}

type Drive interface {
	io.ReadSeeker
	Fd() uintptr
}

type DriveConfig struct {
	Drive          Drive
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

type Header struct {
	Record          int64
	Lastknownrecord int64
	Block           int64
	Lastknownblock  int64
	Deleted         int64
	Typeflag        int64
	Name            string
	Linkname        string
	Size            int64
	Mode            int64
	UID             int64
	Gid             int64
	Uname           string
	Gname           string
	Modtime         time.Time
	Accesstime      time.Time
	Changetime      time.Time
	Devmajor        int64
	Devminor        int64
	Paxrecords      string
	Format          int64
}

type MetadataPersister interface {
	UpsertHeader(ctx context.Context, dbhdr *Header) error
	UpdateHeaderMetadata(ctx context.Context, dbhdr *Header) error
	MoveHeader(ctx context.Context, oldName string, newName string, lastknownrecord, lastknownblock int64) error
	GetHeaders(ctx context.Context) ([]*Header, error)
	GetHeader(ctx context.Context, name string) (*Header, error)
	GetHeaderChildren(ctx context.Context, name string) ([]*Header, error)
	GetRootPath(ctx context.Context) (string, error)
	GetHeaderDirectChildren(ctx context.Context, name string, limit int) ([]*Header, error)
	DeleteHeader(ctx context.Context, name string, lastknownrecord, lastknownblock int64) (*Header, error)
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
