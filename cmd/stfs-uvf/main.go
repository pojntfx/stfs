package main

import (
	"archive/tar"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// See https://github.com/benmcclelland/mtio
const (
	MTIOCTOP = 0x40086d01 // Do magnetic tape operation
	MTEOM    = 12         // Goto end of recorded media (for appending files)
)

// Operation is struct for MTIOCTOP
type Operation struct {
	Op    int16 // Operation ID
	Pad   int16 // Padding to match C structures
	Count int32 // Operation count
}

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	dir := flag.String("dir", ".", "Directory to add to the file")

	flag.Parse()

	isRegular := true
	stat, err := os.Stat(*file)
	if err == nil {
		isRegular = stat.Mode().IsRegular()
	} else {
		if os.IsNotExist(err) {
			isRegular = true
		} else {
			panic(err)
		}
	}

	var f *os.File
	if isRegular {
		f, err = os.OpenFile(*file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}

		// No need to go to end manually due to `os.O_APPEND`
	} else {
		// Go to end of file
		syscall.Syscall(
			syscall.SYS_IOCTL,
			f.Fd(),
			MTIOCTOP,
			uintptr(unsafe.Pointer(
				&Operation{
					Op: MTEOM,
				},
			)),
		)

		f, err = os.OpenFile(*file, os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
		if err != nil {
			panic(err)
		}
	}
	defer f.Close()

	tw := tar.NewWriter(f) // We are not closing the tar writer to prevent writing the trailer

	if err := filepath.Walk(*dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		link := ""
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		hdr.Format = tar.FormatGNU // Required for AccessTime, ChangeTime etc.

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

		hdr.Name = path

		log.Println(hdr)

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(tw, file); err != nil {
			return err
		}

		return nil
	}); err != nil {
		panic(err)
	}
}
