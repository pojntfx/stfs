package operations

import (
	"sync"

	"github.com/pojntfx/stfs/pkg/config"
)

type Operations struct {
	backend  config.BackendConfig
	metadata config.MetadataConfig

	pipes  config.PipeConfig
	crypto config.CryptoConfig

	onHeader func(event *config.HeaderEvent)

	diskOperationLock sync.Mutex
}

func NewOperations(
	backend config.BackendConfig,
	metadata config.MetadataConfig,

	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	onHeader func(event *config.HeaderEvent),
) *Operations {
	return &Operations{
		backend:  backend,
		metadata: metadata,

		pipes:  pipes,
		crypto: crypto,

		onHeader: onHeader,
	}
}
