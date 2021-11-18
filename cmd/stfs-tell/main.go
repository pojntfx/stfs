package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// See https://github.com/benmcclelland/mtio
const (
	MTIOCPOS = 0x80086d03 // Get tape position
)

// Position is struct for MTIOCPOS
type Position struct {
	BlkNo int64 // Current block number
}

func main() {
	file := flag.String("file", "/dev/nst0", "File of tape drive to open")

	flag.Parse()

	f, err := os.OpenFile(*file, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	currentRecord, err := getCurrentRecordFromTape(f)
	if err != nil {
		panic(err)
	}

	fmt.Println(currentRecord)
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
