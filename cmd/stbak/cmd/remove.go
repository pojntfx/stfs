package cmd

import (
	"archive/tar"
	"context"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/converters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
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

		isRegular := true
		stat, err := os.Stat(viper.GetString(tapeFlag))
		if err == nil {
			isRegular = stat.Mode().IsRegular()
		} else {
			if os.IsNotExist(err) {
				isRegular = true
			} else {
				return err
			}
		}

		var f *os.File
		if isRegular {
			f, err = os.OpenFile(viper.GetString(tapeFlag), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}

			// No need to go to end manually due to `os.O_APPEND`
		} else {
			f, err = os.OpenFile(viper.GetString(tapeFlag), os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
			if err != nil {
				return err
			}

			// Go to end of tape
			if err := controllers.GoToEndOfTape(f); err != nil {
				return err
			}
		}
		defer f.Close()

		dirty := false
		tw := tar.NewWriter(f)
		defer func() {
			// Only write the trailer if we wrote to the archive
			if dirty {
				if err := tw.Close(); err != nil {
					panic(err)
				}
			}
		}()

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		dbhdr, err := metadataPersister.DeleteHeader(context.Background(), viper.GetString(nameFlag))
		if err != nil {
			return err
		}

		hdr, err := converters.DBHeaderToTarHeader(dbhdr)
		if err != nil {
			return err
		}

		hdr.Size = 0 // Don't try to seek after the record
		hdr.PAXRecords[pax.STFSRecordVersion] = pax.STFSRecordVersion1
		hdr.PAXRecords[pax.STFSRecordAction] = pax.STFSRecordActionDelete

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		dirty = true

		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}

		if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	removeCmd.PersistentFlags().StringP(nameFlag, "n", "", "Name of the file to remove")

	viper.AutomaticEnv()

	rootCmd.AddCommand(removeCmd)
}
