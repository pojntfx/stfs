package cache

import (
	"github.com/pojntfx/stfs/pkg/config"
)

const (
	CacheTypeMemory = "memory"
	CacheTypeDir    = "dir"
)

var (
	KnownCacheTypes = []string{config.NoneKey, CacheTypeMemory, CacheTypeDir}
)
