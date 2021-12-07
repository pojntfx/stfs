package tape

import "os"

func OpenTapeReadOnly(drive string) (f *os.File, isRegular bool, err error) {
	fileDescription, err := os.Stat(drive)
	if err != nil {
		return nil, false, err
	}

	isRegular = fileDescription.Mode().IsRegular()
	if isRegular {
		f, err = os.Open(drive)
		if err != nil {
			return f, isRegular, err
		}

		return f, isRegular, nil
	}

	f, err = os.OpenFile(drive, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		return f, isRegular, err
	}

	return f, isRegular, nil
}
