package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var operationCmd = &cobra.Command{
	Use:     "operation",
	Aliases: []string{"ope", "o", "op"},
	Short:   "Perform operations on tape or tar file and the index",
}

func init() {
	viper.AutomaticEnv()

	rootCmd.AddCommand(operationCmd)
}
