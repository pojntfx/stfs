package cache

import (
	"github.com/pojntfx/stfs/pkg/config"
)

const (
	FileSystemCacheTypeMemory = "memory"
	FileSystemCacheTypeDir    = "dir"

	WriteCacheTypeMemory = "memory"
	WriteCacheTypeFile   = "file"
)

var (
	KnownFileSystemCacheTypes = []string{config.NoneKey, FileSystemCacheTypeMemory, FileSystemCacheTypeDir}

	KnownWriteCacheTypes = []string{WriteCacheTypeMemory, WriteCacheTypeFile}
)
