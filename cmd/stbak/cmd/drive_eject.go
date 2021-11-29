package cmd

import (
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var driveEjectCmd = &cobra.Command{
	Use:     "eject",
	Aliases: []string{"e"},
	Short:   "Eject tape from drive",
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

		return controllers.EjectTape(f)
	},
}

func init() {
	viper.AutomaticEnv()

	driveCmd.AddCommand(driveEjectCmd)
}
