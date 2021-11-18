package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pojntfx/stfs/pkg/controllers"
)

func main() {
	file := flag.String("file", "/dev/nst0", "File of tape drive to open")

	flag.Parse()

	f, err := os.OpenFile(*file, os.O_RDONLY, os.ModeCharDevice)
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
