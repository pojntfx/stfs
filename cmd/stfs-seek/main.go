package main

import (
	"archive/tar"
	"bufio"
	"flag"
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
	read := flag.Bool("read", false, "Whether to read the next header")

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

	var tr *tar.Reader
	if fileDescription.Mode().IsRegular() {
		// Seek to record and block
		if _, err := f.Seek(int64((*recordSize*blockSize**record)+*block*blockSize), 0); err != nil {
			panic(err)
		}

		tr = tar.NewReader(f)
	} else {
		// Seek to record
		if err := seekToRecordOnTape(f, int32(*record)); err != nil {
			panic(err)
		}

		// Seek to block
		br := bufio.NewReaderSize(f, blockSize**recordSize)
		if _, err := br.Read(make([]byte, *block*blockSize)); err != nil {
			panic(err)
		}

		tr = tar.NewReader(br)
	}

	if *read {
		hdr, err := tr.Next()
		if err != nil {
			panic(err)
		}

		log.Println(hdr)
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
