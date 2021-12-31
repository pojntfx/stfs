package config

import (
	"io"
	"io/fs"
	"os"

	"github.com/pojntfx/stfs/pkg/persisters"
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

type MetadataConfig struct {
	Metadata *persisters.MetadataPersister
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
