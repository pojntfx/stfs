package config

import "errors"

var (
	ErrUnsupportedEncryptionFormat  = errors.New("unsupported encryption format")
	ErrUnsupportedCompressionFormat = errors.New("unsupported compression format")
)
