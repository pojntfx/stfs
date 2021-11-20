package cmd

import (
	"archive/tar"
	"bufio"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/pkg/adapters"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	tapeFlag       = "tape"
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
			if viper.GetBool(overwriteFlag) {
				f, err = os.OpenFile(viper.GetString(tapeFlag), os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					return err
				}
			} else {
				f, err = os.OpenFile(viper.GetString(tapeFlag), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					return err
				}
			}

			// No need to go to end manually due to `os.O_APPEND`
		} else {
			f, err = os.OpenFile(viper.GetString(tapeFlag), os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
			if err != nil {
				return err
			}

			if !viper.GetBool(overwriteFlag) {
				// Go to end of tape
				if err := controllers.GoToEndOfTape(f); err != nil {
					return err
				}
			}
		}
		defer f.Close()

		var tw *tar.Writer
		if isRegular {
			tw = tar.NewWriter(f)
		} else {
			bw := bufio.NewWriterSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
			tw = tar.NewWriter(bw)
		}
		defer tw.Close()

		first := true
		return filepath.Walk(viper.GetString(srcFlag), func(path string, info fs.FileInfo, err error) error {
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
				buf := make([]byte, controllers.BlockSize*viper.GetInt(recordSizeFlag))
				if _, err := io.CopyBuffer(tw, file, buf); err != nil {
					return err
				}
			}

			return nil
		})
	},
}

func init() {
	archiveCmd.PersistentFlags().StringP(tapeFlag, "t", "/dev/nst0", "Tape or tar file to write to")
	archiveCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	archiveCmd.PersistentFlags().StringP(srcFlag, "s", ".", "Directory to archive")
	archiveCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the current position instead of from the end of the tape/file")

	viper.AutomaticEnv()

	rootCmd.AddCommand(archiveCmd)
}
