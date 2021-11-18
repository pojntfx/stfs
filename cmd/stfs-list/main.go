package main

import (
	"archive/tar"
	"bufio"
	"flag"
	"io"
	"log"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/readers"
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

	if fileDescription.Mode().IsRegular() {
		tr := tar.NewReader(f)

		record := int64(0)
		block := int64(0)
		firstRecordOfArchive := int64(0)

		for {
			hdr, err := tr.Next()
			if err != nil {
				// Seek right after the next two blocks to skip the trailer
				if _, err := f.Seek((int64(*recordSize)*controllers.BlockSize*record)+(block+1)*controllers.BlockSize, io.SeekStart); err == nil {
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
			}

			if record == 0 && block == 0 {
				log.Println("Record:", 0, "Block:", 0, "Header:", hdr)
			} else {
				log.Println("Record:", record, "Block:", block, "Header:", hdr)
			}

			curr, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				panic(err)
			}

			nextTotalBlocks := (curr + hdr.Size) / controllers.BlockSize
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
	} else {
		br := bufio.NewReaderSize(f, controllers.BlockSize**recordSize)

		counter := &readers.Counter{Reader: br}
		lastBytesRead := 0
		dirty := false

		record := int64(0)
		block := int64(0)

		for {
			tr := tar.NewReader(counter)
			hdr, err := tr.Next()
			if err != nil {
				if lastBytesRead == counter.BytesRead {
					if dirty {
						// EOD

						break
					}

					if err := controllers.GoToNextFileOnTape(f); err != nil {
						// EOD

						break
					}

					currentRecord, err := controllers.GetCurrentRecordFromTape(f)
					if err != nil {
						panic(err)
					}

					br = bufio.NewReaderSize(f, controllers.BlockSize**recordSize)
					counter = &readers.Counter{Reader: br, BytesRead: (int(currentRecord) * *recordSize * controllers.BlockSize)} // We asume we are at record n, block 0

					dirty = true
				}

				lastBytesRead = counter.BytesRead

				continue
			}

			lastBytesRead = counter.BytesRead

			if hdr.Format == tar.FormatUnknown {
				continue
			}

			dirty = false

			log.Println("Record:", record, "Block:", block, "Header:", hdr)

			nextBytes := int64(counter.BytesRead) + hdr.Size + controllers.BlockSize - 1

			record = nextBytes / (controllers.BlockSize * int64(*recordSize))
			block = (nextBytes - (record * int64(*recordSize) * controllers.BlockSize)) / controllers.BlockSize
		}
	}
}
