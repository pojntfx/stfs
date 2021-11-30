package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/andybalholm/brotli"
	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/pierrec/lz4/v4"
	"github.com/pojntfx/stfs/pkg/adapters"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/counters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	recordSizeFlag       = "record-size"
	srcFlag              = "src"
	overwriteFlag        = "overwrite"
	compressionLevelFlag = "compression-level"

	compressionLevelFastest  = "fastest"
	compressionLevelBalanced = "balanced"
	compressionLevelSmallest = "smallest"
)

var (
	knownCompressionLevels = []string{compressionLevelFastest, compressionLevelBalanced, compressionLevelSmallest}

	errUnknownCompressionLevel = errors.New("unknown compression level")
)

type flusher interface {
	io.WriteCloser

	Flush() error
}

var archiveCmd = &cobra.Command{
	Use:     "archive",
	Aliases: []string{"arc", "a", "c"},
	Short:   "Archive a file or directory to tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		compressionLevelIsKnown := false
		compressionLevel := viper.GetString(compressionLevelFlag)

		for _, candidate := range knownCompressionLevels {
			if compressionLevel == candidate {
				compressionLevelIsKnown = true
			}
		}

		if !compressionLevelIsKnown {
			return errUnknownCompressionLevel
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		lastIndexedRecord := int64(0)
		lastIndexedBlock := int64(0)
		if !viper.GetBool(overwriteFlag) {
			r, b, err := metadataPersister.GetLastIndexedRecordAndBlock(context.Background(), viper.GetInt(recordSizeFlag))
			if err != nil {
				return err
			}

			lastIndexedRecord = r
			lastIndexedBlock = b
		}

		if err := archive(
			viper.GetString(tapeFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(srcFlag),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
			viper.GetString(compressionLevelFlag),
		); err != nil {
			return err
		}

		return index(
			viper.GetString(tapeFlag),
			viper.GetString(metadataFlag),
			viper.GetInt(recordSizeFlag),
			int(lastIndexedRecord),
			int(lastIndexedBlock),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
		)
	},
}

func archive(
	tape string,
	recordSize int,
	src string,
	overwrite bool,
	compressionFormat string,
	compressionLevel string,
) error {
	dirty := false
	tw, isRegular, cleanup, err := openTapeWriter(tape)
	if err != nil {
		return err
	}

	if overwrite {
		if isRegular {
			if err := cleanup(&dirty); err != nil { // dirty will always be false here
				return err
			}

			f, err := os.OpenFile(tape, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}

			// Clear the file's content
			if err := f.Truncate(0); err != nil {
				return err
			}

			if err := f.Close(); err != nil {
				return err
			}

			tw, isRegular, cleanup, err = openTapeWriter(tape)
			if err != nil {
				return err
			}
		} else {
			if err := cleanup(&dirty); err != nil { // dirty will always be false here
				return err
			}

			f, err := os.OpenFile(tape, os.O_WRONLY, os.ModeCharDevice)
			if err != nil {
				return err
			}

			// Seek to the start of the tape
			if err := controllers.SeekToRecordOnTape(f, 0); err != nil {
				return err
			}

			if err := f.Close(); err != nil {
				return err
			}

			tw, isRegular, cleanup, err = openTapeWriter(tape)
			if err != nil {
				return err
			}
		}
	}

	defer cleanup(&dirty)

	first := true
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		link := ""
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		if err := adapters.EnhanceHeader(path, hdr); err != nil {
			return err
		}

		hdr.Name = path
		hdr.Format = tar.FormatPAX

		if info.Mode().IsRegular() {
			switch compressionFormat {
			case compressionFormatGZipKey:
				fallthrough
			case compressionFormatParallelGZipKey:
				// Get the compressed size for the header
				file, err := os.Open(path)
				if err != nil {
					return err
				}

				fileSizeCounter := counters.CounterWriter{
					Writer: io.Discard,
				}

				var gz flusher
				if compressionFormat == compressionFormatGZipKey {
					l := gzip.DefaultCompression
					switch compressionLevel {
					case compressionLevelFastest:
						l = gzip.BestSpeed
					case compressionLevelBalanced:
						l = gzip.DefaultCompression
					case compressionLevelSmallest:
						l = gzip.BestCompression
					}

					gz, err = gzip.NewWriterLevel(&fileSizeCounter, l)
					if err != nil {
						return err
					}
				} else {
					l := pgzip.DefaultCompression
					switch compressionLevel {
					case compressionLevelFastest:
						l = pgzip.BestSpeed
					case compressionLevelBalanced:
						l = pgzip.DefaultCompression
					case compressionLevelSmallest:
						l = pgzip.BestCompression
					}

					gz, err = pgzip.NewWriterLevel(&fileSizeCounter, l)
					if err != nil {
						return err
					}
				}
				if _, err := io.Copy(gz, file); err != nil {
					return err
				}

				if err := gz.Flush(); err != nil {
					return err
				}
				if err := gz.Close(); err != nil {
					return err
				}
				if err := file.Close(); err != nil {
					return err
				}

				if hdr.PAXRecords == nil {
					hdr.PAXRecords = map[string]string{}
				}
				hdr.PAXRecords[pax.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
				hdr.Size = int64(fileSizeCounter.BytesRead)

				hdr.Name += compressionFormatGZipSuffix
			case compressionFormatLZ4Key:
				// Get the compressed size for the header
				file, err := os.Open(path)
				if err != nil {
					return err
				}

				fileSizeCounter := counters.CounterWriter{
					Writer: io.Discard,
				}

				l := lz4.Level5
				switch compressionLevel {
				case compressionLevelFastest:
					l = lz4.Level1
				case compressionLevelBalanced:
					l = lz4.Level5
				case compressionLevelSmallest:
					l = lz4.Level9
				}

				lz := lz4.NewWriter(&fileSizeCounter)
				if err := lz.Apply(lz4.ConcurrencyOption(-1), lz4.CompressionLevelOption(l)); err != nil {
					return err
				}

				if _, err := io.Copy(lz, file); err != nil {
					return err
				}

				if err := lz.Close(); err != nil {
					return err
				}
				if err := file.Close(); err != nil {
					return err
				}

				if hdr.PAXRecords == nil {
					hdr.PAXRecords = map[string]string{}
				}
				hdr.PAXRecords[pax.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
				hdr.Size = int64(fileSizeCounter.BytesRead)

				hdr.Name += compressionFormatLZ4Suffix
			case compressionFormatZStandardKey:
				// Get the compressed size for the header
				file, err := os.Open(path)
				if err != nil {
					return err
				}

				fileSizeCounter := counters.CounterWriter{
					Writer: io.Discard,
				}

				l := zstd.SpeedDefault
				switch compressionLevel {
				case compressionLevelFastest:
					l = zstd.SpeedFastest
				case compressionLevelBalanced:
					l = zstd.SpeedDefault
				case compressionLevelSmallest:
					l = zstd.SpeedBestCompression
				}

				zz, err := zstd.NewWriter(&fileSizeCounter, zstd.WithEncoderLevel(l))
				if err != nil {
					return err
				}

				if _, err := io.Copy(zz, file); err != nil {
					return err
				}

				if err := zz.Flush(); err != nil {
					return err
				}
				if err := zz.Close(); err != nil {
					return err
				}
				if err := file.Close(); err != nil {
					return err
				}

				if hdr.PAXRecords == nil {
					hdr.PAXRecords = map[string]string{}
				}
				hdr.PAXRecords[pax.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
				hdr.Size = int64(fileSizeCounter.BytesRead)

				hdr.Name += compressionFormatZStandardSuffix
			case compressionFormatBrotliKey:
				// Get the compressed size for the header
				file, err := os.Open(path)
				if err != nil {
					return err
				}

				fileSizeCounter := counters.CounterWriter{
					Writer: io.Discard,
				}

				l := brotli.DefaultCompression
				switch compressionLevel {
				case compressionLevelFastest:
					l = brotli.BestSpeed
				case compressionLevelBalanced:
					l = brotli.DefaultCompression
				case compressionLevelSmallest:
					l = brotli.BestCompression
				}

				br := brotli.NewWriterLevel(&fileSizeCounter, l)

				if _, err := io.Copy(br, file); err != nil {
					return err
				}

				if err := br.Flush(); err != nil {
					return err
				}
				if err := br.Close(); err != nil {
					return err
				}
				if err := file.Close(); err != nil {
					return err
				}

				if hdr.PAXRecords == nil {
					hdr.PAXRecords = map[string]string{}
				}
				hdr.PAXRecords[pax.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
				hdr.Size = int64(fileSizeCounter.BytesRead)

				hdr.Name += compressionFormatBrotliSuffix
			case compressionFormatBzip2Key:
				fallthrough
			case compressionFormatBzip2ParallelKey:
				// Get the compressed size for the header
				file, err := os.Open(path)
				if err != nil {
					return err
				}

				fileSizeCounter := counters.CounterWriter{
					Writer: io.Discard,
				}

				l := bzip2.DefaultCompression
				switch compressionLevel {
				case compressionLevelFastest:
					l = bzip2.BestSpeed
				case compressionLevelBalanced:
					l = bzip2.DefaultCompression
				case compressionLevelSmallest:
					l = bzip2.BestCompression
				}

				bz, err := bzip2.NewWriter(&fileSizeCounter, &bzip2.WriterConfig{
					Level: l,
				})
				if err != nil {
					return err
				}

				if _, err := io.Copy(bz, file); err != nil {
					return err
				}

				if err := bz.Close(); err != nil {
					return err
				}
				if err := file.Close(); err != nil {
					return err
				}

				if hdr.PAXRecords == nil {
					hdr.PAXRecords = map[string]string{}
				}
				hdr.PAXRecords[pax.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
				hdr.Size = int64(fileSizeCounter.BytesRead)

				hdr.Name += compressionFormatBzip2Suffix
			case compressionFormatNoneKey:
			default:
				return errUnsupportedCompressionFormat
			}
		}

		if first {
			if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
				return err
			}

			first = false
		}

		if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(-1, -1, hdr)); err != nil {
			return err
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		switch compressionFormat {
		case compressionFormatGZipKey:
			fallthrough
		case compressionFormatParallelGZipKey:
			// Compress and write the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			var gz flusher
			if compressionFormat == compressionFormatGZipKey {
				l := gzip.DefaultCompression
				switch compressionLevel {
				case compressionLevelFastest:
					l = gzip.BestSpeed
				case compressionLevelBalanced:
					l = gzip.DefaultCompression
				case compressionLevelSmallest:
					l = gzip.BestCompression
				}

				gz, err = gzip.NewWriterLevel(tw, l)
				if err != nil {
					return err
				}
			} else {
				l := pgzip.DefaultCompression
				switch compressionLevel {
				case compressionLevelFastest:
					l = pgzip.BestSpeed
				case compressionLevelBalanced:
					l = pgzip.DefaultCompression
				case compressionLevelSmallest:
					l = pgzip.BestCompression
				}

				gz, err = pgzip.NewWriterLevel(tw, l)
				if err != nil {
					return err
				}
			}

			if _, err := io.Copy(gz, file); err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(gz, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(gz, file, buf); err != nil {
					return err
				}
			}

			if err := gz.Flush(); err != nil {
				return err
			}
			if err := gz.Close(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		case compressionFormatLZ4Key:
			// Compress and write the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			l := lz4.Level5
			switch compressionLevel {
			case compressionLevelFastest:
				l = lz4.Level1
			case compressionLevelBalanced:
				l = lz4.Level5
			case compressionLevelSmallest:
				l = lz4.Level9
			}

			lz := lz4.NewWriter(tw)
			if err := lz.Apply(lz4.ConcurrencyOption(-1), lz4.CompressionLevelOption(l)); err != nil {
				return err
			}

			if _, err := io.Copy(lz, file); err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(lz, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(lz, file, buf); err != nil {
					return err
				}
			}

			if err := lz.Close(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		case compressionFormatZStandardKey:
			// Compress and write the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			l := zstd.SpeedDefault
			switch compressionLevel {
			case compressionLevelFastest:
				l = zstd.SpeedFastest
			case compressionLevelBalanced:
				l = zstd.SpeedDefault
			case compressionLevelSmallest:
				l = zstd.SpeedBestCompression
			}

			zz, err := zstd.NewWriter(tw, zstd.WithEncoderLevel(l))
			if err != nil {
				return err
			}

			if _, err := io.Copy(zz, file); err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(zz, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(zz, file, buf); err != nil {
					return err
				}
			}

			if err := zz.Flush(); err != nil {
				return err
			}
			if err := zz.Close(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		case compressionFormatBrotliKey:
			// Compress and write the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			l := brotli.DefaultCompression
			switch compressionLevel {
			case compressionLevelFastest:
				l = brotli.BestSpeed
			case compressionLevelBalanced:
				l = brotli.DefaultCompression
			case compressionLevelSmallest:
				l = brotli.BestCompression
			}

			br := brotli.NewWriterLevel(tw, l)

			if _, err := io.Copy(br, file); err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(br, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(br, file, buf); err != nil {
					return err
				}
			}

			if err := br.Flush(); err != nil {
				return err
			}
			if err := br.Close(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		case compressionFormatBzip2Key:
			fallthrough
		case compressionFormatBzip2ParallelKey:
			// Compress and write the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			l := bzip2.DefaultCompression
			switch compressionLevel {
			case compressionLevelFastest:
				l = bzip2.BestSpeed
			case compressionLevelBalanced:
				l = bzip2.DefaultCompression
			case compressionLevelSmallest:
				l = bzip2.BestCompression
			}

			bz, err := bzip2.NewWriter(tw, &bzip2.WriterConfig{
				Level: l,
			})
			if err != nil {
				return err
			}

			if _, err := io.Copy(bz, file); err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(bz, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(bz, file, buf); err != nil {
					return err
				}
			}

			if err := bz.Close(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		case compressionFormatNoneKey:
			// Write the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(tw, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(tw, file, buf); err != nil {
					return err
				}
			}

			if err := file.Close(); err != nil {
				return err
			}
		default:
			return errUnsupportedCompressionFormat
		}

		dirty = true

		return nil
	})
}

func init() {
	archiveCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	archiveCmd.PersistentFlags().StringP(srcFlag, "s", ".", "File or directory to archive")
	archiveCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the start instead of from the end of the tape or tar file")
	archiveCmd.PersistentFlags().StringP(compressionLevelFlag, "l", compressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", compressionLevelBalanced, knownCompressionLevels))

	viper.AutomaticEnv()

	rootCmd.AddCommand(archiveCmd)
}
