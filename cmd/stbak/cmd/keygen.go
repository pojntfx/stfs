package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/utility"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keygenCmd = &cobra.Command{
	Use:     "keygen",
	Aliases: []string{"key", "k"},
	Short:   "Generate a encryption or signature key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		privkey, pubkey, err := utility.Keygen(
			config.PipeConfig{
				Compression: viper.GetString(compressionFlag),
				Encryption:  viper.GetString(encryptionFlag),
				Signature:   viper.GetString(signatureFlag),
			},
			utility.PasswordConfig{
				Password: viper.GetString(passwordFlag),
			},
		)
		if err != nil {
			return err
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

func init() {
	keygenCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to write the public key to")
	keygenCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to write the private key to")
	keygenCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password to protect the private key with")

	viper.AutomaticEnv()

	rootCmd.AddCommand(keygenCmd)
}
