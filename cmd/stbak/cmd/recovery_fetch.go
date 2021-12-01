package cmd

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/andybalholm/brotli"
	"github.com/cosnicolaou/pbzip2"
	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/pierrec/lz4/v4"
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

var recoveryFetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch a file or directory from tape or tar file by record and block",
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
			viper.GetString(compressionFlag),
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
	compressionFormat string,
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

		// Don't decompress non-regular files
		if !hdr.FileInfo().Mode().IsRegular() {
			if _, err := io.Copy(dstFile, tr); err != nil {
				return err
			}

			return nil
		}

		return decompress(
			tr,
			dstFile,
			compressionFormat,
		)
	}

	return nil
}

func decompress(
	src io.Reader,
	dst io.WriteCloser,
	compressionFormat string,
) error {
	switch compressionFormat {
	case compressionFormatGZipKey:
		fallthrough
	case compressionFormatParallelGZipKey:
		var gz io.ReadCloser
		if compressionFormat == compressionFormatGZipKey {
			g, err := gzip.NewReader(src)
			if err != nil {
				return err
			}
			gz = g
		} else {
			g, err := pgzip.NewReader(src)
			if err != nil {
				return err
			}
			gz = g
		}
		defer gz.Close()

		if _, err := io.Copy(dst, gz); err != nil {
			return err
		}

		if err := dst.Close(); err != nil {
			return err
		}
	case compressionFormatLZ4Key:
		lz := lz4.NewReader(src)
		if err := lz.Apply(lz4.ConcurrencyOption(-1)); err != nil {
			return err
		}

		if _, err := io.Copy(dst, lz); err != nil {
			return err
		}

		if err := dst.Close(); err != nil {
			return err
		}
	case compressionFormatZStandardKey:
		zz, err := zstd.NewReader(src)
		if err != nil {
			return err
		}

		if _, err := io.Copy(dst, zz); err != nil {
			return err
		}

		if err := dst.Close(); err != nil {
			return err
		}
	case compressionFormatBrotliKey:
		br := brotli.NewReader(src)

		if _, err := io.Copy(dst, br); err != nil {
			return err
		}

		if err := dst.Close(); err != nil {
			return err
		}
	case compressionFormatBzip2Key:
		bz, err := bzip2.NewReader(src, nil)
		if err != nil {
			return err
		}

		if _, err := io.Copy(dst, bz); err != nil {
			return err
		}

		if err := dst.Close(); err != nil {
			return err
		}
	case compressionFormatBzip2ParallelKey:
		bz := pbzip2.NewReader(context.Background(), src)

		if _, err := io.Copy(dst, bz); err != nil {
			return err
		}

		if err := dst.Close(); err != nil {
			return err
		}
	case compressionFormatNoneKey:
		if _, err := io.Copy(dst, src); err != nil {
			return err
		}

		if err := dst.Close(); err != nil {
			return err
		}
	default:
		return errUnsupportedCompressionFormat
	}

	return nil
}

func init() {
	recoveryFetchCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	recoveryFetchCmd.PersistentFlags().IntP(recordFlag, "r", 0, "Record to seek too")
	recoveryFetchCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too")
	recoveryFetchCmd.PersistentFlags().StringP(dstFlag, "d", "", "File to restore to (archived name by default)")
	recoveryFetchCmd.PersistentFlags().BoolP(previewFlag, "p", false, "Only read the header")

	viper.AutomaticEnv()

	recoveryCmd.AddCommand(recoveryFetchCmd)
}
