package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"aead.dev/minisign"
	"filippo.io/age"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/andybalholm/brotli"
	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/pierrec/lz4/v4"
	"github.com/pojntfx/stfs/pkg/adapters"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/counters"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/noop"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	recordSizeFlag       = "record-size"
	fromFlag             = "from"
	overwriteFlag        = "overwrite"
	compressionLevelFlag = "compression-level"

	recipientFlag = "recipient"
	identityFlag  = "identity"
	passwordFlag  = "password"

	compressionLevelFastest  = "fastest"
	compressionLevelBalanced = "balanced"
	compressionLevelSmallest = "smallest"
)

var (
	knownCompressionLevels = []string{compressionLevelFastest, compressionLevelBalanced, compressionLevelSmallest}

	errUnknownCompressionLevel     = errors.New("unknown compression level")
	errUnsupportedCompressionLevel = errors.New("unsupported compression level")

	errKeyNotAccessible = errors.New("key not found or accessible")

	errMissingTarHeader = errors.New("tar header is missing")

	errRecipientUnparsable = errors.New("recipient could not be parsed")
)

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

		if err := checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(recipientFlag)); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(signatureFlag), viper.GetString(identityFlag))
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

		pubkey, err := readKey(viper.GetString(encryptionFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := parseRecipient(viper.GetString(encryptionFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := readKey(viper.GetString(signatureFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := parseSignerIdentity(viper.GetString(signatureFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		hdrs, err := archive(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetString(fromFlag),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
			viper.GetString(compressionLevelFlag),
			viper.GetString(encryptionFlag),
			recipient,
			viper.GetString(signatureFlag),
			identity,
		)
		if err != nil {
			return err
		}

		return index(
			viper.GetString(driveFlag),
			viper.GetString(metadataFlag),
			viper.GetInt(recordSizeFlag),
			int(lastIndexedRecord),
			int(lastIndexedBlock),
			viper.GetBool(overwriteFlag),
			viper.GetString(compressionFlag),
			viper.GetString(encryptionFlag),
			func(hdr *tar.Header, i int) error {
				if len(hdrs) <= i {
					return errMissingTarHeader
				}

				*hdr = *hdrs[i]

				return nil
			},
			0,
			func(hdr *tar.Header) error {
				return nil // We sign above, no need to verify
			},
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
	recipient interface{},
	signatureFormat string,
	identity interface{},
) ([]*tar.Header, error) {
	dirty := false
	tw, isRegular, cleanup, err := openTapeWriter(tape)
	if err != nil {
		return []*tar.Header{}, err
	}

	if overwrite {
		if isRegular {
			if err := cleanup(&dirty); err != nil { // dirty will always be false here
				return []*tar.Header{}, err
			}

			f, err := os.OpenFile(tape, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return []*tar.Header{}, err
			}

			// Clear the file's content
			if err := f.Truncate(0); err != nil {
				return []*tar.Header{}, err
			}

			if err := f.Close(); err != nil {
				return []*tar.Header{}, err
			}

			tw, isRegular, cleanup, err = openTapeWriter(tape)
			if err != nil {
				return []*tar.Header{}, err
			}
		} else {
			if err := cleanup(&dirty); err != nil { // dirty will always be false here
				return []*tar.Header{}, err
			}

			f, err := os.OpenFile(tape, os.O_WRONLY, os.ModeCharDevice)
			if err != nil {
				return []*tar.Header{}, err
			}

			// Seek to the start of the tape
			if err := controllers.SeekToRecordOnTape(f, 0); err != nil {
				return []*tar.Header{}, err
			}

			if err := f.Close(); err != nil {
				return []*tar.Header{}, err
			}

			tw, isRegular, cleanup, err = openTapeWriter(tape)
			if err != nil {
				return []*tar.Header{}, err
			}
		}
	}

	defer cleanup(&dirty)

	headers := []*tar.Header{}
	first := true
	return headers, filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
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

			encryptor, err := encrypt(fileSizeCounter, encryptionFormat, recipient)
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

			signer, sign, err := sign(file, signatureFormat, identity)
			if err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(compressor, signer); err != nil {
					return err
				}
			} else {
				buf := make([]byte, controllers.BlockSize*recordSize)
				if _, err := io.CopyBuffer(compressor, signer, buf); err != nil {
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
			if signature := sign(); signature != "" {
				hdr.PAXRecords[pax.STFSRecordSignature] = signature
			}
			hdr.Size = int64(fileSizeCounter.BytesRead)

			hdr.Name, err = addSuffix(hdr.Name, compressionFormat, encryptionFormat)
			if err != nil {
				return err
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

		hdrToAppend := *hdr
		headers = append(headers, &hdrToAppend)

		if err := signHeader(hdr, signatureFormat, identity); err != nil {
			return err
		}

		if err := encryptHeader(hdr, encryptionFormat, recipient); err != nil {
			return err
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		// Compress and write the file
		encryptor, err := encrypt(tw, encryptionFormat, recipient)
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

func checkKeyAccessible(encryptionFormat string, pathToKey string) error {
	if encryptionFormat == noneKey {
		return nil
	}

	if _, err := os.Stat(pathToKey); err != nil {
		return errKeyNotAccessible
	}

	return nil
}

func readKey(encryptionFormat string, pathToKey string) ([]byte, error) {
	if encryptionFormat == noneKey {
		return []byte{}, nil
	}

	return ioutil.ReadFile(pathToKey)
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

func encryptHeader(
	hdr *tar.Header,
	encryptionFormat string,
	recipient interface{},
) error {
	if encryptionFormat == noneKey {
		return nil
	}

	newHdr := &tar.Header{
		Format:     tar.FormatPAX,
		Size:       hdr.Size,
		PAXRecords: map[string]string{},
	}

	wrappedHeader, err := json.Marshal(hdr)
	if err != nil {
		return err
	}

	newHdr.PAXRecords[pax.STFSRecordEmbeddedHeader], err = encryptString(string(wrappedHeader), encryptionFormat, recipient)
	if err != nil {
		return err
	}

	*hdr = *newHdr

	return nil
}

func signHeader(
	hdr *tar.Header,
	signatureFormat string,
	identity interface{},
) error {
	if signatureFormat == noneKey {
		return nil
	}

	newHdr := &tar.Header{
		Format:     tar.FormatPAX,
		Size:       hdr.Size,
		PAXRecords: map[string]string{},
	}

	wrappedHeader, err := json.Marshal(hdr)
	if err != nil {
		return err
	}

	newHdr.PAXRecords[pax.STFSRecordEmbeddedHeader] = string(wrappedHeader)
	newHdr.PAXRecords[pax.STFSRecordSignature], err = signString(newHdr.PAXRecords[pax.STFSRecordEmbeddedHeader], signatureFormat, identity)
	if err != nil {
		return err
	}

	*hdr = *newHdr

	return nil
}

func addSuffix(name string, compressionFormat string, encryptionFormat string) (string, error) {
	switch compressionFormat {
	case compressionFormatGZipKey:
		fallthrough
	case compressionFormatParallelGZipKey:
		name += compressionFormatGZipSuffix
	case compressionFormatLZ4Key:
		name += compressionFormatLZ4Suffix
	case compressionFormatZStandardKey:
		name += compressionFormatZStandardSuffix
	case compressionFormatBrotliKey:
		name += compressionFormatBrotliSuffix
	case compressionFormatBzip2Key:
		fallthrough
	case compressionFormatBzip2ParallelKey:
		name += compressionFormatBzip2Suffix
	case noneKey:
	default:
		return "", errUnsupportedCompressionFormat
	}

	switch encryptionFormat {
	case encryptionFormatAgeKey:
		name += encryptionFormatAgeSuffix
	case encryptionFormatPGPKey:
		name += encryptionFormatPGPSuffix
	case noneKey:
	default:
		return "", errUnsupportedEncryptionFormat
	}

	return name, nil
}

func parseRecipient(
	encryptionFormat string,
	pubkey []byte,
) (interface{}, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		return age.ParseX25519Recipient(string(pubkey))
	case encryptionFormatPGPKey:
		return openpgp.ReadKeyRing(bytes.NewBuffer(pubkey))
	case noneKey:
		return pubkey, nil
	default:
		return nil, errUnsupportedEncryptionFormat
	}
}

func encrypt(
	dst io.Writer,
	encryptionFormat string,
	recipient interface{},
) (io.WriteCloser, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		recipient, ok := recipient.(*age.X25519Recipient)
		if !ok {
			return nil, errRecipientUnparsable
		}

		return age.Encrypt(dst, recipient)
	case encryptionFormatPGPKey:
		recipient, ok := recipient.(openpgp.EntityList)
		if !ok {
			return nil, errRecipientUnparsable
		}

		return openpgp.Encrypt(dst, recipient, nil, nil, nil)
	case noneKey:
		return noop.AddClose(dst), nil
	default:
		return nil, errUnsupportedEncryptionFormat
	}
}

func parseSignerIdentity(
	signatureFormat string,
	privkey []byte,
	password string,
) (interface{}, error) {
	switch signatureFormat {
	case signatureFormatMinisignKey:
		return minisign.DecryptKey(password, privkey)
	case noneKey:
		return privkey, nil
	default:
		return nil, errUnsupportedSignatureFormat
	}
}

func sign(
	src io.Reader,
	signatureFormat string,
	identity interface{},
) (io.Reader, func() string, error) {
	switch signatureFormat {
	case signatureFormatMinisignKey:
		identity, ok := identity.(minisign.PrivateKey)
		if !ok {
			return nil, nil, errIdentityUnparsable
		}

		signer := minisign.NewReader(src)

		return signer, func() string {
			return base64.StdEncoding.EncodeToString(signer.Sign(identity))
		}, nil
	case noneKey:
		return io.NopCloser(src), func() string {
			return ""
		}, nil
	default:
		return nil, nil, errUnsupportedSignatureFormat
	}
}

func encryptString(
	src string,
	encryptionFormat string,
	recipient interface{},
) (string, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		recipient, ok := recipient.(*age.X25519Recipient)
		if !ok {
			return "", errRecipientUnparsable
		}

		out := &bytes.Buffer{}
		w, err := age.Encrypt(out, recipient)
		if err != nil {
			return "", err
		}

		if _, err := io.WriteString(w, src); err != nil {
			return "", err
		}

		if err := w.Close(); err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(out.Bytes()), nil
	case encryptionFormatPGPKey:
		recipient, ok := recipient.(openpgp.EntityList)
		if !ok {
			return "", errRecipientUnparsable
		}

		out := &bytes.Buffer{}
		w, err := openpgp.Encrypt(out, recipient, nil, nil, nil)
		if err != nil {
			return "", err
		}

		if _, err := io.WriteString(w, src); err != nil {
			return "", err
		}

		if err := w.Close(); err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(out.Bytes()), nil
	case noneKey:
		return src, nil
	default:
		return "", errUnsupportedEncryptionFormat
	}
}

func signString(
	src string,
	signatureFormat string,
	identity interface{},
) (string, error) {
	switch signatureFormat {
	case signatureFormatMinisignKey:
		identity, ok := identity.(minisign.PrivateKey)
		if !ok {
			return "", errIdentityUnparsable
		}

		return base64.StdEncoding.EncodeToString(minisign.Sign(identity, []byte(src))), nil
	case noneKey:
		return src, nil
	default:
		return "", errUnsupportedSignatureFormat
	}
}

func compress(
	dst io.Writer,
	compressionFormat string,
	compressionLevel string,
	isRegular bool,
	recordSize int,
) (noop.Flusher, error) {
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

		return noop.AddFlush(lz), nil
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

		return noop.AddFlush(bz), nil
	case noneKey:
		return noop.AddFlush(noop.AddClose(dst)), nil
	default:
		return nil, errUnsupportedCompressionFormat
	}
}

func init() {
	archiveCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	archiveCmd.PersistentFlags().StringP(fromFlag, "f", ".", "File or directory to archive")
	archiveCmd.PersistentFlags().BoolP(overwriteFlag, "o", false, "Start writing from the start instead of from the end of the tape or tar file")
	archiveCmd.PersistentFlags().StringP(compressionLevelFlag, "l", compressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", compressionLevelBalanced, knownCompressionLevels))
	archiveCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to public key of recipient to encrypt for")
	archiveCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key to sign with")
	archiveCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")

	viper.AutomaticEnv()

	rootCmd.AddCommand(archiveCmd)
}
