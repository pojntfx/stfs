package cmd

import (
	"archive/tar"
	"bufio"
	"context"
	"io/ioutil"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/converters"
	"github.com/pojntfx/stfs/pkg/counters"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	nameFlag = "name"
)

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"del", "d", "rm"},
	Short:   "Delete a file or directory from tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetString(encryptionFlag) != encryptionFormatNoneKey {
			if _, err := os.Stat(viper.GetString(recipientFlag)); err != nil {
				return errRecipientNotAccessible
			}
		}

		return nil
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

		return delete(
			viper.GetString(tapeFlag),
			viper.GetString(metadataFlag),
			viper.GetString(nameFlag),
			viper.GetString(encryptionFlag),
			pubkey,
		)
	},
}

func delete(
	tape string,
	metadata string,
	name string,
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

	headersToDelete := []*models.Header{}
	dbhdr, err := metadataPersister.GetHeader(context.Background(), name)
	if err != nil {
		return err
	}
	headersToDelete = append(headersToDelete, dbhdr)

	// If the header refers to a directory, get it's children
	if dbhdr.Typeflag == tar.TypeDir {
		dbhdrs, err := metadataPersister.GetHeaderChildren(context.Background(), name)
		if err != nil {
			return err
		}

		headersToDelete = append(headersToDelete, dbhdrs...)
	}

	// Remove the headers from the index
	if err := metadataPersister.DeleteHeaders(context.Background(), headersToDelete); err != nil {
		return nil
	}

	if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
		return err
	}

	// Append deletion headers to the tape or tar file
	for _, dbhdr := range headersToDelete {
		hdr, err := converters.DBHeaderToTarHeader(dbhdr)
		if err != nil {
			return err
		}

		hdr.Size = 0 // Don't try to seek after the record
		hdr.PAXRecords[pax.STFSRecordVersion] = pax.STFSRecordVersion1
		hdr.PAXRecords[pax.STFSRecordAction] = pax.STFSRecordActionDelete

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

func openTapeWriter(tape string) (tw *tar.Writer, isRegular bool, cleanup func(dirty *bool) error, err error) {
	stat, err := os.Stat(tape)
	if err == nil {
		isRegular = stat.Mode().IsRegular()
	} else {
		if os.IsNotExist(err) {
			isRegular = true
		} else {
			return nil, false, nil, err
		}
	}

	var f *os.File
	if isRegular {
		f, err = os.OpenFile(tape, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, false, nil, err
		}

		// No need to go to end manually due to `os.O_APPEND`
	} else {
		f, err = os.OpenFile(tape, os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
		if err != nil {
			return nil, false, nil, err
		}

		// Go to end of tape
		if err := controllers.GoToEndOfTape(f); err != nil {
			return nil, false, nil, err
		}
	}

	var bw *bufio.Writer
	var counter *counters.CounterWriter
	if isRegular {
		tw = tar.NewWriter(f)
	} else {
		bw = bufio.NewWriterSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
		counter = &counters.CounterWriter{Writer: bw, BytesRead: 0}
		tw = tar.NewWriter(counter)
	}

	return tw, isRegular, func(dirty *bool) error {
		// Only write the trailer if we wrote to the archive
		if *dirty {
			if err := tw.Close(); err != nil {
				return err
			}

			if !isRegular {
				if controllers.BlockSize*viper.GetInt(recordSizeFlag)-counter.BytesRead > 0 {
					// Fill the rest of the record with zeros
					if _, err := bw.Write(make([]byte, controllers.BlockSize*viper.GetInt(recordSizeFlag)-counter.BytesRead)); err != nil {
						return err
					}
				}

				if err := bw.Flush(); err != nil {
					return err
				}
			}
		}

		return f.Close()
	}, nil
}

func init() {
	deleteCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	deleteCmd.PersistentFlags().StringP(nameFlag, "n", "", "Name of the file to remove")
	deleteCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")

	viper.AutomaticEnv()

	rootCmd.AddCommand(deleteCmd)
}
