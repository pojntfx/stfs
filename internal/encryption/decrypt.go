package encryption

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"

	"filippo.io/age"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/pojntfx/stfs/internal/pax"
	"github.com/pojntfx/stfs/pkg/config"
)

func Decrypt(
	src io.Reader,
	encryptionFormat string,
	identity interface{},
) (io.ReadCloser, error) {
	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
		identity, ok := identity.(*age.X25519Identity)
		if !ok {
			return nil, config.ErrIdentityUnparsable
		}

		r, err := age.Decrypt(src, identity)
		if err != nil {
			return nil, err
		}

		return io.NopCloser(r), nil
	case config.EncryptionFormatPGPKey:
		identity, ok := identity.(openpgp.EntityList)
		if !ok {
			return nil, config.ErrIdentityUnparsable
		}

		r, err := openpgp.ReadMessage(src, identity, nil, nil)
		if err != nil {
			return nil, err
		}

		return io.NopCloser(r.UnverifiedBody), nil
	case config.NoneKey:
		return io.NopCloser(src), nil
	default:
		return nil, config.ErrEncryptionFormatUnsupported
	}
}

func DecryptHeader(
	hdr *tar.Header,
	encryptionFormat string,
	identity interface{},
) error {
	if encryptionFormat == config.NoneKey {
		return nil
	}

	if hdr.PAXRecords == nil {
		return config.ErrEmbeddedHeaderMissing
	}

	encryptedEmbeddedHeader, ok := hdr.PAXRecords[pax.STFSRecordEmbeddedHeader]
	if !ok {
		return config.ErrEmbeddedHeaderMissing
	}

	embeddedHeader, err := DecryptString(encryptedEmbeddedHeader, encryptionFormat, identity)
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

func DecryptString(
	src string,
	encryptionFormat string,
	identity interface{},
) (string, error) {
	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
		identity, ok := identity.(*age.X25519Identity)
		if !ok {
			return "", config.ErrIdentityUnparsable
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
	case config.EncryptionFormatPGPKey:
		identity, ok := identity.(openpgp.EntityList)
		if !ok {
			return "", config.ErrIdentityUnparsable
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
	case config.NoneKey:
		return src, nil
	default:
		return "", config.ErrEncryptionFormatUnsupported
	}
}
