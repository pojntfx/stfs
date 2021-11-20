package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	tapeFlag = "tape"
)

var rootCmd = &cobra.Command{
	Use:   "stbak",
	Short: "Simple Tape Backup",
	Long: `Simple Tape Backup (stbak) is a CLI to interact with STFS-managed tapes or tar files.

Find more information at:
https://github.com/pojntfx/stfs`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("stbak")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	},
}

func Execute() {
	rootCmd.PersistentFlags().StringP(tapeFlag, "t", "/dev/nst0", "Tape or tar file to use")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
