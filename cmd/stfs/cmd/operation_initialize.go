package cmd

import (
	"fmt"
	"os"

	"github.com/pojntfx/stfs/internal/check"
	"github.com/pojntfx/stfs/internal/keyext"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/keys"
	"github.com/pojntfx/stfs/pkg/mtio"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var operationInitializeCmd = &cobra.Command{
	Use:     "initialize",
	Aliases: []string{"ini", "i", "init"},
	Short:   "Truncate and initalize a file or directory",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := check.CheckCompressionLevel(viper.GetString(compressionLevelFlag)); err != nil {
			return err
		}

		if err := check.CheckKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(recipientFlag)); err != nil {
			return err
		}

		return check.CheckKeyAccessible(viper.GetString(signatureFlag), viper.GetString(identityFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pubkey, err := keyext.ReadKey(viper.GetString(encryptionFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := keys.ParseRecipient(viper.GetString(encryptionFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := keyext.ReadKey(viper.GetString(signatureFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := keys.ParseSignerIdentity(viper.GetString(signatureFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		mt := mtio.MagneticTapeIO{}
		tm := tape.NewTapeManager(
			viper.GetString(driveFlag),
			mt,
			viper.GetInt(recordSizeFlag),
			true,
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

				MagneticTapeIO: mt,
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

		return ops.Initialize("/", os.ModePerm, viper.GetString(compressionLevelFlag))
	},
}

func init() {
	operationInitializeCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	operationInitializeCmd.PersistentFlags().StringP(compressionLevelFlag, "l", config.CompressionLevelBalancedKey, fmt.Sprintf("Compression level to use (default %v, available are %v)", config.CompressionLevelBalancedKey, config.KnownCompressionLevels))
	operationInitializeCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	operationInitializeCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	operationInitializeCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	operationCmd.AddCommand(operationInitializeCmd)
}
