package keys

import (
	"bytes"
	"io"

	"aead.dev/minisign"
	"filippo.io/age"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/pojntfx/stfs/pkg/config"
)

func ParseIdentity(
	encryptionFormat string,
	privkey []byte,
	password string,
) (interface{}, error) {
	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
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
	case config.EncryptionFormatPGPKey:
		identities, err := openpgp.ReadKeyRing(bytes.NewBuffer(privkey))
		if err != nil {
			return nil, err
		}

		if password != "" {
			for _, identity := range identities {
				if identity.PrivateKey == nil {
					return nil, config.ErrIdentityUnparsable
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
	case config.NoneKey:
		return privkey, nil
	default:
		return nil, config.ErrEncryptionFormatUnsupported
	}
}

func ParseSignerIdentity(
	signatureFormat string,
	privkey []byte,
	password string,
) (interface{}, error) {
	switch signatureFormat {
	case config.SignatureFormatMinisignKey:
		return minisign.DecryptKey(password, privkey)
	case config.SignatureFormatPGPKey:
		return ParseIdentity(signatureFormat, privkey, password)
	case config.NoneKey:
		return privkey, nil
	default:
		return nil, config.ErrSignatureFormatUnsupported
	}
}
