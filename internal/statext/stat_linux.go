//go:build (linux && amd64) || (linux && 386) || (linux && arm) || (linux && arm64) || (linux && ppc64) || (linux && ppc64le) || (linux && riscv64) || (linux && s390x)

package statext

import (
	"archive/tar"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func EnhanceHeader(path string, hdr *tar.Header) error {
	var unixStat syscall.Stat_t
	if err := syscall.Stat(path, &unixStat); err != nil {
		return err
	}

	mtimesec, mtimensec := unixStat.Mtim.Unix()
	atimesec, atimensec := unixStat.Atim.Unix()
	ctimesec, ctimensec := unixStat.Ctim.Unix()

	hdr.ModTime = time.Unix(mtimesec, mtimensec)
	hdr.AccessTime = time.Unix(atimesec, atimensec)
	hdr.ChangeTime = time.Unix(ctimesec, ctimensec)

	hdr.Devmajor = int64(unix.Major(unixStat.Dev))
	hdr.Devminor = int64(unix.Minor(unixStat.Dev))

	return nil
}
