package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/pojntfx/stfs/internal/cache"
	sfs "github.com/pojntfx/stfs/internal/fs"
	"github.com/pojntfx/stfs/internal/ftp"
	"github.com/pojntfx/stfs/internal/keys"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveFTPCmd = &cobra.Command{
	Use:     "ftp",
	Aliases: []string{"f"},
	Short:   "Serve tape or tar file and the index over FTP (read-write)",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := cache.CheckCacheType(viper.GetString(cacheFlag)); err != nil {
			return err
		}

		if err := keys.CheckKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(identityFlag)); err != nil {
			return err
		}

		return keys.CheckKeyAccessible(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pubkey, err := keys.ReadKey(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := keys.ParseSignerRecipient(viper.GetString(signatureFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := keys.ReadKey(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
		if err != nil {
			return err
		}

		identity, err := keys.ParseIdentity(viper.GetString(encryptionFlag), privkey, viper.GetString(passwordFlag))
		if err != nil {
			return err
		}

		tm := tape.NewTapeManager(
			viper.GetString(driveFlag),
			viper.GetInt(recordSizeFlag),
			false,
		)

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			return err
		}

		root, err := metadataPersister.GetRootPath(context.Background())
		if err != nil {
			return err
		}

		logger := logging.NewLogger()

		ops := operations.NewOperations(
			config.BackendConfig{
				GetWriter:   tm.GetWriter,
				CloseWriter: tm.Close,

				GetReader:   tm.GetReader,
				CloseReader: tm.Close,

				GetDrive:   tm.GetDrive,
				CloseDrive: tm.Close,
			},
			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			config.PipeConfig{
				Compression: viper.GetString(compressionFlag),
				Encryption:  viper.GetString(encryptionFlag),
				Signature:   viper.GetString(signatureFlag),
				RecordSize:  viper.GetInt(recordSizeFlag),
			},
			config.CryptoConfig{
				Recipient: recipient,
				Identity:  identity,
				Password:  viper.GetString(passwordFlag),
			},

			logger.PrintHeaderEvent,
		)

		stfs := sfs.NewFileSystem(
			ops,

			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			logger.PrintHeader,
		)

		fs, err := cache.Cache(
			stfs,
			root,
			viper.GetString(cacheFlag),
			viper.GetDuration(cacheDurationFlag),
			viper.GetString(cacheDirFlag),
		)
		if err != nil {
			return err
		}

		srv := ftpserver.NewFtpServer(
			&ftp.FTPServer{
				Settings: &ftpserver.Settings{
					ListenAddr: viper.GetString(laddrFlag),
				},
				FileSystem: fs,
			},
		)

		if viper.GetBool(verboseFlag) {
			srv.Logger = &ftp.Logger{}
		}

		log.Println("Listening on", viper.GetString(laddrFlag))

		return srv.ListenAndServe()
	},
}

func init() {
	serveFTPCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	serveFTPCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")
	serveFTPCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")
	serveFTPCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to the public key to verify with")
	serveFTPCmd.PersistentFlags().StringP(laddrFlag, "a", "localhost:1337", "Listen address")
	serveFTPCmd.PersistentFlags().StringP(cacheFlag, "n", config.NoneKey, fmt.Sprintf("Cache to use (default %v, available are %v)", config.NoneKey, cache.KnownCacheTypes))
	serveFTPCmd.PersistentFlags().DurationP(cacheDurationFlag, "u", time.Hour, "Duration until cache is invalidated")
	serveFTPCmd.PersistentFlags().StringP(cacheDirFlag, "w", filepath.Join(os.TempDir(), "stfs", "cache"), "Directory to use if dir cache is enabled")

	viper.AutomaticEnv()

	serveCmd.AddCommand(serveFTPCmd)
}