package cmd

import (
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	nameFlag = "name"
)

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"del", "d", "rm"},
	Short:   "Delete a file or directory from tape or tar file",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
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

		writer, writerIsRegular, err := tape.OpenTapeWriteOnly(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			false,
		)
		if err != nil {
			return nil
		}
		defer writer.Close()
		reader, readerIsRegular, err := tape.OpenTapeReadOnly(
			viper.GetString(driveFlag),
		)
		if err != nil {
			return nil
		}
		defer reader.Close()

		return operations.Delete(
			config.DriveWriterConfig{
				Drive:          writer,
				DriveIsRegular: writerIsRegular,
			},
			config.DriveReaderConfig{
				Drive:          reader,
				DriveIsRegular: readerIsRegular,
			},
			config.DriveConfig{
				Drive:          reader,
				DriveIsRegular: readerIsRegular,
			},
			config.MetadataConfig{
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
			viper.GetString(nameFlag),

			logging.NewLogger().PrintHeader,
		)
	},
}

func init() {
	deleteCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	deleteCmd.PersistentFlags().StringP(nameFlag, "n", "", "Name of the file to remove")
	deleteCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	deleteCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	deleteCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(deleteCmd)
}
