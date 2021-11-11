package main

import (
	"archive/tar"
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	file := flag.String("file", "/dev/st0", "File (tape drive or tar file) to open")
	blockSize := flag.Int("blockSize", 512, "Size of a block in the tar stream")
	blockSizeMulitlier := flag.Int("blockSizeMultiplier", 20, "Amount of blocks to read from the tar stream at once")

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

	br := bufio.NewReaderSize(f, *blockSize**blockSizeMulitlier)
	tr := tar.NewReader(br)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}

		fmt.Printf(
			"%v %v %v %v %v %v %v %v %v\n",
			header.Mode,

			header.Gname,
			header.Uid,
			header.Gid,

			header.Size,

			header.ModTime,
			header.AccessTime,
			header.ChangeTime,

			header.Name,
		)
	}
}
