package cache

import (
	"os"
	"time"

	"github.com/pojntfx/stfs/pkg/config"
	"github.com/spf13/afero"
)

func Cache(
	base afero.Fs,
	root string,
	cacheType string,
	ttl time.Duration,
	cacheDir string,
) (afero.Fs, error) {
	switch cacheType {
	case CacheTypeMemory:
		return afero.NewCacheOnReadFs(afero.NewBasePathFs(base, root), afero.NewMemMapFs(), ttl), nil
	case CacheTypeDir:
		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
			return nil, err
		}

		return afero.NewCacheOnReadFs(afero.NewBasePathFs(base, root), afero.NewBasePathFs(afero.NewOsFs(), cacheDir), ttl), nil
	case config.NoneKey:
		return afero.NewBasePathFs(base, root), nil
	default:
		return nil, ErrCacheTypeUnsupported
	}
}
