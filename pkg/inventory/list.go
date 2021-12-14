package inventory

import (
	"archive/tar"
	"context"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/config"
)

func List(
	metadata config.MetadataConfig,

	name string,

	onHeader func(hdr *models.Header),
) ([]*tar.Header, error) {
	dbHdrs, err := metadata.Metadata.GetHeaderDirectChildren(context.Background(), name)
	if err != nil {
		return []*tar.Header{}, err
	}

	headers := []*tar.Header{}
	for _, dbhdr := range dbHdrs {
		hdr, err := converters.DBHeaderToTarHeader(dbhdr)
		if err != nil {
			return []*tar.Header{}, err
		}

		if onHeader != nil {
			onHeader(dbhdr)
		}

		headers = append(headers, hdr)
	}

	return headers, nil
}
