package cmd

import (
	"fmt"

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

const (
	recordSizeFlag       = "record-size"
	fromFlag             = "from"
	overwriteFlag        = "overwrite"
	compressionLevelFlag = "compression-level"
	recipientFlag        = "recipient"
	identityFlag         = "identity"
	passwordFlag         = "password"
)

var archiveCmd = &cobra.Command{
	Use:     "archive",
	Aliases: []string{"arc", "a", "c"},
	Short:   "Archive a file or directory to tape or tar file",
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
			viper.GetBool(overwriteFlag),
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

		if _, err := ops.Archive(
			viper.GetString(fromFlag),
			viper.GetString(compressionLevelFlag),
			viper.GetBool(overwriteFlag),
		); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	archiveCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	archiveCmd.PersistentFlags().StringP(fromFlag, "f", ".", "File or directory to archive")
	archiveCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the start instead of from the end of the tape or tar file")
	archiveCmd.PersistentFlags().StringP(compressionLevelFlag, "l", config.CompressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", config.CompressionLevelBalanced, config.KnownCompressionLevels))
	archiveCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	archiveCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	archiveCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(archiveCmd)
}
