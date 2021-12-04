package cmd

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"filippo.io/age"
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

		switch viper.GetString(encryptionFlag) {
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
		default:
			return errKeygenForEncryptionFormatUnsupported
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
