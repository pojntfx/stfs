package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/pojntfx/stfs/examples"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/keys"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/pojntfx/stfs/pkg/utility"
	"github.com/spf13/afero"
)

const (
	verbose            = true
	signaturePassword  = "testSignaturePassword"
	encryptionPassword = "testEncryptionPassword"
)

var (
	recordSizes              = []int{20, 60, 120}
	fileSystemCacheDurations = []time.Duration{time.Minute, time.Hour}
	stfsConfigs              = []stfsConfig{}
)

func init() {
	// TODO: Generate encryption and signature keys here and re-use them
	// for _, signature := range config.KnownSignatureFormats {
	// 	for _, encryption := range config.KnownEncryptionFormats {
	for _, compression := range config.KnownCompressionFormats {
		for _, compressionLevel := range config.KnownCompressionLevels {
			for _, writeCacheType := range config.KnownWriteCacheTypes {
				for _, fileSystemCacheType := range config.KnownFileSystemCacheTypes {
					for _, fileSystemCacheDuration := range fileSystemCacheDurations {
						for _, recordSize := range recordSizes {
							stfsConfigs = append(stfsConfigs, stfsConfig{
								recordSize,
								false,

								config.NoneKey, // encryption,
								config.NoneKey, // signature,
								compression,

								compressionLevel,

								writeCacheType,
								fileSystemCacheType,

								fileSystemCacheDuration,
							})
						}
					}
				}
			}
		}
	}
	// 	}
	// }
}

func createSTFS(
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
	signaturePrivkey := []byte{}
	signaturePubkey := []byte{}

	if signature != config.NoneKey {
		var err error
		signaturePrivkey, signaturePubkey, err = utility.Keygen(
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
	}

	signatureRecipient, err := keys.ParseSignerRecipient(signature, signaturePubkey)
	if err != nil {
		return nil, err
	}

	signatureIdentity, err := keys.ParseSignerIdentity(signature, signaturePrivkey, signaturePassword)
	if err != nil {
		return nil, err
	}

	encryptionPrivkey := []byte{}
	encryptionPubkey := []byte{}

	if encryption != config.NoneKey {
		encryptionPrivkey, encryptionPubkey, err = utility.Keygen(
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

	stfs := NewSTFS(
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
		false,
		readOnly,

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

type stfsConfig struct {
	recordSize int
	readOnly   bool

	signature   string
	encryption  string
	compression string

	compressionLevel string

	writeCache      string
	fileSystemCache string

	fileSystemCacheDuration time.Duration
}

type fsConfig struct {
	stfsConfig stfsConfig
	fs         afero.Fs
}

func createFss() ([]fsConfig, func() error, error) {
	fss := []fsConfig{}

	tmp, err := os.MkdirTemp(os.TempDir(), "stfs-test-*")
	if err != nil {
		return nil, nil, err
	}

	osfsDir := filepath.Join(tmp, "osfs")

	if err := os.MkdirAll(osfsDir, os.ModePerm); err != nil {
		return nil, nil, err
	}

	fss = append(fss, fsConfig{stfsConfig{}, afero.NewBasePathFs(afero.NewOsFs(), osfsDir)})

	for _, config := range stfsConfigs {
		tmp, err := os.MkdirTemp(os.TempDir(), "stfs-test-*")
		if err != nil {
			return nil, nil, err
		}

		drive := filepath.Join(tmp, "drive.tar")
		metadata := filepath.Join(tmp, "metadata.sqlite")

		writeCacheDir := filepath.Join(tmp, "write-cache")
		fileSystemCacheDir := filepath.Join(tmp, "filesystem-cache")

		stfs, err := createSTFS(
			drive,
			metadata,

			config.recordSize,
			config.readOnly,
			verbose,

			config.signature,
			signaturePassword,

			config.encryption,
			encryptionPassword,

			config.compression,
			config.compressionLevel,

			config.writeCache,
			writeCacheDir,

			config.fileSystemCache,
			fileSystemCacheDir,
			config.fileSystemCacheDuration,
		)
		if err != nil {
			return nil, nil, err
		}

		fss = append(fss, fsConfig{config, stfs})
	}

	return fss,
		func() error {
			return os.RemoveAll(tmp)
		},
		nil
}

func runForAllFss(t *testing.T, name string, action func(t *testing.T, fs fsConfig)) {
	fss, cleanup, err := createFss()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	for _, fs := range fss {
		t.Run(fmt.Sprintf(`%v for filesystem with config %v and name %v`, name, fs.stfsConfig, fs.fs.Name()), func(t *testing.T) {
			t.Parallel()

			action(t, fs)
		})
	}
}

func TestSTFS_Create(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Can create /test.txt",
			args{"/test.txt"},
			false,
		},
		// FIXME: STFS can create file in non-existent directory, which should not be possible
		// {
		// 	"Can not create /nonexistent/test.txt",
		// 	args{"/nonexistent/test.txt"},
		// 	true,
		// },
		// FIXME: STFS can create `/` file even if / exists
		// {
		// 	"Can create /",
		// 	args{"/"},
		// 	true,
		// },
	}

	for _, tt := range tests {
		runForAllFss(t, tt.name, func(t *testing.T, fs fsConfig) {
			file, err := fs.fs.Create(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Create() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			want, err := fs.fs.Stat(tt.args.name)
			if err != nil {
				t.Errorf("%v.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			got, err := fs.fs.Stat(file.Name())
			if err != nil {
				t.Errorf("%v.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("%v.Create().Name() = %v, want %v", fs.fs.Name(), got, want)

				return
			}
		})
	}
}
