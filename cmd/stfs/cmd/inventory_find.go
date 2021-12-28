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
	expressionFlag = "expression"
)

var inventoryFindCmd = &cobra.Command{
	Use:     "find",
	Aliases: []string{"fin", "f"},
	Short:   "Find a file or directory on tape or tar file by matching against a regex",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		if _, err := inventory.Find(
			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			viper.GetString(expressionFlag),

			logging.NewCSVLogger().PrintHeader,
		); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	inventoryFindCmd.PersistentFlags().StringP(expressionFlag, "x", "", "Regex to match the file/directory name against")

	viper.AutomaticEnv()

	inventoryCmd.AddCommand(inventoryFindCmd)
}
