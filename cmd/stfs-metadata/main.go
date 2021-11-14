package main

import (
	"database/sql"
	"flag"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pojntfx/stfs/pkg/db/sqlite/migrations/metadata"
	migrate "github.com/rubenv/sql-migrate"
)

//go:generate sqlboiler sqlite3 -o ../../pkg/db/sqlite/models/metadata -c ../../configs/sqlboiler/metadata.toml
//go:generate go-bindata -pkg metadata -o ../../pkg/db/sqlite/migrations/metadata/migrations.go ../../db/sqlite/migrations/metadata

func main() {
	dbPath := flag.String("db", "/tmp/stfs-metadata.sqlite", "Database file to use")

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
}
