package hardware

import (
	"os"

	"github.com/pojntfx/stfs/internal/controllers"
)

func Eject(
	state DriveConfig,
) error {
	f, err := os.OpenFile(state.Drive, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		return err
	}
	defer f.Close()

	return controllers.EjectTape(f)
}
