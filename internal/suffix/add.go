package suffix

import "github.com/pojntfx/stfs/pkg/config"

func AddSuffix(name string, compressionFormat string, encryptionFormat string) (string, error) {
	switch compressionFormat {
	case config.CompressionFormatGZipKey:
		fallthrough
	case config.CompressionFormatParallelGZipKey:
		name += CompressionFormatGZipSuffix
	case config.CompressionFormatLZ4Key:
		name += CompressionFormatLZ4Suffix
	case config.CompressionFormatZStandardKey:
		name += CompressionFormatZStandardSuffix
	case config.CompressionFormatBrotliKey:
		name += CompressionFormatBrotliSuffix
	case config.CompressionFormatBzip2Key:
		fallthrough
	case config.CompressionFormatBzip2ParallelKey:
		name += CompressionFormatBzip2Suffix
	case config.NoneKey:
	default:
		return "", config.ErrCompressionFormatUnsupported
	}

	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
		name += EncryptionFormatAgeSuffix
	case config.EncryptionFormatPGPKey:
		name += EncryptionFormatPGPSuffix
	case config.NoneKey:
	default:
		return "", config.ErrEncryptionFormatUnsupported
	}

	return name, nil
}
