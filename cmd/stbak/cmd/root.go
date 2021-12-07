package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pojntfx/stfs/internal/compression"
	"github.com/pojntfx/stfs/internal/encryption"
	"github.com/pojntfx/stfs/internal/signature"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	driveFlag       = "drive"
	metadataFlag    = "metadata"
	verboseFlag     = "verbose"
	compressionFlag = "compression"
	encryptionFlag  = "encryption"
	signatureFlag   = "signature"
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

		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		if err := compression.CheckCompressionFormat(viper.GetString(compressionFlag)); err != nil {
			return err
		}

		if err := encryption.CheckEncryptionFormat(viper.GetString(encryptionFlag)); err != nil {
			return err
		}

		return signature.CheckSignatureFormat(viper.GetString(signatureFlag))
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
	rootCmd.PersistentFlags().StringP(compressionFlag, "c", config.NoneKey, fmt.Sprintf("Compression format to use (default %v, available are %v)", config.NoneKey, config.KnownCompressionFormats))
	rootCmd.PersistentFlags().StringP(encryptionFlag, "e", config.NoneKey, fmt.Sprintf("Encryption format to use (default %v, available are %v)", config.NoneKey, config.KnownEncryptionFormats))
	rootCmd.PersistentFlags().StringP(signatureFlag, "s", config.NoneKey, fmt.Sprintf("Signature format to use (default %v, available are %v)", config.NoneKey, config.KnownSignatureFormats))

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
