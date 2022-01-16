package hardware

import (
	"github.com/pojntfx/stfs/internal/mtio"
)

func Tell(fd uintptr) (int64, error) {
	return mtio.GetCurrentRecordFromTape(fd)
}
