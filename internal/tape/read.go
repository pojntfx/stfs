package tape

import "os"

func OpenTapeReadOnly(tape string) (f *os.File, isRegular bool, err error) {
	fileDescription, err := os.Stat(tape)
	if err != nil {
		return nil, false, err
	}

	isRegular = fileDescription.Mode().IsRegular()
	if isRegular {
		f, err = os.Open(tape)
		if err != nil {
			return f, isRegular, err
		}

		return f, isRegular, nil
	}

	f, err = os.OpenFile(tape, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		return f, isRegular, err
	}

	return f, isRegular, nil
}
