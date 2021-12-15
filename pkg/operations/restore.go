package operations

import (
	"archive/tar"
	"context"
	"database/sql"
	"path"
	"path/filepath"
	"strings"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/recovery"
)

func (o *Operations) Restore(from string, to string, flatten bool) error {
	o.diskOperationLock.Lock()
	defer o.diskOperationLock.Unlock()

	headersToRestore := []*models.Header{}
	src := strings.TrimSuffix(from, "/")
	dbhdr, err := o.metadataPersister.GetHeader(context.Background(), src)
	if err != nil {
		if err == sql.ErrNoRows {
			src = src + "/"

			dbhdr, err = o.metadataPersister.GetHeader(context.Background(), src)
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
		dbhdrs, err := o.metadataPersister.GetHeaderChildren(context.Background(), src)
		if err != nil {
			return err
		}

		headersToRestore = append(headersToRestore, dbhdrs...)
	}

	reader, err := o.getReader()
	if err != nil {
		return err
	}
	defer o.closeReader()

	drive, err := o.getDrive()
	if err != nil {
		return err
	}
	defer o.closeDrive()

	for _, dbhdr := range headersToRestore {
		if o.onHeader != nil {
			o.onHeader(dbhdr)
		}

		dst := dbhdr.Name
		if to != "" {
			if flatten {
				dst = to
			} else {
				dst = filepath.Join(to, strings.TrimPrefix(dst, from))

				if strings.TrimSuffix(dst, "/") == strings.TrimSuffix(to, "/") {
					dst = filepath.Join(dst, path.Base(dbhdr.Name)) // Append the name so we don't overwrite
				}
			}
		}

		if err := recovery.Fetch(
			reader,
			drive,
			o.pipes,
			o.crypto,

			o.recordSize,
			int(dbhdr.Record),
			int(dbhdr.Block),
			dst,
			false,

			nil,
		); err != nil {
			return err
		}
	}

	return nil
}
