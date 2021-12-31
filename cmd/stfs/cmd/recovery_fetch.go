package cmd

import (
	"io"
	"io/fs"
	"os"

	"github.com/pojntfx/stfs/internal/check"
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/recovery"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	recordFlag  = "record"
	blockFlag   = "block"
	toFlag      = "to"
	previewFlag = "preview"
)

var recoveryFetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch a file or directory from tape or tar file by record and block without the index",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := check.CheckKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(identityFlag)); err != nil {
			return err
		}

		return check.CheckKeyAccessible(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pubkey, err := keys.ReadKey(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := keys.ParseSignerRecipient(viper.GetString(signatureFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := keys.ReadKey(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := keys.ParseIdentity(viper.GetString(encryptionFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		reader, readerIsRegular, err := tape.OpenTapeReadOnly(
			viper.GetString(driveFlag),
		)
		if err != nil {
			return nil
		}
		defer reader.Close()

		return recovery.Fetch(
			config.DriveReaderConfig{
				Drive:          reader,
				DriveIsRegular: readerIsRegular,
			},
			config.DriveConfig{
				Drive:          reader,
				DriveIsRegular: readerIsRegular,
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

			func(path string, mode fs.FileMode) (io.WriteCloser, error) {
				dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, mode)
				if err != nil {
					return nil, err
				}

				if err := dstFile.Truncate(0); err != nil {
					return nil, err
				}

				return dstFile, nil
			},
			func(path string, mode fs.FileMode) error {
				return os.MkdirAll(path, mode)
			},

			viper.GetInt(recordSizeFlag),
			viper.GetInt(recordFlag),
			viper.GetInt(blockFlag),
			viper.GetString(toFlag),
			viper.GetBool(previewFlag),

			logging.NewCSVLogger().PrintHeader,
		)
	},
}

func init() {
	recoveryFetchCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	recoveryFetchCmd.PersistentFlags().IntP(recordFlag, "k", 0, "Record to seek too")
	recoveryFetchCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too")
	recoveryFetchCmd.PersistentFlags().StringP(toFlag, "t", "", "File to restore to (archived name by default)")
	recoveryFetchCmd.PersistentFlags().BoolP(previewFlag, "w", false, "Only read the header")
	recoveryFetchCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")
	recoveryFetchCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")
	recoveryFetchCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to the public key to verify with")

	viper.AutomaticEnv()

	recoveryCmd.AddCommand(recoveryFetchCmd)
}
