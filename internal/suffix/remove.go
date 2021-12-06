package suffix

import (
	"strings"

	"github.com/pojntfx/stfs/pkg/config"
)

func RemoveSuffix(name string, compressionFormat string, encryptionFormat string) (string, error) {
	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
		name = strings.TrimSuffix(name, EncryptionFormatAgeSuffix)
	case config.EncryptionFormatPGPKey:
		name = strings.TrimSuffix(name, EncryptionFormatPGPSuffix)
	case config.NoneKey:
	default:
		return "", config.ErrUnsupportedEncryptionFormat
	}

	switch compressionFormat {
	case config.CompressionFormatGZipKey:
		fallthrough
	case config.CompressionFormatParallelGZipKey:
		name = strings.TrimSuffix(name, CompressionFormatGZipSuffix)
	case config.CompressionFormatLZ4Key:
		name = strings.TrimSuffix(name, CompressionFormatLZ4Suffix)
	case config.CompressionFormatZStandardKey:
		name = strings.TrimSuffix(name, CompressionFormatZStandardSuffix)
	case config.CompressionFormatBrotliKey:
		name = strings.TrimSuffix(name, CompressionFormatBrotliSuffix)
	case config.CompressionFormatBzip2Key:
		fallthrough
	case config.CompressionFormatBzip2ParallelKey:
		name = strings.TrimSuffix(name, CompressionFormatBzip2Suffix)
	case config.NoneKey:
	default:
		return "", config.ErrUnsupportedCompressionFormat
	}

	return name, nil
}
