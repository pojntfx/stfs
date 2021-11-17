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

	blockSize = 512
)

// Operation is struct for MTIOCTOP
type Operation struct {
	Op    int16 // Operation ID
	Pad   int16 // Padding to match C structures
	Count int32 // Operation count
}

func main() {
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	record := flag.Int("record", 0, "Record to seek too")
	block := flag.Int("block", 0, "Block in record to seek too")

	flag.Parse()

	f, err := os.OpenFile(*file, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if _, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		MTIOCTOP,
		uintptr(unsafe.Pointer(
			&Operation{
				Op:    MTSEEK,
				Count: int32(*record),
			},
		)),
	); err != 0 {
		panic(err)
	}

	br := bufio.NewReaderSize(f, blockSize**recordSize)
	if _, err := br.Read(make([]byte, *block*blockSize)); err != nil {
		panic(err)
	}

	tr := tar.NewReader(br)

	hdr, err := tr.Next()
	if err != nil {
		panic(err)
	}

	log.Println(hdr)
}
