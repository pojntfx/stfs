package cmd

import (
	"archive/tar"
	"context"
	"strings"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/pax"
	"github.com/pojntfx/stfs/internal/persisters"
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

		if err := checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(recipientFlag)); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(signatureFlag), viper.GetString(identityFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		pubkey, err := readKey(viper.GetString(encryptionFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := parseRecipient(viper.GetString(encryptionFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := readKey(viper.GetString(signatureFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := parseSignerIdentity(viper.GetString(signatureFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		return move(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(metadataFlag),
			viper.GetString(fromFlag),
			viper.GetString(toFlag),
			viper.GetString(encryptionFlag),
			recipient,
			viper.GetString(signatureFlag),
			identity,
		)
	},
}

func move(
	tape string,
	recordSize int,
	metadata string,
	src string,
	dst string,
	encryptionFormat string,
	recipient interface{},
	signatureFormat string,
	identity interface{},
) error {
	dirty := false
	tw, isRegular, cleanup, err := openTapeWriter(tape, recordSize, false)
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

		if err := signHeader(hdr, isRegular, signatureFormat, identity); err != nil {
			return err
		}

		if err := encryptHeader(hdr, encryptionFormat, recipient); err != nil {
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
	moveCmd.PersistentFlags().StringP(fromFlag, "f", "", "Current path of the file or directory to move")
	moveCmd.PersistentFlags().StringP(toFlag, "t", "", "Path to move the file or directory to")
	moveCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	moveCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	moveCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(moveCmd)
}
