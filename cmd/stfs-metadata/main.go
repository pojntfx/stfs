package main

//go:generate sqlboiler sqlite3 -o ../../pkg/db/sqlite/models/metadata -c ../../configs/sqlboiler/metadata.toml
//go:generate go-bindata -pkg metadata -o ../../pkg/db/sqlite/migrations/metadata/migrations.go ../../db/sqlite/migrations/metadata

import (
	"archive/tar"
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pojntfx/stfs/pkg/db/sqlite/migrations/metadata"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	blockSize = 512
)

type HeaderInBlock struct {
	Record int
	Block  int
	Header string
}

func main() {
	dbPath := flag.String("db", "/tmp/stfs-metadata.sqlite", "Database file to use")
	file := flag.String("file", "/dev/nst0", "File (tape drive or tar file) to open")
	recordSize := flag.Int("recordSize", 20, "Amount of 512-bit blocks per record")
	checkpoint := flag.Int("checkpoint", 0, "Log current record after checkpoint kilobytes have been read")

	flag.Parse()

	leading, _ := filepath.Split(*dbPath)
	if err := os.MkdirAll(leading, os.ModePerm); err != nil {
		panic(err)
	}

	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		panic(err)
	}

	if _, err := migrate.Exec(
		db,
		"sqlite3",
		migrate.AssetMigrationSource{
			Asset:    metadata.Asset,
			AssetDir: metadata.AssetDir,
			Dir:      "../../db/sqlite/migrations/metadata",
		},
		migrate.Up,
	); err != nil {
		panic(err)
	}

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

	record := 0
	for {
		// Lock the current record if requested
		if *checkpoint > 0 && record%*checkpoint == 0 {
			log.Println("Checkpoint:", record)
		}

		// Read exactly one record
		bf := make([]byte, *recordSize*blockSize)
		if _, err := io.ReadFull(f, bf); err != nil {
			if err == io.EOF {
				break
			}

			// Missing trailer (expected for concatenated tars)
			if err == io.ErrUnexpectedEOF {
				break
			}

			panic(err)
		}

		// Get the headers from the record
		for i := 0; i < *recordSize; i++ {
			tr := tar.NewReader(bytes.NewReader(bf[blockSize*i : blockSize*(i+1)]))
			hdr, err := tr.Next()
			if err != nil {
				continue
			}

			if hdr.Format == tar.FormatUnknown {
				// EOF
				break
			}

			dbhdr := &models.Header{
				Typeflag:   int64(hdr.Typeflag),
				Name:       hdr.Name,
				Linkname:   hdr.Linkname,
				Size:       hdr.Size,
				Mode:       hdr.Mode,
				UID:        int64(hdr.Uid),
				Gid:        int64(hdr.Gid),
				Uname:      hdr.Uname,
				Gname:      hdr.Gname,
				Modtime:    hdr.ModTime,
				Accesstime: hdr.AccessTime,
				Changetime: hdr.ChangeTime,
				Devmajor:   hdr.Devmajor,
				Devminor:   hdr.Devminor,
				Format:     int64(hdr.Format),
				Record:     int64(record),
				Block:      int64(i),
			}

			if err := dbhdr.Insert(context.Background(), db, boil.Infer()); err != nil {
				panic(err)
			}

			fmt.Println(dbhdr)
		}

		record++
	}
}
