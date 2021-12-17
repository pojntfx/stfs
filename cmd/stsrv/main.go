package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/internal/fs"
	"github.com/pojntfx/stfs/internal/handlers"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/afero"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	laddr := flag.String("laddr", "localhost:1337", "Listen address")
	dir := flag.String("dir", "/", "Directory to use as the root directory")
	drive := flag.String("drive", "/dev/nst0", "Tape or tar file to use")
	metadata := flag.String("metadata", filepath.Join(home, ".local", "share", "stbak", "var", "lib", "stbak", "metadata.sqlite"), "Metadata database to use")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")

	flag.Parse()

	tm := tape.NewTapeManager(
		*drive,
		*recordSize,
		false,
	)

	metadataPersister := persisters.NewMetadataPersister(*metadata)
	if err := metadataPersister.Open(); err != nil {
		panic(err)
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
			Compression: config.NoneKey,
			Encryption:  config.NoneKey,
			Signature:   config.NoneKey,
			RecordSize:  *recordSize,
		},
		config.CryptoConfig{
			Recipient: []byte{},
			Identity:  []byte{},
			Password:  "",
		},

		logger.PrintHeaderEvent,
	)

	stfs := afero.NewHttpFs(
		fs.NewFileSystem(
			ops,

			config.MetadataConfig{
				Metadata: metadataPersister,
			},

			logger.PrintHeader,
		),
	)

	log.Println("Listening on", *laddr)

	panic(
		http.ListenAndServe(
			*laddr,
			handlers.PanicHandler(
				http.FileServer(
					stfs.Dir(*dir),
				),
			),
		),
	)
}
