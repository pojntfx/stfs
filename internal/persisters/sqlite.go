package persisters

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	migrate "github.com/rubenv/sql-migrate"
)

type SQLite struct {
	DBPath     string
	Migrations migrate.MigrationSource

	db *sql.DB
}

func (s *SQLite) Open() error {
	// Create leading directories for database
	leadingDir, _ := filepath.Split(s.DBPath)
	if err := os.MkdirAll(leadingDir, os.ModePerm); err != nil {
		return err
	}

	// Open the DB
	db, err := sql.Open("sqlite3", s.DBPath)
	if err != nil {
		return err
	}

	// Configure the db
	db.SetMaxOpenConns(1) // Prevent "database locked" errors
	s.db = db

	// Run migrations if set
	if s.Migrations != nil {
		if _, err := migrate.Exec(s.db, "sqlite3", s.Migrations, migrate.Up); err != nil {
			return err
		}
	}

	return nil
}
