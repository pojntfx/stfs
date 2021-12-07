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

func Verify(
	src io.Reader,
	isRegular bool,
	signatureFormat string,
	recipient interface{},
	signature string,
) (io.Reader, func() error, error) {
	switch signatureFormat {
	case config.SignatureFormatMinisignKey:
		if !isRegular {
			return nil, nil, config.ErrSignatureFormatOnlyRegularSupport
		}

		recipient, ok := recipient.(minisign.PublicKey)
		if !ok {
			return nil, nil, config.ErrRecipientUnparsable
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

			return config.ErrSignatureInvalid
		}, nil
	case config.SignatureFormatPGPKey:
		recipients, ok := recipient.(openpgp.EntityList)
		if !ok {
			return nil, nil, config.ErrIdentityUnparsable
		}

		if len(recipients) < 1 {
			return nil, nil, config.ErrIdentityUnparsable
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
			return nil, nil, config.ErrSignatureInvalid
		}

		hash := sig.Hash.New()

		tee := io.TeeReader(src, hash)

		return tee, func() error {
			return recipients[0].PrimaryKey.VerifySignature(hash, sig)
		}, nil
	case config.NoneKey:
		return io.NopCloser(src), func() error {
			return nil
		}, nil
	default:
		return nil, nil, config.ErrSignatureFormatUnsupported
	}
}

func VerifyHeader(
	hdr *tar.Header,
	isRegular bool,
	signatureFormat string,
	recipient interface{},
) error {
	if signatureFormat == config.NoneKey {
		return nil
	}

	if hdr.PAXRecords == nil {
		return config.ErrEmbeddedHeaderMissing
	}

	embeddedHeader, ok := hdr.PAXRecords[pax.STFSRecordEmbeddedHeader]
	if !ok {
		return config.ErrEmbeddedHeaderMissing
	}

	signature, ok := hdr.PAXRecords[pax.STFSRecordSignature]
	if !ok {
		return config.ErrSignatureMissing
	}

	if err := VerifyString(embeddedHeader, isRegular, signatureFormat, recipient, signature); err != nil {
		return err
	}

	var newHdr tar.Header
	if err := json.Unmarshal([]byte(embeddedHeader), &newHdr); err != nil {
		return err
	}

	*hdr = newHdr

	return nil
}

func VerifyString(
	src string,
	isRegular bool,
	signatureFormat string,
	recipient interface{},
	signature string,
) error {
	switch signatureFormat {
	case config.SignatureFormatMinisignKey:
		if !isRegular {
			return config.ErrSignatureFormatOnlyRegularSupport
		}

		recipient, ok := recipient.(minisign.PublicKey)
		if !ok {
			return config.ErrRecipientUnparsable
		}

		decodedSignature, err := base64.StdEncoding.DecodeString(signature)
		if err != nil {
			return err
		}

		if minisign.Verify(recipient, []byte(src), decodedSignature) {
			return nil
		}

		return config.ErrSignatureInvalid
	case config.SignatureFormatPGPKey:
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
	case config.NoneKey:
		return nil
	default:
		return config.ErrSignatureFormatUnsupported
	}
}
