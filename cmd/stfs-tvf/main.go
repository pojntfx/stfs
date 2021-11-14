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

type HeaderInBlock struct {
	Record int
	Block  int
	Header string
}

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	checkpoint := flag.Int("checkpoint", 0, "Log current record after checkpoint kilobytes have been read")

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

	record := 0
	for {
		// Lock the current record if requested
		if *checkpoint > 0 && record%*checkpoint == 0 {
			log.Println("Checkpoint:", record)
		}

		// Read exactly one record
		bf := make([]byte, *recordSize*blockSize)
		if _, err := io.ReadFull(f, bf); err != nil {
			if err == io.EOF {
				break
			}

			// Missing trailer (expected for concatenated tars)
			if err == io.ErrUnexpectedEOF {
				break
			}

			panic(err)
		}

		// Get the headers from the record
		headers := []HeaderInBlock{}
		for i := 0; i < *recordSize; i++ {
			tr := tar.NewReader(bytes.NewReader(bf[blockSize*i : blockSize*(i+1)]))
			hdr, err := tr.Next()
			if err != nil {
				continue
			}

			if hdr.Format == tar.FormatUnknown {
				// EOF
				break
			}

			headers = append(headers, HeaderInBlock{
				Record: record,
				Block:  i,
				Header: fmt.Sprintf("%v", hdr),
			})
		}

		if len(headers) > 0 {
			fmt.Println(headers)
		}

		record++
	}
}
