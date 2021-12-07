package cmd

import (
	"archive/tar"
	"bufio"
	"context"
	"os"

	"github.com/pojntfx/stfs/internal/controllers"
	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/counters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/pax"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/recovery"
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

		identity, err := keys.ParseSignerIdentity(viper.GetString(signatureFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		return delete(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(metadataFlag),
			viper.GetString(nameFlag),
			viper.GetString(encryptionFlag),
			recipient,
			viper.GetString(signatureFlag),
			identity,
		)
	},
}

func delete(
	tape string,
	recordSize int,
	metadata string,
	name string,
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

	lastIndexedRecord, lastIndexedBlock, err := metadataPersister.GetLastIndexedRecordAndBlock(context.Background(), viper.GetInt(recordSizeFlag))
	if err != nil {
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

	if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
		return err
	}

	// Append deletion hdrs to the tape or tar file
	hdrs := []*tar.Header{}
	for _, dbhdr := range headersToDelete {
		hdr, err := converters.DBHeaderToTarHeader(dbhdr)
		if err != nil {
			return err
		}

		hdr.Size = 0 // Don't try to seek after the record
		hdr.PAXRecords[pax.STFSRecordVersion] = pax.STFSRecordVersion1
		hdr.PAXRecords[pax.STFSRecordAction] = pax.STFSRecordActionDelete

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

		if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, -1, -1, hdr)); err != nil {
			return err
		}

		hdrs = append(hdrs, hdr)
	}

	return recovery.Index(
		config.StateConfig{
			Drive:    viper.GetString(driveFlag),
			Metadata: viper.GetString(metadataFlag),
		},
		config.PipeConfig{
			Compression: viper.GetString(compressionFlag),
			Encryption:  viper.GetString(encryptionFlag),
			Signature:   viper.GetString(signatureFlag),
		},
		config.CryptoConfig{
			Recipient: recipient,
			Identity:  identity,
			Password:  viper.GetString(passwordFlag),
		},

		viper.GetInt(recordSizeFlag),
		int(lastIndexedRecord),
		int(lastIndexedBlock),
		false,

		func(hdr *tar.Header, i int) error {
			// Ignore the first header, which is the last header which we already indexed
			if i == 0 {
				return nil
			}

			if len(hdrs) <= i-1 {
				return errMissingTarHeader
			}

			*hdr = *hdrs[i-1]

			return nil
		},
		func(hdr *tar.Header, isRegular bool) error {
			return nil // We sign above, no need to verify
		},
	)
}

func openTapeWriter(tape string, recordSize int, overwrite bool) (tw *tar.Writer, isRegular bool, cleanup func(dirty *bool) error, err error) {
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

		if !overwrite {
			// Go to end of tape
			if err := controllers.GoToEndOfTape(f); err != nil {
				return nil, false, nil, err
			}
		}
	}

	var bw *bufio.Writer
	var counter *counters.CounterWriter
	if isRegular {
		tw = tar.NewWriter(f)
	} else {
		bw = bufio.NewWriterSize(f, controllers.BlockSize*recordSize)
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
				if controllers.BlockSize*recordSize-counter.BytesRead > 0 {
					// Fill the rest of the record with zeros
					if _, err := bw.Write(make([]byte, controllers.BlockSize*recordSize-counter.BytesRead)); err != nil {
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
	deleteCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	deleteCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(deleteCmd)
}
