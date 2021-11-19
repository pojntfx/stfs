package cmd

import (
	"archive/tar"
	"bufio"
	"io"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/readers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List contents of tape or tar file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
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
			tr := tar.NewReader(f)

			record := int64(0)
			block := int64(0)
			firstRecordOfArchive := int64(0)

			for {
				hdr, err := tr.Next()
				if err != nil {
					// Seek right after the next two blocks to skip the trailer
					if _, err := f.Seek((int64(viper.GetInt(recordSizeFlag))*controllers.BlockSize*record)+(block+1)*controllers.BlockSize, io.SeekStart); err == nil {
						tr = tar.NewReader(f)

						hdr, err = tr.Next()
						if err != nil {
							if err == io.EOF {
								break
							}

							return err
						}

						block++
						if block > int64(viper.GetInt(recordSizeFlag)) {
							record++
							block = 0
						}

						firstRecordOfArchive = record
					} else {
						return err
					}
				}

				if record == 0 && block == 0 {
					if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
						return err
					}
				}

				if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(record, block, hdr)); err != nil {
					return err
				}

				curr, err := f.Seek(0, io.SeekCurrent)
				if err != nil {
					return err
				}

				nextTotalBlocks := (curr + hdr.Size) / controllers.BlockSize
				record = nextTotalBlocks / int64(viper.GetInt(recordSizeFlag))

				if record == 0 && block == 0 || record == firstRecordOfArchive {
					block = nextTotalBlocks - (record * int64(viper.GetInt(recordSizeFlag))) // For the first record of the file or archive, the offset of one is not needed
				} else {
					block = nextTotalBlocks - (record * int64(viper.GetInt(recordSizeFlag))) + 1 // +1 because we need to start reading right after the last block
				}

				if block > int64(viper.GetInt(recordSizeFlag)) {
					record++
					block = 0
				}
			}
		} else {
			br := bufio.NewReaderSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))

			counter := &readers.Counter{Reader: br}
			lastBytesRead := 0
			dirty := false

			record := int64(0)
			block := int64(0)

			for {
				tr := tar.NewReader(counter)
				hdr, err := tr.Next()
				if err != nil {
					if lastBytesRead == counter.BytesRead {
						if dirty {
							// EOD

							break
						}

						if err := controllers.GoToNextFileOnTape(f); err != nil {
							// EOD

							break
						}

						currentRecord, err := controllers.GetCurrentRecordFromTape(f)
						if err != nil {
							return err
						}

						br = bufio.NewReaderSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
						counter = &readers.Counter{Reader: br, BytesRead: (int(currentRecord) * viper.GetInt(recordSizeFlag) * controllers.BlockSize)} // We asume we are at record n, block 0

						dirty = true
					}

					lastBytesRead = counter.BytesRead

					continue
				}

				lastBytesRead = counter.BytesRead

				if hdr.Format == tar.FormatUnknown {
					continue
				}

				dirty = false

				if counter.BytesRead == 0 {
					if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
						return err
					}
				}

				if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(record, block, hdr)); err != nil {
					return err
				}

				nextBytes := int64(counter.BytesRead) + hdr.Size + controllers.BlockSize - 1

				record = nextBytes / (controllers.BlockSize * int64(viper.GetInt(recordSizeFlag)))
				block = (nextBytes - (record * int64(viper.GetInt(recordSizeFlag)) * controllers.BlockSize)) / controllers.BlockSize
			}
		}

		return nil
	},
}

func init() {
	listCmd.PersistentFlags().StringP(tapeFlag, "t", "/dev/nst0", "Tape or tar file to read from")
	listCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")

	viper.AutomaticEnv()

	rootCmd.AddCommand(listCmd)
}
