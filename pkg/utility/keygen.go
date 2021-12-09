package utility

import (
	"bytes"
	"crypto/rand"
	"io"

	"aead.dev/minisign"
	"filippo.io/age"
	"github.com/ProtonMail/gopenpgp/v2/armor"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/pojntfx/stfs/pkg/config"
)

func Keygen(
	pipes config.PipeConfig,
	password PasswordConfig,
) (privkey []byte, pubkey []byte, err error) {
	if pipes.Encryption != config.NoneKey {
		priv, pub, err := generateEncryptionKey(pipes.Encryption, password.Password)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		privkey = priv
		pubkey = pub
	} else if pipes.Signature != config.NoneKey {
		priv, pub, err := generateSignatureKey(pipes.Signature, password.Password)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		privkey = priv
		pubkey = pub
	} else {
		return []byte{}, []byte{}, config.ErrKeygenFormatUnsupported
	}

	return privkey, pubkey, nil
}

func generateEncryptionKey(
	encryptionFormat string,
	password string,
) (privkey []byte, pubkey []byte, err error) {
	switch encryptionFormat {
	case config.EncryptionFormatAgeKey:
		identity, err := age.GenerateX25519Identity()
		if err != nil {
			return []byte{}, []byte{}, err
		}

		pub := identity.Recipient().String()
		priv := identity.String()

		if password != "" {
			passwordRecipient, err := age.NewScryptRecipient(password)
			if err != nil {
				return []byte{}, []byte{}, err
			}

			out := &bytes.Buffer{}
			w, err := age.Encrypt(out, passwordRecipient)
			if err != nil {
				return []byte{}, []byte{}, err
			}

			if _, err := io.WriteString(w, priv); err != nil {
				return []byte{}, []byte{}, err
			}

			if err := w.Close(); err != nil {
				return []byte{}, []byte{}, err
			}

			priv = out.String()
		}

		return []byte(priv), []byte(pub), nil
	case config.EncryptionFormatPGPKey:
		armoredIdentity, err := helper.GenerateKey("STFS", "stfs@example.com", []byte(password), "x25519", 0)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		rawIdentity, err := armor.Unarmor(armoredIdentity)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		identity, err := crypto.NewKey([]byte(rawIdentity))
		if err != nil {
			return []byte{}, []byte{}, err
		}

		pub, err := identity.GetPublicKey()
		if err != nil {
			return []byte{}, []byte{}, err
		}

		priv, err := identity.Serialize()
		if err != nil {
			return []byte{}, []byte{}, err
		}

		return priv, pub, nil
	default:
		return []byte{}, []byte{}, config.ErrKeygenFormatUnsupported
	}
}

func generateSignatureKey(
	signatureFormat string,
	password string,
) (privkey []byte, pubkey []byte, err error) {
	switch signatureFormat {
	case config.SignatureFormatMinisignKey:
		pub, rawPriv, err := minisign.GenerateKey(rand.Reader)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		priv, err := minisign.EncryptKey(password, rawPriv)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		return priv, []byte(pub.String()), err
	case config.SignatureFormatPGPKey:
		return generateEncryptionKey(signatureFormat, password)
	default:
		return []byte{}, []byte{}, config.ErrKeygenFormatUnsupported
	}
}
