package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pojntfx/stfs/examples"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/fs"
	"github.com/pojntfx/stfs/pkg/keys"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/pojntfx/stfs/pkg/utility"
	"github.com/spf13/afero"
)

func createFs(
	drive string,
	metadata string,

	recordSize int,
	readOnly bool,
	verbose bool,

	signature string,
	signaturePassword string,

	encryption string,
	encryptionPassword string,

	compression string,
	compressionLevel string,

	writeCache string,
	writeCacheDir string,

	fileSystemCache string,
	fileSystemCacheDir string,
	fileSystemCacheDuration time.Duration,
) (afero.Fs, error) {
	signaturePrivkey, signaturePubkey, err := utility.Keygen(
		config.PipeConfig{
			Signature:  signature,
			Encryption: config.NoneKey,
		},
		config.PasswordConfig{
			Password: signaturePassword,
		},
	)
	if err != nil {
		return nil, err
	}

	signatureRecipient, err := keys.ParseSignerRecipient(signature, signaturePubkey)
	if err != nil {
		return nil, err
	}

	signatureIdentity, err := keys.ParseSignerIdentity(signature, signaturePrivkey, signaturePassword)
	if err != nil {
		return nil, err
	}

	encryptionPrivkey, encryptionPubkey, err := utility.Keygen(
		config.PipeConfig{
			Signature:  config.NoneKey,
			Encryption: encryption,
		},
		config.PasswordConfig{
			Password: encryptionPassword,
		},
	)
	if err != nil {
		return nil, err
	}

	encryptionRecipient, err := keys.ParseRecipient(encryption, encryptionPubkey)
	if err != nil {
		return nil, err
	}

	encryptionIdentity, err := keys.ParseIdentity(encryption, encryptionPrivkey, encryptionPassword)
	if err != nil {
		return nil, err
	}

	tm := tape.NewTapeManager(
		drive,
		recordSize,
		false,
	)

	metadataPersister := persisters.NewMetadataPersister(metadata)
	if err := metadataPersister.Open(); err != nil {
		return nil, err
	}

	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	metadataConfig := config.MetadataConfig{
		Metadata: metadataPersister,
	}
	pipeConfig := config.PipeConfig{
		Compression: compression,
		Encryption:  encryption,
		Signature:   signature,
		RecordSize:  recordSize,
	}
	backendConfig := config.BackendConfig{
		GetWriter:   tm.GetWriter,
		CloseWriter: tm.Close,

		GetReader:   tm.GetReader,
		CloseReader: tm.Close,

		GetDrive:   tm.GetDrive,
		CloseDrive: tm.Close,
	}
	readCryptoConfig := config.CryptoConfig{
		Recipient: signatureRecipient,
		Identity:  encryptionIdentity,
		Password:  encryptionPassword,
	}

	readOps := operations.NewOperations(
		backendConfig,
		metadataConfig,

		pipeConfig,
		readCryptoConfig,

		func(event *config.HeaderEvent) {
			jsonLogger.Debug("Header read", event)
		},
	)

	writeOps := operations.NewOperations(
		backendConfig,
		metadataConfig,

		pipeConfig,
		config.CryptoConfig{
			Recipient: encryptionRecipient,
			Identity:  signatureIdentity,
			Password:  signaturePassword,
		},

		func(event *config.HeaderEvent) {
			jsonLogger.Debug("Header write", event)
		},
	)

	stfs := fs.NewSTFS(
		readOps,
		writeOps,

		config.MetadataConfig{
			Metadata: metadataPersister,
		},

		compressionLevel,
		func() (cache.WriteCache, func() error, error) {
			return cache.NewCacheWrite(
				writeCacheDir,
				writeCache,
			)
		},
		readOnly,
		false,

		func(hdr *config.Header) {
			jsonLogger.Trace("Header transform", hdr)
		},
		jsonLogger,
	)

	root, err := stfs.Initialize("/", os.ModePerm)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheFilesystem(
		stfs,
		root,
		fileSystemCache,
		fileSystemCacheDuration,
		fileSystemCacheDir,
	)
}

func main() {
	tmp, err := os.MkdirTemp(os.TempDir(), "stfs-test-*")
	if err != nil {
		panic(err)
	}

	drive := filepath.Join(tmp, "drive.tar")
	metadata := filepath.Join(tmp, "metadata.sqlite")

	recordSize := 20
	readOnly := false
	verbose := true

	signature := config.SignatureFormatPGPKey
	signaturePassword := "testSignaturePassword"

	encryption := config.EncryptionFormatAgeKey
	encryptionPassword := "testEncryptionPassword"

	compression := config.CompressionFormatZStandardKey
	compressionLevel := config.CompressionLevelFastestKey

	writeCache := config.WriteCacheTypeFile
	writeCacheDir := filepath.Join(tmp, "write-cache")

	fileSystemCache := config.FileSystemCacheTypeDir
	fileSystemCacheDir := filepath.Join(tmp, "filesystem-cache")
	fileSystemCacheDuration := time.Hour

	fs, err := createFs(
		drive,
		metadata,

		recordSize,
		readOnly,
		verbose,

		signature,
		signaturePassword,

		encryption,
		encryptionPassword,

		compression,
		compressionLevel,

		writeCache,
		writeCacheDir,

		fileSystemCache,
		fileSystemCacheDir,
		fileSystemCacheDuration,
	)
	if err != nil {
		panic(err)
	}

	log.Println("stat /")

	stat, err := fs.Stat("/")
	if err != nil {
		panic(err)
	}

	log.Println("Result of stat /:", stat)
}
