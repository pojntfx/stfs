package cmd

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/pojntfx/stfs/internal/check"
	"github.com/pojntfx/stfs/internal/handlers"
	"github.com/pojntfx/stfs/internal/keyext"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	sfs "github.com/pojntfx/stfs/pkg/fs"
	"github.com/pojntfx/stfs/pkg/keys"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	laddrFlag           = "laddr"
	cacheFileSystemFlag = "cache-filesystem-type"
	cacheDirFlag        = "cache-dir"
	cacheDurationFlag   = "cache-duration"
)

var serveHTTPCmd = &cobra.Command{
	Use:     "http",
	Aliases: []string{"htt", "h"},
	Short:   "Serve tape or tar file and the index over HTTP (read-only)",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if err := check.CheckFileSystemCacheType(viper.GetString(cacheFileSystemFlag)); err != nil {
			return err
		}

		if err := check.CheckKeyAccessible(viper.GetString(encryptionFlag), viper.GetString(identityFlag)); err != nil {
			return err
		}

		return check.CheckKeyAccessible(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pubkey, err := keyext.ReadKey(viper.GetString(signatureFlag), viper.GetString(recipientFlag))
		if err != nil {
			return err
		}

		recipient, err := keys.ParseSignerRecipient(viper.GetString(signatureFlag), pubkey)
		if err != nil {
			return err
		}

		privkey, err := keyext.ReadKey(viper.GetString(encryptionFlag), viper.GetString(identityFlag))
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

		jsonLogger := logging.NewJSONLogger(viper.GetInt(verboseFlag))

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
				Recipient: recipient,
				Identity:  identity,
				Password:  viper.GetString(passwordFlag),
			},

			func(event *config.HeaderEvent) {
				jsonLogger.Debug("Header read", event)
			},
		)

		stfs := sfs.NewSTFS(
			readOps,
			nil,

			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			"",    // We never write
			nil,   // We never write
			false, // We never write

			func(hdr *config.Header) {
				jsonLogger.Trace("Header transform", hdr)
			},
			jsonLogger,
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

		jsonLogger.Info("HTTP server listening", map[string]interface{}{
			"laddr": viper.GetString(laddrFlag),
		})

		return http.ListenAndServe(
			viper.GetString(laddrFlag),
			handlers.PanicHandler(
				http.FileServer(
					afero.NewHttpFs(fs),
				),
				jsonLogger,
			),
		)
	},
}

func init() {
	serveHTTPCmd.PersistentFlags().IntP(recordSizeFlag, "z", 20, "Amount of 512-bit blocks per record")
	serveHTTPCmd.PersistentFlags().StringP(identityFlag, "i", "", "Path to private key of recipient that has been encrypted for")
	serveHTTPCmd.PersistentFlags().StringP(passwordFlag, "p", "", "Password for the private key")
	serveHTTPCmd.PersistentFlags().StringP(recipientFlag, "r", "", "Path to the public key to verify with")
	serveHTTPCmd.PersistentFlags().StringP(laddrFlag, "a", "localhost:1337", "Listen address")
	serveHTTPCmd.PersistentFlags().StringP(cacheFileSystemFlag, "n", config.NoneKey, fmt.Sprintf("File system cache to use (default %v, available are %v)", config.NoneKey, config.KnownFileSystemCacheTypes))
	serveHTTPCmd.PersistentFlags().DurationP(cacheDurationFlag, "u", time.Hour, "Duration until cache is invalidated")
	serveHTTPCmd.PersistentFlags().StringP(cacheDirFlag, "w", cacheDir, "Directory to use if dir cache is enabled")

	viper.AutomaticEnv()

	serveCmd.AddCommand(serveHTTPCmd)
}
