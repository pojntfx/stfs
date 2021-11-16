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

	tr := tar.NewReader(f)

	record := int64(0)
	block := int64(0)
	firstRecordOfArchive := int64(0)

	for {
		hdr, err := tr.Next()
		if err != nil {
			// Seek right after the next two blocks to skip the trailer
			if _, err := f.Seek((int64(*recordSize)*blockSize*record)+(block+1)*blockSize, io.SeekStart); err == nil {
				tr = tar.NewReader(f)

				hdr, err = tr.Next()
				if err != nil {
					if err == io.EOF {
						break
					}

					panic(err)
				}

				block++
				if block > int64(*recordSize) {
					record++
					block = 0
				}

				firstRecordOfArchive = record
			} else {
				panic(err)
			}

			// TODO: Seek with matching syscall (`mt seek (int64(*recordSize)*record)+1`)
		}

		// TODO: Do `tell` on tape drive instead, which returns the block - but how do we get the current block? Maybe we have to use the old, iterating method and call.Next after we found the correct record & block.
		curr, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			panic(err)
		}

		if record == 0 && block == 0 {
			log.Println("Record:", 0, "Block:", 0, "Header:", hdr)
		} else {
			log.Println("Record:", record, "Block:", block, "Header:", hdr)
		}

		nextTotalBlocks := (curr + hdr.Size) / blockSize
		record = nextTotalBlocks / int64(*recordSize)

		if record == 0 && block == 0 || record == firstRecordOfArchive {
			block = nextTotalBlocks - (record * int64(*recordSize)) // For the first record of the file or archive, the offset of one is not needed
		} else {
			block = nextTotalBlocks - (record * int64(*recordSize)) + 1 // +1 because we need to start reading right after the last block
		}

		if block > int64(*recordSize) {
			record++
			block = 0
		}
	}
}
