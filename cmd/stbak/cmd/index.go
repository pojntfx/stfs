package cmd

import (
	"archive/tar"
	"bufio"
	"context"
	"io"
	"math"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/converters"
	"github.com/pojntfx/stfs/pkg/counters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var indexCmd = &cobra.Command{
	Use:     "index",
	Aliases: []string{"i"},
	Short:   "Index contents of tape or tar file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(overwriteFlag) {
			f, err := os.OpenFile(viper.GetString(metadataFlag), os.O_WRONLY|os.O_CREATE, 0600)
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

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		fileDescription, err := os.Stat(viper.GetString(tapeFlag))
		if err != nil {
			return err
		}

		var f *os.File
		if fileDescription.Mode().IsRegular() {
			f, err = os.Open(viper.GetString(tapeFlag))
			if err != nil {
				return err
			}
		} else {
			f, err = os.OpenFile(viper.GetString(tapeFlag), os.O_RDONLY, os.ModeCharDevice)
			if err != nil {
				return err
			}
		}
		defer f.Close()

		if fileDescription.Mode().IsRegular() {
			// Seek to record and block
			if _, err := f.Seek(int64((viper.GetInt(recordSizeFlag)*controllers.BlockSize*viper.GetInt(recordFlag))+viper.GetInt(blockFlag)*controllers.BlockSize), 0); err != nil {
				return err
			}

			tr := tar.NewReader(f)

			record := viper.GetInt64(recordFlag)
			block := viper.GetInt64(blockFlag)

			for {
				hdr, err := tr.Next()
				if err != nil {
					// Seek right after the next two blocks to skip the trailer
					if _, err := f.Seek((controllers.BlockSize * 2), io.SeekCurrent); err == nil {
						curr, err := f.Seek(0, io.SeekCurrent)
						if err != nil {
							return err
						}

						nextTotalBlocks := math.Ceil(float64((curr)) / float64(controllers.BlockSize))
						record = int64(nextTotalBlocks) / int64(viper.GetInt(recordSizeFlag))
						block = int64(nextTotalBlocks) - (record * int64(viper.GetInt(recordSizeFlag))) - 2

						if block < 0 {
							record--
							block = int64(viper.GetInt(recordSizeFlag)) - 1
						} else if block >= int64(viper.GetInt(recordSizeFlag)) {
							record++
							block = 0
						}

						// Seek to record and block
						if _, err := f.Seek(int64((viper.GetInt(recordSizeFlag)*controllers.BlockSize*int(record))+int(block)*controllers.BlockSize), io.SeekStart); err != nil {
							return err
						}

						tr = tar.NewReader(f)

						hdr, err = tr.Next()
						if err != nil {
							if err == io.EOF {
								break
							}

							return err
						}
					} else {
						return err
					}
				}

				if err := indexHeader(record, block, hdr, metadataPersister); err != nil {
					return nil
				}

				curr, err := f.Seek(0, io.SeekCurrent)
				if err != nil {
					return err
				}

				nextTotalBlocks := math.Ceil(float64((curr + hdr.Size)) / float64(controllers.BlockSize))
				record = int64(nextTotalBlocks) / int64(viper.GetInt(recordSizeFlag))
				block = int64(nextTotalBlocks) - (record * int64(viper.GetInt(recordSizeFlag)))

				if block > int64(viper.GetInt(recordSizeFlag)) {
					record++
					block = 0
				}
			}
		} else {
			// Seek to record
			if err := controllers.SeekToRecordOnTape(f, int32(viper.GetInt(recordFlag))); err != nil {
				return err
			}

			// Seek to block
			br := bufio.NewReaderSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
			if _, err := br.Read(make([]byte, viper.GetInt(blockFlag)*controllers.BlockSize)); err != nil {
				return err
			}

			record := viper.GetInt64(recordFlag)
			block := viper.GetInt64(blockFlag)

			curr := int64((viper.GetInt(recordSizeFlag) * controllers.BlockSize * viper.GetInt(recordFlag)) + (viper.GetInt(blockFlag) * controllers.BlockSize))
			counter := &counters.CounterReader{Reader: br, BytesRead: int(curr)}

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

						br = bufio.NewReaderSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
						curr := int64(int64(viper.GetInt(recordSizeFlag)) * controllers.BlockSize * record)
						counter := &counters.CounterReader{Reader: br, BytesRead: int(curr)}
						tr = tar.NewReader(counter)

						continue
					} else {
						return err
					}
				}

				if err := indexHeader(record, block, hdr, metadataPersister); err != nil {
					return nil
				}

				curr = int64(counter.BytesRead)

				nextTotalBlocks := math.Ceil(float64((curr + hdr.Size)) / float64(controllers.BlockSize))
				record = int64(nextTotalBlocks) / int64(viper.GetInt(recordSizeFlag))
				block = int64(nextTotalBlocks) - (record * int64(viper.GetInt(recordSizeFlag)))

				if block > int64(viper.GetInt(recordSizeFlag)) {
					record++
					block = 0
				}
			}
		}

		return nil
	},
}

func init() {
	indexCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	indexCmd.PersistentFlags().IntP(recordFlag, "r", 0, "Record to seek too before counting")
	indexCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too before counting")
	indexCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the current position instead of from the end of the tape/file")

	viper.AutomaticEnv()

	rootCmd.AddCommand(indexCmd)
}

func indexHeader(record, block int64, hdr *tar.Header, metadataPersister *persisters.MetadataPersister) error {
	if record == 0 && block == 0 {
		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}
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
		default:
			return pax.ErrUnsupportedAction
		}
	default:
		return pax.ErrUnsupportedVersion
	}

	return nil
}
