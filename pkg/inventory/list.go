package inventory

import (
	"archive/tar"
	"context"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/persisters"
)

func List(
	state MetadataConfig,

	name string,

	onHeader func(hdr *models.Header),
) ([]*tar.Header, error) {
	metadataPersister := persisters.NewMetadataPersister(state.Metadata)
	if err := metadataPersister.Open(); err != nil {
		return []*tar.Header{}, err
	}

	dbHdrs, err := metadataPersister.GetHeaderDirectChildren(context.Background(), name)
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
