package cmd

import (
	"archive/tar"
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/converters"
	"github.com/pojntfx/stfs/pkg/counters"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var recoveryIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index contents of tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		privkey, err := readKey(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := parseIdentity(viper.GetString(encryptionFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		return index(
			viper.GetString(driveFlag),
			viper.GetString(metadataFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetInt(recordFlag),
			viper.GetInt(blockFlag),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
			viper.GetString(encryptionFlag),
			func(hdr *tar.Header, i int) error {
				return decryptHeader(hdr, viper.GetString(encryptionFlag), identity)
			},
			0,
		)
	},
}

func index(
	tape string,
	metadata string,
	recordSize int,
	record int,
	block int,
	overwrite bool,
	compressionFormat string,
	encryptionFormat string,
	decryptHeader func(
		hdr *tar.Header,
		i int,
	) error,
	offset int,
) error {
	if overwrite {
		f, err := os.OpenFile(metadata, os.O_WRONLY|os.O_CREATE, 0600)
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

	metadataPersister := persisters.NewMetadataPersister(metadata)
	if err := metadataPersister.Open(); err != nil {
		return err
	}

	f, isRegular, err := openTapeReadOnly(tape)
	if err != nil {
		return err
	}
	defer f.Close()

	if isRegular {
		// Seek to record and block
		if _, err := f.Seek(int64((recordSize*controllers.BlockSize*record)+block*controllers.BlockSize), 0); err != nil {
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

					nextTotalBlocks := math.Ceil(float64((curr)) / float64(controllers.BlockSize))
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
					if _, err := f.Seek(int64((recordSize*controllers.BlockSize*int(record))+int(block)*controllers.BlockSize), io.SeekStart); err != nil {
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

			if i >= offset {
				if err := decryptHeader(hdr, i-offset); err != nil {
					return err
				}

				if err := indexHeader(record, block, hdr, metadataPersister, compressionFormat, encryptionFormat); err != nil {
					return nil
				}
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

			nextTotalBlocks := math.Ceil(float64(curr+(currAndSize-curr)) / float64(controllers.BlockSize))
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
		if err := controllers.SeekToRecordOnTape(f, int32(record)); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(f, controllers.BlockSize*recordSize)
		if _, err := br.Read(make([]byte, block*controllers.BlockSize)); err != nil {
			return err
		}

		record := int64(record)
		block := int64(block)

		curr := int64((recordSize * controllers.BlockSize * int(record)) + (int(block) * controllers.BlockSize))
		counter := &counters.CounterReader{Reader: br, BytesRead: int(curr)}
		i := 0

		tr := tar.NewReader(counter)
		for {
			hdr, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					if err := controllers.GoToNextFileOnTape(f); err != nil {
						// EOD

						break
					}

					record, err = controllers.GetCurrentRecordFromTape(f)
					if err != nil {
						return err
					}
					block = 0

					br = bufio.NewReaderSize(f, controllers.BlockSize*recordSize)
					curr = int64(int64(recordSize) * controllers.BlockSize * record)
					counter = &counters.CounterReader{Reader: br, BytesRead: int(curr)}
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

				if err := indexHeader(record, block, hdr, metadataPersister, compressionFormat, encryptionFormat); err != nil {
					return nil
				}
			}

			curr = int64(counter.BytesRead)

			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return err
			}

			currAndSize := int64(counter.BytesRead)

			nextTotalBlocks := math.Ceil(float64(curr+(currAndSize-curr)) / float64(controllers.BlockSize))
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
) error {
	if record == 0 && block == 0 {
		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}
	}

	uncompressedSize, ok := hdr.PAXRecords[pax.STFSRecordUncompressedSize]
	if ok {
		size, err := strconv.Atoi(uncompressedSize)
		if err != nil {
			return err
		}

		hdr.Size = int64(size)
	}

	if hdr.FileInfo().Mode().IsRegular() {
		newName, err := removeSuffix(hdr.Name, compressionFormat, encryptionFormat)
		if err != nil {
			return err
		}
		hdr.Name = newName
	}

	if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(record, block, hdr)); err != nil {
		return err
	}

	stfsVersion, ok := hdr.PAXRecords[pax.STFSRecordVersion]
	if !ok {
		stfsVersion = pax.STFSRecordVersion1
	}

	switch stfsVersion {
	case pax.STFSRecordVersion1:
		stfsAction, ok := hdr.PAXRecords[pax.STFSRecordAction]
		if !ok {
			stfsAction = pax.STFSRecordActionCreate
		}

		switch stfsAction {
		case pax.STFSRecordActionCreate:
			dbhdr, err := converters.TarHeaderToDBHeader(record, block, hdr)
			if err != nil {
				return err
			}

			if err := metadataPersister.UpsertHeader(context.Background(), dbhdr); err != nil {
				return err
			}
		case pax.STFSRecordActionDelete:
			if _, err := metadataPersister.DeleteHeader(context.Background(), hdr.Name, true); err != nil {
				return err
			}
		case pax.STFSRecordActionUpdate:
			moveAfterEdits := false
			oldName := hdr.Name
			if _, ok := hdr.PAXRecords[pax.STFSRecordReplacesName]; ok {
				moveAfterEdits = true
				oldName = hdr.PAXRecords[pax.STFSRecordReplacesName]
			}

			var newHdr *models.Header
			if replacesContent, ok := hdr.PAXRecords[pax.STFSRecordReplacesContent]; ok && replacesContent == pax.STFSRecordReplacesContentTrue {
				// Content & metadata update; use the new record & block
				h, err := converters.TarHeaderToDBHeader(record, block, hdr)
				if err != nil {
					return err
				}

				newHdr = h
			} else {
				// Metadata-only update; use the old record & block
				oldHdr, err := metadataPersister.GetHeader(context.Background(), oldName)
				if err != nil {
					return err
				}

				h, err := converters.TarHeaderToDBHeader(oldHdr.Record, oldHdr.Block, hdr)
				if err != nil {
					return err
				}

				newHdr = h
			}

			if err := metadataPersister.UpdateHeaderMetadata(context.Background(), newHdr); err != nil {
				return err
			}

			if moveAfterEdits {
				// Move header
				if err := metadataPersister.MoveHeader(context.Background(), oldName, hdr.Name); err != nil {
					return err
				}
			}

		default:
			return pax.ErrUnsupportedAction
		}
	default:
		return pax.ErrUnsupportedVersion
	}

	return nil
}

func removeSuffix(name string, compressionFormat string, encryptionFormat string) (string, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		name = strings.TrimSuffix(name, encryptionFormatAgeSuffix)
	case encryptionFormatPGPKey:
		name = strings.TrimSuffix(name, encryptionFormatPGPSuffix)
	case noneKey:
	default:
		return "", errUnsupportedEncryptionFormat
	}

	switch compressionFormat {
	case compressionFormatGZipKey:
		fallthrough
	case compressionFormatParallelGZipKey:
		name = strings.TrimSuffix(name, compressionFormatGZipSuffix)
	case compressionFormatLZ4Key:
		name = strings.TrimSuffix(name, compressionFormatLZ4Suffix)
	case compressionFormatZStandardKey:
		name = strings.TrimSuffix(name, compressionFormatZStandardSuffix)
	case compressionFormatBrotliKey:
		name = strings.TrimSuffix(name, compressionFormatBrotliSuffix)
	case compressionFormatBzip2Key:
		fallthrough
	case compressionFormatBzip2ParallelKey:
		name = strings.TrimSuffix(name, compressionFormatBzip2Suffix)
	case noneKey:
	default:
		return "", errUnsupportedCompressionFormat
	}

	return name, nil
}

func openTapeReadOnly(tape string) (f *os.File, isRegular bool, err error) {
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

func init() {
	recoveryIndexCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	recoveryIndexCmd.PersistentFlags().IntP(recordFlag, "k", 0, "Record to seek too before counting")
	recoveryIndexCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too before counting")
	recoveryIndexCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Remove the old index before starting to index")
	recoveryIndexCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")
	recoveryIndexCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	recoveryCmd.AddCommand(recoveryIndexCmd)
}
