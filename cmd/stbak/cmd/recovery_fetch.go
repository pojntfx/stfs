package cmd

import (
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/hardware"
	"github.com/pojntfx/stfs/pkg/recovery"
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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

		return recovery.Fetch(
			hardware.DriveConfig{
				Drive: viper.GetString(driveFlag),
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
			viper.GetInt(recordFlag),
			viper.GetInt(blockFlag),
			viper.GetString(toFlag),
			viper.GetBool(previewFlag),

			true,
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
