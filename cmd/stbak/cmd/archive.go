package cmd

import (
	"archive/tar"
	"bufio"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/pkg/adapters"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/counters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	recordSizeFlag = "record-size"
	srcFlag        = "src"
	overwriteFlag  = "overwrite"
)

var archiveCmd = &cobra.Command{
	Use:     "archive",
	Aliases: []string{"a"},
	Short:   "Archive a directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
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
		)
	},
}

func archive(
	tape string,
	recordSize int,
	src string,
	overwrite bool,
) error {
	isRegular := true
	stat, err := os.Stat(tape)
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
		if overwrite {
			f, err = os.OpenFile(tape, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}

			if err := f.Truncate(0); err != nil {
				return err
			}
		} else {
			f, err = os.OpenFile(tape, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}
		}

		// No need to go to end manually due to `os.O_APPEND`
	} else {
		f, err = os.OpenFile(tape, os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
		if err != nil {
			return err
		}

		if overwrite {
			// Go to start of tape
			if err := controllers.SeekToRecordOnTape(f, 0); err != nil {
				return err
			}
		} else {
			// Go to end of tape
			if err := controllers.GoToEndOfTape(f); err != nil {
				return err
			}
		}
	}
	defer f.Close()

	dirty := false
	var tw *tar.Writer
	var bw *bufio.Writer
	var counter *counters.CounterWriter
	if isRegular {
		tw = tar.NewWriter(f)
	} else {
		bw = bufio.NewWriterSize(f, controllers.BlockSize*recordSize)
		counter = &counters.CounterWriter{Writer: bw, BytesRead: 0}
		tw = tar.NewWriter(counter)
	}
	defer func() {
		// Only write the trailer if we wrote to the archive
		if dirty {
			if err := tw.Close(); err != nil {
				panic(err)
			}

			if !isRegular {
				if controllers.BlockSize*recordSize-counter.BytesRead > 0 {
					// Fill the rest of the record with zeros
					if _, err := bw.Write(make([]byte, controllers.BlockSize*recordSize-counter.BytesRead)); err != nil {
						panic(err)
					}
				}

				if err := bw.Flush(); err != nil {
					panic(err)
				}
			}
		}
	}()

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

		dirty = true

		return nil
	})
}

func init() {
	archiveCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	archiveCmd.PersistentFlags().StringP(srcFlag, "s", ".", "Directory to archive")
	archiveCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the current position instead of from the end of the tape/file")

	viper.AutomaticEnv()

	rootCmd.AddCommand(archiveCmd)
}
