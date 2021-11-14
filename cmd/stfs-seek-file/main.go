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
	file := flag.String("file", "test.tar", "Tar file to open")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	checkpoint := flag.Int("checkpoint", 0, "Log current record after checkpoint kilobytes have been read")
	seek := flag.Int("seek", 0, "Record to seek too")

	flag.Parse()

	bytesToSeek := *recordSize * blockSize * *seek

	f, err := os.Open(*file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	record := 0
	for {
		// Seek to requested record
		if bytesToSeek > 0 && record < *seek {
			if _, err := f.Seek(int64(bytesToSeek), 0); err != nil {
				panic(err)
			}

			record = *seek

			continue
		}

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
