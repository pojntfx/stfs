package cmd

import (
	"github.com/pojntfx/stfs/pkg/hardware"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var driveEjectCmd = &cobra.Command{
	Use:   "eject",
	Short: "Eject tape from drive",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		return hardware.Eject(
			hardware.DriveConfig{
				Drive: viper.GetString(driveFlag),
			},
		)
	},
}

func init() {
	viper.AutomaticEnv()

	driveCmd.AddCommand(driveEjectCmd)
}
