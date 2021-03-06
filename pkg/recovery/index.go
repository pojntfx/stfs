package recovery

import (
	"archive/tar"
	"bufio"
	"context"
	"database/sql"
	"io"
	"io/ioutil"
	"math"
	"strconv"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/internal/suffix"
	"github.com/pojntfx/stfs/pkg/config"
)

func Index(
	reader config.DriveReaderConfig,
	mt config.MagneticTapeIO,
	metadata config.MetadataConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	record int,
	block int,
	overwrite bool,
	initializing bool,
	offset int,

	decryptHeader func(
		hdr *tar.Header,
		i int,
	) error,
	verifyHeader func(
		hdr *tar.Header,
		isRegular bool,
	) error,

	onHeader func(hdr *config.Header),
) error {
	if overwrite {
		if err := metadata.Metadata.PurgeAllHeaders(context.Background()); err != nil {
			return err
		}
	}

	if reader.DriveIsRegular {
		// Seek to record and block
		if _, err := reader.Drive.Seek(int64((pipes.RecordSize*config.MagneticTapeBlockSize*record)+block*config.MagneticTapeBlockSize), 0); err != nil {
			return err
		}

		tr := tar.NewReader(reader.Drive)

		record := int64(record)
		block := int64(block)
		i := 0

		for {
			hdr, err := tr.Next()
			if err != nil {
				for {
					curr, err := reader.Drive.Seek(0, io.SeekCurrent)
					if err != nil {
						return err
					}

					nextTotalBlocks := math.Ceil(float64((curr)) / float64(config.MagneticTapeBlockSize))
					record = int64(nextTotalBlocks) / int64(pipes.RecordSize)
					block = int64(nextTotalBlocks) - (record * int64(pipes.RecordSize))

					if block < 0 {
						record--
						block = int64(pipes.RecordSize) - 1
					} else if block >= int64(pipes.RecordSize) {
						record++
						block = 0
					}

					// Seek to record and block
					if _, err := reader.Drive.Seek(int64((pipes.RecordSize*config.MagneticTapeBlockSize*int(record))+int(block)*config.MagneticTapeBlockSize), io.SeekStart); err != nil {
						return err
					}

					tr = tar.NewReader(reader.Drive)

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
				// Try to skip over the next file mark; this makes it possible to append to a tar file created by i.e. GNU tar
				if _, err := reader.Drive.Read(make([]byte, config.MagneticTapeBlockSize*2)); err != nil {
					if err == io.EOF {
						// EOF
						break
					}

					return err
				}

				continue
			}

			if i >= offset {
				if err := decryptHeader(hdr, i-offset); err != nil {
					return err
				}

				if err := verifyHeader(hdr, reader.DriveIsRegular); err != nil {
					return err
				}

				if err := indexHeader(record, block, hdr, metadata.Metadata, pipes.Compression, pipes.Encryption, initializing, onHeader); err != nil {
					return err
				}
			}

			curr, err := reader.Drive.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}

			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return err
			}

			currAndSize, err := reader.Drive.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}

			nextTotalBlocks := math.Ceil(float64(curr+(currAndSize-curr)) / float64(config.MagneticTapeBlockSize))
			record = int64(nextTotalBlocks) / int64(pipes.RecordSize)
			block = int64(nextTotalBlocks) - (record * int64(pipes.RecordSize))

			if block > int64(pipes.RecordSize) {
				record++
				block = 0
			}

			i++
		}
	} else {
		// Seek to record
		if err := mt.SeekToRecordOnTape(reader.Drive.Fd(), int32(record)); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(reader.Drive, config.MagneticTapeBlockSize*pipes.RecordSize)
		if _, err := br.Read(make([]byte, block*config.MagneticTapeBlockSize)); err != nil {
			return err
		}

		record := int64(record)
		block := int64(block)

		curr := int64((pipes.RecordSize * config.MagneticTapeBlockSize * int(record)) + (int(block) * config.MagneticTapeBlockSize))
		counter := &ioext.CounterReader{Reader: br, BytesRead: int(curr)}
		i := 0

		tr := tar.NewReader(counter)
		for {
			hdr, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					if err := mt.GoToNextFileOnTape(reader.Drive.Fd()); err != nil {
						// EOD

						break
					}

					record, err = mt.GetCurrentRecordFromTape(reader.Drive.Fd())
					if err != nil {
						return err
					}
					block = 0

					br = bufio.NewReaderSize(reader.Drive, config.MagneticTapeBlockSize*pipes.RecordSize)
					curr = int64(int64(pipes.RecordSize) * config.MagneticTapeBlockSize * record)
					counter = &ioext.CounterReader{Reader: br, BytesRead: int(curr)}
					tr = tar.NewReader(counter)

					continue
				} else {
					return err
				}
			}

			if i >= offset {
				if err := decryptHeader(hdr, i-offset); err != nil {
					return err
				}

				if err := verifyHeader(hdr, reader.DriveIsRegular); err != nil {
					return err
				}

				if err := indexHeader(record, block, hdr, metadata.Metadata, pipes.Compression, pipes.Encryption, initializing, onHeader); err != nil {
					return err
				}
			}

			curr = int64(counter.BytesRead)

			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return err
			}

			currAndSize := int64(counter.BytesRead)

			nextTotalBlocks := math.Ceil(float64(curr+(currAndSize-curr)) / float64(config.MagneticTapeBlockSize))
			record = int64(nextTotalBlocks) / int64(pipes.RecordSize)
			block = int64(nextTotalBlocks) - (record * int64(pipes.RecordSize))

			if block > int64(pipes.RecordSize) {
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
	metadataPersister config.MetadataPersister,
	compressionFormat string,
	encryptionFormat string,
	initializing bool,
	onHeader func(hdr *config.Header),
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

		onHeader(converters.DBHeaderToConfigHeader(dbhdr))
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

			if err := metadataPersister.UpsertHeader(context.Background(), converters.DBHeaderToConfigHeader(dbhdr), initializing); err != nil {
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

				if err := metadataPersister.UpdateHeaderMetadata(context.Background(), converters.DBHeaderToConfigHeader(newHdr)); err != nil {
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

					if err := metadataPersister.UpdateHeaderMetadata(context.Background(), converters.DBHeaderToConfigHeader(newHdr)); err != nil {
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
