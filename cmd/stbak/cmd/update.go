package cmd

import (
	"archive/tar"
	"context"
	"fmt"

	"github.com/pojntfx/stfs/internal/compression"
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/recovery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"upd", "u"},
	Short:   "Update a file or directory's content and metadata on tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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
		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		lastIndexedRecord, lastIndexedBlock, err := metadataPersister.GetLastIndexedRecordAndBlock(context.Background(), viper.GetInt(recordSizeFlag))
		if err != nil {
			return err
		}

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

		hdrs, err := operations.Update(
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
			viper.GetString(fromFlag),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionLevelFlag),
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

			func(hdr *tar.Header, i int) error {
				// Ignore the first header, which is the last header which we already indexed
				if i == 0 {
					return nil
				}

				if len(hdrs) <= i-1 {
					return config.ErrTarHeaderMissing
				}

				*hdr = *hdrs[i-1]

				return nil
			},
			func(hdr *tar.Header, isRegular bool) error {
				return nil // We sign above, no need to verify
			},
		)
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
