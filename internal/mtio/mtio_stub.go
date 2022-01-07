//go:build !linux

package mtio

import (
	"os"

	"github.com/pojntfx/stfs/pkg/config"
)

func GetCurrentRecordFromTape(fd uintptr) (int64, error) {
	return -1, config.ErrTapeDrivesUnsupported
}

func GoToEndOfTape(fd uintptr) error {
	return config.ErrTapeDrivesUnsupported
}

func GoToNextFileOnTape(fd uintptr) error {
	return config.ErrTapeDrivesUnsupported
}

func EjectTape(fd uintptr) error {
	return config.ErrTapeDrivesUnsupported
}

func SeekToRecordOnTape(fd uintptr, record int32) error {
	return config.ErrTapeDrivesUnsupported
}
