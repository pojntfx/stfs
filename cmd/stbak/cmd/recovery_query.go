package cmd

import (
	"archive/tar"
	"bufio"
	"io"
	"io/ioutil"
	"math"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/counters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var recoveryQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query contents of tape or tar file without the index",
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

		privkey := []byte{}
		if viper.GetString(encryptionFlag) != encryptionFormatNoneKey {
			p, err := ioutil.ReadFile(viper.GetString(identityFlag))
			if err != nil {
				return err
			}

			privkey = p
		}

		return query(
			viper.GetString(tapeFlag),
			viper.GetInt(recordFlag),
			viper.GetInt(blockFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(encryptionFlag),
			privkey,
		)
	},
}

func query(
	tape string,
	record int,
	block int,
	recordSize int,
	encryptionFormat string,
	privkey []byte,
) error {
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

			if err := decryptHeader(hdr, encryptionFormat, privkey); err != nil {
				return err
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
					curr := int64(int64(recordSize) * controllers.BlockSize * record)
					counter := &counters.CounterReader{Reader: br, BytesRead: int(curr)}
					tr = tar.NewReader(counter)

					continue
				} else {
					return err
				}
			}

			if err := decryptHeader(hdr, encryptionFormat, privkey); err != nil {
				return err
			}

			if record == 0 && block == 0 {
				if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
					return err
				}
			}

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(record, block, hdr)); err != nil {
				return err
			}

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
		}
	}

	return nil
}

func init() {
	recoveryQueryCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	recoveryQueryCmd.PersistentFlags().IntP(recordFlag, "k", 0, "Record to seek too before counting")
	recoveryQueryCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too before counting")
	recoveryQueryCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")

	viper.AutomaticEnv()

	recoveryCmd.AddCommand(recoveryQueryCmd)
}
