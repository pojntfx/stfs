package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/klauspost/pgzip"
	"github.com/pojntfx/stfs/pkg/adapters"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/counters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	recordSizeFlag = "record-size"
	srcFlag        = "src"
	overwriteFlag  = "overwrite"
)

type flusher interface {
	io.WriteCloser

	Flush() error
}

var archiveCmd = &cobra.Command{
	Use:     "archive",
	Aliases: []string{"arc", "a", "c"},
	Short:   "Archive a file or directory to tape or tar file",
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

		lastIndexedRecord := int64(0)
		lastIndexedBlock := int64(0)
		if !viper.GetBool(overwriteFlag) {
			r, b, err := metadataPersister.GetLastIndexedRecordAndBlock(context.Background(), viper.GetInt(recordSizeFlag))
			if err != nil {
				return err
			}

			lastIndexedRecord = r
			lastIndexedBlock = b
		}

		if err := archive(
			viper.GetString(tapeFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(srcFlag),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
		); err != nil {
			return err
		}

		return index(
			viper.GetString(tapeFlag),
			viper.GetString(metadataFlag),
			viper.GetInt(recordSizeFlag),
			int(lastIndexedRecord),
			int(lastIndexedBlock),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
		)
	},
}

func archive(
	tape string,
	recordSize int,
	src string,
	overwrite bool,
	compressionFormat string,
) error {
	dirty := false
	tw, isRegular, cleanup, err := openTapeWriter(tape)
	if err != nil {
		return err
	}

	if overwrite {
		if isRegular {
			if err := cleanup(&dirty); err != nil { // dirty will always be false here
				return err
			}

			f, err := os.OpenFile(tape, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}

			// Clear the file's content
			if err := f.Truncate(0); err != nil {
				return err
			}

			if err := f.Close(); err != nil {
				return err
			}

			tw, isRegular, cleanup, err = openTapeWriter(tape)
			if err != nil {
				return err
			}
		} else {
			if err := cleanup(&dirty); err != nil { // dirty will always be false here
				return err
			}

			f, err := os.OpenFile(tape, os.O_WRONLY, os.ModeCharDevice)
			if err != nil {
				return err
			}

			// Seek to the start of the tape
			if err := controllers.SeekToRecordOnTape(f, 0); err != nil {
				return err
			}

			if err := f.Close(); err != nil {
				return err
			}

			tw, isRegular, cleanup, err = openTapeWriter(tape)
			if err != nil {
				return err
			}
		}
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

		if info.Mode().IsRegular() {
			switch compressionFormat {
			case compressionFormatGZipKey:
				fallthrough
			case compressionFormatParallelGZipKey:
				// Get the compressed size for the header
				file, err := os.Open(path)
				if err != nil {
					return err
				}

				fileSizeCounter := counters.CounterWriter{
					Writer: io.Discard,
				}

				var gz flusher
				if compressionFormat == compressionFormatGZipKey {
					gz = gzip.NewWriter(&fileSizeCounter)
				} else {
					gz = pgzip.NewWriter(&fileSizeCounter)
				}
				if _, err := io.Copy(gz, file); err != nil {
					return err
				}

				if err := gz.Flush(); err != nil {
					return err
				}
				if err := gz.Close(); err != nil {
					return err
				}
				if err := file.Close(); err != nil {
					return err
				}

				if hdr.PAXRecords == nil {
					hdr.PAXRecords = map[string]string{}
				}
				hdr.PAXRecords[pax.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
				hdr.Size = int64(fileSizeCounter.BytesRead)

				hdr.Name += compressionFormatGZipSuffix
			case compressionFormatNoneKey:
			default:
				return errUnsupportedCompressionFormat
			}
		}

		if first {
			if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
				return err
			}

			first = false
		}

		if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
			return err
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		switch compressionFormat {
		case compressionFormatGZipKey:
			fallthrough
		case compressionFormatParallelGZipKey:
			// Compress and write the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			var gz flusher
			if compressionFormat == compressionFormatGZipKey {
				gz = gzip.NewWriter(tw)
			} else {
				gz = pgzip.NewWriter(tw)
			}
			if _, err := io.Copy(gz, file); err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(gz, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(gz, file, buf); err != nil {
					return err
				}
			}

			if err := gz.Flush(); err != nil {
				return err
			}
			if err := gz.Close(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		case compressionFormatNoneKey:
			// Write the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}

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

			if err := file.Close(); err != nil {
				return err
			}
		default:
			return errUnsupportedCompressionFormat
		}

		dirty = true

		return nil
	})
}

func init() {
	archiveCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	archiveCmd.PersistentFlags().StringP(srcFlag, "s", ".", "File or directory to archive")
	archiveCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the start instead of from the end of the tape or tar file")

	viper.AutomaticEnv()

	rootCmd.AddCommand(archiveCmd)
}
