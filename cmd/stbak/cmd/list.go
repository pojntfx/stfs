package cmd

import (
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"lis", "l", "t", "ls"},
	Short:   "List the contents of a directory on tape or tar file ",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		if _, err := inventory.List(
			inventory.MetadataConfig{
				Metadata: viper.GetString(metadataFlag),
			},

			viper.GetString(nameFlag),
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
