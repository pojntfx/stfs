package inventory

import (
	"archive/tar"
	"context"
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/pkg/config"
)

func Stat(
	metadata config.MetadataConfig,

	name string,
	symlink bool,

	onHeader func(hdr *config.Header),
) (*tar.Header, error) {
	name = filepath.ToSlash(name)
	linkname := filepath.ToSlash(name)

	if symlink {
		// Resolve symlink
		link, err := metadata.Metadata.GetHeaderByLinkname(context.Background(), name)
		if err != nil {
			if err == sql.ErrNoRows {
				link, err = metadata.Metadata.GetHeaderByLinkname(context.Background(), strings.TrimSuffix(name, "/")+"/")
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		name = link.Name
		linkname = link.Linkname
	}

	dbhdr, err := metadata.Metadata.GetHeader(context.Background(), name)
	if err != nil {
		if err == sql.ErrNoRows {
			dbhdr, err = metadata.Metadata.GetHeader(context.Background(), strings.TrimSuffix(name, "/")+"/")
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	if symlink {
		dbhdr.Name = linkname
		dbhdr.Linkname = name
	}

	hdr, err := converters.DBHeaderToTarHeader(converters.ConfigHeaderToDBHeader(dbhdr))
	if err != nil {
		return nil, err
	}

	if onHeader != nil {
		onHeader(dbhdr)
	}

	return hdr, nil
}
