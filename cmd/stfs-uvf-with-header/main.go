package main

//go:generate sh -c "mkdir -p ../../pkg/api/proto/v1 && protoc --go_out=paths=source_relative,plugins=grpc:../../pkg/api/proto/v1 -I=../../api/proto/v1 ../../api/proto/v1/*.proto"

import (
	"archive/tar"
	"encoding/base64"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	api "github.com/pojntfx/stfs/pkg/api/proto/v1"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"
)

// See https://github.com/benmcclelland/mtio
const (
	MTIOCTOP = 0x40086d01 // Do magnetic tape operation
	MTEOM    = 12         // Goto end of recorded media (for appending files)
	MTBSR    = 4          // Backward space record

	STFSVersion    = 1
	STFSVersionPAX = "STFS.Version"
	STFSHeaderPAX  = "STFS.Header"
)

// Operation is struct for MTIOCTOP
type Operation struct {
	Op    int16 // Operation ID
	Pad   int16 // Padding to match C structures
	Count int32 // Operation count
}

const (
	blockSize = 512
)

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	dir := flag.String("dir", ".", "Directory to add to the file")

	flag.Parse()

	seekBackwards := int64(-blockSize) // Seek back one block (half a trailer) so we can detect the invalid trailer in `tvf` and seek accordingly
	isRegular := true
	stat, err := os.Stat(*file)
	if err == nil {
		isRegular = stat.Mode().IsRegular()
	} else {
		if os.IsNotExist(err) {
			isRegular = true

			// Create the file
			f, err := os.OpenFile(*file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				panic(err)
			}

			// Create an empty tar archive with a trailer so that we may seek back
			tw := tar.NewWriter(f)
			if err := tw.Close(); err != nil {
				panic(err)
			}

			seekBackwards = -(blockSize * 2) // Overwrite the file completely the first time
		} else {
			panic(err)
		}
	}

	var f *os.File
	if isRegular {
		f, err = os.OpenFile(*file, os.O_RDWR, 0600)
		if err != nil {
			panic(err)
		}

		// Seek backwards into header
		if _, err := f.Seek(seekBackwards, io.SeekEnd); err != nil {
			panic(err)
		}
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

		// Seek backwards into header
		// TODO: Validate that this iterates by block, not by record
		// TODO: Only run this if output of tell syscall != 0
		syscall.Syscall(
			syscall.SYS_IOCTL,
			f.Fd(),
			MTIOCTOP,
			uintptr(unsafe.Pointer(
				&Operation{
					Op:    MTBSR,
					Count: 1,
				},
			)),
		)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

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

		stfsHeader := &api.Header{
			Action: api.Action_CREATE,
		}

		encodedHeader, err := proto.Marshal(stfsHeader)
		if err != nil {
			return err
		}

		hdr.Name = path
		hdr.PAXRecords = map[string]string{
			STFSVersionPAX: strconv.Itoa(STFSVersion),
			STFSHeaderPAX:  base64.StdEncoding.EncodeToString(encodedHeader),
		}
		hdr.Format = tar.FormatPAX

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
