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
	MTIOCTOP = 0x40086d01 // Do magnetic tape operation
	MTFSF    = 1          // Forward space over FileMark, position at first record of next file

	blockSize = 512
)

// Position is struct for MTIOCPOS
type Position struct {
	BlkNo int64 // Current block number
}

// Operation is struct for MTIOCTOP
type Operation struct {
	Op    int16 // Operation ID
	Pad   int16 // Padding to match C structures
	Count int32 // Operation count
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

	if fileDescription.Mode().IsRegular() {
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
	} else {
		br := bufio.NewReaderSize(f, blockSize**recordSize)
		tr := tar.NewReader(br)

		record := int64(0)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				if err := goToNextFileOnTape(f); err != nil {
					panic(err)
				}

				br = bufio.NewReaderSize(f, blockSize**recordSize)
				tr = tar.NewReader(br)

				hdr, err = tr.Next()
				if err != nil {
					if err == io.EOF {
						break
					}

					panic(err)
				}
			}

			if err != nil {
				panic(err)
			}

			if record == 0 {
				log.Println("Record:", 0, "Block:", 0, "Header:", hdr)
			} else {
				log.Println("Record:", record, "Block:", 0, "Header:", hdr)
			}

			curr, err := getCurrentRecordFromTape(f)
			if err != nil {
				panic(err)
			}

			if record == 0 {
				record = ((curr * int64(*recordSize) * blockSize) + hdr.Size) / (int64(*recordSize) * blockSize) // For the first record of the file or archive, the offset of one is not needed
			} else {
				record = ((curr*int64(*recordSize)*blockSize)+hdr.Size)/(int64(*recordSize)*blockSize) + 2 // +2 because we need to start reading right after the last block
			}
		}
	}
}

func goToNextFileOnTape(f *os.File) error {
	if _, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		MTIOCTOP,
		uintptr(unsafe.Pointer(
			&Operation{
				Op:    MTFSF,
				Count: 1,
			},
		)),
	); err != 0 {
		return err
	}

	return nil
}

func getCurrentRecordFromTape(f *os.File) (int64, error) {
	pos := &Position{}
	if _, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		MTIOCPOS,
		uintptr(unsafe.Pointer(pos)),
	); err != 0 {
		return 0, err
	}

	return pos.BlkNo, nil
}
