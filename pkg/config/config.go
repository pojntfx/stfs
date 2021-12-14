package config

import (
	"io"
	"os"

	"github.com/pojntfx/stfs/internal/persisters"
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

type MetadataConfig struct {
	Metadata *persisters.MetadataPersister
}

type PipeConfig struct {
	Compression string
	Encryption  string
	Signature   string
}

type CryptoConfig struct {
	Recipient interface{}
	Identity  interface{}
	Password  string
}

type PasswordConfig struct {
	Password string
}
