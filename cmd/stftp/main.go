package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	golog "github.com/fclairamb/go-log"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/pojntfx/stfs/internal/fs"
	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/afero"
)

var (
	errNoTLS = errors.New("no TLS supported")
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
	enableCache := flag.Bool("cache", true, "Enable in-memory caching")

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

	stfs := fs.NewFileSystem(
		ops,

		config.MetadataConfig{
			Metadata: metadataPersister,
		},

		logger.PrintHeader,
	)

	var fs afero.Fs
	if *enableCache {
		fs = afero.NewCacheOnReadFs(afero.NewBasePathFs(stfs, *dir), afero.NewMemMapFs(), time.Hour)
	} else {
		fs = afero.NewBasePathFs(stfs, *dir)
	}

	srv := ftpserver.NewFtpServer(
		&FTPServer{
			Settings: &ftpserver.Settings{
				ListenAddr: *laddr,
			},
			FileSystem: fs,
		},
	)
	srv.Logger = &Logger{}

	log.Println("Listening on", *laddr)

	panic(srv.ListenAndServe())
}

type FTPServer struct {
	Settings   *ftpserver.Settings
	FileSystem afero.Fs

	clientsLock sync.Mutex
	clients     []ftpserver.ClientContext
}

func (driver *FTPServer) GetSettings() (*ftpserver.Settings, error) {
	return driver.Settings, nil
}

func (driver *FTPServer) GetTLSConfig() (*tls.Config, error) {
	return nil, errNoTLS
}

func (driver *FTPServer) ClientConnected(cc ftpserver.ClientContext) (string, error) {
	driver.clientsLock.Lock()
	defer driver.clientsLock.Unlock()

	driver.clients = append(driver.clients, cc)

	return "", nil
}

func (driver *FTPServer) ClientDisconnected(cc ftpserver.ClientContext) {
	driver.clientsLock.Lock()
	defer driver.clientsLock.Unlock()

	for idx, client := range driver.clients {
		if client.ID() == cc.ID() {
			lastIdx := len(driver.clients) - 1
			driver.clients[idx] = driver.clients[lastIdx]
			driver.clients[lastIdx] = nil
			driver.clients = driver.clients[:lastIdx]

			return
		}
	}
}

func (driver *FTPServer) AuthUser(_ ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	return driver.FileSystem, nil
}

type Logger struct{}

func (l Logger) Debug(event string, keyvals ...interface{}) {
	log.Println(event, keyvals)
}

func (l Logger) Info(event string, keyvals ...interface{}) {
	log.Println(event, keyvals)
}

func (l Logger) Warn(event string, keyvals ...interface{}) {
	log.Println(event, keyvals)
}

func (l Logger) Error(event string, keyvals ...interface{}) {
	log.Println(event, keyvals)
}

func (l Logger) With(keyvals ...interface{}) golog.Logger {
	return l
}
