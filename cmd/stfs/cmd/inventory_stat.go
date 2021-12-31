package cmd

import (
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/inventory"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var inventoryStatCmd = &cobra.Command{
	Use:     "stat",
	Aliases: []string{"sta", "s"},
	Short:   "Get information on a file or directory on tape or tar file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		if _, err := inventory.Stat(
			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			viper.GetString(nameFlag),

			logging.NewCSVLogger().PrintHeader,
		); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	inventoryStatCmd.PersistentFlags().StringP(nameFlag, "n", "", "File or directory to get info for")

	viper.AutomaticEnv()

	inventoryCmd.AddCommand(inventoryStatCmd)
}
