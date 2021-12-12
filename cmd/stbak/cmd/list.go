package cmd

import (
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"lis", "l", "t", "ls"},
	Short:   "List the contents of a directory on tape or tar file ",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if _, err := inventory.List(
			config.MetadataConfig{
				Metadata: viper.GetString(metadataFlag),
			},

			viper.GetString(nameFlag),

			logging.NewLogger().PrintHeader,
		); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	listCmd.PersistentFlags().StringP(nameFlag, "n", "", "Directory to list the contents of")

	viper.AutomaticEnv()

	rootCmd.AddCommand(listCmd)
}
