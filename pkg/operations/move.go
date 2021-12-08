package operations

import (
	"archive/tar"
	"context"
	"strings"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/encryption"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/internal/signature"
	"github.com/pojntfx/stfs/internal/tape"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/recovery"
)

func Move(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	to string,
) error {
	dirty := false
	tw, isRegular, cleanup, err := tape.OpenTapeWriteOnly(state.Drive, recordSize, false)
	if err != nil {
		return err
	}
	defer cleanup(&dirty)

	metadataPersister := persisters.NewMetadataPersister(state.Metadata)
	if err := metadataPersister.Open(); err != nil {
		return err
	}

	lastIndexedRecord, lastIndexedBlock, err := metadataPersister.GetLastIndexedRecordAndBlock(context.Background(), recordSize)
	if err != nil {
		return err
	}

	headersToMove := []*models.Header{}
	dbhdr, err := metadataPersister.GetHeader(context.Background(), from)
	if err != nil {
		return err
	}
	headersToMove = append(headersToMove, dbhdr)

	// If the header refers to a directory, get it's children
	if dbhdr.Typeflag == tar.TypeDir {
		dbhdrs, err := metadataPersister.GetHeaderChildren(context.Background(), from)
		if err != nil {
			return err
		}

		headersToMove = append(headersToMove, dbhdrs...)
	}

	if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
		return err
	}

	// Append move headers to the tape or tar file
	hdrs := []*tar.Header{}
	for _, dbhdr := range headersToMove {
		hdr, err := converters.DBHeaderToTarHeader(dbhdr)
		if err != nil {
			return err
		}

		hdr.Size = 0 // Don't try to seek after the record
		hdr.Name = strings.TrimSuffix(to, "/") + strings.TrimPrefix(hdr.Name, strings.TrimSuffix(from, "/"))
		hdr.PAXRecords[records.STFSRecordVersion] = records.STFSRecordVersion1
		hdr.PAXRecords[records.STFSRecordAction] = records.STFSRecordActionUpdate
		hdr.PAXRecords[records.STFSRecordReplacesName] = dbhdr.Name

		if err := signature.SignHeader(hdr, isRegular, pipes.Signature, crypto.Identity); err != nil {
			return err
		}

		if err := encryption.EncryptHeader(hdr, pipes.Encryption, crypto.Recipient); err != nil {
			return err
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		dirty = true

		if err := formatting.PrintCSV(converters.TARHeaderToCSV(-1, -1, -1, -1, hdr)); err != nil {
			return err
		}

		hdrs = append(hdrs, hdr)
	}

	return recovery.Index(
		state,
		pipes,
		crypto,

		recordSize,
		int(lastIndexedRecord),
		int(lastIndexedBlock),
		false,

		func(hdr *tar.Header, i int) error {
			// Ignore the first header, which is the last header which we already indexed
			if i == 0 {
				return nil
			}

			if len(hdrs) <= i-1 {
				return config.ErrMissingTarHeader
			}

			*hdr = *hdrs[i-1]

			return nil
		},
		func(hdr *tar.Header, isRegular bool) error {
			return nil // We sign above, no need to verify
		},
	)
}
