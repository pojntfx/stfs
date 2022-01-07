package persisters

import (
	"database/sql"
	"os"
	"path/filepath"

	migrate "github.com/rubenv/sql-migrate"
	_ "modernc.org/sqlite"
)

type SQLite struct {
	DBPath     string
	Migrations migrate.MigrationSource

	DB *sql.DB
}

func (s *SQLite) Open() error {
	// Create leading directories for database
	leadingDir, _ := filepath.Split(s.DBPath)
	if err := os.MkdirAll(leadingDir, os.ModePerm); err != nil {
		return err
	}

	// Open the DB
	db, err := sql.Open("sqlite", s.DBPath)
	if err != nil {
		return err
	}

	// Configure the db
	db.SetMaxOpenConns(1) // Prevent "database locked" errors
	s.DB = db

	// Run migrations if set
	if s.Migrations != nil {
		if _, err := migrate.Exec(s.DB, "sqlite3", s.Migrations, migrate.Up); err != nil {
			return err
		}
	}

	return nil
}
