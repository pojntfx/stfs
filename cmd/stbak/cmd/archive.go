package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"filippo.io/age"
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
	keyFlag              = "key"

	compressionLevelFastest  = "fastest"
	compressionLevelBalanced = "balanced"
	compressionLevelSmallest = "smallest"
)

var (
	knownCompressionLevels = []string{compressionLevelFastest, compressionLevelBalanced, compressionLevelSmallest}

	errUnknownCompressionLevel     = errors.New("unknown compression level")
	errUnsupportedCompressionLevel = errors.New("unsupported compression level")

	errKeyNotAccessible = errors.New("key not found or accessible")
)

type flusher interface {
	io.WriteCloser

	Flush() error
}

func nopCloserWriter(w io.Writer) nopCloser {
	return nopCloser{w}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

func nopFlusherWriter(w io.WriteCloser) nopFlusher {
	return nopFlusher{w}
}

type nopFlusher struct {
	io.WriteCloser
}

func (nopFlusher) Flush() error { return nil }

var archiveCmd = &cobra.Command{
	Use:     "archive",
	Aliases: []string{"arc", "a", "c"},
	Short:   "Archive a file or directory to tape or tar file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := checkCompressionLevel(viper.GetString(compressionLevelFlag)); err != nil {
			return err
		}

		if viper.GetString(encryptionFlag) != encryptionFormatNoneKey {
			if _, err := os.Stat(viper.GetString(keyFlag)); err != nil {
				return errKeyNotAccessible
			}
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

		pubkey := []byte{}
		if viper.GetString(encryptionFlag) != encryptionFormatNoneKey {
			p, err := ioutil.ReadFile(viper.GetString(keyFlag))
			if err != nil {
				return err
			}

			pubkey = p
		}

		if err := archive(
			viper.GetString(tapeFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(srcFlag),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
			viper.GetString(compressionLevelFlag),
			viper.GetString(encryptionFlag),
			pubkey,
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
			viper.GetString(encryptionFlag),
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
	encryptionFormat string,
	pubkey []byte,
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
			// Get the compressed size for the header
			fileSizeCounter := &counters.CounterWriter{
				Writer: io.Discard,
			}

			encryptor, err := encrypt(fileSizeCounter, encryptionFormat, pubkey)
			if err != nil {
				return err
			}

			compressor, err := compress(
				encryptor,
				compressionFormat,
				compressionLevel,
				isRegular,
				recordSize,
			)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(compressor, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(compressor, file, buf); err != nil {
					return err
				}
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := compressor.Flush(); err != nil {
				return err
			}

			if err := compressor.Close(); err != nil {
				return err
			}

			if err := encryptor.Close(); err != nil {
				return err
			}

			if hdr.PAXRecords == nil {
				hdr.PAXRecords = map[string]string{}
			}
			hdr.PAXRecords[pax.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
			hdr.Size = int64(fileSizeCounter.BytesRead)

			switch compressionFormat {
			case compressionFormatGZipKey:
				fallthrough
			case compressionFormatParallelGZipKey:
				hdr.Name += compressionFormatGZipSuffix
			case compressionFormatLZ4Key:
				hdr.Name += compressionFormatLZ4Suffix
			case compressionFormatZStandardKey:
				hdr.Name += compressionFormatZStandardSuffix
			case compressionFormatBrotliKey:
				hdr.Name += compressionFormatBrotliSuffix
			case compressionFormatBzip2Key:
				fallthrough
			case compressionFormatBzip2ParallelKey:
				hdr.Name += compressionFormatBzip2Suffix
			case compressionFormatNoneKey:
			default:
				return errUnsupportedCompressionFormat
			}

			switch encryptionFormat {
			case encryptionFormatAgeKey:
				hdr.Name += encryptionFormatAgeSuffix
			case compressionFormatNoneKey:
			default:
				return errUnsupportedEncryptionFormat
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

		// Compress and write the file
		encryptor, err := encrypt(tw, encryptionFormat, pubkey)
		if err != nil {
			return err
		}

		compressor, err := compress(
			encryptor,
			compressionFormat,
			compressionLevel,
			isRegular,
			recordSize,
		)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		if isRegular {
			if _, err := io.Copy(compressor, file); err != nil {
				return err
			}
		} else {
			buf := make([]byte, controllers.BlockSize*recordSize)
			if _, err := io.CopyBuffer(compressor, file, buf); err != nil {
				return err
			}
		}

		if err := file.Close(); err != nil {
			return err
		}

		if err := compressor.Flush(); err != nil {
			return err
		}

		if err := compressor.Close(); err != nil {
			return err
		}

		if err := encryptor.Close(); err != nil {
			return err
		}

		dirty = true

		return nil
	})
}

func checkCompressionLevel(compressionLevel string) error {
	compressionLevelIsKnown := false

	for _, candidate := range knownCompressionLevels {
		if compressionLevel == candidate {
			compressionLevelIsKnown = true
		}
	}

	if !compressionLevelIsKnown {
		return errUnknownCompressionLevel
	}

	return nil
}

func encrypt(
	dst io.Writer,
	encryptionFormat string,
	pubkey []byte,
) (io.WriteCloser, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		recipient, err := age.ParseX25519Recipient(string(pubkey))
		if err != nil {
			return nil, err
		}

		return age.Encrypt(dst, recipient)
	case encryptionFormatNoneKey:
		return nopCloserWriter(dst), nil
	default:
		return nil, errUnsupportedEncryptionFormat
	}
}

func compress(
	dst io.Writer,
	compressionFormat string,
	compressionLevel string,
	isRegular bool,
	recordSize int,
) (flusher, error) {
	switch compressionFormat {
	case compressionFormatGZipKey:
		fallthrough
	case compressionFormatParallelGZipKey:
		if compressionFormat == compressionFormatGZipKey {
			l := gzip.DefaultCompression
			switch compressionLevel {
			case compressionLevelFastest:
				l = gzip.BestSpeed
			case compressionLevelBalanced:
				l = gzip.DefaultCompression
			case compressionLevelSmallest:
				l = gzip.BestCompression
			default:
				return nil, errUnsupportedCompressionLevel
			}

			return gzip.NewWriterLevel(dst, l)
		}

		l := pgzip.DefaultCompression
		switch compressionLevel {
		case compressionLevelFastest:
			l = pgzip.BestSpeed
		case compressionLevelBalanced:
			l = pgzip.DefaultCompression
		case compressionLevelSmallest:
			l = pgzip.BestCompression
		default:
			return nil, errUnsupportedCompressionLevel
		}

		return pgzip.NewWriterLevel(dst, l)
	case compressionFormatLZ4Key:
		l := lz4.Level5
		switch compressionLevel {
		case compressionLevelFastest:
			l = lz4.Level1
		case compressionLevelBalanced:
			l = lz4.Level5
		case compressionLevelSmallest:
			l = lz4.Level9
		default:
			return nil, errUnsupportedCompressionLevel
		}

		lz := lz4.NewWriter(dst)
		if err := lz.Apply(lz4.ConcurrencyOption(-1), lz4.CompressionLevelOption(l)); err != nil {
			return nil, err
		}

		return nopFlusherWriter(lz), nil
	case compressionFormatZStandardKey:
		l := zstd.SpeedDefault
		switch compressionLevel {
		case compressionLevelFastest:
			l = zstd.SpeedFastest
		case compressionLevelBalanced:
			l = zstd.SpeedDefault
		case compressionLevelSmallest:
			l = zstd.SpeedBestCompression
		default:
			return nil, errUnsupportedCompressionLevel
		}

		zz, err := zstd.NewWriter(dst, zstd.WithEncoderLevel(l))
		if err != nil {
			return nil, err
		}

		return zz, nil
	case compressionFormatBrotliKey:
		l := brotli.DefaultCompression
		switch compressionLevel {
		case compressionLevelFastest:
			l = brotli.BestSpeed
		case compressionLevelBalanced:
			l = brotli.DefaultCompression
		case compressionLevelSmallest:
			l = brotli.BestCompression
		default:
			return nil, errUnsupportedCompressionLevel
		}

		br := brotli.NewWriterLevel(dst, l)

		return br, nil
	case compressionFormatBzip2Key:
		fallthrough
	case compressionFormatBzip2ParallelKey:
		l := bzip2.DefaultCompression
		switch compressionLevel {
		case compressionLevelFastest:
			l = bzip2.BestSpeed
		case compressionLevelBalanced:
			l = bzip2.DefaultCompression
		case compressionLevelSmallest:
			l = bzip2.BestCompression
		default:
			return nil, errUnsupportedCompressionLevel
		}

		bz, err := bzip2.NewWriter(dst, &bzip2.WriterConfig{
			Level: l,
		})
		if err != nil {
			return nil, err
		}

		return nopFlusherWriter(bz), nil
	case compressionFormatNoneKey:
		return nopFlusherWriter(nopCloserWriter(dst)), nil
	default:
		return nil, errUnsupportedCompressionFormat
	}
}

func init() {
	archiveCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	archiveCmd.PersistentFlags().StringP(srcFlag, "s", ".", "File or directory to archive")
	archiveCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the start instead of from the end of the tape or tar file")
	archiveCmd.PersistentFlags().StringP(compressionLevelFlag, "l", compressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", compressionLevelBalanced, knownCompressionLevels))
	archiveCmd.PersistentFlags().StringP(keyFlag, "k", "", "Path to public key of recipient to encrypt for")

	viper.AutomaticEnv()

	rootCmd.AddCommand(archiveCmd)
}
