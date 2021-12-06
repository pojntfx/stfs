package inventory

import (
	"archive/tar"
	"context"
	"regexp"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/persisters"
)

func Find(
	state MetadataConfig,

	expression string,
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
	first := true
	for _, dbhdr := range dbHdrs {
		if regexp.MustCompile(expression).Match([]byte(dbhdr.Name)) {
			if first {
				if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
					return []*tar.Header{}, err
				}

				first = false
			}

			hdr, err := converters.DBHeaderToTarHeader(dbhdr)
			if err != nil {
				return []*tar.Header{}, err
			}

			if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(dbhdr.Record, dbhdr.Block, hdr)); err != nil {
				return []*tar.Header{}, err
			}

			headers = append(headers, hdr)
		}
	}

	return headers, nil
}
