//go:build !linux

package mtio

import (
	"os"

	"github.com/pojntfx/stfs/pkg/config"
)

func GetCurrentRecordFromTape(f *os.File) (int64, error) {
	return -1, config.ErrTapeDrivesUnsupported
}

func GoToEndOfTape(f *os.File) error {
	return config.ErrTapeDrivesUnsupported
}

func GoToNextFileOnTape(f *os.File) error {
	return config.ErrTapeDrivesUnsupported
}

func EjectTape(f *os.File) error {
	return config.ErrTapeDrivesUnsupported
}

func SeekToRecordOnTape(f *os.File, record int32) error {
	return config.ErrTapeDrivesUnsupported
}
