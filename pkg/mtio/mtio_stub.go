//go:build !linux

package mtio

import (
	"github.com/pojntfx/stfs/pkg/config"
)

type MagneticTapeIO struct{}

func (t MagneticTapeIO) GetCurrentRecordFromTape(fd uintptr) (int64, error) {
	return -1, config.ErrTapeDrivesUnsupported
}

func (t MagneticTapeIO) GoToEndOfTape(fd uintptr) error {
	return config.ErrTapeDrivesUnsupported
}

func (t MagneticTapeIO) GoToNextFileOnTape(fd uintptr) error {
	return config.ErrTapeDrivesUnsupported
}

func (t MagneticTapeIO) EjectTape(fd uintptr) error {
	return config.ErrTapeDrivesUnsupported
}

func (t MagneticTapeIO) SeekToRecordOnTape(fd uintptr, record int32) error {
	return config.ErrTapeDrivesUnsupported
}
