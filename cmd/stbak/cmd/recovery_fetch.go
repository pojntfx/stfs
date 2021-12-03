package cmd

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/andybalholm/brotli"
	"github.com/cosnicolaou/pbzip2"
	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/pierrec/lz4/v4"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/pax"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	recordFlag  = "record"
	blockFlag   = "block"
	dstFlag     = "dst"
	previewFlag = "preview"
)

var (
	errEmbeddedHeaderMissing = errors.New("embedded header is missing")
)

var recoveryFetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch a file or directory from tape or tar file by record and block",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		privkey, err := readKey(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		return restoreFromRecordAndBlock(
			viper.GetString(tapeFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetInt(recordFlag),
			viper.GetInt(blockFlag),
			viper.GetString(dstFlag),
			viper.GetBool(previewFlag),
			true,
			viper.GetString(compressionFlag),
			viper.GetString(encryptionFlag),
			privkey,
		)
	},
}

func restoreFromRecordAndBlock(
	tape string,
	recordSize int,
	record int,
	block int,
	dst string,
	preview bool,
	showHeader bool,
	compressionFormat string,
	encryptionFormat string,
	privkey []byte,
) error {
	f, isRegular, err := openTapeReadOnly(tape)
	if err != nil {
		return err
	}
	defer f.Close()

	var tr *tar.Reader
	if isRegular {
		// Seek to record and block
		if _, err := f.Seek(int64((recordSize*controllers.BlockSize*record)+block*controllers.BlockSize), io.SeekStart); err != nil {
			return err
		}

		tr = tar.NewReader(f)
	} else {
		// Seek to record
		if err := controllers.SeekToRecordOnTape(f, int32(record)); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(f, controllers.BlockSize*recordSize)
		if _, err := br.Read(make([]byte, block*controllers.BlockSize)); err != nil {
			return err
		}

		tr = tar.NewReader(br)
	}

	hdr, err := tr.Next()
	if err != nil {
		return err
	}

	if err := decryptHeader(hdr, encryptionFormat, privkey); err != nil {
		return err
	}

	if showHeader {
		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}

		if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(int64(record), int64(block), hdr)); err != nil {
			return err
		}
	}

	if !preview {
		if dst == "" {
			dst = filepath.Base(hdr.Name)
		}

		if hdr.Typeflag == tar.TypeDir {
			return os.MkdirAll(dst, hdr.FileInfo().Mode())
		}

		dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}

		if err := dstFile.Truncate(0); err != nil {
			return err
		}

		// Don't decompress non-regular files
		if !hdr.FileInfo().Mode().IsRegular() {
			if _, err := io.Copy(dstFile, tr); err != nil {
				return err
			}

			return nil
		}

		decryptor, err := decrypt(tr, encryptionFormat, privkey)
		if err != nil {
			return err
		}

		decompressor, err := decompress(decryptor, compressionFormat)
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, decompressor); err != nil {
			return err
		}

		if err := decryptor.Close(); err != nil {
			return err
		}

		if err := decompressor.Close(); err != nil {
			return err
		}

		if err := dstFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

func decompress(
	src io.Reader,
	compressionFormat string,
) (io.ReadCloser, error) {
	switch compressionFormat {
	case compressionFormatGZipKey:
		fallthrough
	case compressionFormatParallelGZipKey:
		if compressionFormat == compressionFormatGZipKey {
			return gzip.NewReader(src)
		}

		return pgzip.NewReader(src)
	case compressionFormatLZ4Key:
		lz := lz4.NewReader(src)
		if err := lz.Apply(lz4.ConcurrencyOption(-1)); err != nil {
			return nil, err
		}

		return io.NopCloser(lz), nil
	case compressionFormatZStandardKey:
		zz, err := zstd.NewReader(src)
		if err != nil {
			return nil, err
		}

		return io.NopCloser(zz), nil
	case compressionFormatBrotliKey:
		br := brotli.NewReader(src)

		return io.NopCloser(br), nil
	case compressionFormatBzip2Key:
		return bzip2.NewReader(src, nil)
	case compressionFormatBzip2ParallelKey:
		bz := pbzip2.NewReader(context.Background(), src)

		return io.NopCloser(bz), nil
	case compressionFormatNoneKey:
		return io.NopCloser(src), nil
	default:
		return nil, errUnsupportedCompressionFormat
	}
}

func decryptHeader(
	hdr *tar.Header,
	encryptionFormat string,
	privkey []byte,
) error {
	if encryptionFormat == encryptionFormatNoneKey {
		return nil
	}

	if hdr.PAXRecords == nil {
		return errEmbeddedHeaderMissing
	}

	encryptedEmbeddedHeader, ok := hdr.PAXRecords[pax.STFSEmbeddedHeader]
	if !ok {
		return errEmbeddedHeaderMissing
	}

	embeddedHeader, err := decryptString(encryptedEmbeddedHeader, encryptionFormat, privkey)
	if err != nil {
		return err
	}

	var newHdr tar.Header
	if err := json.Unmarshal([]byte(embeddedHeader), &newHdr); err != nil {
		return err
	}

	*hdr = newHdr

	return nil
}

func decryptString(
	src string,
	encryptionFormat string,
	privkey []byte,
) (string, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		identity, err := age.ParseX25519Identity(string(privkey))
		if err != nil {
			return "", err
		}

		decoded, err := base64.StdEncoding.DecodeString(src)
		if err != nil {
			return "", err
		}

		r, err := age.Decrypt(bytes.NewBufferString(string(decoded)), identity)
		if err != nil {
			return "", err
		}

		out := &bytes.Buffer{}
		if _, err := io.Copy(out, r); err != nil {
			return "", err
		}

		return out.String(), nil
	case encryptionFormatNoneKey:
		return src, nil
	default:
		return "", errUnsupportedEncryptionFormat
	}
}

func decrypt(
	src io.Reader,
	encryptionFormat string,
	privkey []byte,
) (io.ReadCloser, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		identity, err := age.ParseX25519Identity(string(privkey))
		if err != nil {
			return nil, err
		}

		r, err := age.Decrypt(src, identity)
		if err != nil {
			return nil, err
		}

		return io.NopCloser(r), nil
	case encryptionFormatNoneKey:
		return io.NopCloser(src), nil
	default:
		return nil, errUnsupportedEncryptionFormat
	}
}

func init() {
	recoveryFetchCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	recoveryFetchCmd.PersistentFlags().IntP(recordFlag, "k", 0, "Record to seek too")
	recoveryFetchCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too")
	recoveryFetchCmd.PersistentFlags().StringP(dstFlag, "d", "", "File to restore to (archived name by default)")
	recoveryFetchCmd.PersistentFlags().BoolP(previewFlag, "p", false, "Only read the header")
	recoveryFetchCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")

	viper.AutomaticEnv()

	recoveryCmd.AddCommand(recoveryFetchCmd)
}
