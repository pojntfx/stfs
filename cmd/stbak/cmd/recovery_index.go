package cmd

import (
	"archive/tar"

	"github.com/pojntfx/stfs/internal/encryption"
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/internal/signature"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/recovery"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var recoveryIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index contents of tape or tar file",
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

		reader, readerIsRegular, err := tape.OpenTapeReadOnly(
			viper.GetString(driveFlag),
		)
		if err != nil {
			return nil
		}
		defer reader.Close()

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		return recovery.Index(
			config.DriveReaderConfig{
				Drive:          reader,
				DriveIsRegular: readerIsRegular,
			},
			config.DriveConfig{
				Drive:          reader,
				DriveIsRegular: readerIsRegular,
			},
			config.MetadataConfig{
				Metadata: metadataPersister,
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
			viper.GetBool(overwriteFlag),
			0,

			func(hdr *tar.Header, i int) error {
				return encryption.DecryptHeader(hdr, viper.GetString(encryptionFlag), identity)
			},
			func(hdr *tar.Header, isRegular bool) error {
				return signature.VerifyHeader(hdr, isRegular, viper.GetString(signatureFlag), recipient)
			},

			logging.NewLogger().PrintHeader,
		)
	},
}

func init() {
	recoveryIndexCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	recoveryIndexCmd.PersistentFlags().IntP(recordFlag, "k", 0, "Record to seek too before counting")
	recoveryIndexCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too before counting")
	recoveryIndexCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Remove the old index before starting to index")
	recoveryIndexCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")
	recoveryIndexCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")
	recoveryIndexCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to the public key to verify with")

	viper.AutomaticEnv()

	recoveryCmd.AddCommand(recoveryIndexCmd)
}
