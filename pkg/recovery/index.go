package recovery

import (
	"archive/tar"
	"bufio"
	"context"
	"database/sql"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/internal/suffix"
	"github.com/pojntfx/stfs/internal/tape"
	"github.com/pojntfx/stfs/pkg/config"
)

func Index(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,
	overwrite bool,

	decryptHeader func(
		hdr *tar.Header,
		i int,
	) error,
	verifyHeader func(
		hdr *tar.Header,
		isRegular bool,
	) error,

	onHeader func(hdr *models.Header),
) error {
	if overwrite {
		f, err := os.OpenFile(state.Metadata, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		if err := f.Truncate(0); err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}
	}

	metadataPersister := persisters.NewMetadataPersister(state.Metadata)
	if err := metadataPersister.Open(); err != nil {
		return err
	}

	f, isRegular, err := tape.OpenTapeReadOnly(state.Drive)
	if err != nil {
		return err
	}
	defer f.Close()

	if isRegular {
		// Seek to record and block
		if _, err := f.Seek(int64((recordSize*mtio.BlockSize*record)+block*mtio.BlockSize), 0); err != nil {
			return err
		}

		tr := tar.NewReader(f)

		record := int64(record)
		block := int64(block)
		i := 0

		for {
			hdr, err := tr.Next()
			if err != nil {
				for {
					curr, err := f.Seek(0, io.SeekCurrent)
					if err != nil {
						return err
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
						return err
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

			if err := decryptHeader(hdr, i); err != nil {
				return err
			}

			if err := verifyHeader(hdr, isRegular); err != nil {
				return err
			}

			if err := indexHeader(record, block, hdr, metadataPersister, pipes.Compression, pipes.Encryption, onHeader); err != nil {
				return err
			}

			curr, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}

			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return err
			}

			currAndSize, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}

			nextTotalBlocks := math.Ceil(float64(curr+(currAndSize-curr)) / float64(mtio.BlockSize))
			record = int64(nextTotalBlocks) / int64(recordSize)
			block = int64(nextTotalBlocks) - (record * int64(recordSize))

			if block > int64(recordSize) {
				record++
				block = 0
			}

			i++
		}
	} else {
		// Seek to record
		if err := mtio.SeekToRecordOnTape(f, int32(record)); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(f, mtio.BlockSize*recordSize)
		if _, err := br.Read(make([]byte, block*mtio.BlockSize)); err != nil {
			return err
		}

		record := int64(record)
		block := int64(block)

		curr := int64((recordSize * mtio.BlockSize * int(record)) + (int(block) * mtio.BlockSize))
		counter := &ioext.CounterReader{Reader: br, BytesRead: int(curr)}
		i := 0

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
						return err
					}
					block = 0

					br = bufio.NewReaderSize(f, mtio.BlockSize*recordSize)
					curr = int64(int64(recordSize) * mtio.BlockSize * record)
					counter = &ioext.CounterReader{Reader: br, BytesRead: int(curr)}
					tr = tar.NewReader(counter)

					continue
				} else {
					return err
				}
			}

			if err := decryptHeader(hdr, i); err != nil {
				return err
			}

			if err := verifyHeader(hdr, isRegular); err != nil {
				return err
			}

			if err := indexHeader(record, block, hdr, metadataPersister, pipes.Compression, pipes.Encryption, onHeader); err != nil {
				return err
			}

			curr = int64(counter.BytesRead)

			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return err
			}

			currAndSize := int64(counter.BytesRead)

			nextTotalBlocks := math.Ceil(float64(curr+(currAndSize-curr)) / float64(mtio.BlockSize))
			record = int64(nextTotalBlocks) / int64(recordSize)
			block = int64(nextTotalBlocks) - (record * int64(recordSize))

			if block > int64(recordSize) {
				record++
				block = 0
			}

			i++
		}
	}

	return nil
}

func indexHeader(
	record, block int64,
	hdr *tar.Header,
	metadataPersister *persisters.MetadataPersister,
	compressionFormat string,
	encryptionFormat string,
	onHeader func(hdr *models.Header),
) error {
	uncompressedSize, ok := hdr.PAXRecords[records.STFSRecordUncompressedSize]
	if ok {
		size, err := strconv.Atoi(uncompressedSize)
		if err != nil {
			return err
		}

		hdr.Size = int64(size)
	}

	if hdr.FileInfo().Mode().IsRegular() {
		newName, err := suffix.RemoveSuffix(hdr.Name, compressionFormat, encryptionFormat)
		if err != nil {
			return err
		}
		hdr.Name = newName
	}

	if onHeader != nil {
		dbhdr, err := converters.TarHeaderToDBHeader(record, -1, block, -1, hdr)
		if err != nil {
			return err
		}

		onHeader(dbhdr)
	}

	stfsVersion, ok := hdr.PAXRecords[records.STFSRecordVersion]
	if !ok {
		stfsVersion = records.STFSRecordVersion1
	}

	switch stfsVersion {
	case records.STFSRecordVersion1:
		stfsAction, ok := hdr.PAXRecords[records.STFSRecordAction]
		if !ok {
			stfsAction = records.STFSRecordActionCreate
		}

		switch stfsAction {
		case records.STFSRecordActionCreate:
			dbhdr, err := converters.TarHeaderToDBHeader(record, record, block, block, hdr)
			if err != nil {
				return err
			}

			if err := metadataPersister.UpsertHeader(context.Background(), dbhdr); err != nil {
				return err
			}
		case records.STFSRecordActionDelete:
			if _, err := metadataPersister.DeleteHeader(context.Background(), hdr.Name, record, block); err != nil {
				return err
			}
		case records.STFSRecordActionUpdate:
			moveAfterEdits := false
			oldName := hdr.Name
			if _, ok := hdr.PAXRecords[records.STFSRecordReplacesName]; ok {
				moveAfterEdits = true
				oldName = hdr.PAXRecords[records.STFSRecordReplacesName]
			}

			var newHdr *models.Header
			if replacesContent, ok := hdr.PAXRecords[records.STFSRecordReplacesContent]; ok && replacesContent == records.STFSRecordReplacesContentTrue {
				// Content & metadata update; use the new record & block
				h, err := converters.TarHeaderToDBHeader(record, record, block, block, hdr)
				if err != nil {
					return err
				}

				newHdr = h

				if err := metadataPersister.UpdateHeaderMetadata(context.Background(), newHdr); err != nil {
					return err
				}
			} else {
				// Metadata-only update; use the old record & block
				oldHdr, err := metadataPersister.GetHeader(context.Background(), oldName)
				if err == nil {
					h, err := converters.TarHeaderToDBHeader(oldHdr.Record, record, oldHdr.Block, block, hdr)
					if err != nil {
						return err
					}

					newHdr = h

					if err := metadataPersister.UpdateHeaderMetadata(context.Background(), newHdr); err != nil {
						return err
					}
				}

				// To support ignoring previous `Move` operations, we need to ignore non-existent headers here, as moving changes the primary keys
				if err != nil && err != sql.ErrNoRows {
					return err
				}
			}

			if moveAfterEdits {
				// Move header (will be a no-op if the header has been moved before)
				if err := metadataPersister.MoveHeader(context.Background(), oldName, hdr.Name, record, block); err != nil {
					return err
				}
			}

		default:
			return config.ErrSTFSActionUnsupported
		}
	default:
		return config.ErrSTFSVersionUnsupported
	}

	return nil
}
