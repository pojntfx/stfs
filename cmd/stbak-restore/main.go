package main

import (
	"archive/tar"
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/pkg/controllers"
)

func main() {
	drive := flag.String("drive", "/dev/nst0", "Tape or tar file to read from")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	record := flag.Int("record", 0, "Record to seek too")
	block := flag.Int("block", 0, "Block in record to seek too")
	dst := flag.String("dst", "", "File to restore to (archived name by default)")
	preview := flag.Bool("preview", false, "Only read the header")

	flag.Parse()

	fileDescription, err := os.Stat(*drive)
	if err != nil {
		panic(err)
	}

	var f *os.File
	if fileDescription.Mode().IsRegular() {
		f, err = os.Open(*drive)
		if err != nil {
			panic(err)
		}
	} else {
		f, err = os.OpenFile(*drive, os.O_RDONLY, os.ModeCharDevice)
		if err != nil {
			panic(err)
		}
	}
	defer f.Close()

	var tr *tar.Reader
	if fileDescription.Mode().IsRegular() {
		// Seek to record and block
		if _, err := f.Seek(int64((*recordSize*controllers.BlockSize**record)+*block*controllers.BlockSize), 0); err != nil {
			panic(err)
		}

		tr = tar.NewReader(f)
	} else {
		// Seek to record
		if err := controllers.SeekToRecordOnTape(f, int32(*record)); err != nil {
			panic(err)
		}

		// Seek to block
		br := bufio.NewReaderSize(f, controllers.BlockSize**recordSize)
		if _, err := br.Read(make([]byte, *block*controllers.BlockSize)); err != nil {
			panic(err)
		}

		tr = tar.NewReader(br)
	}

	hdr, err := tr.Next()
	if err != nil {
		panic(err)
	}

	log.Println(hdr)

	if !*preview {
		if *dst == "" {
			*dst = filepath.Base(hdr.Name)
		}

		dstFile, err := os.OpenFile(*dst, os.O_WRONLY|os.O_CREATE, hdr.FileInfo().Mode())
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, tr); err != nil {
			panic(err)
		}
	}
}
