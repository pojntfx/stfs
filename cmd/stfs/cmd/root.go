package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pojntfx/stfs/internal/check"
	"github.com/pojntfx/stfs/internal/logging"
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
	Use:   "stfs",
	Short: "Simple Tape File System",
	Long: `Simple Tape File System (STFS), a file system for tapes and tar files.

Find more information at:
https://github.com/pojntfx/stfs`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		viper.SetEnvPrefix("stfs")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if verbosity := viper.GetInt(verboseFlag); verbosity >= 4 {
			boil.DebugMode = true
			boil.DebugWriter = logging.NewJSONLoggerWriter(verbosity, "SQL Query", "query")
		}

		if err := check.CheckCompressionFormat(viper.GetString(compressionFlag)); err != nil {
			return err
		}

		if err := check.CheckEncryptionFormat(viper.GetString(encryptionFlag)); err != nil {
			return err
		}

		return check.CheckSignatureFormat(viper.GetString(signatureFlag))
	},
}

func Execute() error {
	// Get default working dir
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	metadataPath := filepath.Join(home, ".local", "share", "stfs", "var", "lib", "stfs", "metadata.sqlite")

	rootCmd.PersistentFlags().StringP(driveFlag, "d", "/dev/nst0", "Tape or tar file to use")
	rootCmd.PersistentFlags().StringP(metadataFlag, "m", metadataPath, "Metadata database to use")
	rootCmd.PersistentFlags().IntP(verboseFlag, "v", 2, fmt.Sprintf("Verbosity level (default %v, available are %v)", 2, []int{0, 1, 2, 3, 4}))
	rootCmd.PersistentFlags().StringP(compressionFlag, "c", config.NoneKey, fmt.Sprintf("Compression format to use (default %v, available are %v)", config.NoneKey, config.KnownCompressionFormats))
	rootCmd.PersistentFlags().StringP(encryptionFlag, "e", config.NoneKey, fmt.Sprintf("Encryption format to use (default %v, available are %v)", config.NoneKey, config.KnownEncryptionFormats))
	rootCmd.PersistentFlags().StringP(signatureFlag, "s", config.NoneKey, fmt.Sprintf("Signature format to use (default %v, available are %v)", config.NoneKey, config.KnownSignatureFormats))

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		return err
	}

	viper.AutomaticEnv()

	return rootCmd.Execute()
}
