package hardware

import (
	"github.com/pojntfx/stfs/internal/mtio"
)

func Eject(fd uintptr) error {
	return mtio.EjectTape(fd)
}
