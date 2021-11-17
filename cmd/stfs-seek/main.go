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
	MTIOCTOP = 0x40086d01 // Do magnetic tape operation
	MTSEEK   = 22         // Seek to block
	MTIOCPOS = 0x80086d03 // Get tape position

	blockSize = 512
)

// Operation is struct for MTIOCTOP
type Operation struct {
	Op    int16 // Operation ID
	Pad   int16 // Padding to match C structures
	Count int32 // Operation count
}

// Position is struct for MTIOCPOS
type Position struct {
	BlkNo int64 // Current block number
}

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	record := flag.Int("record", 0, "Record to seek too")
	block := flag.Int("block", 0, "Block in record to seek too")

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

		// Seek to record & block
		if _, err := f.Seek(int64((*recordSize*blockSize**record)+*block*blockSize), 0); err != nil {
			panic(err)
		}
	} else {
		f, err = os.OpenFile(*file, os.O_RDONLY, os.ModeCharDevice)
		if err != nil {
			panic(err)
		}

		// Seek to record (we can't seek to block on tape)
		// TODO: Seek to next header here too; currently this only works for the start of each archive/after a filemark
		if err := seekToRecordOnTape(f, int32(*record)); err != nil {
			panic(err)
		}
	}
	defer f.Close()

	var tr *tar.Reader
	if fileDescription.Mode().IsRegular() {
		tr = tar.NewReader(f)
	} else {
		br := bufio.NewReaderSize(f, blockSize**recordSize)
		tr = tar.NewReader(br)
	}

	for {
		hdr, err := tr.Next()
		if err != nil {
			panic(err)
		}

		log.Println(hdr)

		currentRecord := int64(0)
		if fileDescription.Mode().IsRegular() {
			curr, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				panic(err)
			}

			currentRecord = ((curr + hdr.Size) / blockSize) / int64(*recordSize)
		} else {
			currentRecord, err = getCurrentRecordFromTape(f)
			if err != nil {
				panic(err)
			}
		}

		if currentRecord > int64(*record) {
			break
		}
	}
}

func seekToRecordOnTape(f *os.File, record int32) error {
	if _, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		MTIOCTOP,
		uintptr(unsafe.Pointer(
			&Operation{
				Op:    MTSEEK,
				Count: record,
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
