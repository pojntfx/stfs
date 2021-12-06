package operations

import (
	"archive/tar"

	"github.com/pojntfx/stfs/pkg/config"
)

func Archive(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	overwrite bool,
	compressionLevel string,
) ([]*tar.Header, error)

func Restore(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	to string,
	flatten bool,
) error

func Update(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	overwrite bool,
	compressionLevel string,
) error

func Delete(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	name string,
) error

func Move(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	to string,
) error
