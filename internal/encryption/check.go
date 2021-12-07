package encryption

import "github.com/pojntfx/stfs/pkg/config"

func CheckEncryptionFormat(encryptionFormat string) error {
	encryptionFormatIsKnown := false

	for _, candidate := range config.KnownEncryptionFormats {
		if encryptionFormat == candidate {
			encryptionFormatIsKnown = true
		}
	}

	if !encryptionFormatIsKnown {
		return config.ErrEncryptionFormatUnknown
	}

	return nil
}
