package hardware

import "github.com/pojntfx/stfs/pkg/config"

func Tell(mt config.MagneticTapeIO, fd uintptr) (int64, error) {
	return mt.GetCurrentRecordFromTape(fd)
}
