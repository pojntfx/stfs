package main

import (
	"bufio"
	"flag"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
)

func main() {
	drive := flag.String("drive", "/dev/nst0", "Tape drive to seek on")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	record := flag.Int("record", 0, "Record to seek too")
	block := flag.Int("block", 0, "Block in record to seek too")

	flag.Parse()

	f, err := os.OpenFile(*drive, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Seek to record
	if err := controllers.SeekToRecordOnTape(f, int32(*record)); err != nil {
		panic(err)
	}

	// Seek to block
	br := bufio.NewReaderSize(f, controllers.BlockSize**recordSize)
	if _, err := br.Read(make([]byte, *block*controllers.BlockSize)); err != nil {
		panic(err)
	}
}
