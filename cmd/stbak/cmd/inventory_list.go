package cmd

import (
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	limitFlag = "limit"
)

var inventoryListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"lis", "l", "t", "ls"},
	Short:   "List the contents of a directory on tape or tar file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		if _, err := inventory.List(
			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			viper.GetString(nameFlag),
			viper.GetInt(limitFlag),

			logging.NewLogger().PrintHeader,
		); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	inventoryListCmd.PersistentFlags().StringP(nameFlag, "n", "", "Directory to list the contents of")
	inventoryListCmd.PersistentFlags().IntP(limitFlag, "l", -1, "Maximum amount of files to list (-1 lists all)")

	viper.AutomaticEnv()

	inventoryCmd.AddCommand(inventoryListCmd)
}
