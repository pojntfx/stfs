package cmd

import (
	"context"
	"log"

	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	nameFlag = "name"
)

var removeCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"r"},
	Short:   "Remove a file from tape or tar file and index",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(dbFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		hdr, err := metadataPersister.DeleteHeader(context.Background(), viper.GetString(nameFlag))
		if err != nil {
			return err
		}

		log.Println(hdr.Record, hdr.Block)

		return nil
	},
}

func init() {
	removeCmd.PersistentFlags().StringP(nameFlag, "n", "", "Name of the file to remove")

	viper.AutomaticEnv()

	rootCmd.AddCommand(removeCmd)
}
