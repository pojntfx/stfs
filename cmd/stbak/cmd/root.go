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
	tapeFlag        = "tape"
	metadataFlag    = "metadata"
	verboseFlag     = "verbose"
	compressionFlag = "compression"

	compressionFormatNoneKey = "none"

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

	encryptionFormatNoneKey = "none"

	encryptionFormatAgeKey    = "age"
	encryptionFormatAgeSuffix = ".age"

	encryptionFormatPGPKey    = "pgp"
	encryptionFormatPGPSuffix = ".pgp"
)

var (
	knownCompressionFormats = []string{compressionFormatNoneKey, compressionFormatGZipKey, compressionFormatParallelGZipKey, compressionFormatLZ4Key, compressionFormatZStandardKey, compressionFormatBrotliKey, compressionFormatBzip2Key, compressionFormatBzip2ParallelKey}

	errUnknownCompressionFormat     = errors.New("unknown compression format")
	errUnsupportedCompressionFormat = errors.New("unsupported compression format")

	knownEncryptionFormats = []string{encryptionFormatNoneKey, encryptionFormatAgeKey, encryptionFormatPGPKey}

	errUnknownEncryptionFormat              = errors.New("unknown encryption format")
	errUnsupportedEncryptionFormat          = errors.New("unsupported encryption format")
	errKeygenForEncryptionFormatUnsupported = errors.New("can not generate keys for this encryption format")
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

	rootCmd.PersistentFlags().StringP(tapeFlag, "t", "/dev/nst0", "Tape or tar file to use")
	rootCmd.PersistentFlags().StringP(metadataFlag, "m", metadataPath, "Metadata database to use")
	rootCmd.PersistentFlags().BoolP(verboseFlag, "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringP(compressionFlag, "c", compressionFormatNoneKey, fmt.Sprintf("Compression format to use (default %v, available are %v)", compressionFormatNoneKey, knownCompressionFormats))
	rootCmd.PersistentFlags().StringP(encryptionFlag, "e", encryptionFormatNoneKey, fmt.Sprintf("Encryption format to use (default %v, available are %v)", encryptionFormatNoneKey, knownEncryptionFormats))

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
