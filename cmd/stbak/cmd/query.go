package cmd

import (
	"archive/tar"
	"context"
	"encoding/json"

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

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(dbFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		headers, err := metadataPersister.GetHeaders(context.Background())
		if err != nil {
			return err
		}

		for i, hdr := range headers {
			if i == 0 {
				if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
					return err
				}
			}

			paxRecords := map[string]string{}
			if err := json.Unmarshal([]byte(hdr.Paxrecords), &paxRecords); err != nil {
				return err
			}

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(hdr.Record, hdr.Block, &tar.Header{
				Typeflag:   byte(hdr.Typeflag),
				Name:       hdr.Name,
				Linkname:   hdr.Linkname,
				Size:       hdr.Size,
				Mode:       hdr.Mode,
				Uid:        int(hdr.UID),
				Gid:        int(hdr.Gid),
				Uname:      hdr.Uname,
				Gname:      hdr.Gname,
				ModTime:    hdr.Modtime,
				AccessTime: hdr.Accesstime,
				ChangeTime: hdr.Changetime,
				Devmajor:   hdr.Devmajor,
				Devminor:   hdr.Devminor,
				PAXRecords: paxRecords,
				Format:     tar.Format(hdr.Format),
			})); err != nil {
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
