package config

import "errors"

var (
	ErrEncryptionFormatUnknown  = errors.New("encryption format unknown")
	ErrSignatureFormatUnknown   = errors.New("signature format unknown")
	ErrCompressionFormatUnknown = errors.New("compression format unknown")

	ErrEncryptionFormatUnsupported  = errors.New("encryption format unsupported")
	ErrSignatureFormatUnsupported   = errors.New("signature format unsupported")
	ErrCompressionFormatUnsupported = errors.New("compression format unsupported")

	ErrSignatureFormatRegularOnly = errors.New("signature format only supports regular files, not tape drives")
	ErrSignatureInvalid           = errors.New("signature invalid")
	ErrSignatureMissing           = errors.New("signature missing")

	ErrCompressionFormatRegularOnly              = errors.New("compression format only supports regular files, not tape drives")
	ErrCompressionFormatRequiresLargerRecordSize = errors.New("compression format requires larger record size")
	ErrCompressionLevelUnsupported               = errors.New("compression level unsupported")
	ErrCompressionLevelUnknown                   = errors.New("compression level unknown")

	ErrIdentityUnparsable  = errors.New("identity could not be parsed")
	ErrRecipientUnparsable = errors.New("recipient could not be parsed")

	ErrKeygenFormatUnsupported = errors.New("key generation for format unsupported")

	ErrTarHeaderMissing         = errors.New("tar header missing")
	ErrTarHeaderEmbeddedMissing = errors.New("embedded tar header missing")

	ErrTapeDrivesUnsupported = errors.New("system unsupported for tape drives")

	ErrSTFSVersionUnsupported = errors.New("STFS version unsupported")
	ErrSTFSActionUnsupported  = errors.New("STFS action unsupported")

	ErrNotImplemented = errors.New("not implemented")
	ErrIsDirectory    = errors.New("is a directory")
	ErrIsFile         = errors.New("is a file")

	ErrFileSystemCacheTypeUnsupported = errors.New("file system cache type unsupported")
	ErrFileSystemCacheTypeUnknown     = errors.New("file system cache type unknown")

	ErrWriteCacheTypeUnsupported = errors.New("write cache type unsupported")
	ErrWriteCacheTypeUnknown     = errors.New("write cache type unknown")

	ErrNoRootDirectory = errors.New("root directory could not be found")
)
