//go:build js || windows

package adapters

import (
	"archive/tar"
)

func EnhanceHeader(path string, hdr *tar.Header) error {
	return nil
}
