package cmd

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/recovery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	recordSizeFlag       = "record-size"
	fromFlag             = "from"
	overwriteFlag        = "overwrite"
	compressionLevelFlag = "compression-level"

	recipientFlag = "recipient"
	identityFlag  = "identity"
	passwordFlag  = "password"
)

var (
	knownCompressionLevels = []string{config.CompressionLevelFastest, config.CompressionLevelBalanced, config.CompressionLevelSmallest}

	errUnknownCompressionLevel     = errors.New("unknown compression level")
	errUnsupportedCompressionLevel = errors.New("unsupported compression level")

	errKeyNotAccessible = errors.New("key not found or accessible")

	errMissingTarHeader = errors.New("tar header is missing")

	errRecipientUnparsable = errors.New("recipient could not be parsed")

	errCompressionFormatRequiresLargerRecordSize = errors.New("this compression format requires a larger record size")

	errCompressionFormatOnlyRegularSupport = errors.New("this compression format only supports regular files, not i.e. tape drives")

	errSignatureFormatOnlyRegularSupport = errors.New("this signature format only supports regular files, not i.e. tape drives")
)

var archiveCmd = &cobra.Command{
	Use:     "archive",
	Aliases: []string{"arc", "a", "c"},
	Short:   "Archive a file or directory to tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := checkCompressionLevel(viper.GetString(compressionLevelFlag)); err != nil {
			return err
		}

		if err := checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(recipientFlag)); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(signatureFlag), viper.GetString(identityFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		lastIndexedRecord := int64(0)
		lastIndexedBlock := int64(0)
		if !viper.GetBool(overwriteFlag) {
			r, b, err := metadataPersister.GetLastIndexedRecordAndBlock(context.Background(), viper.GetInt(recordSizeFlag))
			if err != nil {
				return err
			}

			lastIndexedRecord = r
			lastIndexedBlock = b
		}

		pubkey, err := readKey(viper.GetString(encryptionFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := keys.ParseRecipient(viper.GetString(encryptionFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := readKey(viper.GetString(signatureFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := keys.ParseSignerIdentity(viper.GetString(signatureFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		hdrs, err := operations.Archive(
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
			viper.GetBool(overwriteFlag),

			func(hdr *tar.Header, i int) error {
				// Ignore the first header, which is the last header which we already indexed
				if i == 0 {
					return nil
				}

				if len(hdrs) <= i-1 {
					return errMissingTarHeader
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

func checkKeyAccessible(encryptionFormat string, pathToKey string) error {
	if encryptionFormat == noneKey {
		return nil
	}

	if _, err := os.Stat(pathToKey); err != nil {
		return errKeyNotAccessible
	}

	return nil
}

func readKey(encryptionFormat string, pathToKey string) ([]byte, error) {
	if encryptionFormat == noneKey {
		return []byte{}, nil
	}

	return ioutil.ReadFile(pathToKey)
}

func checkCompressionLevel(compressionLevel string) error {
	compressionLevelIsKnown := false

	for _, candidate := range knownCompressionLevels {
		if compressionLevel == candidate {
			compressionLevelIsKnown = true
		}
	}

	if !compressionLevelIsKnown {
		return errUnknownCompressionLevel
	}

	return nil
}

func init() {
	archiveCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	archiveCmd.PersistentFlags().StringP(fromFlag, "f", ".", "File or directory to archive")
	archiveCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the start instead of from the end of the tape or tar file")
	archiveCmd.PersistentFlags().StringP(compressionLevelFlag, "l", config.CompressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", config.CompressionLevelBalanced, knownCompressionLevels))
	archiveCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	archiveCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	archiveCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(archiveCmd)
}
