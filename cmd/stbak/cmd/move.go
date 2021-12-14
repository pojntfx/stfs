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

var moveCmd = &cobra.Command{
	Use:     "move",
	Aliases: []string{"mov", "m", "mv"},
	Short:   "Move a file or directory on tape or tar file",
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

		tm := tape.NewTapeManager(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			false,
		)

		ops := operations.NewOperations(
			tm.GetWriter,
			tm.Close,

			tm.GetReader,
			tm.Close,

			tm.GetDrive,
			tm.Close,
		)

		return ops.Move(
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
			viper.GetString(fromFlag),
			viper.GetString(toFlag),

			logging.NewLogger().PrintHeader,
		)
	},
}

func init() {
	moveCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	moveCmd.PersistentFlags().StringP(fromFlag, "f", "", "Current path of the file or directory to move")
	moveCmd.PersistentFlags().StringP(toFlag, "t", "", "Path to move the file or directory to")
	moveCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	moveCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	moveCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(moveCmd)
}
