package main

import (
	"archive/tar"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	blockSize  = 20
	headerSize = 512
)

type HeaderInBlock struct {
	OffsetInBlock int
	Header        string
}

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")

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

	// Read exactly one block
	bf := make([]byte, blockSize*headerSize)
	if _, err := io.ReadFull(f, bf); err != nil {
		panic(err)
	}

	// Get the headers from the block
	headers := []HeaderInBlock{}
	for i := 0; i < blockSize; i++ {
		tr := tar.NewReader(bytes.NewReader(bf[headerSize*i : headerSize*(i+1)]))
		hdr, err := tr.Next()
		if err != nil {
			continue
		}

		headers = append(headers, HeaderInBlock{
			OffsetInBlock: i,
			Header:        fmt.Sprintf("%v", hdr),
		})
	}

	if len(headers) > 0 {
		fmt.Println(headers)
	}
}
