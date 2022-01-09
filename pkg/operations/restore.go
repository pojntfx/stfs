package operations

import (
	"archive/tar"
	"context"
	"database/sql"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/recovery"
)

func (o *Operations) Restore(
	getDst func(path string, mode fs.FileMode) (io.WriteCloser, error),
	mkdirAll func(path string, mode fs.FileMode) error,

	from string,
	to string,
	flatten bool,
) error {
	from, to = filepath.ToSlash(from), filepath.ToSlash(to)

	o.diskOperationLock.Lock()
	defer o.diskOperationLock.Unlock()

	headersToRestore := []*config.Header{}
	src := strings.TrimSuffix(from, "/")
	dbhdr, err := o.metadata.Metadata.GetHeader(context.Background(), src)
	if err != nil {
		if err == sql.ErrNoRows {
			src = src + "/"

			dbhdr, err = o.metadata.Metadata.GetHeader(context.Background(), src)
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
		dbhdrs, err := o.metadata.Metadata.GetHeaderChildren(context.Background(), src)
		if err != nil {
			return err
		}

		headersToRestore = append(headersToRestore, dbhdrs...)
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

	for _, dbhdr := range headersToRestore {
		if o.onHeader != nil {
			o.onHeader(&config.HeaderEvent{
				Type:    config.HeaderEventTypeRestore,
				Indexed: true,
				Header:  dbhdr,
			})
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

			getDst,
			mkdirAll,

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
