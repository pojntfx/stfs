package persisters

//go:generate sqlboiler sqlite3 -o ../../internal/db/sqlite/models/metadata -c ../../configs/sqlboiler/metadata.yaml
//go:generate go-bindata -pkg metadata -o ../../internal/db/sqlite/migrations/metadata/migrations.go ../../db/sqlite/migrations/metadata

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/pojntfx/stfs/internal/converters"
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
	sqlite *ipersisters.SQLite

	root              string
	rootIsEmptyString bool
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
		"",
		false,
	}
}

func (p *MetadataPersister) Open() error {
	if err := p.sqlite.Open(); err != nil {
		return err
	}

	root, err := p.GetRootPath(context.Background())

	// Ignore if root directory can't be found, which can happen i.e. on initial archiving
	if err == config.ErrNoRootDirectory {
		return nil
	}

	if err != nil {
		return err
	}

	p.root = root

	return nil
}

func (p *MetadataPersister) GetRootPath(ctx context.Context) (string, error) {
	// Cache the root directory
	if p.root != "" {
		return p.root, nil
	}

	root := models.Header{}

	if err := queries.Raw(
		fmt.Sprintf(
			`select min(length(%v) - length(replace(%v, "/", ""))) as depth, name from %v where %v != 1`,
			models.HeaderColumns.Name,
			models.HeaderColumns.Name,
			models.TableNames.Headers,
			models.HeaderColumns.Deleted,
		),
	).Bind(ctx, p.sqlite.DB, &root); err != nil {
		if strings.Contains(err.Error(), "converting NULL to string is unsupported") {
			return "", config.ErrNoRootDirectory
		}

		return "", err
	}

	p.root = root.Name

	return root.Name, nil
}

func (p *MetadataPersister) UpsertHeader(ctx context.Context, dbhdr *config.Header, initializing bool) error {
	idbhdr := converters.ConfigHeaderToDBHeader(dbhdr)

	hdr := *idbhdr
	if !initializing {
		hdr.Name = p.getSanitizedPath(ctx, idbhdr.Name)
	}

	if _, err := models.Headers(
		qm.Where(models.HeaderColumns.Name+" = ?", hdr.Name),
		qm.Where(models.HeaderColumns.Linkname+" = ?", hdr.Linkname),
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).One(ctx, p.sqlite.DB); err != nil {
		if err == sql.ErrNoRows {
			if err := hdr.Insert(ctx, p.sqlite.DB, boil.Infer()); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if _, err := hdr.Update(ctx, p.sqlite.DB, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func (p *MetadataPersister) UpdateHeaderMetadata(ctx context.Context, dbhdr *config.Header) error {
	idbhdr := converters.ConfigHeaderToDBHeader(dbhdr)

	hdr := *idbhdr
	hdr.Name = p.getSanitizedPath(ctx, idbhdr.Name)

	if _, err := hdr.Update(ctx, p.sqlite.DB, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func (p *MetadataPersister) MoveHeader(ctx context.Context, oldName string, newName string, lastknownrecord, lastknownblock int64) error {
	newName = p.getSanitizedPath(ctx, newName)
	oldName = p.getSanitizedPath(ctx, oldName)

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
	).ExecContext(ctx, p.sqlite.DB)
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
				models.HeaderColumns.Name,
				models.HeaderColumns.Lastknownrecord,
				models.HeaderColumns.Lastknownblock,
				models.HeaderColumns.Name,
			),
			newName,
			lastknownrecord,
			lastknownblock,
			oldName,
		).ExecContext(ctx, p.sqlite.DB); err != nil {
			return err
		}
	}

	return nil
}

func (p *MetadataPersister) GetHeaders(ctx context.Context) ([]*config.Header, error) {
	dbhdrs, err := models.Headers(
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).All(ctx, p.sqlite.DB)
	if err != nil {
		return []*config.Header{}, err
	}

	hdrs := []*config.Header{}
	for _, dbhdr := range dbhdrs {
		hdrs = append(hdrs, converters.DBHeaderToConfigHeader(dbhdr))
	}

	return hdrs, nil
}

func (p *MetadataPersister) GetHeader(ctx context.Context, name string) (*config.Header, error) {
	name = p.getSanitizedPath(ctx, name)

	hdr, err := models.Headers(
		qm.Where(models.HeaderColumns.Name+" = ?", name),
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).One(ctx, p.sqlite.DB)
	if err != nil {
		return nil, err
	}

	return converters.DBHeaderToConfigHeader(hdr), nil
}

func (p *MetadataPersister) GetHeaderByLinkname(ctx context.Context, linkname string) (*config.Header, error) {
	linkname = p.getSanitizedPath(ctx, linkname)

	hdr, err := models.Headers(
		qm.Where(models.HeaderColumns.Linkname+" = ?", linkname),
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).One(ctx, p.sqlite.DB)
	if err != nil {
		return nil, err
	}

	return converters.DBHeaderToConfigHeader(hdr), nil
}

func (p *MetadataPersister) GetHeaderChildren(ctx context.Context, name string) ([]*config.Header, error) {
	name = p.getSanitizedPath(ctx, name)

	headers, err := models.Headers(
		qm.Where(models.HeaderColumns.Name+" like ?", strings.TrimSuffix(name, "/")+"/%"), // Prevent double trailing slashes
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).All(ctx, p.sqlite.DB)
	if err != nil {
		return nil, err
	}

	outhdrs := []*config.Header{}
	for _, hdr := range headers {
		prefix := strings.TrimSuffix(hdr.Name, "/")
		if name != prefix && name != prefix+"/" {
			outhdrs = append(outhdrs, converters.DBHeaderToConfigHeader(hdr))
		}
	}

	return outhdrs, nil
}

func (p *MetadataPersister) GetHeaderDirectChildren(ctx context.Context, name string, limit int) ([]*config.Header, error) {
	name = p.getSanitizedPath(ctx, name)
	prefix := strings.TrimSuffix(name, "/") + "/"
	rootDepth := 0

	// We want <=, not <
	if limit > 0 {
		limit++
	}

	// Root node
	if pathext.IsRoot(name, false) {
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
		).Bind(ctx, p.sqlite.DB, &depth); err != nil {
			if err == sql.ErrNoRows {
				return []*config.Header{}, nil
			}

			return nil, err
		}

		rootDepth = int(depth.Depth)
	}

	getHeaders := func(prefix string, useLinkname bool) ([]*config.Header, error) {
		pk := models.HeaderColumns.Name
		exclude := fmt.Sprintf(`%v = ""`, models.HeaderColumns.Linkname)
		if useLinkname {
			pk = models.HeaderColumns.Linkname
			exclude = "1" // No need to exclude anything
		}
		headers := []*config.Header{}

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
	and %v
    and not %v in ('', '.', '/', './')`,
			models.HeaderColumns.Record,
			models.HeaderColumns.Lastknownrecord,
			models.HeaderColumns.Block,
			models.HeaderColumns.Lastknownblock,
			models.HeaderColumns.Deleted,
			models.HeaderColumns.Typeflag,
			pk,
			exclude,
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
			pk,
			pk,
			models.TableNames.Headers,
			pk,
			pk,
			models.HeaderColumns.Deleted,
			exclude,
			pk,
		)

		if limit > 0 {
			if err := queries.Raw(
				query+`limit ?`,
				prefix,
				prefix,
				prefix+"%",
				rootDepth,
				rootDepth+1,
				limit+1, // +1 to accomodate the parent directory if it exists
			).Bind(ctx, p.sqlite.DB, &headers); err != nil {
				if err == sql.ErrNoRows {
					return headers, nil
				}

				return nil, err
			}
		} else if limit <= 0 {
			if err := queries.Raw(
				query,
				prefix,
				prefix,
				prefix+"%",
				rootDepth,
				rootDepth+1,
			).Bind(ctx, p.sqlite.DB, &headers); err != nil {
				if err == sql.ErrNoRows {
					return headers, nil
				}

				return nil, err
			}
		}

		return headers, nil
	}

	headers := []*config.Header{}

	nameHeaders, err := getHeaders(prefix, false)
	if err != nil {
		return nil, err
	}

	rawLinknameHeaders, err := getHeaders(prefix, true)
	if err != nil {
		return nil, err
	}

	linknameHeaders := []*config.Header{}
	for _, link := range rawLinknameHeaders {
		name := link.Name
		linkname := link.Linkname

		target, err := p.GetHeader(ctx, name)
		if err != nil {
			if err == sql.ErrNoRows {
				link.Name = linkname
				link.Linkname = name

				linknameHeaders = append(linknameHeaders, link)

				continue
			} else {
				return nil, err
			}
		}

		target.Name = linkname
		target.Linkname = name

		linknameHeaders = append(linknameHeaders, target)
	}

	headers = append(headers, nameHeaders...)
	headers = append(headers, linknameHeaders...)

	outhdrs := []*config.Header{}
	for _, hdr := range headers {
		prefix := strings.TrimSuffix(hdr.Name, "/")
		if name != prefix && name != prefix+"/" {
			outhdrs = append(outhdrs, hdr)
		}
	}

	if limit <= 0 || len(outhdrs) < limit || len(outhdrs) == 0 {
		return outhdrs, nil
	}

	return outhdrs[:limit-1], nil
}

func (p *MetadataPersister) DeleteHeader(ctx context.Context, name string, lastknownrecord, lastknownblock int64) (*config.Header, error) {
	name = p.getSanitizedPath(ctx, name)

	hdr, err := models.Headers(
		qm.Where(models.HeaderColumns.Name+" = ?", name),
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).One(ctx, p.sqlite.DB)
	if err != nil {
		return nil, err
	}

	hdr.Deleted = 1
	hdr.Lastknownrecord = lastknownrecord
	hdr.Lastknownblock = lastknownblock

	if _, err := hdr.Update(ctx, p.sqlite.DB, boil.Infer()); err != nil {
		return nil, err
	}

	return converters.DBHeaderToConfigHeader(hdr), nil
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
	).Bind(ctx, p.sqlite.DB, &header); err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, nil
		}

		return 0, 0, err
	}

	return header.Lastknownrecord, header.Lastknownblock, nil
}

func (p *MetadataPersister) PurgeAllHeaders(ctx context.Context) error {
	if _, err := models.Headers().DeleteAll(ctx, p.sqlite.DB); err != nil {
		return err
	}

	p.root = ""
	p.rootIsEmptyString = false

	return nil
}

func (p *MetadataPersister) headerExistsExact(ctx context.Context, name string) error {
	exists, err := models.Headers(
		qm.Where(models.HeaderColumns.Name+" = ?", name),
		qm.Where(models.HeaderColumns.Deleted+" != 1"),
	).Exists(ctx, p.sqlite.DB)
	if err != nil {
		return err
	}

	if !exists {
		return sql.ErrNoRows
	}

	return nil
}

func (p *MetadataPersister) getSanitizedPath(ctx context.Context, name string) string {
	// If root is queried, return actual root
	if pathext.IsRoot(name, false) {
		return p.root
	}

	// If root has not been set, the incoming path is absolute and no header with the exact name "" (empty string) exists, assume it is root
	if p.root == "" && strings.HasPrefix(name, "/") && !p.rootIsEmptyString {
		if err := p.headerExistsExact(ctx, ""); err != nil {
			p.root = name

			return p.root
		} else {
			p.rootIsEmptyString = true
		}
	}

	// Keep absolute paths untouched if root is also absolute
	if strings.HasPrefix(p.root, "/") && strings.HasPrefix(name, "/") {
		return name
	}

	// Get correct root prefix
	prefix := ""
	if p.root == "" {
		prefix = ""
	} else if p.root == "." {
		prefix = "."
	} else if p.root == "/" {
		prefix = "/"
	} else {
		return "./" + filepath.Clean(strings.TrimPrefix(name, "/")) // Special case: There is no root directory, only files, and the files start with `./`
	}

	return path.Join(prefix, strings.TrimPrefix(name, "/"))
}
