package persisters

//go:generate sqlboiler sqlite3 -o ../db/sqlite/models/metadata -c ../../../configs/sqlboiler/metadata.yaml
//go:generate go-bindata -pkg metadata -o ../db/sqlite/migrations/metadata/migrations.go ../../../db/sqlite/migrations/metadata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pojntfx/stfs/pkg/db/sqlite/migrations/metadata"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
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

func (p *MetadataPersister) GetHeader(ctx context.Context, name string) (*models.Header, error) {
	return models.FindHeader(ctx, p.db, name)
}

func (p *MetadataPersister) GetHeaderChildren(ctx context.Context, name string) (models.HeaderSlice, error) {
	return models.Headers(
		qm.Where(models.HeaderColumns.Name+" like ?", strings.TrimSuffix(name, "/")+"/%"), // Prevent double trailing slashes
	).All(ctx, p.db)
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

func (p *MetadataPersister) DeleteHeaders(ctx context.Context, hdrs models.HeaderSlice) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, hdr := range hdrs {
		if _, err := hdr.Delete(ctx, tx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *MetadataPersister) GetLastIndexedRecordAndBlock(ctx context.Context, recordSize int) (int64, int64, error) {
	var header models.Header
	if err := queries.Raw(
		fmt.Sprintf(
			`select %v, %v, ((%v*$1)+%v) as location from %v order by location desc limit 1`,
			models.HeaderColumns.Record,
			models.HeaderColumns.Block,
			models.HeaderColumns.Record,
			models.HeaderColumns.Block,
			models.TableNames.Headers,
		),
		recordSize,
	).Bind(ctx, p.db, &header); err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, nil
		}

		return 0, 0, err
	}

	return header.Record, header.Block, nil
}
