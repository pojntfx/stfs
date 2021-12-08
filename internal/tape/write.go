package tape

import (
	"archive/tar"
	"bufio"
	"os"

	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/mtio"
)

func OpenTapeWriteOnly(drive string, recordSize int, overwrite bool) (tw *tar.Writer, isRegular bool, cleanup func(dirty *bool) error, err error) {
	stat, err := os.Stat(drive)
	if err == nil {
		isRegular = stat.Mode().IsRegular()
	} else {
		if os.IsNotExist(err) {
			isRegular = true
		} else {
			return nil, false, nil, err
		}
	}

	var f *os.File
	if isRegular {
		f, err = os.OpenFile(drive, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, false, nil, err
		}

		// No need to go to end manually due to `os.O_APPEND`
	} else {
		f, err = os.OpenFile(drive, os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
		if err != nil {
			return nil, false, nil, err
		}

		if !overwrite {
			// Go to end of tape
			if err := mtio.GoToEndOfTape(f); err != nil {
				return nil, false, nil, err
			}
		}
	}

	var bw *bufio.Writer
	var counter *ioext.CounterWriter
	if isRegular {
		tw = tar.NewWriter(f)
	} else {
		bw = bufio.NewWriterSize(f, mtio.BlockSize*recordSize)
		counter = &ioext.CounterWriter{Writer: bw, BytesRead: 0}
		tw = tar.NewWriter(counter)
	}

	return tw, isRegular, func(dirty *bool) error {
		// Only write the trailer if we wrote to the archive
		if *dirty {
			if err := tw.Close(); err != nil {
				return err
			}

			if !isRegular {
				if mtio.BlockSize*recordSize-counter.BytesRead > 0 {
					// Fill the rest of the record with zeros
					if _, err := bw.Write(make([]byte, mtio.BlockSize*recordSize-counter.BytesRead)); err != nil {
						return err
					}
				}

				if err := bw.Flush(); err != nil {
					return err
				}
			}
		}

		return f.Close()
	}, nil
}
