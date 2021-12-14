package cmd

import (
	"fmt"

	"github.com/pojntfx/stfs/internal/compression"
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"upd", "u"},
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

		ops := operations.NewOperations(
			tm.GetWriter,
			tm.Close,

			tm.GetReader,
			tm.Close,

			tm.GetDrive,
			tm.Close,
		)

		if _, err := ops.Update(
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
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionLevelFlag),

			logging.NewLogger().PrintHeader,
		); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	updateCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	updateCmd.PersistentFlags().StringP(fromFlag, "f", "", "Path of the file or directory to update")
	updateCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Replace the content on the tape or tar file")
	updateCmd.PersistentFlags().StringP(compressionLevelFlag, "l", config.CompressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", config.CompressionLevelBalanced, config.KnownCompressionLevels))
	updateCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	updateCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	updateCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(updateCmd)
}
