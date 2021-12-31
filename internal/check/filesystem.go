package check

import "github.com/pojntfx/stfs/pkg/config"

func CheckFileSystemCacheType(cacheType string) error {
	cacheTypeIsKnown := false

	for _, candidate := range config.KnownFileSystemCacheTypes {
		if cacheType == candidate {
			cacheTypeIsKnown = true
		}
	}

	if !cacheTypeIsKnown {
		return config.ErrFileSystemCacheTypeUnknown
	}

	return nil
}

func CheckWriteCacheType(cacheType string) error {
	cacheTypeIsKnown := false

	for _, candidate := range config.KnownWriteCacheTypes {
		if cacheType == candidate {
			cacheTypeIsKnown = true
		}
	}

	if !cacheTypeIsKnown {
		return config.ErrWriteCacheTypeUnknown
	}

	return nil
}
