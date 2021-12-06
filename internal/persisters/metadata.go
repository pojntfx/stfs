package persisters

//go:generate sqlboiler sqlite3 -o ../db/sqlite/models/metadata -c ../../configs/sqlboiler/metadata.yaml
//go:generate go-bindata -pkg metadata -o ../db/sqlite/migrations/metadata/migrations.go ../../db/sqlite/migrations/metadata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pojntfx/stfs/internal/db/sqlite/migrations/metadata"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type depth struct {
	Depth int64 `boil:"depth" json:"depth" toml:"depth" yaml:"depth"`
}

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
				Dir:      "../../db/sqlite/migrations/metadata",
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

func (p *MetadataPersister) UpdateHeaderMetadata(ctx context.Context, dbhdr *models.Header) error {
	if _, err := dbhdr.Update(ctx, p.db, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func (p *MetadataPersister) moveHeader(ctx context.Context, tx boil.ContextExecutor, oldName string, newName string) error {
	// We can't do this with `dbhdr.Update` because we are renaming the primary key
	if _, err := queries.Raw(
		fmt.Sprintf(
			` update %v set %v = ? where %v = ?;`,
			models.TableNames.Headers,
			models.HeaderColumns.Name,
			models.HeaderColumns.Name,
		),
		newName,
		oldName,
	).ExecContext(ctx, tx); err != nil {
		return err
	}

	return nil
}

func (p *MetadataPersister) MoveHeader(ctx context.Context, oldName string, newName string) error {
	return p.moveHeader(ctx, p.db, oldName, newName)
}

func (p *MetadataPersister) MoveHeaders(ctx context.Context, hdrs models.HeaderSlice, oldName string, newName string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, hdr := range hdrs {
		if err := p.moveHeader(ctx, tx, hdr.Name, strings.TrimSuffix(newName, "/")+strings.TrimPrefix(hdr.Name, strings.TrimSuffix(oldName, "/"))); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *MetadataPersister) GetHeaders(ctx context.Context) (models.HeaderSlice, error) {
	return models.Headers(
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).All(ctx, p.db)
}

func (p *MetadataPersister) GetHeader(ctx context.Context, name string) (*models.Header, error) {
	return models.FindHeader(ctx, p.db, name)
}

func (p *MetadataPersister) GetHeaderChildren(ctx context.Context, name string) (models.HeaderSlice, error) {
	return models.Headers(
		qm.Where(models.HeaderColumns.Name+" like ?", strings.TrimSuffix(name, "/")+"/%"), // Prevent double trailing slashes
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).All(ctx, p.db)
}

func (p *MetadataPersister) GetHeaderDirectChildren(ctx context.Context, name string) (models.HeaderSlice, error) {
	prefix := strings.TrimSuffix(name, "/") + "/"
	rootDepth := 0
	headers := models.HeaderSlice{}

	// Root node
	if name == "" || name == "." || name == "/" || name == "./" {
		prefix = ""
		depth := depth{}

		if err := queries.Raw(
			fmt.Sprintf(
				`select min(length(%v) - length(replace(%v, "/", ""))) as depth from %v where %v != 1`,
				models.HeaderColumns.Name,
				models.HeaderColumns.Name,
				models.TableNames.Headers,
				models.HeaderColumns.Deleted,
			),
		).Bind(ctx, p.db, &depth); err != nil {
			if err == sql.ErrNoRows {
				return headers, nil
			}

			return nil, err
		}

		rootDepth = int(depth.Depth)
	}

	if err := queries.Raw(
		fmt.Sprintf(
			`select %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v,
    length(replace(%v, ?, '')) - length(replace(replace(%v, ?, ''), '/', '')) as depth
from %v
where %v like ?
    and (
        depth = ?
        or (
            %v like '%%/'
            and depth = ?
        )
    )
	and %v != 1
    and not %v in ('', '.', '/', './')`,
			models.HeaderColumns.Record,
			models.HeaderColumns.Lastknownrecord,
			models.HeaderColumns.Block,
			models.HeaderColumns.Lastknownblock,
			models.HeaderColumns.Deleted,
			models.HeaderColumns.Typeflag,
			models.HeaderColumns.Name,
			models.HeaderColumns.Linkname,
			models.HeaderColumns.Size,
			models.HeaderColumns.Mode,
			models.HeaderColumns.UID,
			models.HeaderColumns.Gid,
			models.HeaderColumns.Uname,
			models.HeaderColumns.Gname,
			models.HeaderColumns.Modtime,
			models.HeaderColumns.Accesstime,
			models.HeaderColumns.Changetime,
			models.HeaderColumns.Devmajor,
			models.HeaderColumns.Devminor,
			models.HeaderColumns.Paxrecords,
			models.HeaderColumns.Format,
			models.HeaderColumns.Name,
			models.HeaderColumns.Name,
			models.TableNames.Headers,
			models.HeaderColumns.Name,
			models.HeaderColumns.Name,
			models.HeaderColumns.Deleted,
			models.HeaderColumns.Name,
		),
		prefix,
		prefix,
		prefix+"%",
		rootDepth,
		rootDepth+1,
	).Bind(ctx, p.db, &headers); err != nil {
		if err == sql.ErrNoRows {
			return headers, nil
		}

		return nil, err
	}

	return headers, nil
}

func (p *MetadataPersister) DeleteHeader(ctx context.Context, name string, lastknownrecord, lastknownblock int64, ignoreNotExists bool) (*models.Header, error) {
	hdr, err := models.FindHeader(ctx, p.db, name)
	if err != nil {
		if err == sql.ErrNoRows && ignoreNotExists {
			return nil, nil
		}

		return nil, err
	}

	if hdr != nil && hdr.Deleted == 1 && !ignoreNotExists {
		return nil, sql.ErrNoRows
	}

	hdr.Deleted = 1
	hdr.Lastknownrecord = lastknownrecord
	hdr.Lastknownblock = lastknownblock

	if _, err := hdr.Update(ctx, p.db, boil.Infer()); err != nil {
		return nil, err
	}

	return hdr, nil
}

func (p *MetadataPersister) GetLastIndexedRecordAndBlock(ctx context.Context, recordSize int) (int64, int64, error) {
	var header models.Header
	if err := queries.Raw(
		fmt.Sprintf(
			`select %v, %v, ((%v*$1)+%v) as location from %v order by location desc limit 1`, // We include deleted headers here as they are still physically on the tape and have to be considered when re-indexing
			models.HeaderColumns.Lastknownrecord,
			models.HeaderColumns.Lastknownblock,
			models.HeaderColumns.Lastknownrecord,
			models.HeaderColumns.Lastknownblock,
			models.TableNames.Headers,
		),
		recordSize,
	).Bind(ctx, p.db, &header); err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, nil
		}

		return 0, 0, err
	}

	return header.Lastknownrecord, header.Lastknownblock, nil
}
