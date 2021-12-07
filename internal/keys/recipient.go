package keys

import (
	"bytes"

	"aead.dev/minisign"
	"filippo.io/age"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/pojntfx/stfs/pkg/config"
)

func ParseRecipient(
	encryptionFormat string,
	pubkey []byte,
) (interface{}, error) {
	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
		return age.ParseX25519Recipient(string(pubkey))
	case config.EncryptionFormatPGPKey:
		return openpgp.ReadKeyRing(bytes.NewBuffer(pubkey))
	case config.NoneKey:
		return pubkey, nil
	default:
		return nil, config.ErrEncryptionFormatUnsupported
	}
}

func ParseSignerRecipient(
	signatureFormat string,
	pubkey []byte,
) (interface{}, error) {
	switch signatureFormat {
	case config.SignatureFormatMinisignKey:
		var recipient minisign.PublicKey
		if err := recipient.UnmarshalText(pubkey); err != nil {
			return nil, err
		}

		return recipient, nil
	case config.SignatureFormatPGPKey:
		return ParseRecipient(signatureFormat, pubkey)
	case config.NoneKey:
		return pubkey, nil
	default:
		return nil, config.ErrSignatureFormatUnsupported
	}
}
