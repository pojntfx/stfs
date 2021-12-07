package signature

import "github.com/pojntfx/stfs/pkg/config"

func CheckSignatureFormat(signatureFormat string) error {
	signatureFormatIsKnown := false

	for _, candidate := range config.KnownSignatureFormats {
		if signatureFormat == candidate {
			signatureFormatIsKnown = true
		}
	}

	if !signatureFormatIsKnown {
		return config.ErrSignatureFormatUnknown
	}

	return nil
}
