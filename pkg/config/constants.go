package config

const (
	NoneKey = ""

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

	CompressionLevelFastestKey  = "fastest"
	CompressionLevelBalancedKey = "balanced"
	CompressionLevelSmallestKey = "smallest"

	HeaderEventTypeArchive = "archive"
	HeaderEventTypeDelete  = "delete"
	HeaderEventTypeMove    = "move"
	HeaderEventTypeRestore = "restore"
	HeaderEventTypeUpdate  = "update"

	FileSystemNameSTFS = "STFS"

	FileSystemCacheTypeMemory = "memory"
	FileSystemCacheTypeDir    = "dir"

	WriteCacheTypeMemory = "memory"
	WriteCacheTypeFile   = "file"

	MagneticTapeBlockSize = 512
)

var (
	KnownCompressionLevels = []string{CompressionLevelFastestKey, CompressionLevelBalancedKey, CompressionLevelSmallestKey}

	KnownCompressionFormats = []string{NoneKey, CompressionFormatGZipKey, CompressionFormatParallelGZipKey, CompressionFormatLZ4Key, CompressionFormatZStandardKey, CompressionFormatBrotliKey, CompressionFormatBzip2Key, CompressionFormatBzip2ParallelKey}

	KnownEncryptionFormats = []string{NoneKey, EncryptionFormatAgeKey, EncryptionFormatPGPKey}

	KnownSignatureFormats = []string{NoneKey, SignatureFormatMinisignKey, SignatureFormatPGPKey}

	KnownFileSystemCacheTypes = []string{NoneKey, FileSystemCacheTypeMemory, FileSystemCacheTypeDir}

	KnownWriteCacheTypes = []string{WriteCacheTypeMemory, WriteCacheTypeFile}
)
