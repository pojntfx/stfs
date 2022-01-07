package hardware

import (
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/pkg/config"
)

func Eject(
	state config.DriveConfig,
) error {
	return mtio.EjectTape(state.Drive.Fd())
}
