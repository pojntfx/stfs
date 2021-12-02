package cmd

import (
	"archive/tar"
	"context"
	"io/ioutil"
	"strings"

	"github.com/pojntfx/stfs/pkg/converters"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var moveCmd = &cobra.Command{
	Use:     "move",
	Aliases: []string{"mov", "m", "mv"},
	Short:   "Move a file or directory on tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(recipientFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		pubkey := []byte{}
		if viper.GetString(encryptionFlag) != encryptionFormatNoneKey {
			p, err := ioutil.ReadFile(viper.GetString(recipientFlag))
			if err != nil {
				return err
			}

			pubkey = p
		}

		return move(
			viper.GetString(tapeFlag),
			viper.GetString(metadataFlag),
			viper.GetString(srcFlag),
			viper.GetString(dstFlag),
			viper.GetString(encryptionFlag),
			pubkey,
		)
	},
}

func move(
	tape string,
	metadata string,
	src string,
	dst string,
	encryptionFormat string,
	pubkey []byte,
) error {
	dirty := false
	tw, _, cleanup, err := openTapeWriter(tape)
	if err != nil {
		return err
	}
	defer cleanup(&dirty)

	metadataPersister := persisters.NewMetadataPersister(metadata)
	if err := metadataPersister.Open(); err != nil {
		return err
	}

	headersToMove := []*models.Header{}
	dbhdr, err := metadataPersister.GetHeader(context.Background(), src)
	if err != nil {
		return err
	}
	headersToMove = append(headersToMove, dbhdr)

	// If the header refers to a directory, get it's children
	if dbhdr.Typeflag == tar.TypeDir {
		dbhdrs, err := metadataPersister.GetHeaderChildren(context.Background(), src)
		if err != nil {
			return err
		}

		headersToMove = append(headersToMove, dbhdrs...)
	}

	// Move the headers in the index
	if err := metadataPersister.MoveHeaders(context.Background(), headersToMove, src, dst); err != nil {
		return nil
	}

	if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
		return err
	}

	// Append move headers to the tape or tar file
	for _, dbhdr := range headersToMove {
		hdr, err := converters.DBHeaderToTarHeader(dbhdr)
		if err != nil {
			return err
		}

		hdr.Size = 0 // Don't try to seek after the record
		hdr.Name = strings.TrimSuffix(dst, "/") + strings.TrimPrefix(hdr.Name, strings.TrimSuffix(src, "/"))
		hdr.PAXRecords[pax.STFSRecordVersion] = pax.STFSRecordVersion1
		hdr.PAXRecords[pax.STFSRecordAction] = pax.STFSRecordActionUpdate
		hdr.PAXRecords[pax.STFSRecordReplacesName] = dbhdr.Name

		if err := encryptHeader(hdr, encryptionFormat, pubkey); err != nil {
			return err
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		dirty = true

		if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	moveCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	moveCmd.PersistentFlags().StringP(srcFlag, "s", "", "Current path of the file or directory to move")
	moveCmd.PersistentFlags().StringP(dstFlag, "d", "", "Path to move the file or directory to")
	moveCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")

	viper.AutomaticEnv()

	rootCmd.AddCommand(moveCmd)
}
