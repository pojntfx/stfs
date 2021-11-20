package cmd

import (
	"fmt"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tellCmd = &cobra.Command{
	Use:     "tell",
	Aliases: []string{"t"},
	Short:   "Get the current record (tape only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		f, err := os.OpenFile(viper.GetString(tapeFlag), os.O_RDONLY, os.ModeCharDevice)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		currentRecord, err := controllers.GetCurrentRecordFromTape(f)
		if err != nil {
			panic(err)
		}

		fmt.Println(currentRecord)

		return nil
	},
}

func init() {
	tellCmd.PersistentFlags().StringP(tapeFlag, "t", "/dev/nst0", "Tape drive to get the current record from")

	viper.AutomaticEnv()

	rootCmd.AddCommand(tellCmd)
}