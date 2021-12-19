package cache

func CheckCacheType(cacheType string) error {
	cacheTypeIsKnown := false

	for _, candidate := range KnownCacheTypes {
		if cacheType == candidate {
			cacheTypeIsKnown = true
		}
	}

	if !cacheTypeIsKnown {
		return ErrCacheTypeUnknown
	}

	return nil
}
