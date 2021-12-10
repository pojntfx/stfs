package operations

import (
	"archive/tar"
	"context"
	"database/sql"
	"path"
	"path/filepath"
	"strings"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/hardware"
	"github.com/pojntfx/stfs/pkg/recovery"
)

func Restore(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	to string,
	flatten bool,

	onHeader func(hdr *models.Header),
) error {
	metadataPersister := persisters.NewMetadataPersister(state.Metadata)
	if err := metadataPersister.Open(); err != nil {
		return err
	}

	headersToRestore := []*models.Header{}
	src := strings.TrimSuffix(from, "/")
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

	for _, dbhdr := range headersToRestore {
		if onHeader != nil {
			onHeader(dbhdr)
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
			hardware.DriveConfig{
				Drive: state.Drive,
			},
			pipes,
			crypto,

			recordSize,
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
