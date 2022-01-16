package cmd

import (
	"fmt"

	"github.com/pojntfx/stfs/pkg/hardware"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var driveTellCmd = &cobra.Command{
	Use:   "tell",
	Short: "Get the current record on the tape",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		reader, _, err := tape.OpenTapeReadOnly(
			viper.GetString(driveFlag),
		)
		if err != nil {
			return nil
		}
		defer reader.Close()

		currentRecord, err := hardware.Tell(reader.Fd())
		if err != nil {
			return err
		}

		fmt.Println(currentRecord)

		return nil
	},
}

func init() {
	viper.AutomaticEnv()

	driveCmd.AddCommand(driveTellCmd)
}
