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
)

var (
	knownCompressionFormats = []string{compressionFormatNoneKey, compressionFormatGZipKey, compressionFormatParallelGZipKey, compressionFormatLZ4Key}

	errUnknownCompressionFormat     = errors.New("unknown compression format")
	errUnsupportedCompressionFormat = errors.New("unsupported compression format")
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

		compressionIsKnown := false
		chosenCompression := viper.GetString(compressionFlag)

		for _, candidate := range knownCompressionFormats {
			if chosenCompression == candidate {
				compressionIsKnown = true
			}
		}

		if !compressionIsKnown {
			return errUnknownCompressionFormat
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
	rootCmd.PersistentFlags().StringP(compressionFlag, "c", compressionFormatNoneKey, fmt.Sprintf("Compression format to use (default none, available are %v)", knownCompressionFormats))

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
