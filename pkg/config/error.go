package config

import "errors"

var (
	ErrEncryptionFormatUnsupported = errors.New("unsupported encryption format")

	ErrIdentityUnparsable  = errors.New("recipient could not be parsed")
	ErrRecipientUnparsable = errors.New("recipient could not be parsed")

	ErrEmbeddedHeaderMissing = errors.New("embedded header is missing")

	ErrSignatureFormatUnsupported        = errors.New("unsupported signature format")
	ErrSignatureFormatOnlyRegularSupport = errors.New("this signature format only supports regular files, not i.e. tape drives")
	ErrSignatureInvalid                  = errors.New("signature is invalid")
	ErrSignatureMissing                  = errors.New("signature is missing")

	ErrKeygenForFormatUnsupported = errors.New("can not generate keys for this format")

	ErrCompressionFormatUnsupported              = errors.New("unsupported compression format")
	ErrCompressionFormatOnlyRegularSupport       = errors.New("this compression format only supports regular files, not i.e. tape drives")
	ErrCompressionFormatRequiresLargerRecordSize = errors.New("this compression format requires a larger record size")
	ErrCompressionLevelUnsupported               = errors.New("compression level is unsupported")

	ErrMissingTarHeader = errors.New("tar header is missing")
)
