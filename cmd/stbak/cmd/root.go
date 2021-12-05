package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	driveFlag    = "drive"
	metadataFlag = "metadata"
	verboseFlag  = "verbose"

	compressionFlag = "compression"

	noneKey = "none"

	compressionFormatGZipKey    = "gzip"
	compressionFormatGZipSuffix = ".gz"

	compressionFormatParallelGZipKey = "parallelgzip"

	compressionFormatLZ4Key    = "lz4"
	compressionFormatLZ4Suffix = ".lz4"

	compressionFormatZStandardKey    = "zstandard"
	compressionFormatZStandardSuffix = ".zst"

	compressionFormatBrotliKey    = "brotli"
	compressionFormatBrotliSuffix = ".br"

	compressionFormatBzip2Key    = "bzip2"
	compressionFormatBzip2Suffix = ".bz2"

	compressionFormatBzip2ParallelKey = "parallelbzip2"

	encryptionFlag = "encryption"

	encryptionFormatAgeKey    = "age"
	encryptionFormatAgeSuffix = ".age"

	encryptionFormatPGPKey    = "pgp"
	encryptionFormatPGPSuffix = ".pgp"

	signatureFlag = "signature"

	signatureFormatMinisignKey = "minisign"
)

var (
	knownCompressionFormats = []string{noneKey, compressionFormatGZipKey, compressionFormatParallelGZipKey, compressionFormatLZ4Key, compressionFormatZStandardKey, compressionFormatBrotliKey, compressionFormatBzip2Key, compressionFormatBzip2ParallelKey}

	errUnknownCompressionFormat     = errors.New("unknown compression format")
	errUnsupportedCompressionFormat = errors.New("unsupported compression format")

	knownEncryptionFormats = []string{noneKey, encryptionFormatAgeKey, encryptionFormatPGPKey}

	errUnknownEncryptionFormat     = errors.New("unknown encryption format")
	errUnsupportedEncryptionFormat = errors.New("unsupported encryption format")
	errKeygenForFormatUnsupported  = errors.New("can not generate keys for this format")

	knownSignatureFormats = []string{noneKey, signatureFormatMinisignKey}

	errUnknownSignatureFormat     = errors.New("unknown signature format")
	errUnsupportedSignatureFormat = errors.New("unsupported signature format")
)

var rootCmd = &cobra.Command{
	Use:   "stbak",
	Short: "Simple Tape Backup",
	Long: `Simple Tape Backup (stbak) is a CLI to interact with STFS-managed tapes, tar files and indexes.

Find more information at:
https://github.com/pojntfx/stfs`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		viper.SetEnvPrefix("stbak")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

		compressionFormatIsKnown := false
		compressionFormat := viper.GetString(compressionFlag)

		for _, candidate := range knownCompressionFormats {
			if compressionFormat == candidate {
				compressionFormatIsKnown = true
			}
		}

		if !compressionFormatIsKnown {
			return errUnknownCompressionFormat
		}

		encryptionFormatIsKnown := false
		encryptionFormat := viper.GetString(encryptionFlag)

		for _, candidate := range knownEncryptionFormats {
			if encryptionFormat == candidate {
				encryptionFormatIsKnown = true
			}
		}

		if !encryptionFormatIsKnown {
			return errUnknownEncryptionFormat
		}

		signatureFormatIsKnown := false
		signatureFormat := viper.GetString(signatureFlag)

		for _, candidate := range knownSignatureFormats {
			if signatureFormat == candidate {
				signatureFormatIsKnown = true
			}
		}

		if !signatureFormatIsKnown {
			return errUnknownSignatureFormat
		}

		return nil
	},
}

func Execute() {
	// Get default working dir
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	metadataPath := filepath.Join(home, ".local", "share", "stbak", "var", "lib", "stbak", "metadata.sqlite")

	rootCmd.PersistentFlags().StringP(driveFlag, "d", "/dev/nst0", "Tape or tar file to use")
	rootCmd.PersistentFlags().StringP(metadataFlag, "m", metadataPath, "Metadata database to use")
	rootCmd.PersistentFlags().BoolP(verboseFlag, "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringP(compressionFlag, "c", noneKey, fmt.Sprintf("Compression format to use (default %v, available are %v)", noneKey, knownCompressionFormats))
	rootCmd.PersistentFlags().StringP(encryptionFlag, "e", noneKey, fmt.Sprintf("Encryption format to use (default %v, available are %v)", noneKey, knownEncryptionFormats))
	rootCmd.PersistentFlags().StringP(signatureFlag, "s", noneKey, fmt.Sprintf("Signature format to use (default %v, available are %v)", noneKey, knownSignatureFormats))

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
