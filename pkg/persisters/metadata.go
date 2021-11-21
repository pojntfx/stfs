package persisters

//go:generate sqlboiler sqlite3 -o ../db/sqlite/models/metadata -c ../../../configs/sqlboiler/metadata.yaml
//go:generate go-bindata -pkg metadata -o ../db/sqlite/migrations/metadata/migrations.go ../../../db/sqlite/migrations/metadata

import (
	"context"
	"database/sql"

	"github.com/pojntfx/stfs/pkg/db/sqlite/migrations/metadata"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type MetadataPersister struct {
	*SQLite
}

func NewMetadataPersister(dbPath string) *MetadataPersister {
	return &MetadataPersister{
		&SQLite{
			DBPath: dbPath,
			Migrations: migrate.AssetMigrationSource{
				Asset:    metadata.Asset,
				AssetDir: metadata.AssetDir,
				Dir:      "../../../db/sqlite/migrations/metadata",
			},
		},
	}
}

func (p *MetadataPersister) UpsertHeader(ctx context.Context, dbhdr *models.Header) error {
	if _, err := models.FindHeader(ctx, p.db, dbhdr.Name, models.HeaderColumns.Name); err != nil {
		if err == sql.ErrNoRows {
			if err := dbhdr.Insert(ctx, p.db, boil.Infer()); err != nil {
				return err
			}

			return nil
		}

		return err
	}

	if _, err := dbhdr.Update(ctx, p.db, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func (p *MetadataPersister) GetHeaders(ctx context.Context) (models.HeaderSlice, error) {
	return models.Headers().All(ctx, p.db)
}

func (p *MetadataPersister) DeleteHeader(ctx context.Context, name string, ignoreNotExists bool) (*models.Header, error) {
	hdr, err := models.FindHeader(ctx, p.db, name)
	if err != nil {
		if err == sql.ErrNoRows && ignoreNotExists {
			return nil, nil
		}

		return nil, err
	}

	if _, err := hdr.Delete(ctx, p.db); err != nil {
		return nil, err
	}

	return hdr, nil
}
