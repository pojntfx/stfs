package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var driveCmd = &cobra.Command{
	Use:     "drive",
	Aliases: []string{"dri", "d"},
	Short:   "Manage tape drives",
}

func init() {
	viper.AutomaticEnv()

	rootCmd.AddCommand(driveCmd)
}
