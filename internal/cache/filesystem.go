package cache

import (
	"os"
	"time"

	"github.com/pojntfx/stfs/internal/pathext"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/spf13/afero"
)

func NewCacheFilesystem(
	base afero.Fs,
	root string,
	cacheType string,
	ttl time.Duration,
	cacheDir string,
) (afero.Fs, error) {
	switch cacheType {
	case FileSystemCacheTypeMemory:
		if pathext.IsRoot(root) {
			return afero.NewCacheOnReadFs(base, afero.NewMemMapFs(), ttl), nil
		}

		return afero.NewCacheOnReadFs(afero.NewBasePathFs(base, root), afero.NewMemMapFs(), ttl), nil
	case FileSystemCacheTypeDir:
		if err := os.RemoveAll(cacheDir); err != nil {
			return nil, err
		}

		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
			return nil, err
		}

		if pathext.IsRoot(root) {
			return afero.NewCacheOnReadFs(base, afero.NewBasePathFs(afero.NewOsFs(), cacheDir), ttl), nil
		}

		return afero.NewCacheOnReadFs(afero.NewBasePathFs(base, root), afero.NewBasePathFs(afero.NewOsFs(), cacheDir), ttl), nil
	case config.NoneKey:
		if pathext.IsRoot(root) {
			return base, nil
		}

		return afero.NewBasePathFs(base, root), nil
	default:
		return nil, ErrFileSystemCacheTypeUnsupported
	}
}
