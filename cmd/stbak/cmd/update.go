package cmd

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pojntfx/stfs/pkg/adapters"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/counters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"upd", "u"},
	Short:   "Update a file or directory's content and metadata on tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := checkCompressionLevel(viper.GetString(compressionLevelFlag)); err != nil {
			return err
		}

		if viper.GetString(encryptionFlag) != encryptionFormatNoneKey {
			if _, err := os.Stat(viper.GetString(recipientFlag)); err != nil {
				return errRecipientNotAccessible
			}

			if _, err := os.Stat(viper.GetString(identityFlag)); err != nil {
				return errIdentityNotAccessible
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		lastIndexedRecord, lastIndexedBlock, err := metadataPersister.GetLastIndexedRecordAndBlock(context.Background(), viper.GetInt(recordSizeFlag))
		if err != nil {
			return err
		}

		pubkey := []byte{}
		privkey := []byte{}
		if viper.GetString(encryptionFlag) != encryptionFormatNoneKey {
			p, err := ioutil.ReadFile(viper.GetString(recipientFlag))
			if err != nil {
				return err
			}

			pubkey = p

			privkey, err = ioutil.ReadFile(viper.GetString(identityFlag))
			if err != nil {
				return err
			}
		}

		if err := update(
			viper.GetString(tapeFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(srcFlag),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
			viper.GetString(compressionLevelFlag),
			viper.GetString(encryptionFlag),
			pubkey,
		); err != nil {
			return err
		}

		return index(
			viper.GetString(tapeFlag),
			viper.GetString(metadataFlag),
			viper.GetInt(recordSizeFlag),
			int(lastIndexedRecord),
			int(lastIndexedBlock),
			false,
			viper.GetString(compressionFlag),
			viper.GetString(encryptionFlag),
			privkey,
		)
	},
}

func update(
	tape string,
	recordSize int,
	src string,
	replacesContent bool,
	compressionFormat string,
	compressionLevel string,
	encryptionFormat string,
	pubkey []byte,
) error {
	dirty := false
	tw, isRegular, cleanup, err := openTapeWriter(tape)
	if err != nil {
		return err
	}
	defer cleanup(&dirty)

	first := true
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		link := ""
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		if err := adapters.EnhanceHeader(path, hdr); err != nil {
			return err
		}

		hdr.Name = path
		hdr.Format = tar.FormatPAX
		if hdr.PAXRecords == nil {
			hdr.PAXRecords = map[string]string{}
		}
		hdr.PAXRecords[pax.STFSRecordVersion] = pax.STFSRecordVersion1
		hdr.PAXRecords[pax.STFSRecordAction] = pax.STFSRecordActionUpdate

		if info.Mode().IsRegular() && replacesContent {
			// Get the compressed size for the header
			fileSizeCounter := &counters.CounterWriter{
				Writer: io.Discard,
			}

			encryptor, err := encrypt(fileSizeCounter, encryptionFormat, pubkey)
			if err != nil {
				return err
			}

			compressor, err := compress(
				encryptor,
				compressionFormat,
				compressionLevel,
				isRegular,
				recordSize,
			)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(compressor, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(compressor, file, buf); err != nil {
					return err
				}
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := compressor.Flush(); err != nil {
				return err
			}

			if err := compressor.Close(); err != nil {
				return err
			}

			if err := encryptor.Close(); err != nil {
				return err
			}

			if hdr.PAXRecords == nil {
				hdr.PAXRecords = map[string]string{}
			}
			hdr.PAXRecords[pax.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
			hdr.Size = int64(fileSizeCounter.BytesRead)

			hdr.Name, err = addSuffix(hdr.Name, compressionFormat, encryptionFormat)
			if err != nil {
				return err
			}
		}

		if first {
			if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
				return err
			}

			first = false
		}

		if replacesContent {
			hdr.PAXRecords[pax.STFSRecordReplacesContent] = pax.STFSRecordReplacesContentTrue

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
				return err
			}

			if err := encryptHeader(hdr, encryptionFormat, pubkey); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			// Compress and write the file
			encryptor, err := encrypt(tw, encryptionFormat, pubkey)
			if err != nil {
				return err
			}

			compressor, err := compress(
				encryptor,
				compressionFormat,
				compressionLevel,
				isRegular,
				recordSize,
			)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(compressor, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(compressor, file, buf); err != nil {
					return err
				}
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := compressor.Flush(); err != nil {
				return err
			}

			if err := compressor.Close(); err != nil {
				return err
			}

			if err := encryptor.Close(); err != nil {
				return err
			}
		} else {
			hdr.Size = 0 // Don't try to seek after the record

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
				return err
			}

			if err := encryptHeader(hdr, encryptionFormat, pubkey); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		}

		dirty = true

		return nil
	})
}

func init() {
	updateCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	updateCmd.PersistentFlags().StringP(srcFlag, "s", "", "Path of the file or directory to update")
	updateCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Replace the content on the tape or tar file")
	updateCmd.PersistentFlags().StringP(compressionLevelFlag, "l", compressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", compressionLevelBalanced, knownCompressionLevels))
	updateCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	updateCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")

	viper.AutomaticEnv()

	rootCmd.AddCommand(updateCmd)
}
