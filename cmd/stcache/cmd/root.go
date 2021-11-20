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
	rootCmd.PersistentFlags().StringP(tapeFlag, "t", "/dev/nst0", "Tape or tar file to use")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
