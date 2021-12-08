//go:build freebsd

package statext

import (
	"archive/tar"
	"syscall"

	"golang.org/x/sys/unix"
)

func EnhanceHeader(path string, hdr *tar.Header) error {
	var unixStat syscall.Stat_t
	if err := syscall.Stat(path, &unixStat); err != nil {
		return err
	}

	hdr.Devmajor = int64(unix.Major(unixStat.Dev))
	hdr.Devminor = int64(unix.Minor(unixStat.Dev))

	return nil
}
