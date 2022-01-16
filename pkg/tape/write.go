package tape

import (
	"os"

	"github.com/pojntfx/stfs/pkg/config"
)

func OpenTapeWriteOnly(
	drive string,
	mt config.MagneticTapeIO,
	recordSize int,
	overwrite bool,
) (f *os.File, isRegular bool, err error) {
	stat, err := os.Stat(drive)
	if err == nil {
		isRegular = stat.Mode().IsRegular()
	} else {
		if os.IsNotExist(err) {
			isRegular = true
		} else {
			return nil, false, err
		}
	}

	if overwrite {
		if isRegular {
			f, err := os.OpenFile(drive, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return nil, false, err
			}

			// Clear the file's content
			if err := f.Truncate(0); err != nil {
				return nil, false, err
			}

			if err := f.Close(); err != nil {
				return nil, false, err
			}
		} else {
			f, err := os.OpenFile(drive, os.O_WRONLY, os.ModeCharDevice)
			if err != nil {
				return nil, false, err
			}

			// Seek to the start of the tape
			if err := mt.SeekToRecordOnTape(f.Fd(), 0); err != nil {
				return nil, false, err
			}

			if err := f.Close(); err != nil {
				return nil, false, err
			}
		}
	}

	if isRegular {
		f, err = os.OpenFile(drive, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, false, err
		}

		// No need to go to end manually due to `os.O_APPEND`
	} else {
		f, err = os.OpenFile(drive, os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
		if err != nil {
			return nil, false, err
		}

		if !overwrite {
			// Go to end of tape
			if err := mt.GoToEndOfTape(f.Fd()); err != nil {
				return nil, false, err
			}
		}
	}

	return f, isRegular, nil
}
