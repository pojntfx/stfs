package recovery

import (
	"archive/tar"

	"github.com/pojntfx/stfs/pkg/config"
)

func Fetch(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,
	to string,
	preview string,
) error

func Index(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,
	overwrite bool,
) error

func Query(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,
) ([]*tar.Header, error)
