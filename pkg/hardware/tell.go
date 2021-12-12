package hardware

import (
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/pkg/config"
)

func Tell(
	state config.DriveConfig,
) (int64, error) {
	return mtio.GetCurrentRecordFromTape(state.Drive)
}
