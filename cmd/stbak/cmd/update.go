package cmd

import (
	"archive/tar"
	"context"
	"os"

	"github.com/pojntfx/stfs/pkg/adapters"
	"github.com/pojntfx/stfs/pkg/converters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	contentFlag = "content"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"u"},
	Short:   "Update a file's content and metadata",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		// dirty := false
		// tw, _, cleanup, err := openTapeWriter(viper.GetString(tapeFlag))
		// if err != nil {
		// 	return err
		// }
		// defer cleanup(&dirty)

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		stat, err := os.Stat(viper.GetString(contentFlag))
		if err != nil {
			return err
		}

		link := ""
		if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
			if link, err = os.Readlink(viper.GetString(srcFlag)); err != nil {
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(stat, link)
		if err != nil {
			return err
		}

		if err := adapters.EnhanceHeader(viper.GetString(srcFlag), hdr); err != nil {
			return err
		}

		hdr.Name = viper.GetString(srcFlag)
		hdr.Format = tar.FormatPAX

		if !viper.GetBool(contentFlag) {
			oldhdr, err := metadataPersister.GetHeader(context.Background(), viper.GetString(srcFlag))
			if err != nil {
				return err
			}

			newHdr, err := converters.TarHeaderToDBHeader(-1, -1, hdr)
			if err != nil {
				return err
			}

			// Metadata-only update; use the old record & block
			newHdr.Record = oldhdr.Record
			newHdr.Block = oldhdr.Block

			// Add the update header to the index
			if err := metadataPersister.UpdateHeaderMetadata(context.Background(), newHdr); err != nil {
				return nil
			}

			if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
				return err
			}
		}

		// TODO: Append the headers to the tape
		// Append update headers to the tape/tar file
		// hdr, err := converters.DBHeaderToTarHeader(dbhdr)
		// if err != nil {
		// 	return err
		// }

		// hdr.Size = 0 // Don't try to seek after the record
		// hdr.Name = strings.TrimSuffix(viper.GetString(dstFlag), "/") + strings.TrimPrefix(hdr.Name, strings.TrimSuffix(viper.GetString(srcFlag), "/"))
		// hdr.PAXRecords[pax.STFSRecordVersion] = pax.STFSRecordVersion1
		// hdr.PAXRecords[pax.STFSRecordAction] = pax.STFSRecordActionUpdate
		// hdr.PAXRecords[pax.STFSRecordReplacesName] = dbhdr.Name

		// if err := tw.WriteHeader(hdr); err != nil {
		// 	return err
		// }

		// dirty = true

		// if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
		// 	return err
		// }

		return nil
	},
}

func init() {
	updateCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	updateCmd.PersistentFlags().StringP(srcFlag, "s", "", "Path of the file or directory to update")
	updateCmd.PersistentFlags().BoolP(contentFlag, "c", false, "Replace the content on the tape/tar file")

	viper.AutomaticEnv()

	rootCmd.AddCommand(updateCmd)
}
