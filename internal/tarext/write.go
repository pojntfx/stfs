package tarext

import (
	"archive/tar"
	"bufio"
	"io"

	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/mtio"
)

func NewTapeWriter(f io.Writer, isRegular bool, recordSize int) (tw *tar.Writer, cleanup func(dirty *bool) error, err error) {
	var bw *bufio.Writer
	var counter *ioext.CounterWriter
	if isRegular {
		tw = tar.NewWriter(f)
	} else {
		bw = bufio.NewWriterSize(f, mtio.BlockSize*recordSize)
		counter = &ioext.CounterWriter{Writer: bw, BytesRead: 0}
		tw = tar.NewWriter(counter)
	}

	return tw, func(dirty *bool) error {
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

		return nil
	}, nil
}
