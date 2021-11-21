package cmd

import (
	"context"

	"github.com/pojntfx/stfs/pkg/converters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var queryCmd = &cobra.Command{
	Use:     "query",
	Aliases: []string{"q"},
	Short:   "Query the contents of an index",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		headers, err := metadataPersister.GetHeaders(context.Background())
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
	viper.AutomaticEnv()

	rootCmd.AddCommand(queryCmd)
}
