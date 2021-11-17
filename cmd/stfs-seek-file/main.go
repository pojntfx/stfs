package main

import (
	"archive/tar"
	"flag"
	"io"
	"log"
	"os"
)

const (
	blockSize = 512
)

func main() {
	file := flag.String("file", "test.tar", "Tar file to open")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	record := flag.Int("record", 0, "Record to seek too")
	block := flag.Int("block", 0, "Block in record to seek too")

	flag.Parse()

	bytesToSeek := (*recordSize * blockSize * *record) + *block*blockSize

	f, err := os.Open(*file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if _, err := f.Seek(int64(bytesToSeek), 0); err != nil {
		panic(err)
	}

	tr := tar.NewReader(f)

	for {
		hdr, err := tr.Next()
		if err != nil {
			panic(err)
		}

		log.Println(hdr)

		curr, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			panic(err)
		}

		if currentRecord := ((curr + hdr.Size) / blockSize) / int64(*recordSize); currentRecord > int64(*record) {
			break
		}
	}
}
