package hardware

import (
	"os"

	"github.com/pojntfx/stfs/internal/controllers"
)

func Tell(
	state DriveConfig,
) (int64, error) {
	f, err := os.OpenFile(state.Drive, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		return -1, err
	}
	defer f.Close()

	return controllers.GetCurrentRecordFromTape(f)
}