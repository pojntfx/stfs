package cmd

import (
	"archive/tar"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/pkg/adapters"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
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

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		lastIndexedRecord, lastIndexedBlock, err := metadataPersister.GetLastIndexedRecordAndBlock(context.Background(), viper.GetInt(recordSizeFlag))
		if err != nil {
			return err
		}

		if err := update(
			viper.GetString(tapeFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(srcFlag),
			viper.GetBool(contentFlag),
		); err != nil {
			return err
		}

		return index(
			viper.GetString(tapeFlag),
			viper.GetString(metadataFlag),
			viper.GetInt(recordSizeFlag),
			int(lastIndexedRecord),
			int(lastIndexedBlock),
			false,
		)
	},
}

func update(
	tape string,
	recordSize int,
	src string,
	replacesContent bool,
) error {
	dirty := false
	tw, isRegular, cleanup, err := openTapeWriter(tape)
	if err != nil {
		return err
	}
	defer cleanup(&dirty)

	first := true
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		link := ""
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		if err := adapters.EnhanceHeader(path, hdr); err != nil {
			return err
		}

		hdr.Name = path
		hdr.Format = tar.FormatPAX
		if hdr.PAXRecords == nil {
			hdr.PAXRecords = map[string]string{}
		}
		hdr.PAXRecords[pax.STFSRecordVersion] = pax.STFSRecordVersion1
		hdr.PAXRecords[pax.STFSRecordAction] = pax.STFSRecordActionUpdate

		if first {
			if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
				return err
			}

			first = false
		}

		if replacesContent {
			hdr.PAXRecords[pax.STFSRecordReplacesContent] = pax.STFSRecordReplacesContentTrue

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if isRegular {
				if _, err := io.Copy(tw, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(tw, file, buf); err != nil {
					return err
				}
			}
		} else {
			hdr.Size = 0 // Don't try to seek after the record

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		}

		dirty = true

		return nil
	})
}

func init() {
	updateCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	updateCmd.PersistentFlags().StringP(srcFlag, "s", "", "Path of the file or directory to update")
	updateCmd.PersistentFlags().BoolP(contentFlag, "c", false, "Replace the content on the tape/tar file")

	viper.AutomaticEnv()

	rootCmd.AddCommand(updateCmd)
}
