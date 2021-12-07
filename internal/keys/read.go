package keys

import (
	"io/ioutil"

	"github.com/pojntfx/stfs/pkg/config"
)

func ReadKey(encryptionFormat string, pathToKey string) ([]byte, error) {
	if encryptionFormat == config.NoneKey {
		return []byte{}, nil
	}

	return ioutil.ReadFile(pathToKey)
}
