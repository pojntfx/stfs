package recovery

import (
	"archive/tar"
	"bufio"
	"io"
	"io/ioutil"
	"math"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/encryption"
	"github.com/pojntfx/stfs/pkg/signature"
)

func Query(
	state config.DriveConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,

	onHeader func(hdr *config.Header),
) ([]*tar.Header, error) {
	headers := []*tar.Header{}

	if state.DriveIsRegular {
		// Seek to record and block
		if _, err := state.Drive.Seek(int64((recordSize*mtio.BlockSize*record)+block*mtio.BlockSize), 0); err != nil {
			return []*tar.Header{}, err
		}

		tr := tar.NewReader(state.Drive)

		record := int64(record)
		block := int64(block)

		for {
			hdr, err := tr.Next()
			if err != nil {
				for {
					curr, err := state.Drive.Seek(0, io.SeekCurrent)
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
					if _, err := state.Drive.Seek(int64((recordSize*mtio.BlockSize*int(record))+int(block)*mtio.BlockSize), io.SeekStart); err != nil {
						return []*tar.Header{}, err
					}

					tr = tar.NewReader(state.Drive)

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

			if err := signature.VerifyHeader(hdr, state.DriveIsRegular, pipes.Signature, crypto.Recipient); err != nil {
				return []*tar.Header{}, err
			}

			if onHeader != nil {
				dbhdr, err := converters.TarHeaderToDBHeader(record, -1, block, -1, hdr)
				if err != nil {
					return []*tar.Header{}, err
				}

				onHeader(converters.DBHeaderToConfigHeader(dbhdr))
			}

			headers = append(headers, hdr)

			curr, err := state.Drive.Seek(0, io.SeekCurrent)
			if err != nil {
				return []*tar.Header{}, err
			}

			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return []*tar.Header{}, err
			}

			currAndSize, err := state.Drive.Seek(0, io.SeekCurrent)
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
		if err := mtio.SeekToRecordOnTape(state.Drive, int32(record)); err != nil {
			return []*tar.Header{}, err
		}

		// Seek to block
		br := bufio.NewReaderSize(state.Drive, mtio.BlockSize*recordSize)
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
					if err := mtio.GoToNextFileOnTape(state.Drive); err != nil {
						// EOD

						break
					}

					record, err = mtio.GetCurrentRecordFromTape(state.Drive)
					if err != nil {
						return []*tar.Header{}, err
					}
					block = 0

					br = bufio.NewReaderSize(state.Drive, mtio.BlockSize*recordSize)
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

			if err := signature.VerifyHeader(hdr, state.DriveIsRegular, pipes.Signature, crypto.Recipient); err != nil {
				return []*tar.Header{}, err
			}

			if onHeader != nil {
				dbhdr, err := converters.TarHeaderToDBHeader(record, -1, block, -1, hdr)
				if err != nil {
					return []*tar.Header{}, err
				}

				onHeader(converters.DBHeaderToConfigHeader(dbhdr))
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
