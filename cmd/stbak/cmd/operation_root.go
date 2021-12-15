package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var operationCmd = &cobra.Command{
	Use:   "operation",
	Short: "Perform operations on tape or tar file and the index",
}

func init() {
	viper.AutomaticEnv()

	rootCmd.AddCommand(operationCmd)
}
