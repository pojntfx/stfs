package recovery

import (
	"archive/tar"
	"bufio"
	"io"
	"io/ioutil"
	"math"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/encryption"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/internal/signature"
	"github.com/pojntfx/stfs/internal/tape"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/hardware"
)

func Query(
	state hardware.DriveConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,
) ([]*tar.Header, error) {
	f, isRegular, err := tape.OpenTapeReadOnly(state.Drive)
	if err != nil {
		return []*tar.Header{}, err
	}
	defer f.Close()

	headers := []*tar.Header{}

	if isRegular {
		// Seek to record and block
		if _, err := f.Seek(int64((recordSize*mtio.BlockSize*record)+block*mtio.BlockSize), 0); err != nil {
			return []*tar.Header{}, err
		}

		tr := tar.NewReader(f)

		record := int64(record)
		block := int64(block)

		for {
			hdr, err := tr.Next()
			if err != nil {
				for {
					curr, err := f.Seek(0, io.SeekCurrent)
					if err != nil {
						return []*tar.Header{}, err
					}

					nextTotalBlocks := math.Ceil(float64((curr)) / float64(mtio.BlockSize))
					record = int64(nextTotalBlocks) / int64(recordSize)
					block = int64(nextTotalBlocks) - (record * int64(recordSize))

					if block < 0 {
						record--
						block = int64(recordSize) - 1
					} else if block >= int64(recordSize) {
						record++
						block = 0
					}

					// Seek to record and block
					if _, err := f.Seek(int64((recordSize*mtio.BlockSize*int(record))+int(block)*mtio.BlockSize), io.SeekStart); err != nil {
						return []*tar.Header{}, err
					}

					tr = tar.NewReader(f)

					hdr, err = tr.Next()
					if err != nil {
						if err == io.EOF {
							// EOF

							break
						}

						continue
					}

					break
				}
			}

			if hdr == nil {
				// EOF

				break
			}

			if err := encryption.DecryptHeader(hdr, pipes.Encryption, crypto.Identity); err != nil {
				return []*tar.Header{}, err
			}

			if err := signature.VerifyHeader(hdr, isRegular, pipes.Signature, crypto.Recipient); err != nil {
				return []*tar.Header{}, err
			}

			if record == 0 && block == 0 {
				if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
					return []*tar.Header{}, err
				}
			}

			if err := formatting.PrintCSV(converters.TARHeaderToCSV(record, -1, block, -1, hdr)); err != nil {
				return []*tar.Header{}, err
			}

			headers = append(headers, hdr)

			curr, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return []*tar.Header{}, err
			}

			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return []*tar.Header{}, err
			}

			currAndSize, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return []*tar.Header{}, err
			}

			nextTotalBlocks := math.Ceil(float64(curr+(currAndSize-curr)) / float64(mtio.BlockSize))
			record = int64(nextTotalBlocks) / int64(recordSize)
			block = int64(nextTotalBlocks) - (record * int64(recordSize))

			if block > int64(recordSize) {
				record++
				block = 0
			}
		}
	} else {
		// Seek to record
		if err := mtio.SeekToRecordOnTape(f, int32(record)); err != nil {
			return []*tar.Header{}, err
		}

		// Seek to block
		br := bufio.NewReaderSize(f, mtio.BlockSize*recordSize)
		if _, err := br.Read(make([]byte, block*mtio.BlockSize)); err != nil {
			return []*tar.Header{}, err
		}

		record := int64(record)
		block := int64(block)

		curr := int64((recordSize * mtio.BlockSize * int(record)) + (int(block) * mtio.BlockSize))
		counter := &ioext.CounterReader{Reader: br, BytesRead: int(curr)}

		tr := tar.NewReader(counter)
		for {
			hdr, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					if err := mtio.GoToNextFileOnTape(f); err != nil {
						// EOD

						break
					}

					record, err = mtio.GetCurrentRecordFromTape(f)
					if err != nil {
						return []*tar.Header{}, err
					}
					block = 0

					br = bufio.NewReaderSize(f, mtio.BlockSize*recordSize)
					curr := int64(int64(recordSize) * mtio.BlockSize * record)
					counter := &ioext.CounterReader{Reader: br, BytesRead: int(curr)}
					tr = tar.NewReader(counter)

					continue
				} else {
					return []*tar.Header{}, err
				}
			}

			if err := encryption.DecryptHeader(hdr, pipes.Encryption, crypto.Identity); err != nil {
				return []*tar.Header{}, err
			}

			if err := signature.VerifyHeader(hdr, isRegular, pipes.Signature, crypto.Recipient); err != nil {
				return []*tar.Header{}, err
			}

			if record == 0 && block == 0 {
				if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
					return []*tar.Header{}, err
				}
			}

			if err := formatting.PrintCSV(converters.TARHeaderToCSV(record, -1, block, -1, hdr)); err != nil {
				return []*tar.Header{}, err
			}

			headers = append(headers, hdr)

			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return []*tar.Header{}, err
			}

			currAndSize := int64(counter.BytesRead)

			nextTotalBlocks := math.Ceil(float64(curr+(currAndSize-curr)) / float64(mtio.BlockSize))
			record = int64(nextTotalBlocks) / int64(recordSize)
			block = int64(nextTotalBlocks) - (record * int64(recordSize))

			if block > int64(recordSize) {
				record++
				block = 0
			}
		}
	}

	return headers, nil
}
