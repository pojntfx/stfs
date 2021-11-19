package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
)

func main() {
	drive := flag.String("drive", "/dev/nst0", "Tape drive to get position from")

	flag.Parse()

	f, err := os.OpenFile(*drive, os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	currentRecord, err := controllers.GetCurrentRecordFromTape(f)
	if err != nil {
		panic(err)
	}

	fmt.Println(currentRecord)
}
