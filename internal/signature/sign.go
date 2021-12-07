package signature

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"

	"aead.dev/minisign"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/pojntfx/stfs/internal/pax"
	"github.com/pojntfx/stfs/pkg/config"
)

func Sign(
	src io.Reader,
	isRegular bool,
	signatureFormat string,
	identity interface{},
) (io.Reader, func() (string, error), error) {
	switch signatureFormat {
	case config.SignatureFormatMinisignKey:
		if !isRegular {
			return nil, nil, config.ErrSignatureFormatOnlyRegularSupport
		}

		identity, ok := identity.(minisign.PrivateKey)
		if !ok {
			return nil, nil, config.ErrIdentityUnparsable
		}

		signer := minisign.NewReader(src)

		return signer, func() (string, error) {
			return base64.StdEncoding.EncodeToString(signer.Sign(identity)), nil
		}, nil
	case config.SignatureFormatPGPKey:
		identities, ok := identity.(openpgp.EntityList)
		if !ok {
			return nil, nil, config.ErrIdentityUnparsable
		}

		if len(identities) < 1 {
			return nil, nil, config.ErrIdentityUnparsable
		}

		// See openpgp.DetachSign
		var c *packet.Config
		signingKey, ok := identities[0].SigningKeyById(c.Now(), c.SigningKey())
		if !ok || signingKey.PrivateKey == nil || signingKey.PublicKey == nil {
			return nil, nil, config.ErrIdentityUnparsable
		}

		sig := new(packet.Signature)
		sig.SigType = packet.SigTypeBinary
		sig.PubKeyAlgo = signingKey.PrivateKey.PubKeyAlgo
		sig.Hash = c.Hash()
		sig.CreationTime = c.Now()
		sigLifetimeSecs := c.SigLifetime()
		sig.SigLifetimeSecs = &sigLifetimeSecs
		sig.IssuerKeyId = &signingKey.PrivateKey.KeyId

		hash := sig.Hash.New()

		return io.TeeReader(src, hash), func() (string, error) {
			if err := sig.Sign(hash, signingKey.PrivateKey, c); err != nil {
				return "", err
			}

			out := &bytes.Buffer{}
			if err := sig.Serialize(out); err != nil {
				return "", err
			}

			return base64.StdEncoding.EncodeToString(out.Bytes()), nil
		}, nil
	case config.NoneKey:
		return src, func() (string, error) {
			return "", nil
		}, nil
	default:
		return nil, nil, config.ErrSignatureFormatUnsupported
	}
}

func SignHeader(
	hdr *tar.Header,
	isRegular bool,
	signatureFormat string,
	identity interface{},
) error {
	if signatureFormat == config.NoneKey {
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
	newHdr.PAXRecords[pax.STFSRecordSignature], err = SignString(newHdr.PAXRecords[pax.STFSRecordEmbeddedHeader], isRegular, signatureFormat, identity)
	if err != nil {
		return err
	}

	*hdr = *newHdr

	return nil
}

func SignString(
	src string,
	isRegular bool,
	signatureFormat string,
	identity interface{},
) (string, error) {
	switch signatureFormat {
	case config.SignatureFormatMinisignKey:
		if !isRegular {
			return "", config.ErrSignatureFormatOnlyRegularSupport
		}

		identity, ok := identity.(minisign.PrivateKey)
		if !ok {
			return "", config.ErrIdentityUnparsable
		}

		return base64.StdEncoding.EncodeToString(minisign.Sign(identity, []byte(src))), nil
	case config.SignatureFormatPGPKey:
		identities, ok := identity.(openpgp.EntityList)
		if !ok {
			return "", config.ErrIdentityUnparsable
		}

		if len(identities) < 1 {
			return "", config.ErrIdentityUnparsable
		}

		out := &bytes.Buffer{}
		if err := openpgp.DetachSign(out, identities[0], bytes.NewBufferString(src), nil); err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(out.Bytes()), nil
	case config.NoneKey:
		return src, nil
	default:
		return "", config.ErrSignatureFormatUnsupported
	}
}
