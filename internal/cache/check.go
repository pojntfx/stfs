package cache

func CheckFileSystemCacheType(cacheType string) error {
	cacheTypeIsKnown := false

	for _, candidate := range KnownFileSystemCacheTypes {
		if cacheType == candidate {
			cacheTypeIsKnown = true
		}
	}

	if !cacheTypeIsKnown {
		return ErrFileSystemCacheTypeUnknown
	}

	return nil
}

func CheckWriteCacheType(cacheType string) error {
	cacheTypeIsKnown := false

	for _, candidate := range KnownWriteCacheTypes {
		if cacheType == candidate {
			cacheTypeIsKnown = true
		}
	}

	if !cacheTypeIsKnown {
		return ErrWriteCacheTypeUnknown
	}

	return nil
}
