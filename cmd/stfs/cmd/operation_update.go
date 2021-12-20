package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/internal/compression"
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var operationUpdateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"upd", "u", "put"},
	Short:   "Update a file or directory's content and metadata on tape or tar file",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := compression.CheckCompressionLevel(viper.GetString(compressionLevelFlag)); err != nil {
			return err
		}

		if err := keys.CheckKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(recipientFlag)); err != nil {
			return err
		}

		return keys.CheckKeyAccessible(viper.GetString(signatureFlag), viper.GetString(identityFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pubkey, err := keys.ReadKey(viper.GetString(encryptionFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := keys.ParseRecipient(viper.GetString(encryptionFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := keys.ReadKey(viper.GetString(signatureFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := keys.ParseSignerIdentity(viper.GetString(signatureFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		tm := tape.NewTapeManager(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			false,
		)

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		ops := operations.NewOperations(
			config.BackendConfig{
				GetWriter:   tm.GetWriter,
				CloseWriter: tm.Close,

				GetReader:   tm.GetReader,
				CloseReader: tm.Close,

				GetDrive:   tm.GetDrive,
				CloseDrive: tm.Close,
			},
			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			config.PipeConfig{
				Compression: viper.GetString(compressionFlag),
				Encryption:  viper.GetString(encryptionFlag),
				Signature:   viper.GetString(signatureFlag),
				RecordSize:  viper.GetInt(recordSizeFlag),
			},
			config.CryptoConfig{
				Recipient: recipient,
				Identity:  identity,
				Password:  viper.GetString(passwordFlag),
			},

			logging.NewLogger().PrintHeaderEvent,
		)

		files := make(chan config.FileConfig)
		errs := make(chan error)
		go func() {
			if err := filepath.Walk(viper.GetString(fromFlag), func(path string, info fs.FileInfo, err error) error {
				path = filepath.ToSlash(path)

				if err != nil {
					return err
				}

				link := ""
				if info.Mode()&os.ModeSymlink == os.ModeSymlink {
					if link, err = os.Readlink(path); err != nil {
						return err
					}
				}

				files <- config.FileConfig{
					GetFile: func() (io.ReadSeekCloser, error) {
						return os.Open(path)
					},
					Info: info,
					Path: path,
					Link: link,
				}

				return nil
			}); err != nil {
				errs <- err

				return
			}

			errs <- io.EOF
		}()

		if _, err := ops.Update(
			func() (config.FileConfig, error) {
				select {
				case file := <-files:
					return file, err
				case err := <-errs:
					return config.FileConfig{}, err
				}
			},
			viper.GetString(compressionLevelFlag),
			viper.GetBool(overwriteFlag),
		); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	operationUpdateCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	operationUpdateCmd.PersistentFlags().StringP(fromFlag, "f", "", "Path of the file or directory to update")
	operationUpdateCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Replace the content on the tape or tar file")
	operationUpdateCmd.PersistentFlags().StringP(compressionLevelFlag, "l", config.CompressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", config.CompressionLevelBalanced, config.KnownCompressionLevels))
	operationUpdateCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	operationUpdateCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	operationUpdateCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	operationCmd.AddCommand(operationUpdateCmd)
}
