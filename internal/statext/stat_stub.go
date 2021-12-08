//go:build (linux && mips64) || (linux && mips) || (linux && mipsle) || (linux && mips64le) || !(darwin || dragonfly || freebsd || linux)

package statext

import (
	"archive/tar"
)

func EnhanceHeader(path string, hdr *tar.Header) error {
	return nil
}
