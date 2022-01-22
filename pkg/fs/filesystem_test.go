package fs

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pojntfx/stfs/examples"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/keys"
	"github.com/pojntfx/stfs/pkg/mtio"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/pojntfx/stfs/pkg/utility"
	"github.com/spf13/afero"
	"github.com/volatiletech/sqlboiler/v4/boil"
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

type symFs interface {
	afero.Fs
	afero.Symlinker
}

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

	if verbose {
		boil.DebugMode = false
		boil.DebugWriter = os.Stderr
	}

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
	mt := mtio.MagneticTapeIO{}
	tm := tape.NewTapeManager(
		drive,
		mt,
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

		MagneticTapeIO: mt,
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
		false,

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

func createFss(initialize bool, withCache bool, withOsFs bool) ([]fsConfig, error) {
	fss := []fsConfig{}

	baseTmp, err := os.MkdirTemp(os.TempDir(), "stfs-test-*")
	if err != nil {
		return nil, err
	}

	if withOsFs {
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
	}

	for _, cfg := range stfsConfigs {
		if !withCache && cfg.fileSystemCache != config.NoneKey {
			continue
		}

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

			cfg.recordSize,
			cfg.readOnly,
			verbose,

			cfg.signature,
			cfg.signatureRecipient,
			cfg.signatureIdentity,

			cfg.encryption,
			cfg.encryptionRecipient,
			cfg.encryptionIdentity,

			cfg.compression,
			cfg.compressionLevel,

			cfg.writeCache,
			writeCacheDir,

			cfg.fileSystemCache,
			fileSystemCacheDir,
			cfg.fileSystemCacheDuration,

			initialize,
		)
		if err != nil {
			return nil, err
		}

		fss = append(fss, fsConfig{
			cfg,
			stfs,
			func() error {
				return os.RemoveAll(tmp)
			},
		})
	}

	return fss, nil
}

func runTestForAllFss(t *testing.T, name string, initialize bool, withCache bool, withOsFs bool, action func(t *testing.T, fs fsConfig)) {
	fss, err := createFss(initialize, withCache, withOsFs)
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
	fss, err := createFss(true, true, true)
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.Name(); got != tt.want {
				t.Errorf("%v.Name() = %v, want %v", t.Name(), got, tt.want)
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
	prepare func(symFs) error
}{
	{
		"Can create file /test.txt",
		createArgs{"/test.txt"},
		false,
		func(sf symFs) error { return nil },
	},
	{
		"Can create file /test.txt/",
		createArgs{"/test.txt/"},
		false,
		func(sf symFs) error { return nil },
	},
	{
		"Can not create existing file /",
		createArgs{"/"},
		true,
		func(sf symFs) error { return nil },
	},
	{
		"Can create file ' '",
		createArgs{" "},
		false,
		func(sf symFs) error { return nil },
	},
	{
		"Can create file ''",
		createArgs{""},
		true,
		func(sf symFs) error { return nil },
	},
	{
		"Can not create /nonexistent/test.txt",
		createArgs{"/nonexistent/test.txt"},
		true,
		func(sf symFs) error { return nil },
	},
	{
		"Can not create file in place of existing directory /mydir",
		createArgs{"/mydir"},
		true,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
	},
	{
		"Can not create file in place of symlink to root",
		createArgs{"/existingsymlink"},
		true,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
	},
	{
		"Can create file in place of broken symlink /brokensymlink",
		createArgs{"/brokensymlink"},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/test.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
	},
	{
		"Can create file in place of existing symlink /existingsymlink to file",
		createArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			file, err := sf.Create("/test.txt")
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
	},
	{
		"Can not create file in place of existing symlink /existingsymlink to directory",
		createArgs{"/existingsymlink"},
		true,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
	},
}

func TestSTFS_Create(t *testing.T) {
	for _, tt := range createTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, true, true, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			file, err := symFs.Create(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Create() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				want, err := symFs.Stat(tt.args.name)
				if err != nil {
					t.Errorf("%v.Stat() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}

				if file == nil {
					t.Errorf("%v.Create() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}

				got, err := symFs.Stat(file.Name())
				if err != nil {
					t.Errorf("%v.Stat() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}

				if !reflect.DeepEqual(got, want) {
					t.Errorf("%v.Create().Name() = %v, want %v", symFs.Name(), got, want)

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
		tt := tt

		runTestForAllFss(t, tt.name, false, true, true, func(t *testing.T, fs fsConfig) {
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
				t.Errorf("%v.Initialize() error = %v, wantErr %v", f.Name(), err, tt.wantErr)

				return
			}
			if gotRoot != tt.wantRoot {
				t.Errorf("%v.Initialize() = %v, want %v", f.Name(), gotRoot, tt.wantRoot)
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
	prepare func(symFs) error
}{
	{
		"Can create directory /test",
		mkdirArgs{"/test", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
	},
	{
		"Can create directory /test with different permissions",
		mkdirArgs{"/test", 0666},
		false,
		func(sf symFs) error { return nil },
	},
	{
		"Can not create existing directory /",
		mkdirArgs{"/", os.ModePerm},
		true,
		func(sf symFs) error { return nil },
	},
	{
		"Can create directory ' '",
		mkdirArgs{" ", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
	},
	{
		"Can create directory ''",
		mkdirArgs{"", os.ModePerm},
		true,
		func(sf symFs) error { return nil },
	},
	{
		"Can not create /nonexistent/test",
		mkdirArgs{"/nonexistent/test", os.ModePerm},
		true,
		func(sf symFs) error { return nil },
	},
	{
		"Can not create directory in place of existing directory /mydir",
		mkdirArgs{"/mydir", os.ModePerm},
		true,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
	},
	{
		"Can not create directory in place of existing existing file /myfile",
		mkdirArgs{"/myfile", os.ModePerm},
		true,
		func(sf symFs) error {
			file, err := sf.Create("/myfile")
			if err != nil {
				return err
			}

			return file.Close()
		},
	},
	{
		"Can not create directory in place of symlink to root",
		mkdirArgs{"/existingsymlink", os.ModePerm},
		true,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
	},
	{
		"Can not create directory in place of broken symlink /brokensymlink",
		mkdirArgs{"/brokensymlink", os.ModePerm},
		true,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/test.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
	},
	{
		"Can not create directory in place of existing symlink /existingsymlink to directory",
		mkdirArgs{"/existingsymlink", os.ModePerm},
		true,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
	},
}

func TestSTFS_Mkdir(t *testing.T) {
	for _, tt := range mkdirTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, true, true, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			if err := symFs.Mkdir(tt.args.name, tt.args.perm); (err != nil) != tt.wantErr {
				t.Errorf("%v.Mkdir() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)
			}

			if !tt.wantErr {
				want, err := symFs.Stat(tt.args.name)
				if err != nil {
					t.Errorf("%v.Stat() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}

				if want == nil {
					t.Errorf("%v.Stat() returned %v, want !nil", symFs.Name(), want)
				}
			}
		})
	}
}

type mkdirAllArgs struct {
	name string
	perm os.FileMode
}

var mkdirAllTests = []struct {
	name    string
	args    mkdirAllArgs
	wantErr bool
	prepare func(symFs) error
	lstat   bool
}{
	{
		"Can create directory /test.txt",
		mkdirAllArgs{"/test.txt", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can create directory /test.txt with different permissions",
		mkdirAllArgs{"/test.txt", 0666},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can create existing directory /",
		mkdirAllArgs{"/", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can create directory ' '",
		mkdirAllArgs{" ", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can create directory ''",
		mkdirAllArgs{"", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can create /nonexistent/test.txt",
		mkdirAllArgs{"/nonexistent/test.txt", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can create /nested/second/test.txt",
		mkdirAllArgs{"/nested/second/test.txt", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can create /nested//test.txt",
		mkdirAllArgs{"/nested//test.txt", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can create ///test.txt",
		mkdirAllArgs{"///test.txt", os.ModePerm},
		false,
		func(sf symFs) error { return nil },
		false,
	},
	{
		"Can not create directory in place of existing existing file /myfile",
		mkdirAllArgs{"/myfile", os.ModePerm},
		true,
		func(sf symFs) error {
			file, err := sf.Create("/myfile")
			if err != nil {
				return err
			}

			return file.Close()
		},
		false,
	},
	{
		"Can create directory in place of existing directory /mydir",
		mkdirAllArgs{"/mydir", os.ModePerm},
		false,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can create directory in place of symlink to root",
		mkdirAllArgs{"/existingsymlink", os.ModePerm},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		true,
	},
	{
		"Can not create directory in place of broken symlink /brokensymlink",
		mkdirAllArgs{"/brokensymlink", os.ModePerm},
		true,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/test.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
		true,
	},
	{
		"Can not create directory in place of existing symlink /existingsymlink to directory",
		mkdirAllArgs{"/existingsymlink", os.ModePerm},
		false,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		true,
	},
}

func TestSTFS_MkdirAll(t *testing.T) {
	for _, tt := range mkdirAllTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, true, true, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			if err := symFs.MkdirAll(tt.args.name, tt.args.perm); (err != nil) != tt.wantErr {
				t.Errorf("%v.MkdirAll() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)
			}

			if !tt.wantErr {
				if tt.lstat {
					want, _, err := symFs.LstatIfPossible(tt.args.name)
					if err != nil {
						t.Errorf("%v.LstatIfPossible() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

						return
					}

					if want == nil {
						t.Errorf("%v.LstatIfPossible() returned %v, want !nil", symFs.Name(), want)
					}
				} else {
					want, err := symFs.Stat(tt.args.name)
					if err != nil {
						t.Errorf("%v.Stat() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

						return
					}

					if want == nil {
						t.Errorf("%v.Stat() returned %v, want !nil", symFs.Name(), want)
					}
				}
			}
		})
	}
}

type openArgs struct {
	name string
}

var openTests = []struct {
	name    string
	args    openArgs
	wantErr bool
	prepare func(symFs) error
	check   func(afero.File) error
}{
	{
		"Can open /",
		openArgs{"/"},
		false,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
	},
	{
		"Can not open ' '",
		openArgs{" "},
		true,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
	},
	{
		"Can open ''",
		openArgs{""},
		false,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
	},
	{
		"Can not open /test.txt without creating it",
		openArgs{"/test.txt"},
		true,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
	},
	{
		"Can open /test.txt after creating it",
		openArgs{"/test.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			want := "/test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
	},
	{
		"Can not open /mydir/test.txt without creating it",
		openArgs{"/mydir/test.txt"},
		true,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
	},
	{
		"Can open /mydir after creating it",
		openArgs{"/mydir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			want := "/mydir"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
	},
	{
		"Can open /mydir/test.txt after creating it",
		openArgs{"/mydir/test.txt"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			want := "/mydir/test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
	},
	{
		"Can open symlink to root",
		openArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f afero.File) error { return nil },
	},
	{
		"Can not open broken symlink to /test.txt",
		openArgs{"/brokensymlink"},
		true,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/test.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f afero.File) error { return nil },
	},
	{
		"Can open symlink /existingsymlink to directory",
		openArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f afero.File) error { return nil },
	},
	{
		"Can open symlink /existingsymlink to file",
		openArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			file, err := sf.Create("test.txt")
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f afero.File) error { return nil },
	},
}

func TestSTFS_Open(t *testing.T) {
	for _, tt := range openTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, true, true, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			got, err := symFs.Open(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Open() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", symFs.Name(), err)

				return
			}
		})
	}
}

type openFileArgs struct {
	name string
	flag int
	perm os.FileMode
}

var openFileTests = []struct {
	name            string
	args            openFileArgs
	wantErr         bool
	prepare         func(symFs) error
	check           func(afero.File) error
	checkAfterError bool
	withCache       bool
	withOsFs        bool
}{
	{
		"Can open /",
		openFileArgs{"/", os.O_RDONLY, 0},
		false,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
		false,
		false, // FIXME: Can't open this with in-memory or file cache (will need a upstream fix in CacheOnReadFs)
		true,
	},
	{
		"Can not open /test.txt without creating it",
		openFileArgs{"/test.txt", os.O_RDONLY, 0},
		true,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can open /test.txt if O_CREATE is set",
		openFileArgs{"/test.txt", os.O_CREATE, os.ModePerm},
		false,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can open /test.txt after creating it",
		openFileArgs{"/test.txt", os.O_RDONLY, 0},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			want := "/test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not open /mydir/test.txt without creating it",
		openFileArgs{"/mydir/test.txt", os.O_RDONLY, 0},
		true,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can not open /mydir/test.txt if O_CREATE is set",
		openFileArgs{"/mydir/test.txt", os.O_CREATE, os.ModePerm},
		true,
		func(f symFs) error { return nil },
		func(f afero.File) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can open /mydir/test.txt after creating it",
		openFileArgs{"/mydir/test.txt", os.O_RDONLY, 0},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			want := "/mydir/test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not write to /test.txt if O_RDONLY is set",
		openFileArgs{"/test.txt", os.O_RDONLY, 0},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			want := "/test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			if _, err := f.Write([]byte("test content")); err == nil {
				return errors.New("could write to read-only file")
			}

			return nil
		},
		true,
		true,
		true,
	},
	{
		"Can write to /test.txt if O_WRONLY is set",
		openFileArgs{"/test.txt", os.O_WRONLY, 0},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			want := "/test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			if _, err := f.Write([]byte("test content")); err != nil {
				return errors.New("could not write to write-only file")
			}

			return nil
		},
		true,
		true,
		true,
	},
	{
		"Can write to /test.txt if O_RDWR is set",
		openFileArgs{"/test.txt", os.O_RDWR, 0},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			want := "/test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			if _, err := f.Write([]byte("test content")); err != nil {
				return errors.New("could not write to read-write file")
			}

			return nil
		},
		true,
		true,
		true,
	},
	{
		"Can open symlink to root",
		openFileArgs{"/existingsymlink", os.O_RDONLY, 0},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f afero.File) error { return nil },
		true,
		true,
		true,
	},
	{
		"Can not open broken symlink to /test.txt",
		openFileArgs{"/brokensymlink", os.O_RDONLY, 0},
		true,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/test.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f afero.File) error { return nil },
		true,
		true,
		true,
	},
	{
		"Can open symlink /existingsymlink to directory",
		openFileArgs{"/existingsymlink", os.O_RDONLY, 0},
		false,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f afero.File) error { return nil },
		true,
		true,
		true,
	},
	{
		"Can open symlink /existingsymlink to file",
		openFileArgs{"/existingsymlink", os.O_RDONLY, 0},
		false,
		func(sf symFs) error {
			file, err := sf.Create("/test.txt")
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f afero.File) error { return nil },
		true,
		true,
		true,
	},
}

func TestSTFS_OpenFile(t *testing.T) {
	for _, tt := range openFileTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			got, err := symFs.OpenFile(tt.args.name, tt.args.flag, tt.args.perm)
			if (err != nil) != tt.wantErr {
				if !tt.checkAfterError {
					t.Errorf("%v.OpenFile() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", symFs.Name(), err)

				return
			}
		})
	}
}

type removeArgs struct {
	name string
}

var removeTests = []struct {
	name            string
	args            removeArgs
	wantErr         bool
	prepare         func(symFs) error
	check           func(symFs) error
	checkAfterError bool
}{
	{
		"Can remove /",
		removeArgs{"/"},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can remove ''",
		removeArgs{""},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can not remove ' '",
		removeArgs{" "},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can not remove /test.txt if does not exist",
		removeArgs{"/test.txt"},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can remove /test.txt if does exist",
		removeArgs{"/test.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can not remove /mydir/test.txt if does not exist",
		removeArgs{"/mydir/test.txt"},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can not remove /mydir/test.txt if does not exist, but the parent exists",
		removeArgs{"/mydir/test.txt"},
		true,
		func(f symFs) error {
			return f.Mkdir("/mydir", os.ModePerm)
		},
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can remove /mydir/test.txt if does exist",
		removeArgs{"/mydir/test.txt"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove /mydir if it is a directory and empty",
		removeArgs{"/mydir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			return nil
		},
		false,
	},
	{
		"Can not remove /mydir if it is a directory and not empty",
		removeArgs{"/mydir"},
		true,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			return nil
		},
		false,
	},
	{
		"Can not remove /mydir/subdir if it is a directory and not empty",
		removeArgs{"/mydir/subdir"},
		true,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir/subdir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/subdir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir/subdir/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/mydir/subdir"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove /mydir/subdir if it is a directory and empty",
		removeArgs{"/mydir/subdir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir/subdir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir/subdir"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove symlink to root",
		removeArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/"); err != nil {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove broken symlink to /test.txt",
		removeArgs{"/brokensymlink"},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/test.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/brokensymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove symlink /existingsymlink to directory without removing the link's target",
		removeArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/mydir"); err != nil {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove symlink /existingsymlink to file without removing the link's target",
		removeArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			file, err := sf.Create("/test.txt")
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		false,
	},
}

func TestSTFS_Remove(t *testing.T) {
	for _, tt := range removeTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, true, true, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			if err := symFs.Remove(tt.args.name); (err != nil) != tt.wantErr {
				if !tt.checkAfterError {
					t.Errorf("%v.Remove() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}
			}

			if err := tt.check(symFs); err != nil {
				t.Errorf("%v check() error = %v", symFs.Name(), err)

				return
			}
		})
	}
}

type removeAllArgs struct {
	name string
}

var removeAllTests = []struct {
	name            string
	args            removeAllArgs
	wantErr         bool
	prepare         func(symFs) error
	check           func(symFs) error
	checkAfterError bool
}{
	{
		"Can remove /",
		removeAllArgs{"/"},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can remove ''",
		removeAllArgs{""},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can not remove ' '",
		removeAllArgs{" "},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can not remove /test.txt if does not exist",
		removeAllArgs{"/test.txt"},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can remove /test.txt if does exist",
		removeAllArgs{"/test.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can not remove /mydir/test.txt if does not exist",
		removeAllArgs{"/mydir/test.txt"},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can not remove /mydir/test.txt if does not exist, but the parent exists",
		removeAllArgs{"/mydir/test.txt"},
		false,
		func(f symFs) error {
			return f.Mkdir("/mydir", os.ModePerm)
		},
		func(f symFs) error { return nil },
		false,
	},
	{
		"Can remove /mydir/test.txt if does exist",
		removeAllArgs{"/mydir/test.txt"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove /mydir if it is a directory and empty",
		removeAllArgs{"/mydir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			return nil
		},
		false,
	},
	{
		"Can not remove /mydir if it is a directory and not empty",
		removeAllArgs{"/mydir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			return nil
		},
		false,
	},
	{
		"Can not remove /mydir/subdir if it is a directory and not empty",
		removeAllArgs{"/mydir/subdir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir/subdir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/subdir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir/subdir/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/mydir/subdir"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove /mydir/subdir if it is a directory and empty",
		removeAllArgs{"/mydir/subdir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir/subdir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir/subdir"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove symlink to root",
		removeAllArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/"); err != nil {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove broken symlink to /test.txt",
		removeAllArgs{"/brokensymlink"},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/test.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/brokensymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove symlink /existingsymlink to directory without removing the link's target",
		removeAllArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/mydir"); err != nil {
				return err
			}

			return nil
		},
		false,
	},
	{
		"Can remove symlink /existingsymlink to file without removing the link's target",
		removeAllArgs{"/existingsymlink"},
		false,
		func(sf symFs) error {
			file, err := sf.Create("/test.txt")
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		false,
	},
}

func TestSTFS_RemoveAll(t *testing.T) {
	for _, tt := range removeAllTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, true, true, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			if err := symFs.RemoveAll(tt.args.name); (err != nil) != tt.wantErr {
				if !tt.checkAfterError {
					t.Errorf("%v.RemoveAll() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}
			}

			if err := tt.check(symFs); err != nil {
				t.Errorf("%v check() error = %v", symFs.Name(), err)

				return
			}
		})
	}
}

type renameArgs struct {
	oldname string
	newname string
}

var renameTests = []struct {
	name            string
	args            renameArgs
	wantErr         bool
	prepare         func(symFs) error
	check           func(symFs) error
	checkAfterError bool
	withCache       bool
	withOsFs        bool
}{
	{
		"Can not rename / to /mydir",
		renameArgs{"/", "/mydri"},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can not rename / to /",
		renameArgs{"/", "/"},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can not rename '' to ''",
		renameArgs{"", ""},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can not rename remove ' ' to ' '",
		renameArgs{" ", " "},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can rename /test.txt to /new.txt if does exist",
		renameArgs{"/test.txt", "/new.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			info, err := f.Stat("/new.txt")
			if err != nil {
				return err
			}

			want := "new.txt"
			got := info.Name()

			if want != got {
				return fmt.Errorf("renamed file has wrong name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not rename /test.txt to /new.txt if does exist",
		renameArgs{"/test.txt", "/new.txt"},
		true,
		func(f symFs) error {
			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/new.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can move empty directory /myolddir to /mydir",
		renameArgs{"/myolddir", "/mydir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/myolddir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
		false, // FIXME: Can't rename with in-memory or file cache (will need a upstream fix in CacheOnReadFs; `error = is a directory`)
		true,
	},
	{
		"Can move non-empty directory /myolddir to /mydir",
		renameArgs{"/myolddir", "/mydir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/myolddir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/myolddir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
		false, // FIXME: Can't rename with in-memory or file cache (will need a upstream fix in CacheOnReadFs; `error = is a directory`)
		true,
	},
	{
		"Can not rename /test.txt to /mydir/new.txt if new parent drectory does not exist",
		renameArgs{"/test.txt", "/mydir/new.txt"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/mydir/new.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can rename /test.txt to /mydir/new.txt if new parent drectory does exist",
		renameArgs{"/test.txt", "/mydir/new.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			info, err := f.Stat("/mydir/new.txt")
			if err != nil {
				return err
			}

			want := "new.txt"
			got := info.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can rename /test.txt to /test.txt if does exist",
		renameArgs{"/test.txt", "/test.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not rename move /test.txt to /test.txt if does not exist",
		renameArgs{"/test.txt", "/test.txt"},
		true,
		func(f symFs) error {
			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can rename /test.txt to /existing.txt if source and target both exist",
		renameArgs{"/test.txt", "/existing.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/existing.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, err := f.Stat("/test.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/existing.txt"); err != nil {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not rename /test.txt to /mydir if source is file and target is directory",
		renameArgs{"/test.txt", "/mydir"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not rename /mydir to /test.txt if source is directory and target is file",
		renameArgs{"/mydir", "/test.txt"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can rename /test.txt to /test.txt/",
		renameArgs{"/test.txt", "/test.txt/"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can rename symlink /existingsymlink to root to /newexistingsymlink",
		renameArgs{"/existingsymlink", "/newexistingsymlink"},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/"); err != nil {
				return err
			}

			if _, _, err := f.LstatIfPossible("/newexistingsymlink"); err != nil {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can rename broken symlink /brokensymlink to /test.txt to /newbrokensymlink",
		renameArgs{"/brokensymlink", "/newbrokensymlink"},
		false,
		func(sf symFs) error {
			if err := sf.SymlinkIfPossible("/test.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/brokensymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, _, err := f.LstatIfPossible("/newbrokensymlink"); err != nil {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can rename symlink /existingsymlink to directory to /newexistingsymlink without removing the link's target",
		renameArgs{"/existingsymlink", "/newexistingsymlink"},
		false,
		func(sf symFs) error {
			if err := sf.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/mydir"); err != nil {
				return err
			}

			if _, _, err := f.LstatIfPossible("/newexistingsymlink"); err != nil {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can rename symlink /existingsymlink to file to /newexistingsymlink without removing the link's target",
		renameArgs{"/existingsymlink", "/newexistingsymlink"},
		false,
		func(sf symFs) error {
			file, err := sf.Create("/test.txt")
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := sf.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/existingsymlink"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if _, err := f.Stat("/test.txt"); err != nil {
				return err
			}

			if _, _, err := f.LstatIfPossible("/newexistingsymlink"); err != nil {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
}

func TestSTFS_Rename(t *testing.T) {
	for _, tt := range renameTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			if err := symFs.Rename(tt.args.oldname, tt.args.newname); (err != nil) != tt.wantErr {
				if !tt.checkAfterError {
					t.Errorf("%v.Rename() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}
			}

			if err := tt.check(symFs); err != nil {
				t.Errorf("%v check() error = %v", symFs.Name(), err)

				return
			}
		})
	}
}

type statArgs struct {
	name string
}

var statTests = []struct {
	name      string
	args      statArgs
	wantErr   bool
	prepare   func(afero.Fs) error
	check     func(os.FileInfo) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can stat /",
		statArgs{"/"},
		false,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error {
			if dir, _ := path.Split(f.Name()); !(dir == "/" || dir == "") {
				return fmt.Errorf("invalid dir part of path %v, should be ''", dir)

			}

			return nil
		},
		true,
		true,
	},
	{
		"Can not stat /test.txt without creating it",
		statArgs{"/test.txt"},
		true,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error { return nil },
		true,
		true,
	},
	{
		"Can stat /test.txt after creating it",
		statArgs{"/test.txt"},
		false,
		func(f afero.Fs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can not stat /mydir/test.txt without creating it",
		statArgs{"/mydir/test.txt"},
		true,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error { return nil },
		true,
		true,
	},
	{
		"Can stat /mydir/test.txt after creating it",
		statArgs{"/mydir/test.txt"},
		false,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Result of stat /test.txt after creating it matches provided values",
		statArgs{"/test.txt"},
		false,
		func(f afero.Fs) error {
			file, err := f.OpenFile("/test.txt", os.O_CREATE, os.ModePerm)
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f os.FileInfo) error {
			wantName := "test.txt"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			wantPerm := os.ModePerm
			gotPerm := f.Mode().Perm()

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		false, // FIXME: With cache enabled, the permissions don't match
		false, // FIXME: With the OsFs, the permissions don't match
	},
}

func TestSTFS_Stat(t *testing.T) {
	for _, tt := range statTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			if err := tt.prepare(fs.fs); err != nil {
				t.Errorf("%v prepare() error = %v", fs.fs.Name(), err)

				return
			}

			got, err := fs.fs.Stat(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

type chmodArgs struct {
	name string
	mode os.FileMode
}

var chmodTests = []struct {
	name      string
	args      chmodArgs
	wantErr   bool
	prepare   func(afero.Fs) error
	check     func(f os.FileInfo) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can chmod / to 0666",
		chmodArgs{"/", 0666},
		false,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error {
			if dir, _ := path.Split(f.Name()); !(dir == "/" || dir == "") {
				return fmt.Errorf("invalid dir part of path %v, should be ''", dir)

			}

			wantPerm := fs.FileMode(0666)
			gotPerm := f.Mode().Perm()

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		false, // FIXME: With cache enabled, directories can't be `chmod`ed
		true,
	},
	{
		"Can chmod /test.txt to 0666 if it exists",
		chmodArgs{"/test.txt", 0666},
		false,
		func(f afero.Fs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantPerm := fs.FileMode(0666)
			gotPerm := f.Mode().Perm()

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can chmod /test.txt to 0777 if it exists",
		chmodArgs{"/test.txt", 0777},
		false,
		func(f afero.Fs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantPerm := fs.FileMode(0777)
			gotPerm := f.Mode().Perm()

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can not chmod /test.txt without creating it",
		chmodArgs{"/test.txt", 0666},
		true,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error { return nil },
		true,
		true,
	},
	{
		"Can not chmod /mydir/test.txt without creating it",
		chmodArgs{"/mydir/test.txt", 0666},
		true,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error { return nil },
		true,
		true,
	},
	{
		"Can chmod /mydir/test.txt to 0666 after creating it",
		chmodArgs{"/mydir/test.txt", 0666},
		false,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantPerm := fs.FileMode(0666)
			gotPerm := f.Mode().Perm()

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		true,
		true,
	},
}

func TestSTFS_Chmod(t *testing.T) {
	for _, tt := range chmodTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			if err := tt.prepare(fs.fs); err != nil {
				t.Errorf("%v prepare() error = %v", fs.fs.Name(), err)

				return
			}

			if err := fs.fs.Chmod(tt.args.name, tt.args.mode); (err != nil) != tt.wantErr {
				t.Errorf("%v.Chmod() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			got, err := fs.fs.Stat(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

type chownArgs struct {
	name string
	uid  int
	gid  int
}

var chownTests = []struct {
	name      string
	args      chownArgs
	wantErr   bool
	prepare   func(afero.Fs) error
	check     func(f os.FileInfo) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can chown /test.txt to 11, 11 if it exists",
		chownArgs{"/test.txt", 11, 11},
		false,
		func(f afero.Fs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantGID := 11
			wantUID := 11

			gotSys, ok := f.Sys().(*Stat)
			if !ok {
				return errors.New("could not get fs.Stat from FileInfo.Sys()")
			}

			gotGID := int(gotSys.Gid)
			gotUID := int(gotSys.Uid)

			if wantGID != gotGID {
				return fmt.Errorf("invalid GID, got %v, want %v", gotGID, wantGID)
			}

			if wantUID != gotUID {
				return fmt.Errorf("invalid UID, got %v, want %v", gotUID, wantUID)
			}

			return nil
		},
		false,
		false, // FIXME: With cache enabled, files and directories can't be `chmod`ed
	},
	{
		"Can chown /mydir/test.txt to 11, 11 if it exists",
		chownArgs{"/mydir/test.txt", 11, 11},
		false,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantGID := 11
			wantUID := 11

			gotSys, ok := f.Sys().(*Stat)
			if !ok {
				return errors.New("could not get fs.Stat from FileInfo.Sys()")
			}

			gotGID := int(gotSys.Gid)
			gotUID := int(gotSys.Uid)

			if wantGID != gotGID {
				return fmt.Errorf("invalid GID, got %v, want %v", gotGID, wantGID)
			}

			if wantUID != gotUID {
				return fmt.Errorf("invalid UID, got %v, want %v", gotUID, wantUID)
			}

			return nil
		},
		false,
		false, // FIXME: With cache enabled, files and directories can't be `chmod`ed
	},
	{
		"Can chown /mydir to 11, 11 if it exists",
		chownArgs{"/mydir", 11, 11},
		false,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "mydir"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantGID := 11
			wantUID := 11

			gotSys, ok := f.Sys().(*Stat)
			if !ok {
				return errors.New("could not get fs.Stat from FileInfo.Sys()")
			}

			gotGID := int(gotSys.Gid)
			gotUID := int(gotSys.Uid)

			if wantGID != gotGID {
				return fmt.Errorf("invalid GID, got %v, want %v", gotGID, wantGID)
			}

			if wantUID != gotUID {
				return fmt.Errorf("invalid UID, got %v, want %v", gotUID, wantUID)
			}

			return nil
		},
		false,
		false, // FIXME: With cache enabled, files and directories can't be `chmod`ed
	},
	{
		"Can not chown /test.txt without creating it",
		chownArgs{"/test.txt", 11, 11},
		true,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error { return nil },
		true,
		true,
	},
	{
		"Can not chown /mydir/test.txt without creating it",
		chownArgs{"/mydir/test.txt", 11, 11},
		true,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error { return nil },
		true,
		true,
	},
}

func TestSTFS_Chown(t *testing.T) {
	for _, tt := range chownTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			if err := tt.prepare(fs.fs); err != nil {
				t.Errorf("%v prepare() error = %v", fs.fs.Name(), err)

				return
			}

			if err := fs.fs.Chown(tt.args.name, tt.args.uid, tt.args.gid); (err != nil) != tt.wantErr {
				t.Errorf("%v.Chown() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			got, err := fs.fs.Stat(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

type chtimesArgs struct {
	name  string
	atime time.Time
	mtime time.Time
}

var chtimesTests = []struct {
	name      string
	args      chtimesArgs
	wantErr   bool
	prepare   func(afero.Fs) error
	check     func(f os.FileInfo) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can chtimes /test.txt to 2021-12-23, 2022-01-14, if it exists",
		chtimesArgs{"/test.txt", time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC), time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)},
		false,
		func(f afero.Fs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantAtime := time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC)
			wantMtime := time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)

			gotSys, ok := f.Sys().(*Stat)
			if !ok {
				return errors.New("could not get fs.Stat from FileInfo.Sys()")
			}

			gotAtime := time.Unix(0, gotSys.Atim.Nano())
			gotMtime := f.ModTime()

			if !wantAtime.Equal(gotAtime) {
				return fmt.Errorf("invalid Atime, got %v, want %v", gotAtime, wantAtime)
			}

			if !wantMtime.Equal(gotMtime) {
				return fmt.Errorf("invalid Mtime, got %v, want %v", gotMtime, wantMtime)
			}

			return nil
		},
		false, // FIXME: Can't cast to `Stat` struct if cache is enabled
		false, // FIXME: Can't cast to `Stat` struct if OsFs is enabled
	},
	{
		"Can chtimes /mydir/test.txt to 2021-12-23, 2022-01-14, if it exists",
		chtimesArgs{"/mydir/test.txt", time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC), time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)},
		false,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "test.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantAtime := time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC)
			wantMtime := time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)

			gotSys, ok := f.Sys().(*Stat)
			if !ok {
				return errors.New("could not get fs.Stat from FileInfo.Sys()")
			}

			gotAtime := time.Unix(0, gotSys.Atim.Nano())
			gotMtime := f.ModTime()

			if !wantAtime.Equal(gotAtime) {
				return fmt.Errorf("invalid Atime, got %v, want %v", gotAtime, wantAtime)
			}

			if !wantMtime.Equal(gotMtime) {
				return fmt.Errorf("invalid Mtime, got %v, want %v", gotMtime, wantMtime)
			}

			return nil
		},
		false, // FIXME: Can't cast to `Stat` struct if cache is enabled
		false, // FIXME: Can't cast to `Stat` struct if OsFs is enabled
	},
	{
		"Can chtimes /mydir to 2021-12-23, 2022-01-14, if it exists",
		chtimesArgs{"/mydir", time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC), time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)},
		false,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			want := "mydir"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			wantAtime := time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC)
			wantMtime := time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)

			gotSys, ok := f.Sys().(*Stat)
			if !ok {
				return errors.New("could not get fs.Stat from FileInfo.Sys()")
			}

			gotAtime := time.Unix(0, gotSys.Atim.Nano())
			gotMtime := f.ModTime()

			if !wantAtime.Equal(gotAtime) {
				return fmt.Errorf("invalid Atime, got %v, want %v", gotAtime, wantAtime)
			}

			if !wantMtime.Equal(gotMtime) {
				return fmt.Errorf("invalid Mtime, got %v, want %v", gotMtime, wantMtime)
			}

			return nil
		},
		false, // FIXME: Can't cast to `Stat` struct if cache is enabled
		false, // FIXME: Can't cast to `Stat` struct if OsFs is enabled
	},
	{
		"Can not chtimes /test.txt without creating it",
		chtimesArgs{"/test.txt", time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC), time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)},
		true,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error { return nil },
		true,
		true,
	},
	{
		"Can not chtimes /mydir/test.txt without creating it",
		chtimesArgs{"/mydir/test.txt", time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC), time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)},
		true,
		func(f afero.Fs) error { return nil },
		func(f os.FileInfo) error { return nil },
		true,
		true,
	},
}

func TestSTFS_Chtimes(t *testing.T) {
	for _, tt := range chtimesTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			if err := tt.prepare(fs.fs); err != nil {
				t.Errorf("%v prepare() error = %v", fs.fs.Name(), err)

				return
			}

			if err := fs.fs.Chtimes(tt.args.name, tt.args.atime, tt.args.mtime); (err != nil) != tt.wantErr {
				t.Errorf("%v.Chtimes() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			got, err := fs.fs.Stat(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

type lstatArgs struct {
	name []string
}

var lstatTests = []struct {
	name      string
	args      lstatArgs
	wantErr   bool
	prepare   func(*STFS) error
	check     func(os.FileInfo, int) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can not lstat /",
		lstatArgs{[]string{"/"}},
		true,
		func(f *STFS) error { return nil },
		func(f os.FileInfo, i int) error { return nil },
		true,
		true,
	},
	{
		"Can not lstat /test.txt without creating it",
		lstatArgs{[]string{"/test.txt"}},
		true,
		func(f *STFS) error { return nil },
		func(f os.FileInfo, i int) error { return nil },
		true,
		true,
	},
	{
		"Can lstat /test2.txt after creating /test.txt and symlinking it",
		lstatArgs{[]string{"/test2.txt"}},
		false,
		func(f *STFS) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/test2.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo, i int) error {
			want := "test2.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can not lstat /mydir/test.txt without creating it",
		lstatArgs{[]string{"/mydir/test.txt"}},
		true,
		func(f *STFS) error { return nil },
		func(f os.FileInfo, i int) error { return nil },
		true,
		true,
	},
	{
		"Can lstat /mydir/test2.txt after creating /mydir/test.txt and symlinking it",
		lstatArgs{[]string{"/mydir/test2.txt"}},
		false,
		func(f *STFS) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir/test.txt", "/mydir/test2.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo, i int) error {
			want := "test2.txt"
			got := f.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Result of lstat /test2.txt after creating /test.txt and symlinking it matches provided values",
		lstatArgs{[]string{"/test2.txt"}},
		false,
		func(f *STFS) error {
			file, err := f.OpenFile("/test.txt", os.O_CREATE, os.ModePerm)
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/test2.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo, i int) error {
			wantName := "test2.txt"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			wantPerm := os.ModePerm
			gotPerm := f.Mode().Perm()

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Result of lstat /test2.txt, /test3.txt and /test4.txt after creating /test.txt and symlinking it matches provided values",
		lstatArgs{[]string{"/test2.txt", "/test3.txt", "/test4.txt"}},
		false,
		func(f *STFS) error {
			file, err := f.OpenFile("/test.txt", os.O_CREATE, os.ModePerm)
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/test2.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/test3.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/test4.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo, i int) error {
			wantName := "test2.txt"
			if i == 1 {
				wantName = "test3.txt"
			} else if i == 2 {
				wantName = "test4.txt"
			}

			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			wantPerm := os.ModePerm
			gotPerm := f.Mode().Perm()

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		true,
		true,
	},
}

func TestSTFS_Lstat(t *testing.T) {
	for _, tt := range lstatTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			stfs, ok := fs.fs.(*STFS)
			if !ok {
				return
			}

			if err := tt.prepare(stfs); err != nil {
				t.Errorf("%v prepare() error = %v", stfs.Name(), err)

				return
			}

			for i, arg := range tt.args.name {
				got, possible, err := stfs.LstatIfPossible(arg)
				if !possible {
					t.Errorf("%v.LstatIfPossible() possible = %v, want %v", stfs.Name(), possible, true)
				}

				if (err != nil) != tt.wantErr {
					t.Errorf("%v.LstatIfPossible() error = %v, wantErr %v", stfs.Name(), err, tt.wantErr)

					return
				}

				if err := tt.check(got, i); err != nil {
					t.Errorf("%v check() error = %v", stfs.Name(), err)

					return
				}
			}
		})
	}
}

type symlinkArgs struct {
	oldname string
	newname string
}

var symlinkTests = []struct {
	name            string
	args            symlinkArgs
	wantErr         bool
	prepare         func(symFs) error
	check           func(symFs) error
	checkAfterError bool
	withCache       bool
	withOsFs        bool
}{
	{
		"Can symlink / to /mydir",
		symlinkArgs{"/", "/mydir"},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error {
			info, _, err := f.LstatIfPossible("/mydir")
			if err != nil {
				return err
			}

			want := "mydir"
			got := info.Name()

			if want != got {
				return fmt.Errorf("symlinked file has wrong name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not symlink / to /",
		symlinkArgs{"/", "/"},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can not symlink '' to ''",
		symlinkArgs{"", ""},
		true,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can symlink ' ' to ' '",
		symlinkArgs{" ", " "},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible(" "); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can symlink /test.txt to /new.txt if does exist",
		symlinkArgs{"/test.txt", "/new.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			info, _, err := f.LstatIfPossible("/new.txt")
			if err != nil {
				return err
			}

			want := "new.txt"
			got := info.Name()

			if want != got {
				return fmt.Errorf("symlinked file has wrong name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can symlink /test.txt to /new.txt if does not exist",
		symlinkArgs{"/test.txt", "/new.txt"},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can symlink directory /myolddir to /mydir",
		symlinkArgs{"/myolddir", "/mydir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/myolddir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			info, _, err := f.LstatIfPossible("/mydir")
			if err != nil {
				return err
			}

			want := "mydir"
			got := info.Name()

			if want != got {
				return fmt.Errorf("symlinked directory has wrong name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		false,
		true,
	},
	{
		"Can symlink non-empty directory /myolddir to /mydir",
		symlinkArgs{"/myolddir", "/mydir"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/myolddir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/myolddir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			info, _, err := f.LstatIfPossible("/mydir")
			if err != nil {
				return err
			}

			want := "mydir"
			got := info.Name()

			if want != got {
				return fmt.Errorf("symlinked directory has wrong name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		false,
		true,
	},
	{
		"Can not symlink /test.txt to /mydir/new.txt if new parent drectory does not exist",
		symlinkArgs{"/test.txt", "/mydir/new.txt"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			if _, _, err := f.LstatIfPossible("/mydir/new.txt"); !errors.Is(err, os.ErrNotExist) {
				return err
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can symlink /test.txt to /mydir/new.txt if new parent drectory does exist",
		symlinkArgs{"/test.txt", "/mydir/new.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error {
			info, _, err := f.LstatIfPossible("/mydir/new.txt")
			if err != nil {
				return err
			}

			want := "new.txt"
			got := info.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not symlink /test.txt to /test.txt if does exist",
		symlinkArgs{"/test.txt", "/test.txt"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can symlink /test.txt to /test.txt if does not exist",
		symlinkArgs{"/test.txt", "/test.txt"},
		false,
		func(f symFs) error { return nil },
		func(f symFs) error {
			info, _, err := f.LstatIfPossible("/test.txt")
			if err != nil {
				return err
			}

			want := "test.txt"
			got := info.Name()

			if want != got {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		false,
		true,
		true,
	},
	{
		"Can not symlink /test.txt to /existing.txt if source and target both exist",
		symlinkArgs{"/test.txt", "/existing.txt"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/existing.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can not symlink /test.txt to /mydir if source is file and target is directory",
		symlinkArgs{"/test.txt", "/mydir"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can not symlink /mydir to /test.txt if source is directory and target is file",
		symlinkArgs{"/mydir", "/test.txt"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
	{
		"Can not symlink /test.txt to /test.txt/",
		symlinkArgs{"/test.txt", "/test.txt/"},
		true,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f symFs) error { return nil },
		false,
		true,
		true,
	},
}

func TestSTFS_Symlink(t *testing.T) {
	for _, tt := range symlinkTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			if err := symFs.SymlinkIfPossible(tt.args.oldname, tt.args.newname); (err != nil) != tt.wantErr {
				if !tt.checkAfterError {
					t.Errorf("%v.SymlinkIfPossible() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

					return
				}
			}

			if err := tt.check(symFs); err != nil {
				t.Errorf("%v check() error = %v", symFs.Name(), err)

				return
			}
		})
	}
}

type readlinkArgs struct {
	name string
}

var readlinkTests = []struct {
	name      string
	args      readlinkArgs
	wantErr   bool
	prepare   func(symFs) error
	check     func(string) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can not readlink /",
		readlinkArgs{"/"},
		true,
		func(f symFs) error { return nil },
		func(got string) error { return nil },
		true,
		true,
	},
	{
		"Can not readlink /test.txt without creating it",
		readlinkArgs{"/test.txt"},
		true,
		func(f symFs) error { return nil },
		func(got string) error { return nil },
		true,
		true,
	},
	{
		"Can readlink /test2.txt after creating /test.txt and symlinking it",
		readlinkArgs{"/test2.txt"},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/test2.txt"); err != nil {
				return err
			}

			return nil
		},
		func(got string) error {
			want := "test.txt"

			if !strings.HasSuffix(got, want) {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can not readlink /mydir/test.txt without creating it",
		readlinkArgs{"/mydir/test.txt"},
		true,
		func(f symFs) error { return nil },
		func(got string) error { return nil },
		true,
		true,
	},
	{
		"Can readlink /mydir/test2.txt after creating /mydir/test.txt and symlinking it",
		readlinkArgs{"/mydir/test2.txt"},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir/test.txt", "/mydir/test2.txt"); err != nil {
				return err
			}

			return nil
		},
		func(got string) error {
			want := "test.txt"

			if !strings.HasSuffix(got, want) {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Result of readlink /test2.txt after creating /test.txt and symlinking it matches provided values",
		readlinkArgs{"/test2.txt"},
		false,
		func(f symFs) error {
			file, err := f.OpenFile("/test.txt", os.O_CREATE, os.ModePerm)
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/test2.txt"); err != nil {
				return err
			}

			return nil
		},
		func(got string) error {
			want := "test.txt"

			if !strings.HasSuffix(got, want) {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
}

func TestSTFS_Readlink(t *testing.T) {
	for _, tt := range readlinkTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			got, err := symFs.ReadlinkIfPossible(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.ReadlinkIfPossible() error = %v, wantErr %v", symFs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", symFs.Name(), err)

				return
			}
		})
	}
}
