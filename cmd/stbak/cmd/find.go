package cmd

import (
	"context"
	"regexp"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	expressionFlag = "expression"
)

var findCmd = &cobra.Command{
	Use:     "find",
	Aliases: []string{"fin", "f"},
	Short:   "Find a file or directory on tape or tar file by matching against a regex",
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

		headers, err := metadataPersister.GetHeaders(context.Background())
		if err != nil {
			return err
		}

		first := true
		for _, dbhdr := range headers {
			if regexp.MustCompile(viper.GetString(expressionFlag)).Match([]byte(dbhdr.Name)) {
				if first {
					if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
						return err
					}

					first = false
				}

				hdr, err := converters.DBHeaderToTarHeader(dbhdr)
				if err != nil {
					return err
				}

				if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(dbhdr.Record, dbhdr.Block, hdr)); err != nil {
					return err
				}
			}
		}

		return nil
	},
}

func init() {
	findCmd.PersistentFlags().StringP(expressionFlag, "x", "", "Regex to match the file/directory name against")

	viper.AutomaticEnv()

	rootCmd.AddCommand(findCmd)
}
