package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var recoveryCmd = &cobra.Command{
	Use:     "recovery",
	Aliases: []string{"rec", "r"},
	Short:   "Recover tapes or tar files",
}

func init() {
	viper.AutomaticEnv()

	rootCmd.AddCommand(recoveryCmd)
}
