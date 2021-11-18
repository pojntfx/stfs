package main

import (
	"archive/tar"
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
)

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	record := flag.Int("record", 0, "Record to seek too")
	block := flag.Int("block", 0, "Block in record to seek too")
	read := flag.Bool("read", false, "Whether to read the next header")

	flag.Parse()

	fileDescription, err := os.Stat(*file)
	if err != nil {
		panic(err)
	}

	var f *os.File
	if fileDescription.Mode().IsRegular() {
		f, err = os.Open(*file)
		if err != nil {
			panic(err)
		}
	} else {
		f, err = os.OpenFile(*file, os.O_RDONLY, os.ModeCharDevice)
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

	if *read {
		hdr, err := tr.Next()
		if err != nil {
			panic(err)
		}

		log.Println(hdr)
	}
}
