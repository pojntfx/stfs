package inventory

import (
	"archive/tar"
	"context"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/pkg/config"
)

func List(
	metadata config.MetadataConfig,

	name string,
	limit int,

	onHeader func(hdr *config.Header),
) ([]*tar.Header, error) {
	dbHdrs, err := metadata.Metadata.GetHeaderDirectChildren(context.Background(), name, limit)
	if err != nil {
		return []*tar.Header{}, err
	}

	headers := []*tar.Header{}
	for _, dbhdr := range dbHdrs {
		hdr, err := converters.DBHeaderToTarHeader(converters.ConfigHeaderToDBHeader(dbhdr))
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
