package hardware

import "github.com/pojntfx/stfs/pkg/config"

func Eject(mt config.MagneticTapeIO, fd uintptr) error {
	return mt.EjectTape(fd)
}
