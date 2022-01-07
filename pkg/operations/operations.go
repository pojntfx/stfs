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

func (o *Operations) GetBackend() config.BackendConfig {
	return o.backend
}

func (o *Operations) GetMetadata() config.MetadataConfig {
	return o.metadata
}

func (o *Operations) GetPipes() config.PipeConfig {
	return o.pipes
}

func (o *Operations) GetCrypto() config.CryptoConfig {
	return o.crypto
}
