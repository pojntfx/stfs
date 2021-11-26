package cmd

import (
	"archive/tar"
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	recordFlag  = "record"
	blockFlag   = "block"
	dstFlag     = "dst"
	previewFlag = "preview"
)

var restoreCmd = &cobra.Command{
	Use:     "restore",
	Aliases: []string{"r"},
	Short:   "Restore a file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		f, isRegular, err := openTapeReadOnly(viper.GetString(tapeFlag))
		if err != nil {
			return err
		}
		defer f.Close()

		var tr *tar.Reader
		if isRegular {
			// Seek to record and block
			if _, err := f.Seek(int64((viper.GetInt(recordSizeFlag)*controllers.BlockSize*viper.GetInt(recordFlag))+viper.GetInt(blockFlag)*controllers.BlockSize), io.SeekStart); err != nil {
				return err
			}

			tr = tar.NewReader(f)
		} else {
			// Seek to record
			if err := controllers.SeekToRecordOnTape(f, int32(viper.GetInt(recordFlag))); err != nil {
				return err
			}

			// Seek to block
			br := bufio.NewReaderSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
			if _, err := br.Read(make([]byte, viper.GetInt(blockFlag)*controllers.BlockSize)); err != nil {
				return err
			}

			tr = tar.NewReader(br)
		}

		hdr, err := tr.Next()
		if err != nil {
			return err
		}

		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}

		if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(int64(viper.GetInt(recordFlag)), int64(viper.GetInt(blockFlag)), hdr)); err != nil {
			return err
		}

		if !viper.GetBool(previewFlag) {
			dst := viper.GetString(dstFlag)
			if dst == "" {
				dst = filepath.Base(hdr.Name)
			}

			dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}

			if _, err := io.Copy(dstFile, tr); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	restoreCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	restoreCmd.PersistentFlags().IntP(recordFlag, "r", 0, "Record to seek too")
	restoreCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too")
	restoreCmd.PersistentFlags().StringP(dstFlag, "d", "", "File to restore to (archived name by default)")
	restoreCmd.PersistentFlags().BoolP(previewFlag, "p", false, "Only read the header")

	viper.AutomaticEnv()

	rootCmd.AddCommand(restoreCmd)
}
