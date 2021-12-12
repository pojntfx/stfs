package inventory

import (
	"archive/tar"
	"context"
	"regexp"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
)

func Find(
	state config.MetadataConfig,

	expression string,

	onHeader func(hdr *models.Header),
) ([]*tar.Header, error) {
	metadataPersister := persisters.NewMetadataPersister(state.Metadata)
	if err := metadataPersister.Open(); err != nil {
		return []*tar.Header{}, err
	}

	dbHdrs, err := metadataPersister.GetHeaders(context.Background())
	if err != nil {
		return []*tar.Header{}, err
	}

	headers := []*tar.Header{}
	for _, dbhdr := range dbHdrs {
		if regexp.MustCompile(expression).Match([]byte(dbhdr.Name)) {
			hdr, err := converters.DBHeaderToTarHeader(dbhdr)
			if err != nil {
				return []*tar.Header{}, err
			}

			if onHeader != nil {
				onHeader(dbhdr)
			}

			headers = append(headers, hdr)
		}
	}

	return headers, nil
}
