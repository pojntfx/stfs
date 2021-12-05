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

		pubkey := ""
		privkey := ""

		if encryptionFormat := viper.GetString(encryptionFlag); encryptionFormat != noneKey {
			switch encryptionFormat {
			case encryptionFormatAgeKey:
				identity, err := age.GenerateX25519Identity()
				if err != nil {
					return err
				}

				pubkey = identity.Recipient().String()
				privkey = identity.String()

				if password := viper.GetString(passwordFlag); password != "" {
					passwordRecipient, err := age.NewScryptRecipient(password)
					if err != nil {
						return err
					}

					out := &bytes.Buffer{}
					w, err := age.Encrypt(out, passwordRecipient)
					if err != nil {
						return err
					}

					if _, err := io.WriteString(w, privkey); err != nil {
						return err
					}

					if err := w.Close(); err != nil {
						return err
					}

					privkey = out.String()
				}
			case encryptionFormatPGPKey:
				armoredIdentity, err := helper.GenerateKey("STFS", "stfs@example.com", []byte(viper.GetString(passwordFlag)), "x25519", 0)
				if err != nil {
					return err
				}

				rawIdentity, err := armor.Unarmor(armoredIdentity)
				if err != nil {
					return err
				}

				identity, err := crypto.NewKey([]byte(rawIdentity))
				if err != nil {
					return err
				}

				pub, err := identity.GetPublicKey()
				if err != nil {
					return err
				}

				priv, err := identity.Serialize()
				if err != nil {
					return err
				}

				pubkey = string(pub)
				privkey = string(priv)
			default:
				return errKeygenForFormatUnsupported
			}
		} else if signatureFormat := viper.GetString(signatureFlag); signatureFormat != noneKey {
			switch signatureFormat {
			case signatureFormatMinisignKey:
				pub, rawPriv, err := minisign.GenerateKey(rand.Reader)
				if err != nil {
					return err
				}

				priv, err := minisign.EncryptKey(viper.GetString(passwordFlag), rawPriv)
				if err != nil {
					return err
				}

				pubkey = pub.String()
				privkey = string(priv)
			default:
				return errKeygenForFormatUnsupported
			}
		} else {
			return errKeygenForFormatUnsupported
		}

		// Write pubkey (read/writable by everyone)
		if err := os.MkdirAll(filepath.Dir(viper.GetString(recipientFlag)), os.ModePerm); err != nil {
			return err
		}

		if err := ioutil.WriteFile(viper.GetString(recipientFlag), []byte(pubkey), os.ModePerm); err != nil {
			return err
		}

		// Write privkey (read/writable only by the owner)
		if err := os.MkdirAll(filepath.Dir(viper.GetString(identityFlag)), 0700); err != nil {
			return err
		}

		return ioutil.WriteFile(viper.GetString(identityFlag), []byte(privkey), 0600)
	},
}

func init() {
	keygenCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to write the public key to")
	keygenCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to write the private key to")
	keygenCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password to protect the private key with")

	viper.AutomaticEnv()

	rootCmd.AddCommand(keygenCmd)
}
