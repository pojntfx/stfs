package operations

import (
	"archive/tar"
	"context"
	"path"
	"path/filepath"
	"strings"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/encryption"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/internal/signature"
	"github.com/pojntfx/stfs/internal/tarext"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/recovery"
)

func (o *Operations) Move(from string, to string) error {
	from, to = filepath.ToSlash(from), filepath.ToSlash(to)

	// Ignore no-op move operation
	if from == to {
		return nil
	}

	o.diskOperationLock.Lock()
	defer o.diskOperationLock.Unlock()

	writer, err := o.backend.GetWriter()
	if err != nil {
		return err
	}

	dirty := false
	tw, cleanup, err := tarext.NewTapeWriter(writer.Drive, writer.DriveIsRegular, o.pipes.RecordSize)
	if err != nil {
		return err
	}

	lastIndexedRecord, lastIndexedBlock, err := o.metadata.Metadata.GetLastIndexedRecordAndBlock(context.Background(), o.pipes.RecordSize)
	if err != nil {
		return err
	}

	headersToMove := []*config.Header{}
	dbhdr, err := o.metadata.Metadata.GetHeader(context.Background(), from)
	if err != nil {
		return err
	}
	headersToMove = append(headersToMove, dbhdr)

	// Prevent moving from relative to absolute path
	if path.IsAbs(to) && !path.IsAbs(dbhdr.Name) {
		to = strings.TrimPrefix(to, "/")
	}

	// Ignore no-op move operation
	if from == to {
		return nil
	}

	// If the header refers to a directory, get it's children
	if dbhdr.Typeflag == tar.TypeDir {
		dbhdrs, err := o.metadata.Metadata.GetHeaderChildren(context.Background(), from)
		if err != nil {
			return err
		}

		headersToMove = append(headersToMove, dbhdrs...)
	}

	// Append move headers to the tape or tar file
	hdrs := []tar.Header{}
	for _, dbhdr := range headersToMove {
		hdr, err := converters.DBHeaderToTarHeader(converters.ConfigHeaderToDBHeader(dbhdr))
		if err != nil {
			return err
		}

		hdr.Size = 0 // Don't try to seek after the record
		hdr.Name = path.Join(to, strings.TrimPrefix(strings.TrimPrefix(dbhdr.Name, "/"), strings.TrimPrefix(from, "/")))
		hdr.PAXRecords[records.STFSRecordVersion] = records.STFSRecordVersion1
		hdr.PAXRecords[records.STFSRecordAction] = records.STFSRecordActionUpdate
		hdr.PAXRecords[records.STFSRecordReplacesName] = dbhdr.Name

		hdrs = append(hdrs, *hdr)

		if o.onHeader != nil {
			dbhdr, err := converters.TarHeaderToDBHeader(-1, -1, -1, -1, hdr)
			if err != nil {
				return err
			}

			o.onHeader(&config.HeaderEvent{
				Type:    config.HeaderEventTypeMove,
				Indexed: false,
				Header:  converters.DBHeaderToConfigHeader(dbhdr),
			})
		}

		if err := signature.SignHeader(hdr, writer.DriveIsRegular, o.pipes.Signature, o.crypto.Identity); err != nil {
			return err
		}

		if err := encryption.EncryptHeader(hdr, o.pipes.Encryption, o.crypto.Recipient); err != nil {
			return err
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		dirty = true
	}

	if err := cleanup(&dirty); err != nil {
		return err
	}

	if err := o.backend.CloseWriter(); err != nil {
		return err
	}

	reader, err := o.backend.GetReader()
	if err != nil {
		return err
	}
	defer o.backend.CloseReader()

	drive, err := o.backend.GetDrive()
	if err != nil {
		return err
	}
	defer o.backend.CloseDrive()

	return recovery.Index(
		reader,
		drive,
		o.metadata,
		o.pipes,
		o.crypto,

		o.pipes.RecordSize,
		int(lastIndexedRecord),
		int(lastIndexedBlock),
		false,
		1, // Ignore the first header, which is the last header which we already indexed

		func(hdr *tar.Header, i int) error {
			if len(hdrs) <= i {
				return config.ErrTarHeaderMissing
			}

			*hdr = hdrs[i]

			return nil
		},
		func(hdr *tar.Header, isRegular bool) error {
			return nil // We sign above, no need to verify
		},

		func(hdr *config.Header) {
			o.onHeader(&config.HeaderEvent{
				Type:    config.HeaderEventTypeMove,
				Indexed: true,
				Header:  hdr,
			})
		},
	)
}
