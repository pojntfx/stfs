package operations

import (
	"github.com/pojntfx/stfs/pkg/config"
)

func Delete(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	name string,
) error {
	return nil
}

func Move(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	to string,
) error {
	return nil
}
