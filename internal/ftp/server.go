package ftp

import (
	"crypto/tls"
	"errors"
	"sync"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"
)

var (
	ErrNoTLS = errors.New("no TLS supported")
)

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
	return nil, ErrNoTLS
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
