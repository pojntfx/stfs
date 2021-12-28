package cmd

import (
	"io"
	"io/fs"
	"os"

	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flattenFlag = "flatten"
)

var operationRestoreCmd = &cobra.Command{
	Use:     "restore",
	Aliases: []string{"res", "r", "x", "get", "extract"},
	Short:   "Restore a file or directory from tape or tar file",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := keys.CheckKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(identityFlag)); err != nil {
			return err
		}

		return keys.CheckKeyAccessible(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
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

			logging.NewCSVLogger().PrintHeaderEvent,
		)

		return ops.Restore(
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

			viper.GetString(fromFlag),
			viper.GetString(toFlag),
			viper.GetBool(flattenFlag),
		)
	},
}

func init() {
	operationRestoreCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	operationRestoreCmd.PersistentFlags().StringP(fromFlag, "f", "", "File or directory to restore")
	operationRestoreCmd.PersistentFlags().StringP(toFlag, "t", "", "File or directory restore to (archived name by default)")
	operationRestoreCmd.PersistentFlags().BoolP(flattenFlag, "a", false, "Ignore the folder hierarchy on the tape or tar file")
	operationRestoreCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")
	operationRestoreCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")
	operationRestoreCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to the public key to verify with")

	viper.AutomaticEnv()

	operationCmd.AddCommand(operationRestoreCmd)
}
