package recovery

import (
	"archive/tar"

	"github.com/pojntfx/stfs/pkg/config"
)

func Query(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,
) ([]*tar.Header, error) {
	return nil, nil
}
