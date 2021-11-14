package main

import (
	"flag"
	"os"
	"syscall"
	"unsafe"
)

// See https://github.com/benmcclelland/mtio
const (
	MTIOCTOP = 0x40086d01 // Do magnetic tape operation
	MTEOM    = 12         // Goto end of recorded media (for appending files)
)

// Operation is struct for MTIOCTOP
type Operation struct {
	Op    int16 // Operation ID
	Pad   int16 // Padding to match C structures
	Count int32 // Operation count
}

func main() {
	file := flag.String("file", "/dev/nst0", "File of tape drive to open")

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
				Op: MTEOM,
			},
		)),
	)
}
