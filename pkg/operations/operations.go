package operations

import (
	"sync"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/config"
)

type Operations struct {
	backend  config.BackendConfig
	metadata config.MetadataConfig

	pipes  config.PipeConfig
	crypto config.CryptoConfig

	onHeader func(hdr *models.Header)

	diskOperationLock sync.Mutex
}

func NewOperations(
	backend config.BackendConfig,
	metadata config.MetadataConfig,

	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	onHeader func(hdr *models.Header),
) *Operations {
	return &Operations{
		backend:  backend,
		metadata: metadata,

		pipes:  pipes,
		crypto: crypto,

		onHeader: onHeader,
	}
}
