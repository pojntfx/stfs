package cmd

import (
	"archive/tar"
	"context"
	"database/sql"
	"path"
	"path/filepath"
	"strings"

	"github.com/pojntfx/stfs/pkg/converters"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	flattenFlag = "flatten"
)

var restoreCmd = &cobra.Command{
	Use:     "restore",
	Aliases: []string{"res", "r", "x"},
	Short:   "Restore a file or directory from tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
	},
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

		privkey, err := readKey(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := parseIdentity(viper.GetString(encryptionFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		headersToRestore := []*models.Header{}
		src := strings.TrimSuffix(viper.GetString(fromFlag), "/")
		dbhdr, err := metadataPersister.GetHeader(context.Background(), src)
		if err != nil {
			if err == sql.ErrNoRows {
				src = src + "/"

				dbhdr, err = metadataPersister.GetHeader(context.Background(), src)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
		headersToRestore = append(headersToRestore, dbhdr)

		// If the header refers to a directory, get it's children
		if dbhdr.Typeflag == tar.TypeDir {
			dbhdrs, err := metadataPersister.GetHeaderChildren(context.Background(), src)
			if err != nil {
				return err
			}

			headersToRestore = append(headersToRestore, dbhdrs...)
		}

		for i, dbhdr := range headersToRestore {
			if i == 0 {
				if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
					return err
				}
			}

			hdr, err := converters.DBHeaderToTarHeader(dbhdr)
			if err != nil {
				return err
			}

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(dbhdr.Record, dbhdr.Block, hdr)); err != nil {
				return err
			}

			dst := dbhdr.Name
			if viper.GetString(toFlag) != "" {
				if viper.GetBool(flattenFlag) {
					dst = viper.GetString(toFlag)
				} else {
					dst = filepath.Join(viper.GetString(toFlag), strings.TrimPrefix(dst, viper.GetString(fromFlag)))

					if strings.TrimSuffix(dst, "/") == strings.TrimSuffix(viper.GetString(toFlag), "/") {
						dst = filepath.Join(dst, path.Base(dbhdr.Name)) // Append the name so we don't overwrite
					}
				}
			}

			if err := restoreFromRecordAndBlock(
				viper.GetString(driveFlag),
				viper.GetInt(recordSizeFlag),
				int(dbhdr.Record),
				int(dbhdr.Block),
				dst,
				false,
				false,
				viper.GetString(compressionFlag),
				viper.GetString(encryptionFlag),
				identity,
			); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	restoreCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	restoreCmd.PersistentFlags().StringP(fromFlag, "f", "", "File or directory to restore")
	restoreCmd.PersistentFlags().StringP(toFlag, "t", "", "File or directory restore to (archived name by default)")
	restoreCmd.PersistentFlags().BoolP(flattenFlag, "a", false, "Ignore the folder hierarchy on the tape or tar file")
	restoreCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")
	restoreCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(restoreCmd)
}
