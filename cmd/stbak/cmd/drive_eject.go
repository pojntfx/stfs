package cmd

import (
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/hardware"
	"github.com/pojntfx/stfs/pkg/tape"
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

		reader, readerIsRegular, err := tape.OpenTapeReadOnly(
			viper.GetString(driveFlag),
		)
		if err != nil {
			return nil
		}
		defer reader.Close()

		return hardware.Eject(
			config.DriveConfig{
				Drive:          reader,
				DriveIsRegular: readerIsRegular,
			},
		)
	},
}

func init() {
	viper.AutomaticEnv()

	driveCmd.AddCommand(driveEjectCmd)
}
