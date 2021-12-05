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

	"aead.dev/minisign"
	"filippo.io/age"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
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
	toFlag      = "to"
	previewFlag = "preview"
)

var (
	errEmbeddedHeaderMissing = errors.New("embedded header is missing")

	errIdentityUnparsable = errors.New("recipient could not be parsed")

	errInvalidSignature = errors.New("invalid signature")

	errSignatureMissing = errors.New("missing signature")
)

var recoveryFetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch a file or directory from tape or tar file by record and block",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := checkKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(identityFlag)); err != nil {
			return err
		}

		return checkKeyAccessible(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		pubkey, err := readKey(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := parseSignerRecipient(viper.GetString(signatureFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := readKey(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := parseIdentity(viper.GetString(encryptionFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		return restoreFromRecordAndBlock(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			viper.GetInt(recordFlag),
			viper.GetInt(blockFlag),
			viper.GetString(toFlag),
			viper.GetBool(previewFlag),
			true,
			viper.GetString(compressionFlag),
			viper.GetString(encryptionFlag),
			identity,
			viper.GetString(signatureFlag),
			recipient,
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
	identity interface{},
	signatureFormat string,
	recipient interface{},
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

	if err := decryptHeader(hdr, encryptionFormat, identity); err != nil {
		return err
	}

	if err := verifyHeader(hdr, isRegular, signatureFormat, recipient); err != nil {
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

		decryptor, err := decrypt(tr, encryptionFormat, identity)
		if err != nil {
			return err
		}

		decompressor, err := decompress(decryptor, compressionFormat)
		if err != nil {
			return err
		}

		signature := ""
		if hdr.PAXRecords != nil {
			if s, ok := hdr.PAXRecords[pax.STFSRecordSignature]; ok {
				signature = s
			}
		}

		verifier, verify, err := verify(decompressor, isRegular, signatureFormat, recipient, signature)
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, verifier); err != nil {
			return err
		}

		if err := verify(); err != nil {
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
	case noneKey:
		return io.NopCloser(src), nil
	default:
		return nil, errUnsupportedCompressionFormat
	}
}

func decryptHeader(
	hdr *tar.Header,
	encryptionFormat string,
	identity interface{},
) error {
	if encryptionFormat == noneKey {
		return nil
	}

	if hdr.PAXRecords == nil {
		return errEmbeddedHeaderMissing
	}

	encryptedEmbeddedHeader, ok := hdr.PAXRecords[pax.STFSRecordEmbeddedHeader]
	if !ok {
		return errEmbeddedHeaderMissing
	}

	embeddedHeader, err := decryptString(encryptedEmbeddedHeader, encryptionFormat, identity)
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

func verifyHeader(
	hdr *tar.Header,
	isRegular bool,
	signatureFormat string,
	recipient interface{},
) error {
	if signatureFormat == noneKey {
		return nil
	}

	if hdr.PAXRecords == nil {
		return errEmbeddedHeaderMissing
	}

	embeddedHeader, ok := hdr.PAXRecords[pax.STFSRecordEmbeddedHeader]
	if !ok {
		return errEmbeddedHeaderMissing
	}

	signature, ok := hdr.PAXRecords[pax.STFSRecordSignature]
	if !ok {
		return errSignatureMissing
	}

	if err := verifyString(embeddedHeader, isRegular, signatureFormat, recipient, signature); err != nil {
		return err
	}

	var newHdr tar.Header
	if err := json.Unmarshal([]byte(embeddedHeader), &newHdr); err != nil {
		return err
	}

	*hdr = newHdr

	return nil
}

func parseIdentity(
	encryptionFormat string,
	privkey []byte,
	password string,
) (interface{}, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		if password != "" {
			passwordIdentity, err := age.NewScryptIdentity(password)
			if err != nil {
				return nil, err
			}

			r, err := age.Decrypt(bytes.NewBuffer(privkey), passwordIdentity)
			if err != nil {
				return nil, err
			}

			out := &bytes.Buffer{}
			if _, err := io.Copy(out, r); err != nil {
				return nil, err
			}

			privkey = out.Bytes()
		}

		return age.ParseX25519Identity(string(privkey))
	case encryptionFormatPGPKey:
		identities, err := openpgp.ReadKeyRing(bytes.NewBuffer(privkey))
		if err != nil {
			return nil, err
		}

		if password != "" {
			for _, identity := range identities {
				if identity.PrivateKey == nil {
					return nil, errIdentityUnparsable
				}

				if err := identity.PrivateKey.Decrypt([]byte(password)); err != nil {
					return nil, err
				}

				for _, subkey := range identity.Subkeys {
					if err := subkey.PrivateKey.Decrypt([]byte(password)); err != nil {
						return nil, err
					}
				}
			}
		}

		return identities, nil
	case noneKey:
		return privkey, nil
	default:
		return nil, errUnsupportedEncryptionFormat
	}
}

func decryptString(
	src string,
	encryptionFormat string,
	identity interface{},
) (string, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		identity, ok := identity.(*age.X25519Identity)
		if !ok {
			return "", errIdentityUnparsable
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
	case encryptionFormatPGPKey:
		identity, ok := identity.(openpgp.EntityList)
		if !ok {
			return "", errIdentityUnparsable
		}

		decoded, err := base64.StdEncoding.DecodeString(src)
		if err != nil {
			return "", err
		}

		r, err := openpgp.ReadMessage(bytes.NewBufferString(string(decoded)), identity, nil, nil)
		if err != nil {
			return "", err
		}

		out := &bytes.Buffer{}
		if _, err := io.Copy(out, r.UnverifiedBody); err != nil {
			return "", err
		}

		return out.String(), nil
	case noneKey:
		return src, nil
	default:
		return "", errUnsupportedEncryptionFormat
	}
}

func decrypt(
	src io.Reader,
	encryptionFormat string,
	identity interface{},
) (io.ReadCloser, error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
		identity, ok := identity.(*age.X25519Identity)
		if !ok {
			return nil, errIdentityUnparsable
		}

		r, err := age.Decrypt(src, identity)
		if err != nil {
			return nil, err
		}

		return io.NopCloser(r), nil
	case encryptionFormatPGPKey:
		identity, ok := identity.(openpgp.EntityList)
		if !ok {
			return nil, errIdentityUnparsable
		}

		r, err := openpgp.ReadMessage(src, identity, nil, nil)
		if err != nil {
			return nil, err
		}

		return io.NopCloser(r.UnverifiedBody), nil
	case noneKey:
		return io.NopCloser(src), nil
	default:
		return nil, errUnsupportedEncryptionFormat
	}
}

func parseSignerRecipient(
	signatureFormat string,
	pubkey []byte,
) (interface{}, error) {
	switch signatureFormat {
	case signatureFormatMinisignKey:
		var recipient minisign.PublicKey
		if err := recipient.UnmarshalText(pubkey); err != nil {
			return nil, err
		}

		return recipient, nil
	case signatureFormatPGPKey:
		return parseRecipient(signatureFormat, pubkey)
	case noneKey:
		return pubkey, nil
	default:
		return nil, errUnsupportedSignatureFormat
	}
}

func verify(
	src io.Reader,
	isRegular bool,
	signatureFormat string,
	recipient interface{},
	signature string,
) (io.Reader, func() error, error) {
	switch signatureFormat {
	case signatureFormatMinisignKey:
		if !isRegular {
			return nil, nil, errSignatureFormatOnlyRegularSupport
		}

		recipient, ok := recipient.(minisign.PublicKey)
		if !ok {
			return nil, nil, errRecipientUnparsable
		}

		verifier := minisign.NewReader(src)

		return verifier, func() error {
			decodedSignature, err := base64.StdEncoding.DecodeString(signature)
			if err != nil {
				return err
			}

			if verifier.Verify(recipient, decodedSignature) {
				return nil
			}

			return errInvalidSignature
		}, nil
	case signatureFormatPGPKey:
		recipients, ok := recipient.(openpgp.EntityList)
		if !ok {
			return nil, nil, errIdentityUnparsable
		}

		if len(recipients) < 1 {
			return nil, nil, errIdentityUnparsable
		}

		decodedSignature, err := base64.StdEncoding.DecodeString(signature)
		if err != nil {
			return nil, nil, err
		}

		reader := packet.NewReader(bytes.NewBuffer(decodedSignature))
		pkt, err := reader.Next()
		if err != nil {
			return nil, nil, err
		}

		sig, ok := pkt.(*packet.Signature)
		if !ok {
			return nil, nil, errInvalidSignature
		}

		hash := sig.Hash.New()

		tee := io.TeeReader(src, hash)

		return tee, func() error {
			return recipients[0].PrimaryKey.VerifySignature(hash, sig)
		}, nil
	case noneKey:
		return io.NopCloser(src), func() error {
			return nil
		}, nil
	default:
		return nil, nil, errUnsupportedSignatureFormat
	}
}

func verifyString(
	src string,
	isRegular bool,
	signatureFormat string,
	recipient interface{},
	signature string,
) error {
	switch signatureFormat {
	case signatureFormatMinisignKey:
		if !isRegular {
			return errSignatureFormatOnlyRegularSupport
		}

		recipient, ok := recipient.(minisign.PublicKey)
		if !ok {
			return errRecipientUnparsable
		}

		decodedSignature, err := base64.StdEncoding.DecodeString(signature)
		if err != nil {
			return err
		}

		if minisign.Verify(recipient, []byte(src), decodedSignature) {
			return nil
		}

		return errInvalidSignature
	case signatureFormatPGPKey:
		recipients, ok := recipient.(openpgp.EntityList)
		if !ok {
			return nil
		}

		if len(recipients) < 1 {
			return nil
		}

		decodedSignature, err := base64.StdEncoding.DecodeString(signature)
		if err != nil {
			return nil
		}

		reader := packet.NewReader(bytes.NewBuffer(decodedSignature))
		pkt, err := reader.Next()
		if err != nil {
			return nil
		}

		sig, ok := pkt.(*packet.Signature)
		if !ok {
			return nil
		}

		hash := sig.Hash.New()

		if _, err := io.Copy(hash, bytes.NewBufferString(src)); err != nil {
			return err
		}

		return recipients[0].PrimaryKey.VerifySignature(hash, sig)
	case noneKey:
		return nil
	default:
		return errUnsupportedSignatureFormat
	}
}

func init() {
	recoveryFetchCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	recoveryFetchCmd.PersistentFlags().IntP(recordFlag, "k", 0, "Record to seek too")
	recoveryFetchCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too")
	recoveryFetchCmd.PersistentFlags().StringP(toFlag, "t", "", "File to restore to (archived name by default)")
	recoveryFetchCmd.PersistentFlags().BoolP(previewFlag, "w", false, "Only read the header")
	recoveryFetchCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")
	recoveryFetchCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")
	recoveryFetchCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to the public key to verify with")

	viper.AutomaticEnv()

	recoveryCmd.AddCommand(recoveryFetchCmd)
}
