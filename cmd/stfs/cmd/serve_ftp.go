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

const (
	encryptionIdentityFlag  = "encryption-identity"
	encryptionPasswordFlag  = "encryption-password"
	encryptionRecipientFlag = "encryption-recipient"

	signatureIdentityFlag  = "signature-identity"
	signaturePasswordFlag  = "signature-password"
	signatureRecipientFlag = "signature-recipient"

	cacheWriteFlag = "cache-write-type"
)

var (
	cacheDir = filepath.Join(os.TempDir(), "stfs")
)

var serveFTPCmd = &cobra.Command{
	Use:     "ftp",
	Aliases: []string{"f"},
	Short:   "Serve tape or tar file and the index over FTP (read-write)",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := cache.CheckFileSystemCacheType(viper.GetString(cacheFileSystemFlag)); err != nil {
			return err
		}

		if err := cache.CheckWriteCacheType(viper.GetString(cacheWriteFlag)); err != nil {
			return err
		}

		if err := keys.CheckKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(encryptionIdentityFlag)); err != nil {
			return err
		}

		if err := keys.CheckKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(encryptionRecipientFlag)); err != nil {
			return err
		}

		if err := keys.CheckKeyAccessible(viper.GetString(signatureFlag), viper.GetString(signatureIdentityFlag)); err != nil {
			return err
		}

		return keys.CheckKeyAccessible(viper.GetString(signatureFlag), viper.GetString(signatureRecipientFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		signaturePubkey, err := keys.ReadKey(viper.GetString(signatureFlag), viper.GetString(signatureRecipientFlag))
		if err != nil {
			return err
		}

		signaturePrivkey, err := keys.ReadKey(viper.GetString(signatureFlag), viper.GetString(signatureIdentityFlag))
		if err != nil {
			return err
		}

		encryptionPubkey, err := keys.ReadKey(viper.GetString(encryptionFlag), viper.GetString(encryptionRecipientFlag))
		if err != nil {
			return err
		}

		encryptionPrivkey, err := keys.ReadKey(viper.GetString(encryptionFlag), viper.GetString(encryptionIdentityFlag))
		if err != nil {
			return err
		}

		signatureRecipient, err := keys.ParseSignerRecipient(viper.GetString(signatureFlag), signaturePubkey)
		if err != nil {
			return err
		}

		signatureIdentity, err := keys.ParseSignerIdentity(viper.GetString(signatureFlag), signaturePrivkey, viper.GetString(signaturePasswordFlag))
		if err != nil {
			return err
		}

		encryptionRecipient, err := keys.ParseRecipient(viper.GetString(encryptionFlag), encryptionPubkey)
		if err != nil {
			return err
		}

		encryptionIdentity, err := keys.ParseIdentity(viper.GetString(encryptionFlag), encryptionPrivkey, viper.GetString(encryptionPasswordFlag))
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

		readOps := operations.NewOperations(
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
				Recipient: signatureRecipient,
				Identity:  encryptionIdentity,
				Password:  viper.GetString(encryptionPasswordFlag),
			},

			logger.PrintHeaderEvent,
		)

		writeOps := operations.NewOperations(
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
				Recipient: encryptionRecipient,
				Identity:  signatureIdentity,
				Password:  viper.GetString(signaturePasswordFlag),
			},

			logger.PrintHeaderEvent,
		)

		stfs := sfs.NewFileSystem(
			readOps,
			writeOps,

			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			viper.GetString(compressionLevelFlag),
			func() (sfs.WriteCache, func() error, error) {
				return cache.NewCacheWrite(
					filepath.Join(viper.GetString(cacheDirFlag), "write"),
					viper.GetString(cacheWriteFlag),
				)
			},
			true, // FTP needs read permission for `STOR` command even if O_WRONLY is set

			logger.PrintHeader,
		)

		fs, err := cache.NewCacheFilesystem(
			stfs,
			root,
			viper.GetString(cacheFileSystemFlag),
			viper.GetDuration(cacheDurationFlag),
			filepath.Join(viper.GetString(cacheDirFlag), "filesystem"),
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

	serveFTPCmd.PersistentFlags().StringP(encryptionIdentityFlag, "i", "", "Path to private key to decrypt with")
	serveFTPCmd.PersistentFlags().StringP(encryptionPasswordFlag, "p", "", "Password for the private key to decrypt with")
	serveFTPCmd.PersistentFlags().StringP(encryptionRecipientFlag, "t", "", "Path to public key of recipient to encrypt with")

	serveFTPCmd.PersistentFlags().StringP(signatureIdentityFlag, "g", "", "Path to private key to sign with")
	serveFTPCmd.PersistentFlags().StringP(signaturePasswordFlag, "x", "", "Password for the private key to sign with")
	serveFTPCmd.PersistentFlags().StringP(signatureRecipientFlag, "r", "", "Path to the public key to verify with")

	serveFTPCmd.PersistentFlags().StringP(compressionLevelFlag, "l", config.CompressionLevelBalanced, fmt.Sprintf("Compression level to use (default %v, available are %v)", config.CompressionLevelBalanced, config.KnownCompressionLevels))
	serveFTPCmd.PersistentFlags().StringP(laddrFlag, "a", "localhost:1337", "Listen address")
	serveFTPCmd.PersistentFlags().StringP(cacheFileSystemFlag, "n", config.NoneKey, fmt.Sprintf("File system cache to use (default %v, available are %v)", config.NoneKey, cache.KnownFileSystemCacheTypes))
	serveFTPCmd.PersistentFlags().StringP(cacheWriteFlag, "q", cache.WriteCacheTypeFile, fmt.Sprintf("Write cache to use (default %v, available are %v)", cache.WriteCacheTypeFile, cache.KnownWriteCacheTypes))
	serveFTPCmd.PersistentFlags().DurationP(cacheDurationFlag, "u", time.Hour, "Duration until cache is invalidated")
	serveFTPCmd.PersistentFlags().StringP(cacheDirFlag, "w", cacheDir, "Directory to use if dir cache is enabled")

	viper.AutomaticEnv()

	serveCmd.AddCommand(serveFTPCmd)
}
