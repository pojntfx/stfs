package inventory

import (
	"archive/tar"
	"context"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/persisters"
)

func List(
	state MetadataConfig,

	name string,
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
	for i, dbhdr := range dbHdrs {
		if i == 0 {
			if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
				return []*tar.Header{}, err
			}
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

	return headers, nil
}
