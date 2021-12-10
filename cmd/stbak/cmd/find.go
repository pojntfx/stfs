package cmd

import (
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	expressionFlag = "expression"
)

var findCmd = &cobra.Command{
	Use:     "find",
	Aliases: []string{"fin", "f"},
	Short:   "Find a file or directory on tape or tar file by matching against a regex",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if _, err := inventory.Find(
			inventory.MetadataConfig{
				Metadata: viper.GetString(metadataFlag),
			},

			viper.GetString(expressionFlag),

			logging.NewLogger().PrintHeader,
		); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	findCmd.PersistentFlags().StringP(expressionFlag, "x", "", "Regex to match the file/directory name against")

	viper.AutomaticEnv()

	rootCmd.AddCommand(findCmd)
}
