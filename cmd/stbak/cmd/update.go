package cmd

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pojntfx/stfs/internal/adapters"
	"github.com/pojntfx/stfs/internal/controllers"
	"github.com/pojntfx/stfs/internal/counters"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/pax"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/recovery"
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

		if err := checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(recipientFlag)); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(signatureFlag), viper.GetString(identityFlag))
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

		pubkey, err := readKey(viper.GetString(encryptionFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := parseRecipient(viper.GetString(encryptionFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := readKey(viper.GetString(signatureFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := parseSignerIdentity(viper.GetString(signatureFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		hdrs, err := update(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(fromFlag),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
			viper.GetString(compressionLevelFlag),
			viper.GetString(encryptionFlag),
			recipient,
			viper.GetString(signatureFlag),
			identity,
		)
		if err != nil {
			return err
		}

		return recovery.Index(
			config.StateConfig{
				Drive:    viper.GetString(driveFlag),
				Metadata: viper.GetString(metadataFlag),
			},
			config.PipeConfig{
				Compression: viper.GetString(compressionFlag),
				Encryption:  viper.GetString(encryptionFlag),
				Signature:   viper.GetString(signatureFlag),
			},
			config.CryptoConfig{
				Recipient: recipient,
				Identity:  identity,
				Password:  viper.GetString(passwordFlag),
			},

			viper.GetInt(recordSizeFlag),
			int(lastIndexedRecord),
			int(lastIndexedBlock),
			false,

			1,
			func(hdr *tar.Header, i int) error {
				if len(hdrs) <= i {
					return errMissingTarHeader
				}

				*hdr = *hdrs[i]

				return nil
			},
			func(hdr *tar.Header, isRegular bool) error {
				return nil // We sign above, no need to verify
			},
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
	recipient interface{},
	signatureFormat string,
	identity interface{},
) ([]*tar.Header, error) {
	dirty := false
	tw, isRegular, cleanup, err := openTapeWriter(tape, recordSize, false)
	if err != nil {
		return []*tar.Header{}, err
	}
	defer cleanup(&dirty)

	headers := []*tar.Header{}
	first := true
	return headers, filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
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

			encryptor, err := encrypt(fileSizeCounter, encryptionFormat, recipient)
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

			signer, sign, err := sign(file, isRegular, signatureFormat, identity)
			if err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(compressor, signer); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(compressor, signer, buf); err != nil {
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
			signature, err := sign()
			if err != nil {
				return err
			}

			if signature != "" {
				hdr.PAXRecords[pax.STFSRecordSignature] = signature
			}
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

			hdrToAppend := *hdr
			headers = append(headers, &hdrToAppend)

			if err := signHeader(hdr, isRegular, signatureFormat, identity); err != nil {
				return err
			}

			if err := encryptHeader(hdr, encryptionFormat, recipient); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			// Compress and write the file
			encryptor, err := encrypt(tw, encryptionFormat, recipient)
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

			hdrToAppend := *hdr
			headers = append(headers, &hdrToAppend)

			if err := signHeader(hdr, isRegular, signatureFormat, identity); err != nil {
				return err
			}

			if err := encryptHeader(hdr, encryptionFormat, recipient); err != nil {
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
	updateCmd.PersistentFlags().StringP(fromFlag, "f", "", "Path of the file or directory to update")
	updateCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Replace the content on the tape or tar file")
	updateCmd.PersistentFlags().StringP(compressionLevelFlag, "l", compressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", compressionLevelBalanced, knownCompressionLevels))
	updateCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	updateCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	updateCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(updateCmd)
}
