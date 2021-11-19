package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "stcache",
	Short: "Simple Tape Cache",
	Long: `Simple Tape Cache (stcache) is a CLI to interact with STFS-managed indexes of tapes or tar files.

Find more information at:
https://github.com/pojntfx/stfs`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("stcache")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
