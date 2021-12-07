package operations

import (
	"github.com/pojntfx/stfs/pkg/config"
)

func Update(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	overwrite bool,
	compressionLevel string,
) error {
	return nil
}

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
