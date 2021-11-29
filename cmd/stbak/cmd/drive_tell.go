package cmd

import (
	"fmt"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var driveTellCmd = &cobra.Command{
	Use:     "tell",
	Aliases: []string{"t"},
	Short:   "Get the current record on the tape",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
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
	viper.AutomaticEnv()

	driveCmd.AddCommand(driveTellCmd)
}
