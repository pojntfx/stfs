package main

import (
	"archive/tar"
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"
)

// See https://github.com/benmcclelland/mtio
const (
	MTIOCPOS = 0x80086d03 // Get tape position

	blockSize = 512
)

// Position is struct for MTIOCPOS
type Position struct {
	BlkNo int64 // Current block number
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

	br := bufio.NewReaderSize(f, blockSize**recordSize)
	tr := tar.NewReader(br)

	record := int64(0)
	block := int64(0)

	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			// Seek one block backwards (half into the trailer) into trailer
			if fileDescription.Mode().IsRegular() {
				if _, err := f.Seek((int64(*recordSize)*blockSize*record)+block*blockSize, io.SeekStart); err == nil {
					tr = tar.NewReader(br)

					hdr, err = tr.Next()
					if err != nil {
						panic(err)
					}
				} else {
					panic(err)
				}
			} else {
				panic(err)
			}
		}

		curr := int64(0)
		if fileDescription.Mode().IsRegular() {
			curr, err = f.Seek(0, io.SeekCurrent)
			if err != nil {
				panic(err)
			}
		} else {
			pos := &Position{}

			syscall.Syscall(
				syscall.SYS_IOCTL,
				f.Fd(),
				MTIOCPOS,
				uintptr(unsafe.Pointer(pos)),
			)

			// TODO: Ensure that this is in fact the block, not just the record
			curr = pos.BlkNo * blockSize
		}

		if record == 0 && block == 0 {
			log.Println("Record:", 0, "Block:", 0, "Header:", hdr)
		} else {
			log.Println("Record:", record, "Block:", block, "Header:", hdr)
		}

		nextTotalBlocks := (curr + hdr.Size) / blockSize

		if record == 0 && block == 0 {
			record = nextTotalBlocks / int64(*recordSize)
			block = nextTotalBlocks - (record * int64(*recordSize)) // For the first record, the offset of one is not needed
		} else {
			record = nextTotalBlocks / int64(*recordSize)
			block = nextTotalBlocks - (record * int64(*recordSize)) + 1 // +1 because we need to start reading right after the last block
		}

		if block > int64(*recordSize) {
			record++
			block = 0
		}
	}
}
