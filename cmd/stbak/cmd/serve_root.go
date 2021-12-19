package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"ser", "s", "srv"},
	Short:   "Serve tape or tar file and the index",
}

func init() {
	viper.AutomaticEnv()

	rootCmd.AddCommand(serveCmd)
}
