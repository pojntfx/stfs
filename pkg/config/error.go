package config

import "errors"

var (
	ErrUnsupportedCompressionFormat = errors.New("unsupported compression format")
	ErrUnsupportedEncryptionFormat  = errors.New("unsupported encryption format")
	ErrUnsupportedSignatureFormat   = errors.New("unsupported signature format")

	ErrIdentityUnparsable  = errors.New("recipient could not be parsed")
	ErrRecipientUnparsable = errors.New("recipient could not be parsed")

	ErrEmbeddedHeaderMissing = errors.New("embedded header is missing")

	ErrSignatureFormatOnlyRegularSupport = errors.New("this signature format only supports regular files, not i.e. tape drives")
	ErrSignatureInvalid                  = errors.New("signature invalid")
	ErrSignatureMissing                  = errors.New("signature missing")

	ErrKeygenForFormatUnsupported = errors.New("can not generate keys for this format")
)
