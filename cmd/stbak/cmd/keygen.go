package cmd

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"aead.dev/minisign"
	"filippo.io/age"
	"github.com/ProtonMail/gopenpgp/v2/armor"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var keygenCmd = &cobra.Command{
	Use:     "keygen",
	Aliases: []string{"key", "k"},
	Short:   "Restore a file or directory from tape or tar file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if viper.GetBool(verboseFlag) {
			boil.DebugMode = true
		}

		var privkey []byte
		var pubkey []byte

		if encryptionFormat := viper.GetString(encryptionFlag); encryptionFormat != noneKey {
			priv, pub, err := generateEncryptionKey(encryptionFormat, viper.GetString(passwordFlag))
			if err != nil {
				return err
			}

			privkey = priv
			pubkey = pub
		} else if signatureFormat := viper.GetString(signatureFlag); signatureFormat != noneKey {
			priv, pub, err := generateSignatureKey(signatureFormat, viper.GetString(passwordFlag))
			if err != nil {
				return err
			}

			privkey = priv
			pubkey = pub
		} else {
			return errKeygenForFormatUnsupported
		}

		// Write pubkey (read/writable by everyone)
		if err := os.MkdirAll(filepath.Dir(viper.GetString(recipientFlag)), os.ModePerm); err != nil {
			return err
		}

		if err := ioutil.WriteFile(viper.GetString(recipientFlag), pubkey, os.ModePerm); err != nil {
			return err
		}

		// Write privkey (read/writable only by the owner)
		if err := os.MkdirAll(filepath.Dir(viper.GetString(identityFlag)), 0700); err != nil {
			return err
		}

		return ioutil.WriteFile(viper.GetString(identityFlag), privkey, 0600)
	},
}

func generateEncryptionKey(
	encryptionFormat string,
	password string,
) (privkey []byte, pubkey []byte, err error) {
	switch encryptionFormat {
	case encryptionFormatAgeKey:
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
	case encryptionFormatPGPKey:
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
		return []byte{}, []byte{}, errKeygenForFormatUnsupported
	}
}

func generateSignatureKey(
	signatureFormat string,
	password string,
) (privkey []byte, pubkey []byte, err error) {
	switch signatureFormat {
	case signatureFormatMinisignKey:
		pub, rawPriv, err := minisign.GenerateKey(rand.Reader)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		priv, err := minisign.EncryptKey(password, rawPriv)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		return priv, []byte(pub.String()), err
	case signatureFormatPGPKey:
		return generateEncryptionKey(signatureFormat, password)
	default:
		return []byte{}, []byte{}, errKeygenForFormatUnsupported
	}
}

func init() {
	keygenCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to write the public key to")
	keygenCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to write the private key to")
	keygenCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password to protect the private key with")

	viper.AutomaticEnv()

	rootCmd.AddCommand(keygenCmd)
}
