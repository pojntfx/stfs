package cmd

import (
	"archive/tar"
	"context"
	"strings"

	"github.com/pojntfx/stfs/pkg/converters"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var moveCmd = &cobra.Command{
	Use:     "move",
	Aliases: []string{"m"},
	Short:   "Move a file or directory on tape or tar file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
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

		headersToMove := []*models.Header{}
		dbhdr, err := metadataPersister.GetHeader(context.Background(), viper.GetString(srcFlag))
		if err != nil {
			return err
		}
		headersToMove = append(headersToMove, dbhdr)

		// If the header refers to a directory, get it's children
		if dbhdr.Typeflag == tar.TypeDir {
			dbhdrs, err := metadataPersister.GetHeaderChildren(context.Background(), viper.GetString(srcFlag))
			if err != nil {
				return err
			}

			headersToMove = append(headersToMove, dbhdrs...)
		}

		// Move the headers in the index
		if err := metadataPersister.MoveHeaders(context.Background(), headersToMove, viper.GetString(srcFlag), viper.GetString(dstFlag)); err != nil {
			return nil
		}

		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}

		// Append move headers to the tape or tar file
		for _, dbhdr := range headersToMove {
			hdr, err := converters.DBHeaderToTarHeader(dbhdr)
			if err != nil {
				return err
			}

			hdr.Size = 0 // Don't try to seek after the record
			hdr.Name = strings.TrimSuffix(viper.GetString(dstFlag), "/") + strings.TrimPrefix(hdr.Name, strings.TrimSuffix(viper.GetString(srcFlag), "/"))
			hdr.PAXRecords[pax.STFSRecordVersion] = pax.STFSRecordVersion1
			hdr.PAXRecords[pax.STFSRecordAction] = pax.STFSRecordActionUpdate
			hdr.PAXRecords[pax.STFSRecordReplacesName] = dbhdr.Name

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

func init() {
	moveCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	moveCmd.PersistentFlags().StringP(srcFlag, "s", "", "Current path of the file or directory to move")
	moveCmd.PersistentFlags().StringP(dstFlag, "d", "", "Path to move the file or directory to")

	viper.AutomaticEnv()

	rootCmd.AddCommand(moveCmd)
}
