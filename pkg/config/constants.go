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
)
