package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var inventoryCmd = &cobra.Command{
	Use:     "inventory",
	Aliases: []string{"inv", "i"},
	Short:   "Get contents and metadata of tape or tar file from the index",
}

func init() {
	viper.AutomaticEnv()

	rootCmd.AddCommand(inventoryCmd)
}
