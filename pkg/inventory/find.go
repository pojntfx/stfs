package inventory

import (
	"archive/tar"
	"context"
	"regexp"

	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/config"
)

func Find(
	metadata config.MetadataConfig,

	expression string,

	onHeader func(hdr *models.Header),
) ([]*tar.Header, error) {
	dbHdrs, err := metadata.Metadata.GetHeaders(context.Background())
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
