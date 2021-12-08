package hardware

import (
	"os"

	"github.com/pojntfx/stfs/internal/mtio"
)

func Eject(
	state DriveConfig,
) error {
	f, err := os.OpenFile(state.Drive, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		return err
	}
	defer f.Close()

	return mtio.EjectTape(f)
}
