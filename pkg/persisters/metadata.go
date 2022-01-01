package persisters

//go:generate sqlboiler sqlite3 -o ../db/sqlite/models/metadata -c ../../configs/sqlboiler/metadata.yaml
//go:generate go-bindata -pkg metadata -o ../db/sqlite/migrations/metadata/migrations.go ../../db/sqlite/migrations/metadata

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/pojntfx/stfs/internal/db/sqlite/migrations/metadata"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/pathext"
	ipersisters "github.com/pojntfx/stfs/internal/persisters"
	"github.com/pojntfx/stfs/pkg/config"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type depth struct {
	Depth int64 `boil:"depth" json:"depth" toml:"depth" yaml:"depth"`
}

type MetadataPersister struct {
	*ipersisters.SQLite
}

func NewMetadataPersister(dbPath string) *MetadataPersister {
	return &MetadataPersister{
		&ipersisters.SQLite{
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
	hdr := *dbhdr

	if _, err := models.FindHeader(ctx, p.DB, hdr.Name, models.HeaderColumns.Name); err != nil {
		if err == sql.ErrNoRows {
			if _, err := models.FindHeader(ctx, p.DB, p.withRelativeRoot(ctx, hdr.Name), models.HeaderColumns.Name); err == nil {
				hdr.Name = p.withRelativeRoot(ctx, hdr.Name)
			} else {
				if err := hdr.Insert(ctx, p.DB, boil.Infer()); err != nil {
					return err
				}

				return nil
			}

		} else {
			return err
		}
	}

	if _, err := hdr.Update(ctx, p.DB, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func (p *MetadataPersister) UpdateHeaderMetadata(ctx context.Context, dbhdr *models.Header) error {
	if _, err := dbhdr.Update(ctx, p.DB, boil.Infer()); err != nil {
		if err == sql.ErrNoRows {
			hdr := *dbhdr
			hdr.Name = p.withRelativeRoot(ctx, dbhdr.Name)

			if _, err := hdr.Update(ctx, p.DB, boil.Infer()); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (p *MetadataPersister) MoveHeader(ctx context.Context, oldName string, newName string, lastknownrecord, lastknownblock int64) error {
	// We can't do this with `dbhdr.Update` because we are renaming the primary key
	n, err := queries.Raw(
		fmt.Sprintf(
			`update %v set %v = ?, %v = ?, %v = ? where %v = ?;`,
			models.TableNames.Headers,
			models.HeaderColumns.Name,
			models.HeaderColumns.Lastknownrecord,
			models.HeaderColumns.Lastknownblock,
			models.HeaderColumns.Name,
		),
		newName,
		lastknownrecord,
		lastknownblock,
		oldName,
	).ExecContext(ctx, p.DB)
	if err != nil {
		return err
	}

	written, err := n.RowsAffected()
	if err != nil {
		return err
	}

	if written < 1 {
		if _, err := queries.Raw(
			fmt.Sprintf(
				`update %v set %v = ?, %v = ?, %v = ? where %v = ?;`,
				models.TableNames.Headers,
				p.withRelativeRoot(ctx, models.HeaderColumns.Name),
				models.HeaderColumns.Lastknownrecord,
				models.HeaderColumns.Lastknownblock,
				p.withRelativeRoot(ctx, models.HeaderColumns.Name),
			),
			newName,
			lastknownrecord,
			lastknownblock,
			oldName,
		).ExecContext(ctx, p.DB); err != nil {
			return err
		}
	}

	return nil
}

func (p *MetadataPersister) GetHeaders(ctx context.Context) (models.HeaderSlice, error) {
	return models.Headers(
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).All(ctx, p.DB)
}

func (p *MetadataPersister) GetHeader(ctx context.Context, name string) (*models.Header, error) {
	hdr, err := models.Headers(
		qm.Where(models.HeaderColumns.Name+" = ?", name),
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).One(ctx, p.DB)
	if err != nil {
		if err == sql.ErrNoRows {
			hdr, err = models.Headers(
				qm.Where(models.HeaderColumns.Name+" = ?", p.withRelativeRoot(ctx, name)),
				qm.Where(models.HeaderColumns.Deleted+" != 1"),
			).One(ctx, p.DB)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return hdr, nil
}

func (p *MetadataPersister) GetHeaderChildren(ctx context.Context, name string) (models.HeaderSlice, error) {
	headers, err := models.Headers(
		qm.Where(models.HeaderColumns.Name+" like ?", strings.TrimSuffix(name, "/")+"/%"), // Prevent double trailing slashes
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).All(ctx, p.DB)
	if err != nil {
		return nil, err
	}

	if len(headers) < 1 {
		headers, err = models.Headers(
			qm.Where(models.HeaderColumns.Name+" like ?", p.withRelativeRoot(ctx, strings.TrimSuffix(name, "/")+"/%")), // Prevent double trailing slashes
			qm.Where(models.HeaderColumns.Deleted+" != 1"),
		).All(ctx, p.DB)
		if err != nil {
			return nil, err
		}
	}

	outhdrs := models.HeaderSlice{}
	for _, hdr := range headers {
		prefix := strings.TrimSuffix(hdr.Name, "/")
		if name != prefix && name != prefix+"/" {
			outhdrs = append(outhdrs, hdr)
		}
	}

	return outhdrs, nil
}

func (p *MetadataPersister) GetRootPath(ctx context.Context) (string, error) {
	root := models.Header{}

	if err := queries.Raw(
		fmt.Sprintf(
			`select min(length(%v) - length(replace(%v, "/", ""))) as depth, name from %v where %v != 1`,
			models.HeaderColumns.Name,
			models.HeaderColumns.Name,
			models.TableNames.Headers,
			models.HeaderColumns.Deleted,
		),
	).Bind(ctx, p.DB, &root); err != nil {
		if strings.Contains(err.Error(), "converting NULL to string is unsupported") {
			return "", config.ErrNoRootDirectory
		}

		return "", err
	}

	return root.Name, nil
}

func (p *MetadataPersister) GetHeaderDirectChildren(ctx context.Context, name string, limit int) (models.HeaderSlice, error) {
	prefix := strings.TrimSuffix(name, "/") + "/"
	rootDepth := 0
	headers := models.HeaderSlice{}

	// Root node
	if pathext.IsRoot(name) {
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
		).Bind(ctx, p.DB, &depth); err != nil {
			if err == sql.ErrNoRows {
				return headers, nil
			}

			return nil, err
		}

		rootDepth = int(depth.Depth)
	}

	getHeaders := func(prefix string) (models.HeaderSlice, error) {
		query := fmt.Sprintf(
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
		)

		if limit < 0 {
			if err := queries.Raw(
				query+`limit ?`,
				prefix,
				prefix,
				prefix+"%",
				rootDepth,
				rootDepth+1,
				limit+1, // +1 to accomodate the parent directory if it exists
			).Bind(ctx, p.DB, &headers); err != nil {
				if err == sql.ErrNoRows {
					return headers, nil
				}

				return nil, err
			}
		}

		if err := queries.Raw(
			query,
			prefix,
			prefix,
			prefix+"%",
			rootDepth,
			rootDepth+1,
		).Bind(ctx, p.DB, &headers); err != nil {
			if err == sql.ErrNoRows {
				return headers, nil
			}

			return nil, err
		}

		return headers, nil
	}

	headers, err := getHeaders(prefix)
	if err != nil {
		headers, err = getHeaders(p.withRelativeRoot(ctx, prefix))
		if err == sql.ErrNoRows {
			return headers, nil
		}

		if err != nil {
			return nil, err
		}
	}

	outhdrs := models.HeaderSlice{}
	for _, hdr := range headers {
		prefix := strings.TrimSuffix(hdr.Name, "/")
		if name != prefix && name != prefix+"/" {
			outhdrs = append(outhdrs, hdr)
		}
	}

	if limit < 0 || len(outhdrs) < limit {
		return outhdrs, nil
	}

	return outhdrs[:limit-1], nil
}

func (p *MetadataPersister) DeleteHeader(ctx context.Context, name string, lastknownrecord, lastknownblock int64) (*models.Header, error) {
	hdr, err := models.FindHeader(ctx, p.DB, name)
	if err != nil {
		if err == sql.ErrNoRows {
			hdr, err = models.FindHeader(ctx, p.DB, p.withRelativeRoot(ctx, name))
			if err == sql.ErrNoRows {
				return nil, err
			}

			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	hdr.Deleted = 1
	hdr.Lastknownrecord = lastknownrecord
	hdr.Lastknownblock = lastknownblock

	if _, err := hdr.Update(ctx, p.DB, boil.Infer()); err != nil {
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
	).Bind(ctx, p.DB, &header); err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, nil
		}

		return 0, 0, err
	}

	return header.Lastknownrecord, header.Lastknownblock, nil
}

func (p *MetadataPersister) PurgeAllHeaders(ctx context.Context) error {
	if _, err := models.Headers().DeleteAll(ctx, p.DB); err != nil {
		return err
	}

	return nil
}

func (p *MetadataPersister) headerExistsExact(ctx context.Context, name string) error {
	exists, err := models.Headers(
		qm.Where(models.HeaderColumns.Name+" = ?", name),
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).Exists(ctx, p.DB)
	if err != nil {
		return err
	}

	if !exists {
		return sql.ErrNoRows
	}

	return nil
}

func (p *MetadataPersister) withRelativeRoot(ctx context.Context, root string) string {
	prefix := ""
	if err := p.headerExistsExact(ctx, ""); err == nil {
		prefix = ""
	} else if err := p.headerExistsExact(ctx, "."); err == nil {
		prefix = "."
	} else if err := p.headerExistsExact(ctx, "/"); err == nil {
		prefix = "/"
	} else {
		prefix = "./" // Special case: There is no root directory, only files, and the files start with `./`
	}

	if pathext.IsRoot(root) {
		return prefix
	}

	if prefix == "./" {
		// Special case: There is no root directory, only files, and the files start with `./`; we can't do path.Join, as `./asdf.txt` would be shortened to `asdf.txt`
		return prefix + filepath.Clean(strings.TrimPrefix(root, "/"))
	}

	return path.Join(prefix, filepath.Clean(strings.TrimPrefix(root, "/")))
}
