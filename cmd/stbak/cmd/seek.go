package cmd

import (
	"bufio"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var seekCmd = &cobra.Command{
	Use:     "seek",
	Aliases: []string{"s"},
	Short:   "Seek to a record and block (tape only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		f, err := os.OpenFile(viper.GetString(tapeFlag), os.O_RDONLY, os.ModeCharDevice)
		if err != nil {
			return err
		}
		defer f.Close()

		// Seek to record
		if err := controllers.SeekToRecordOnTape(f, int32(viper.GetInt(recordFlag))); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
		if _, err := br.Read(make([]byte, viper.GetInt(blockFlag)*controllers.BlockSize)); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	seekCmd.PersistentFlags().StringP(tapeFlag, "t", "/dev/nst0", "Tape drive to seek on")
	seekCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	seekCmd.PersistentFlags().IntP(recordFlag, "r", 0, "Record to seek too")
	seekCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too")

	viper.AutomaticEnv()

	rootCmd.AddCommand(seekCmd)
}
