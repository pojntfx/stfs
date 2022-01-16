package compression

import (
	"compress/gzip"
	"io"
	"math"

	"github.com/andybalholm/brotli"
	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/pierrec/lz4/v4"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/pkg/config"
)

func Compress(
	dst io.Writer,
	compressionFormat string,
	compressionLevel string,
	isRegular bool,
	recordSize int,
) (ioext.FlusherWriter, error) {
	switch compressionFormat {
	case config.CompressionFormatGZipKey:
		fallthrough
	case config.CompressionFormatParallelGZipKey:
		if compressionFormat == config.CompressionFormatGZipKey {
			if !isRegular {
				maxSize := getNearestPowerOf2Lower(config.MagneticTapeBlockSize * recordSize)

				if maxSize < 65535 { // See https://www.daylight.com/meetings/mug00/Sayle/gzip.html#:~:text=Stored%20blocks%20are%20allowed%20to,size%20of%20the%20gzip%20header.
					return nil, config.ErrCompressionFormatRequiresLargerRecordSize
				}
			}

			l := gzip.DefaultCompression
			switch compressionLevel {
			case config.CompressionLevelFastestKey:
				l = gzip.BestSpeed
			case config.CompressionLevelBalancedKey:
				l = gzip.DefaultCompression
			case config.CompressionLevelSmallestKey:
				l = gzip.BestCompression
			default:
				return nil, config.ErrCompressionLevelUnsupported
			}

			return gzip.NewWriterLevel(dst, l)
		}

		if !isRegular {
			return nil, config.ErrCompressionFormatRegularOnly // "device or resource busy"
		}

		l := pgzip.DefaultCompression
		switch compressionLevel {
		case config.CompressionLevelFastestKey:
			l = pgzip.BestSpeed
		case config.CompressionLevelBalancedKey:
			l = pgzip.DefaultCompression
		case config.CompressionLevelSmallestKey:
			l = pgzip.BestCompression
		default:
			return nil, config.ErrCompressionLevelUnsupported
		}

		return pgzip.NewWriterLevel(dst, l)
	case config.CompressionFormatLZ4Key:
		l := lz4.Level5
		switch compressionLevel {
		case config.CompressionLevelFastestKey:
			l = lz4.Level1
		case config.CompressionLevelBalancedKey:
			l = lz4.Level5
		case config.CompressionLevelSmallestKey:
			l = lz4.Level9
		default:
			return nil, config.ErrCompressionLevelUnsupported
		}

		opts := []lz4.Option{lz4.CompressionLevelOption(l), lz4.ConcurrencyOption(-1)}
		if !isRegular {
			maxSize := getNearestPowerOf2Lower(config.MagneticTapeBlockSize * recordSize)

			if uint32(maxSize) < uint32(lz4.Block64Kb) {
				return nil, config.ErrCompressionFormatRequiresLargerRecordSize
			}

			if uint32(maxSize) < uint32(lz4.Block256Kb) {
				opts = append(opts, lz4.BlockSizeOption(lz4.Block64Kb))
			} else if uint32(maxSize) < uint32(lz4.Block1Mb) {
				opts = append(opts, lz4.BlockSizeOption(lz4.Block256Kb))
			} else if uint32(maxSize) < uint32(lz4.Block4Mb) {
				opts = append(opts, lz4.BlockSizeOption(lz4.Block1Mb))
			} else {
				opts = append(opts, lz4.BlockSizeOption(lz4.Block4Mb))
			}
		}

		lz := lz4.NewWriter(dst)
		if err := lz.Apply(opts...); err != nil {
			return nil, err
		}

		return ioext.AddFlushNop(lz), nil
	case config.CompressionFormatZStandardKey:
		l := zstd.SpeedDefault
		switch compressionLevel {
		case config.CompressionLevelFastestKey:
			l = zstd.SpeedFastest
		case config.CompressionLevelBalancedKey:
			l = zstd.SpeedDefault
		case config.CompressionLevelSmallestKey:
			l = zstd.SpeedBestCompression
		default:
			return nil, config.ErrCompressionLevelUnsupported
		}

		opts := []zstd.EOption{zstd.WithEncoderLevel(l)}
		if !isRegular {
			opts = append(opts, zstd.WithWindowSize(getNearestPowerOf2Lower(config.MagneticTapeBlockSize*recordSize)))
		}

		zz, err := zstd.NewWriter(dst, opts...)
		if err != nil {
			return nil, err
		}

		return zz, nil
	case config.CompressionFormatBrotliKey:
		if !isRegular {
			return nil, config.ErrCompressionFormatRegularOnly // "cannot allocate memory"
		}

		l := brotli.DefaultCompression
		switch compressionLevel {
		case config.CompressionLevelFastestKey:
			l = brotli.BestSpeed
		case config.CompressionLevelBalancedKey:
			l = brotli.DefaultCompression
		case config.CompressionLevelSmallestKey:
			l = brotli.BestCompression
		default:
			return nil, config.ErrCompressionLevelUnsupported
		}

		br := brotli.NewWriterLevel(dst, l)

		return br, nil
	case config.CompressionFormatBzip2Key:
		fallthrough
	case config.CompressionFormatBzip2ParallelKey:
		l := bzip2.DefaultCompression
		switch compressionLevel {
		case config.CompressionLevelFastestKey:
			l = bzip2.BestSpeed
		case config.CompressionLevelBalancedKey:
			l = bzip2.DefaultCompression
		case config.CompressionLevelSmallestKey:
			l = bzip2.BestCompression
		default:
			return nil, config.ErrCompressionLevelUnsupported
		}

		bz, err := bzip2.NewWriter(dst, &bzip2.WriterConfig{
			Level: l,
		})
		if err != nil {
			return nil, err
		}

		return ioext.AddFlushNop(bz), nil
	case config.NoneKey:
		return ioext.AddFlushNop(ioext.AddCloseNopToWriter(dst)), nil
	default:
		return nil, config.ErrCompressionFormatUnsupported
	}
}

func getNearestPowerOf2Lower(n int) int {
	return int(math.Pow(2, float64(getNearestLogOf2Lower(n)))) // Truncation is intentional, see https://www.geeksforgeeks.org/highest-power-2-less-equal-given-number/
}

func getNearestLogOf2Lower(n int) int {
	return int(math.Log2(float64(n))) // Truncation is intentional, see https://www.geeksforgeeks.org/highest-power-2-less-equal-given-number/
}
