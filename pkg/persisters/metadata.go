package persisters

//go:generate sqlboiler sqlite3 -o ../db/sqlite/models/metadata -c ../../../configs/sqlboiler/metadata.yaml
//go:generate go-bindata -pkg metadata -o ../db/sqlite/migrations/metadata/migrations.go ../../../db/sqlite/migrations/metadata

import (
	"archive/tar"
	"context"
	"database/sql"
	"encoding/json"

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

func (c *MetadataPersister) UpsertHeader(ctx context.Context, record, block int64, hdr *tar.Header) error {
	paxRecords, err := json.Marshal(hdr.PAXRecords)
	if err != nil {
		return err
	}

	dbhdr := models.Header{
		Record:     record,
		Block:      block,
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
		Paxrecords: string(paxRecords),
		Format:     int64(hdr.Format),
	}

	if _, err := models.FindHeader(ctx, c.db, dbhdr.Name, models.HeaderColumns.Name); err != nil {
		if err == sql.ErrNoRows {
			if err := dbhdr.Insert(ctx, c.db, boil.Infer()); err != nil {
				return err
			}

			return nil
		}

		return err
	}

	if _, err := dbhdr.Update(ctx, c.db, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func (p *MetadataPersister) GetHeaders(ctx context.Context) (models.HeaderSlice, error) {
	return models.Headers().All(ctx, p.db)
}
