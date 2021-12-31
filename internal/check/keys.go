package check

import (
	"errors"
	"os"

	"github.com/pojntfx/stfs/pkg/config"
)

var (
	ErrKeyNotAccessible = errors.New("key not found or accessible")
)

func CheckKeyAccessible(encryptionFormat string, pathToKey string) error {
	if encryptionFormat == config.NoneKey {
		return nil
	}

	if _, err := os.Stat(pathToKey); err != nil {
		return ErrKeyNotAccessible
	}

	return nil
}
