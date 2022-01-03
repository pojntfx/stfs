package main

import (
	"archive/tar"
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/encryption"
	sfs "github.com/pojntfx/stfs/pkg/fs"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/recovery"
	"github.com/pojntfx/stfs/pkg/signature"
	"github.com/pojntfx/stfs/pkg/tape"

	golog "github.com/fclairamb/go-log"
)

type logger struct {
	verbose bool
}

func (l logger) log(level, event string, keyvals ...interface{}) {
	k, _ := json.Marshal(keyvals)

	log.Println(level, event, string(k))
}

func (l logger) Trace(event string, keyvals ...interface{}) {
	if l.verbose {
		l.log("TRACE", event, keyvals)
	}
}

func (l logger) Debug(event string, keyvals ...interface{}) {
	if l.verbose {
		l.log("DEBUG", event, keyvals)
	}
}

func (l logger) Info(event string, keyvals ...interface{}) {
	l.log("INFO", event, keyvals)
}

func (l logger) Warn(event string, keyvals ...interface{}) {
	l.log("WARN", event, keyvals)
}

func (l logger) Error(event string, keyvals ...interface{}) {
	l.log("ERROR", event, keyvals)
}

func (l logger) With(keyvals ...interface{}) golog.Logger {
	return l
}

func main() {
	driveFlag := flag.String("drive", "/dev/nst0", "Tape or tar file to use")
	recordSizeFlag := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	metadataFlag := flag.String("metadata", filepath.Join(os.TempDir(), "metadata.sqlite"), "Metadata database to use")
	writeCacheFlag := flag.String("writeCache", filepath.Join(os.TempDir(), "stfs-write-cache"), "Directory to use for write cache")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	tm := tape.NewTapeManager(
		*driveFlag,
		*recordSizeFlag,
		false,
	)

	metadataPersister := persisters.NewMetadataPersister(*metadataFlag)
	if err := metadataPersister.Open(); err != nil {
		panic(err)
	}

	l := &logger{
		verbose: *verboseFlag,
	}

	metadataConfig := config.MetadataConfig{
		Metadata: metadataPersister,
	}
	pipeConfig := config.PipeConfig{
		Compression: config.NoneKey,
		Encryption:  config.NoneKey,
		Signature:   config.NoneKey,
		RecordSize:  *recordSizeFlag,
	}
	backendConfig := config.BackendConfig{
		GetWriter:   tm.GetWriter,
		CloseWriter: tm.Close,

		GetReader:   tm.GetReader,
		CloseReader: tm.Close,

		GetDrive:   tm.GetDrive,
		CloseDrive: tm.Close,
	}
	readCryptoConfig := config.CryptoConfig{}

	readOps := operations.NewOperations(
		backendConfig,
		metadataConfig,

		pipeConfig,
		readCryptoConfig,

		func(event *config.HeaderEvent) {
			l.Debug("Header read", event)
		},
	)
	writeOps := operations.NewOperations(
		backendConfig,
		metadataConfig,

		pipeConfig,
		config.CryptoConfig{},

		func(event *config.HeaderEvent) {
			l.Debug("Header write", event)
		},
	)

	stfs := sfs.NewSTFS(
		readOps,
		writeOps,

		config.MetadataConfig{
			Metadata: metadataPersister,
		},

		config.CompressionLevelFastest,
		func() (cache.WriteCache, func() error, error) {
			return cache.NewCacheWrite(
				*writeCacheFlag,
				config.WriteCacheTypeFile,
			)
		},
		false,

		func(hdr *config.Header) {
			l.Trace("Header transform", hdr)
		},
		l,
	)

	root, err := metadataPersister.GetRootPath(context.Background())
	if err != nil {
		if err == config.ErrNoRootDirectory {
			root = "/"

			drive, err := tm.GetDrive()
			if err == nil {
				err = recovery.Index(
					config.DriveReaderConfig{
						Drive:          drive.Drive,
						DriveIsRegular: drive.DriveIsRegular,
					},
					config.DriveConfig{
						Drive:          drive.Drive,
						DriveIsRegular: drive.DriveIsRegular,
					},
					metadataConfig,
					pipeConfig,
					readCryptoConfig,

					*recordSizeFlag,
					0,
					0,
					true,
					0,

					func(hdr *tar.Header, i int) error {
						return encryption.DecryptHeader(hdr, config.NoneKey, nil)
					},
					func(hdr *tar.Header, isRegular bool) error {
						return signature.VerifyHeader(hdr, isRegular, config.NoneKey, nil)
					},

					func(hdr *config.Header) {
						l.Debug("Header read", hdr)
					},
				)
				if err != nil {
					if err := tm.Close(); err != nil {
						panic(err)
					}

					if err := stfs.MkdirRoot(root, os.ModePerm); err != nil {
						panic(err)
					}
				}
			} else if os.IsNotExist(err) {
				if err := tm.Close(); err != nil {
					panic(err)
				}

				if err := stfs.MkdirRoot(root, os.ModePerm); err != nil {
					panic(err)
				}
			} else {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	fs, err := cache.NewCacheFilesystem(
		stfs,
		root,
		config.NoneKey,
		0,
		"",
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

	log.Println("open /")

	dir, err := fs.Open("/")
	if err != nil {
		panic(err)
	}

	log.Println("Result of open /:", dir)

	log.Println("readdir /")

	children, err := dir.Readdir(-1)
	if err != nil {
		panic(err)
	}

	log.Println("Result of readdir /:", children)

	log.Println("create /test.txt")

	file, err := fs.Create("/test.txt")
	if err != nil {
		panic(err)
	}

	log.Println("Result of create /test.txt:", file)

	log.Println("writeString /test.txt")

	n, err := file.WriteString("Hello, world!")
	if err != nil {
		panic(err)
	}

	log.Println("Result of writeString /test.txt:", n)

	if err := file.Close(); err != nil {
		panic(err)
	}

	log.Println("readdir /")

	children, err = dir.Readdir(-1)
	if err != nil {
		panic(err)
	}

	log.Println("Result of readdir /:", children)

	if err := dir.Close(); err != nil {
		panic(err)
	}
}
