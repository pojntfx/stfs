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
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	recordFlag  = "record"
	blockFlag   = "block"
	dstFlag     = "dst"
	previewFlag = "preview"
)

var recoveryRestoreCmd = &cobra.Command{
	Use:     "restore",
	Aliases: []string{"r"},
	Short:   "Restore a file or directory from tape or tar file by record and block",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		return restoreFromRecordAndBlock(
			viper.GetString(tapeFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetInt(recordFlag),
			viper.GetInt(blockFlag),
			viper.GetString(dstFlag),
			viper.GetBool(previewFlag),
			true,
		)
	},
}

func restoreFromRecordAndBlock(
	tape string,
	recordSize int,
	record int,
	block int,
	dst string,
	preview bool,
	showHeader bool,
) error {
	f, isRegular, err := openTapeReadOnly(tape)
	if err != nil {
		return err
	}
	defer f.Close()

	var tr *tar.Reader
	if isRegular {
		// Seek to record and block
		if _, err := f.Seek(int64((recordSize*controllers.BlockSize*record)+block*controllers.BlockSize), io.SeekStart); err != nil {
			return err
		}

		tr = tar.NewReader(f)
	} else {
		// Seek to record
		if err := controllers.SeekToRecordOnTape(f, int32(record)); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(f, controllers.BlockSize*recordSize)
		if _, err := br.Read(make([]byte, block*controllers.BlockSize)); err != nil {
			return err
		}

		tr = tar.NewReader(br)
	}

	hdr, err := tr.Next()
	if err != nil {
		return err
	}

	if showHeader {
		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}

		if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(int64(record), int64(block), hdr)); err != nil {
			return err
		}
	}

	if !preview {
		if dst == "" {
			dst = filepath.Base(hdr.Name)
		}

		if hdr.Typeflag == tar.TypeDir {
			return os.MkdirAll(dst, hdr.FileInfo().Mode())
		}

		dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}

		if err := dstFile.Truncate(0); err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, tr); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	recoveryRestoreCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	recoveryRestoreCmd.PersistentFlags().IntP(recordFlag, "r", 0, "Record to seek too")
	recoveryRestoreCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too")
	recoveryRestoreCmd.PersistentFlags().StringP(dstFlag, "d", "", "File to restore to (archived name by default)")
	recoveryRestoreCmd.PersistentFlags().BoolP(previewFlag, "p", false, "Only read the header")

	viper.AutomaticEnv()

	recoveryCmd.AddCommand(recoveryRestoreCmd)
}
