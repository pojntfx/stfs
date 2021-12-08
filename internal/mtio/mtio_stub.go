//go:build !linux

package mtio

import (
	"os"

	"github.com/pojntfx/stfs/pkg/config"
)

func GetCurrentRecordFromTape(f *os.File) (int64, error) {
	return -1, config.ErrDrivesUnsupported
}

func GoToEndOfTape(f *os.File) error {
	return config.ErrDrivesUnsupported
}

func GoToNextFileOnTape(f *os.File) error {
	return config.ErrDrivesUnsupported
}

func EjectTape(f *os.File) error {
	return config.ErrDrivesUnsupported
}

func SeekToRecordOnTape(f *os.File, record int32) error {
	return config.ErrDrivesUnsupported
}
