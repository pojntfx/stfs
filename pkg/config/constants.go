package config

const (
	NoneKey = "none"

	CompressionFormatGZipKey          = "gzip"
	CompressionFormatParallelGZipKey  = "parallelgzip"
	CompressionFormatLZ4Key           = "lz4"
	CompressionFormatZStandardKey     = "zstandard"
	CompressionFormatBrotliKey        = "brotli"
	CompressionFormatBzip2Key         = "bzip2"
	CompressionFormatBzip2ParallelKey = "parallelbzip2"

	EncryptionFormatAgeKey = "age"
	EncryptionFormatPGPKey = "pgp"

	SignatureFormatMinisignKey = "minisign"
	SignatureFormatPGPKey      = "pgp"

	CompressionLevelFastest  = "fastest"
	CompressionLevelBalanced = "balanced"
	CompressionLevelSmallest = "smallest"

	HeaderEventTypeArchive = 0
	HeaderEventTypeDelete  = 1
	HeaderEventTypeMove    = 2
	HeaderEventTypeRestore = 3
	HeaderEventTypeUpdate  = 4
)

var (
	KnownCompressionLevels = []string{CompressionLevelFastest, CompressionLevelBalanced, CompressionLevelSmallest}

	KnownCompressionFormats = []string{NoneKey, CompressionFormatGZipKey, CompressionFormatParallelGZipKey, CompressionFormatLZ4Key, CompressionFormatZStandardKey, CompressionFormatBrotliKey, CompressionFormatBzip2Key, CompressionFormatBzip2ParallelKey}

	KnownEncryptionFormats = []string{NoneKey, EncryptionFormatAgeKey, EncryptionFormatPGPKey}

	KnownSignatureFormats = []string{NoneKey, SignatureFormatMinisignKey, SignatureFormatPGPKey}
)
