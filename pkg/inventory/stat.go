package inventory

import (
	"archive/tar"
	"context"
	"database/sql"
	"strings"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/config"
)

func Stat(
	metadata config.MetadataConfig,

	name string,

	onHeader func(hdr *models.Header),
) (*tar.Header, error) {
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

	hdr, err := converters.DBHeaderToTarHeader(dbhdr)
	if err != nil {
		return nil, err
	}

	if onHeader != nil {
		onHeader(dbhdr)
	}

	return hdr, nil
}