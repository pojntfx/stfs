package cmd

import (
	"archive/tar"
	"bufio"
	"context"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/converters"
	"github.com/pojntfx/stfs/pkg/counters"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
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

		dirty := false
		tw, _, cleanup, err := openTapeWriter(viper.GetString(tapeFlag))
		if err != nil {
			return err
		}
		defer cleanup(&dirty)

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		headersToDelete := []*models.Header{}
		dbhdr, err := metadataPersister.GetHeader(context.Background(), viper.GetString(nameFlag))
		if err != nil {
			return err
		}
		headersToDelete = append(headersToDelete, dbhdr)

		// If the header refers to a directory, get it's children
		if dbhdr.Typeflag == tar.TypeDir {
			dbhdrs, err := metadataPersister.GetHeaderChildren(context.Background(), viper.GetString(nameFlag))
			if err != nil {
				return err
			}

			headersToDelete = append(headersToDelete, dbhdrs...)
		}

		// Remove the headers from the index
		if err := metadataPersister.DeleteHeaders(context.Background(), headersToDelete); err != nil {
			return nil
		}

		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}

		// Append deletion headers to the tape/tar file
		for _, dbhdr := range headersToDelete {
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

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
				return err
			}
		}

		return nil
	},
}

func openTapeWriter(tape string) (tw *tar.Writer, isRegular bool, cleanup func(dirty *bool) error, err error) {
	stat, err := os.Stat(tape)
	if err == nil {
		isRegular = stat.Mode().IsRegular()
	} else {
		if os.IsNotExist(err) {
			isRegular = true
		} else {
			return nil, false, nil, err
		}
	}

	var f *os.File
	if isRegular {
		f, err = os.OpenFile(tape, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, false, nil, err
		}

		// No need to go to end manually due to `os.O_APPEND`
	} else {
		f, err = os.OpenFile(tape, os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
		if err != nil {
			return nil, false, nil, err
		}

		// Go to end of tape
		if err := controllers.GoToEndOfTape(f); err != nil {
			return nil, false, nil, err
		}
	}

	var bw *bufio.Writer
	var counter *counters.CounterWriter
	if isRegular {
		tw = tar.NewWriter(f)
	} else {
		bw = bufio.NewWriterSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
		counter = &counters.CounterWriter{Writer: bw, BytesRead: 0}
		tw = tar.NewWriter(counter)
	}

	return tw, isRegular, func(dirty *bool) error {
		// Only write the trailer if we wrote to the archive
		if *dirty {
			if err := tw.Close(); err != nil {
				return err
			}

			if !isRegular {
				if controllers.BlockSize*viper.GetInt(recordSizeFlag)-counter.BytesRead > 0 {
					// Fill the rest of the record with zeros
					if _, err := bw.Write(make([]byte, controllers.BlockSize*viper.GetInt(recordSizeFlag)-counter.BytesRead)); err != nil {
						return err
					}
				}

				if err := bw.Flush(); err != nil {
					return err
				}
			}
		}

		return f.Close()
	}, nil
}

func init() {
	removeCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	removeCmd.PersistentFlags().StringP(nameFlag, "n", "", "Name of the file to remove")

	viper.AutomaticEnv()

	rootCmd.AddCommand(removeCmd)
}
