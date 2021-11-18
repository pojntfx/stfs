package main

//go:generate sh -c "mkdir -p ../../pkg/api/proto/v1 && protoc --go_out=paths=source_relative,plugins=grpc:../../pkg/api/proto/v1 -I=../../api/proto/v1 ../../api/proto/v1/*.proto"

import (
	"archive/tar"
	"bufio"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/pojntfx/stfs/pkg/controllers"
	"golang.org/x/sys/unix"
)

const (
	stfsVersionPAX = "STFS.Version"
	stfsVersion    = 1

	stfsActionPAX    = "STFS.Action"
	stfsActionCreate = "CREATE"
	stfsActionUpdate = "UPDATE"
	stfsActionDelete = "DELETE"

	stfsReplacesPAX = "STFS.Replaces"
)

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	dir := flag.String("dir", ".", "Directory to add to the file")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	overwrite := flag.Bool("overwrite", false, "Whether to start writing from the current position instead of from the end of the tape")

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
		if *overwrite {
			f, err = os.OpenFile(*file, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				panic(err)
			}
		} else {
			f, err = os.OpenFile(*file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				panic(err)
			}
		}

		// No need to go to end manually due to `os.O_APPEND`
	} else {
		f, err = os.OpenFile(*file, os.O_APPEND|os.O_WRONLY, os.ModeCharDevice)
		if err != nil {
			panic(err)
		}

		if !*overwrite {
			// Go to end of tape
			if err := controllers.GoToEndOfTape(f); err != nil {
				panic(err)
			}
		}
	}
	defer f.Close()

	var tw *tar.Writer
	if isRegular {
		tw = tar.NewWriter(f)
	} else {
		bw := bufio.NewWriterSize(f, controllers.BlockSize**recordSize)
		tw = tar.NewWriter(bw)
	}
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
		hdr.PAXRecords = map[string]string{
			stfsVersionPAX:  strconv.Itoa(stfsVersion),
			stfsActionPAX:   stfsActionUpdate,
			stfsReplacesPAX: "",
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

		if isRegular {
			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		} else {
			buf := make([]byte, controllers.BlockSize**recordSize)
			if _, err := io.CopyBuffer(tw, file, buf); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		panic(err)
	}
}
