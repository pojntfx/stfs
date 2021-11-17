package main

import (
	"archive/tar"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	blockSize = 512
)

type Wrapper struct {
	Record int64
	Block  int64
	Header string
}

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")

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

	record := int64(0)
	block := int64(0)
	for {
		// Read exactly one record
		bf := make([]byte, *recordSize*blockSize)
		if _, err := io.ReadFull(f, bf); err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		// Get the headers from the record
		for currentBlock := 0; currentBlock < *recordSize; currentBlock++ {
			tr := tar.NewReader(bytes.NewReader(bf[blockSize*currentBlock : blockSize*(currentBlock+1)]))
			hdr, err := tr.Next() // TODO: Read PAX header info by a plain `tar tvf` before; reads should still be efficient even when seeking back as reads are cached
			if err != nil {
				continue
			}

			wrapper := &Wrapper{
				Record: record,
				Block:  block,
				Header: fmt.Sprintf("%v", hdr),
			}

			log.Println(wrapper)

			curr := (record * int64(*recordSize) * blockSize) + int64(currentBlock)*blockSize

			nextTotalBlocks := (curr + hdr.Size) / blockSize
			record := nextTotalBlocks / int64(*recordSize)

			block = nextTotalBlocks - (record * int64(*recordSize)) + 1
			if block > int64(*recordSize) {
				record++
				block = 0
			}
		}

		record++
	}
}
