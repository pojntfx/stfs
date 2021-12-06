package utility

import "github.com/pojntfx/stfs/pkg/config"

func Keygen(
	pipes config.PipeConfig,
	crypto config.CryptoConfig,
) (privkey []byte, pubkey []byte, err error)
