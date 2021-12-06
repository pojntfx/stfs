package compression

import (
	"compress/gzip"
	"context"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/cosnicolaou/pbzip2"
	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/pierrec/lz4/v4"
	"github.com/pojntfx/stfs/pkg/config"
)

func Decompress(
	src io.Reader,
	compressionFormat string,
) (io.ReadCloser, error) {
	switch compressionFormat {
	case config.CompressionFormatGZipKey:
		fallthrough
	case config.CompressionFormatParallelGZipKey:
		if compressionFormat == config.CompressionFormatGZipKey {
			return gzip.NewReader(src)
		}

		return pgzip.NewReader(src)
	case config.CompressionFormatLZ4Key:
		lz := lz4.NewReader(src)
		if err := lz.Apply(lz4.ConcurrencyOption(-1)); err != nil {
			return nil, err
		}

		return io.NopCloser(lz), nil
	case config.CompressionFormatZStandardKey:
		zz, err := zstd.NewReader(src)
		if err != nil {
			return nil, err
		}

		return io.NopCloser(zz), nil
	case config.CompressionFormatBrotliKey:
		br := brotli.NewReader(src)

		return io.NopCloser(br), nil
	case config.CompressionFormatBzip2Key:
		return bzip2.NewReader(src, nil)
	case config.CompressionFormatBzip2ParallelKey:
		bz := pbzip2.NewReader(context.Background(), src)

		return io.NopCloser(bz), nil
	case config.NoneKey:
		return io.NopCloser(src), nil
	default:
		return nil, config.ErrUnsupportedCompressionFormat
	}
}
