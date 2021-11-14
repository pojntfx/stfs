package main

import (
	"flag"
	"os"
	"syscall"
	"unsafe"
)

// See https://github.com/benmcclelland/mtio/blob/f929531fb4fe6433f7198ccd89d1c1414ef8fa3f/mtst.go#L46
const (
	MTIOCTOP = 0x40086d01 // Do magnetic tape operation
	MTSEEK   = 22         // Seek to block
)

// Operation is struct for MTIOCTOP
type Operation struct {
	op int16 // Operation ID

	pad int16 // Padding to match C structures

	count int32 // Operation count
}

func main() {
	file := flag.String("file", "/dev/nst0", "File of tape drive to open")
	record := flag.Int("record", 0, "Record to seek too")

	flag.Parse()

	f, err := os.OpenFile(*file, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		MTIOCTOP,
		uintptr(unsafe.Pointer(
			&Operation{
				op:    MTSEEK,
				count: int32(*record),
			},
		)),
	)
}
