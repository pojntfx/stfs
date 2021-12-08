package encryption

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"

	"filippo.io/age"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/pkg/config"
)

func Encrypt(
	dst io.Writer,
	encryptionFormat string,
	recipient interface{},
) (io.WriteCloser, error) {
	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
		recipient, ok := recipient.(*age.X25519Recipient)
		if !ok {
			return nil, config.ErrRecipientUnparsable
		}

		return age.Encrypt(dst, recipient)
	case config.EncryptionFormatPGPKey:
		recipient, ok := recipient.(openpgp.EntityList)
		if !ok {
			return nil, config.ErrRecipientUnparsable
		}

		return openpgp.Encrypt(dst, recipient, nil, nil, nil)
	case config.NoneKey:
		return ioext.AddClose(dst), nil
	default:
		return nil, config.ErrEncryptionFormatUnsupported
	}
}

func EncryptHeader(
	hdr *tar.Header,
	encryptionFormat string,
	recipient interface{},
) error {
	if encryptionFormat == config.NoneKey {
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

	newHdr.PAXRecords[records.STFSRecordEmbeddedHeader], err = EncryptString(string(wrappedHeader), encryptionFormat, recipient)
	if err != nil {
		return err
	}

	*hdr = *newHdr

	return nil
}

func EncryptString(
	src string,
	encryptionFormat string,
	recipient interface{},
) (string, error) {
	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
		recipient, ok := recipient.(*age.X25519Recipient)
		if !ok {
			return "", config.ErrRecipientUnparsable
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
	case config.EncryptionFormatPGPKey:
		recipient, ok := recipient.(openpgp.EntityList)
		if !ok {
			return "", config.ErrRecipientUnparsable
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
	case config.NoneKey:
		return src, nil
	default:
		return "", config.ErrEncryptionFormatUnsupported
	}
}
