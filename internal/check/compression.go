package check

import (
	"github.com/pojntfx/stfs/pkg/config"
)

func CheckCompressionFormat(compressionFormat string) error {
	compressionFormatIsKnown := false

	for _, candidate := range config.KnownCompressionFormats {
		if compressionFormat == candidate {
			compressionFormatIsKnown = true
		}
	}

	if !compressionFormatIsKnown {
		return config.ErrCompressionFormatUnknown
	}

	return nil
}

func CheckCompressionLevel(compressionLevel string) error {
	compressionLevelIsKnown := false

	for _, candidate := range config.KnownCompressionLevels {
		if compressionLevel == candidate {
			compressionLevelIsKnown = true
		}
	}

	if !compressionLevelIsKnown {
		return config.ErrCompressionLevelUnknown
	}

	return nil
}
