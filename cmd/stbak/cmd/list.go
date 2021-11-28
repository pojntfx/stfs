package cmd

import (
	"context"

	"github.com/pojntfx/stfs/pkg/converters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List the contents of a directory in the index",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		headers, err := metadataPersister.GetHeaderDirectChildren(context.Background(), viper.GetString(nameFlag))
		if err != nil {
			return err
		}

		for i, dbhdr := range headers {
			if i == 0 {
				if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
					return err
				}
			}

			hdr, err := converters.DBHeaderToTarHeader(dbhdr)
			if err != nil {
				return err
			}

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(dbhdr.Record, dbhdr.Block, hdr)); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	listCmd.PersistentFlags().StringP(nameFlag, "n", "", "Directory to list the contents of")

	viper.AutomaticEnv()

	rootCmd.AddCommand(listCmd)
}
