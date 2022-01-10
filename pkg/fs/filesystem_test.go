package fs

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
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
	readOnly           = false
	verbose            = false
	signaturePassword  = "testSignaturePassword"
	encryptionPassword = "testEncryptionPassword"
)

var (
	recordSizes              = []int{20}                  // This leads to more permutations, but tests multiple: []int{20, 60, 120}
	fileSystemCacheDurations = []time.Duration{time.Hour} // This leads to more permutations, but tests multiple: []time.Duration{time.Minute, time.Hour}
	stfsConfigs              = []stfsConfig{}
)

type stfsConfig struct {
	recordSize int
	readOnly   bool

	signature          string
	signatureRecipient interface{}
	signatureIdentity  interface{}

	encryption          string
	encryptionRecipient interface{}
	encryptionIdentity  interface{}

	compression string

	compressionLevel string

	writeCache      string
	fileSystemCache string

	fileSystemCacheDuration time.Duration
}

type stfsPermutation struct {
	recordSize int
	readOnly   bool

	signature        string
	encryption       string
	compression      string
	compressionLevel string

	writeCache      string
	fileSystemCache string

	fileSystemCacheDuration time.Duration
}

type fsConfig struct {
	stfsConfig stfsConfig
	fs         afero.Fs
	cleanup    func() error
}

type cryptoConfig struct {
	name      string
	recipient interface{}
	identity  interface{}
}

func TestMain(m *testing.M) {
	flag.Parse() // So that `testing.Short` can be called, see https://go-review.googlesource.com/c/go/+/7604/
	if testing.Short() {
		for _, writeCacheType := range config.KnownWriteCacheTypes {
			for _, fileSystemCacheType := range config.KnownFileSystemCacheTypes {
				for _, fileSystemCacheDuration := range fileSystemCacheDurations {
					for _, recordSize := range recordSizes {
						stfsConfigs = append(stfsConfigs, stfsConfig{
							recordSize,
							readOnly,

							config.NoneKey,
							nil,
							nil,

							config.NoneKey,
							nil,
							nil,

							config.NoneKey,

							config.CompressionLevelFastestKey,

							writeCacheType,
							fileSystemCacheType,

							fileSystemCacheDuration,
						})
					}
				}
			}
		}
	} else {
		signatures := []cryptoConfig{}
		encryptions := []cryptoConfig{}

		cacheRoot := filepath.Join(os.TempDir(), "stfs-filesystem-test-key-cache")
		if err := os.MkdirAll(cacheRoot, os.ModePerm); err != nil {
			panic(err)
		}

		var wg sync.WaitGroup

		for _, signature := range config.KnownSignatureFormats {
			wg.Add(1)

			go func(signature string) {
				generateKeys := false
				signaturePrivkeyPath := filepath.Join(cacheRoot, "signature-key-"+signature+".priv")
				signaturePubkeyPath := filepath.Join(cacheRoot, "signature-key-"+signature+".pub")

				signaturePubkey := []byte{}
				signaturePrivkey, err := ioutil.ReadFile(signaturePrivkeyPath)
				if err == nil {
					signaturePubkey, err = ioutil.ReadFile(signaturePubkeyPath)
					if err != nil {
						generateKeys = true
					}
				} else {
					generateKeys = true
				}

				if signature != config.NoneKey && generateKeys {
					log.Println("Generating signature keys for format", signature)

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
						panic(err)
					}

					if err := ioutil.WriteFile(signaturePrivkeyPath, signaturePrivkey, os.ModePerm); err != nil {
						panic(err)
					}
					if err := ioutil.WriteFile(signaturePubkeyPath, signaturePubkey, os.ModePerm); err != nil {
						panic(err)
					}
				}

				signatureRecipient, err := keys.ParseSignerRecipient(signature, signaturePubkey)
				if err != nil {
					panic(err)
				}

				signatureIdentity, err := keys.ParseSignerIdentity(signature, signaturePrivkey, signaturePassword)
				if err != nil {
					panic(err)
				}

				signatures = append(signatures, cryptoConfig{signature, signatureRecipient, signatureIdentity})

				wg.Done()
			}(signature)
		}

		for _, encryption := range config.KnownEncryptionFormats {
			wg.Add(1)

			go func(encryption string) {
				generateKeys := false
				encryptionPrivkeyPath := filepath.Join(cacheRoot, "encryption-key-"+encryption+".priv")
				encryptionPubkeyPath := filepath.Join(cacheRoot, "encryption-key-"+encryption+".pub")

				encryptionPubkey := []byte{}
				encryptionPrivkey, err := ioutil.ReadFile(encryptionPrivkeyPath)
				if err == nil {
					encryptionPubkey, err = ioutil.ReadFile(encryptionPubkeyPath)
					if err != nil {
						generateKeys = true
					}
				} else {
					generateKeys = true
				}

				if encryption != config.NoneKey && generateKeys {
					log.Println("Generating encryption keys for format", encryption)

					var err error
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
						panic(err)
					}

					if err := ioutil.WriteFile(encryptionPrivkeyPath, encryptionPrivkey, os.ModePerm); err != nil {
						panic(err)
					}
					if err := ioutil.WriteFile(encryptionPubkeyPath, encryptionPubkey, os.ModePerm); err != nil {
						panic(err)
					}
				}

				encryptionRecipient, err := keys.ParseRecipient(encryption, encryptionPubkey)
				if err != nil {
					panic(err)
				}

				encryptionIdentity, err := keys.ParseIdentity(encryption, encryptionPrivkey, encryptionPassword)
				if err != nil {
					panic(err)
				}

				encryptions = append(encryptions, cryptoConfig{encryption, encryptionRecipient, encryptionIdentity})

				wg.Done()
			}(encryption)
		}

		wg.Wait()

		for _, signature := range signatures {
			for _, encryption := range encryptions {
				for _, compression := range config.KnownCompressionFormats {
					for _, compressionLevel := range config.KnownCompressionLevels {
						for _, writeCacheType := range config.KnownWriteCacheTypes {
							for _, fileSystemCacheType := range config.KnownFileSystemCacheTypes {
								for _, fileSystemCacheDuration := range fileSystemCacheDurations {
									for _, recordSize := range recordSizes {
										stfsConfigs = append(stfsConfigs, stfsConfig{
											recordSize,
											readOnly,

											signature.name,
											signature.recipient,
											signature.identity,

											encryption.name,
											encryption.recipient,
											encryption.identity,

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
			}
		}
	}

	log.Println("Starting filesystem tests for", len(stfsConfigs), "filesystem permutations")

	os.Exit(m.Run())
}

func createSTFS(
	drive string,
	metadata string,

	recordSize int,
	readOnly bool,
	verbose bool,

	signature string,
	signatureRecipient interface{},
	signatureIdentity interface{},

	encryption string,
	encryptionRecipient interface{},
	encryptionIdentity interface{},

	compression string,
	compressionLevel string,

	writeCache string,
	writeCacheDir string,

	fileSystemCache string,
	fileSystemCacheDir string,
	fileSystemCacheDuration time.Duration,

	initialize bool,
) (afero.Fs, error) {
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
		readOnly,

		func(hdr *config.Header) {
			jsonLogger.Trace("Header transform", hdr)
		},
		jsonLogger,
	)

	root := ""
	if initialize {
		var err error
		root, err = stfs.Initialize("/", os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	return cache.NewCacheFilesystem(
		stfs,
		root,
		fileSystemCache,
		fileSystemCacheDuration,
		fileSystemCacheDir,
	)
}

func createFss(initialize bool) ([]fsConfig, error) {
	fss := []fsConfig{}

	baseTmp, err := os.MkdirTemp(os.TempDir(), "stfs-test-*")
	if err != nil {
		return nil, err
	}

	tmp := filepath.Join(baseTmp, "osfs")

	if err := os.MkdirAll(tmp, os.ModePerm); err != nil {
		return nil, err
	}

	fss = append(fss, fsConfig{
		stfsConfig{},
		afero.NewBasePathFs(afero.NewOsFs(), tmp),
		func() error {
			return os.RemoveAll(tmp)
		},
	})

	for _, config := range stfsConfigs {
		tmp, err := os.MkdirTemp(baseTmp, "fs-*")
		if err != nil {
			return nil, err
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
			config.signatureRecipient,
			config.signatureIdentity,

			config.encryption,
			config.encryptionRecipient,
			config.encryptionIdentity,

			config.compression,
			config.compressionLevel,

			config.writeCache,
			writeCacheDir,

			config.fileSystemCache,
			fileSystemCacheDir,
			config.fileSystemCacheDuration,

			initialize,
		)
		if err != nil {
			return nil, err
		}

		fss = append(fss, fsConfig{
			config,
			stfs,
			func() error {
				return os.RemoveAll(tmp)
			},
		})
	}

	return fss, nil
}

func runTestForAllFss(t *testing.T, name string, initialize bool, action func(t *testing.T, fs fsConfig)) {
	fss, err := createFss(initialize)
	if err != nil {
		t.Fatal(err)

		return
	}

	for _, fs := range fss {
		t.Run(fmt.Sprintf(`%v filesystem=%v config=%v`, name, fs.fs.Name(), stfsPermutation{
			fs.stfsConfig.recordSize,
			fs.stfsConfig.readOnly,

			fs.stfsConfig.signature,
			fs.stfsConfig.encryption,
			fs.stfsConfig.compression,
			fs.stfsConfig.compressionLevel,

			fs.stfsConfig.writeCache,
			fs.stfsConfig.fileSystemCache,

			fs.stfsConfig.fileSystemCacheDuration,
		}), func(t *testing.T) {
			fs := fs

			t.Parallel()

			action(t, fs)

			if err := fs.cleanup(); err != nil {
				t.Fatal(err)

				return
			}
		})
	}
}

// func runBenchmarkForAllFss(b *testing.B, name string, initialize bool, action func(b *testing.B, fs fsConfig)) {
// 	fss, err := createFss(initialize)
// 	if err != nil {
// 		b.Fatal(err)

// 		return
// 	}

// 	for _, fs := range fss {
// 		b.Run(fmt.Sprintf(`%v filesystem=%v config=%v`, name, fs.fs.Name(), stfsPermutation{
// 			fs.stfsConfig.recordSize,
// 			fs.stfsConfig.readOnly,

// 			fs.stfsConfig.signature,
// 			fs.stfsConfig.encryption,
// 			fs.stfsConfig.compression,
// 			fs.stfsConfig.compressionLevel,

// 			fs.stfsConfig.writeCache,
// 			fs.stfsConfig.fileSystemCache,

// 			fs.stfsConfig.fileSystemCacheDuration,
// 		}), func(b *testing.B) {
// 			fs := fs

// 			action(b, fs)
// 		})

// 		if err := fs.cleanup(); err != nil {
// 			b.Fatal(err)

// 			return
// 		}
// 	}
// }

func TestSTFS_Name(t *testing.T) {
	fss, err := createFss(true)
	if err != nil {
		t.Fatal(err)

		return
	}
	defer func() {
		for _, fs := range fss {
			if err := fs.cleanup(); err != nil {
				t.Fatal(err)

				return
			}
		}
	}()

	tests := []struct {
		name string
		f    afero.Fs
		want string
	}{
		{
			"Returns correct filesystem name",
			fss[1].fs, // This is the first STFS config, [0] is the BasePathFs
			config.FileSystemNameSTFS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.Name(); got != tt.want {
				t.Errorf("STFS.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

type createArgs struct {
	name string
}

var createTests = []struct {
	name    string
	args    createArgs
	wantErr bool
}{
	{
		"Can create file /test.txt",
		createArgs{"/test.txt"},
		false,
	},
	{
		"Can not create existing file/directory /",
		createArgs{"/"},
		true,
	},
	{
		"Can not create file ' '",
		createArgs{" "},
		true,
	},
	{
		"Can not create file ''",
		createArgs{""},
		true,
	},
	// FIXME: STFS can create file in non-existent directory, which should not be possible
	// {
	// 	"Can not create /nonexistent/test.txt",
	// 	createArgs{"/nonexistent/test.txt"},
	// 	true,
	// },
}

func TestSTFS_Create(t *testing.T) {
	for _, tt := range createTests {
		runTestForAllFss(t, tt.name, true, func(t *testing.T, fs fsConfig) {
			file, err := fs.fs.Create(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Create() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				want, err := fs.fs.Stat(tt.args.name)
				if err != nil {
					t.Errorf("%v.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

					return
				}

				if file == nil {
					t.Errorf("%v.Create() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

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
			}
		})
	}
}

type initializeArgs struct {
	rootProposal string
	rootPerm     os.FileMode
}

var initializeTests = []struct {
	name     string
	args     initializeArgs
	wantRoot string
	wantErr  bool
}{
	{
		"Can create absolute root directory /",
		initializeArgs{"/", os.ModePerm},
		"/",
		false,
	},
	{
		"Can create absolute root directory /test",
		initializeArgs{"/test", os.ModePerm},
		"/test",
		false,
	},
	{
		"Can create absolute root directory /test/yes",
		initializeArgs{"/test/yes", os.ModePerm},
		"/test/yes",
		false,
	},
	{
		"Can create relative root directory ' '",
		initializeArgs{" ", os.ModePerm},
		" ",
		false,
	},
	{
		"Can create relative root directory ''",
		initializeArgs{"", os.ModePerm},
		"",
		false,
	},
	{
		"Can create relative root directory .",
		initializeArgs{".", os.ModePerm},
		".",
		false,
	},
	{
		"Can create relative root directory test",
		initializeArgs{"test", os.ModePerm},
		"test",
		false,
	},
	{
		"Can create absolute root directory test/yes",
		initializeArgs{"test/yes", os.ModePerm},
		"test/yes",
		false,
	},
}

func TestSTFS_Initialize(t *testing.T) {
	for _, tt := range initializeTests {
		runTestForAllFss(t, tt.name, false, func(t *testing.T, fs fsConfig) {
			f, ok := fs.fs.(*STFS)
			if !ok {
				if fs.fs.Name() == config.FileSystemNameSTFS {
					t.Fatal("Initialize function missing from filesystem")

					return
				}

				// Skip non-STFS filesystems
				return
			}

			gotRoot, err := f.Initialize(tt.args.rootProposal, tt.args.rootPerm)
			if (err != nil) != tt.wantErr {
				t.Errorf("STFS.Initialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRoot != tt.wantRoot {
				t.Errorf("STFS.Initialize() = %v, want %v", gotRoot, tt.wantRoot)
			}
		})
	}
}

type mkdirArgs struct {
	name string
	perm os.FileMode
}

var mkdirTests = []struct {
	name    string
	args    mkdirArgs
	wantErr bool
}{
	{
		"Can create directory /test.txt",
		mkdirArgs{"/test.txt", os.ModePerm},
		false,
	},
	{
		"Can create directory /test.txt with different permissions",
		mkdirArgs{"/test.txt", 0666},
		false,
	},
	{
		"Can not create existing directory /",
		mkdirArgs{"/", os.ModePerm},
		true,
	},
	{
		"Can not create directory ' '",
		mkdirArgs{" ", os.ModePerm},
		true,
	},
	{
		"Can not create directory ''",
		mkdirArgs{"", os.ModePerm},
		true,
	},
	// FIXME: STFS can create directory in non-existent directory, which should not be possible
	// {
	// 	"Can not create /nonexistent/test.txt",
	// 	mkdirArgs{"/nonexistent/test.txt", os.ModePerm},
	// 	true,
	// },
}

func TestSTFS_Mkdir(t *testing.T) {
	for _, tt := range mkdirTests {
		runTestForAllFss(t, tt.name, true, func(t *testing.T, fs fsConfig) {
			if err := fs.fs.Mkdir(tt.args.name, tt.args.perm); (err != nil) != tt.wantErr {
				t.Errorf("STFS.Mkdir() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				want, err := fs.fs.Stat(tt.args.name)
				if err != nil {
					t.Errorf("%v.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

					return
				}

				if want == nil {
					t.Errorf("%v.Stat() returned %v, want !nil", fs.fs.Name(), want)
				}
			}
		})
	}
}
