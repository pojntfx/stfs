// Code generated by SQLBoiler 4.16.2 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/queries/qmhelper"
	"github.com/volatiletech/strmangle"
)

// Header is an object representing the database table.
type Header struct {
	Record          int64     `boil:"record" json:"record" toml:"record" yaml:"record"`
	Lastknownrecord int64     `boil:"lastknownrecord" json:"lastknownrecord" toml:"lastknownrecord" yaml:"lastknownrecord"`
	Block           int64     `boil:"block" json:"block" toml:"block" yaml:"block"`
	Lastknownblock  int64     `boil:"lastknownblock" json:"lastknownblock" toml:"lastknownblock" yaml:"lastknownblock"`
	Deleted         int64     `boil:"deleted" json:"deleted" toml:"deleted" yaml:"deleted"`
	Typeflag        int64     `boil:"typeflag" json:"typeflag" toml:"typeflag" yaml:"typeflag"`
	Name            string    `boil:"name" json:"name" toml:"name" yaml:"name"`
	Linkname        string    `boil:"linkname" json:"linkname" toml:"linkname" yaml:"linkname"`
	Size            int64     `boil:"size" json:"size" toml:"size" yaml:"size"`
	Mode            int64     `boil:"mode" json:"mode" toml:"mode" yaml:"mode"`
	UID             int64     `boil:"uid" json:"uid" toml:"uid" yaml:"uid"`
	Gid             int64     `boil:"gid" json:"gid" toml:"gid" yaml:"gid"`
	Uname           string    `boil:"uname" json:"uname" toml:"uname" yaml:"uname"`
	Gname           string    `boil:"gname" json:"gname" toml:"gname" yaml:"gname"`
	Modtime         time.Time `boil:"modtime" json:"modtime" toml:"modtime" yaml:"modtime"`
	Accesstime      time.Time `boil:"accesstime" json:"accesstime" toml:"accesstime" yaml:"accesstime"`
	Changetime      time.Time `boil:"changetime" json:"changetime" toml:"changetime" yaml:"changetime"`
	Devmajor        int64     `boil:"devmajor" json:"devmajor" toml:"devmajor" yaml:"devmajor"`
	Devminor        int64     `boil:"devminor" json:"devminor" toml:"devminor" yaml:"devminor"`
	Paxrecords      string    `boil:"paxrecords" json:"paxrecords" toml:"paxrecords" yaml:"paxrecords"`
	Format          int64     `boil:"format" json:"format" toml:"format" yaml:"format"`

	R *headerR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L headerL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var HeaderColumns = struct {
	Record          string
	Lastknownrecord string
	Block           string
	Lastknownblock  string
	Deleted         string
	Typeflag        string
	Name            string
	Linkname        string
	Size            string
	Mode            string
	UID             string
	Gid             string
	Uname           string
	Gname           string
	Modtime         string
	Accesstime      string
	Changetime      string
	Devmajor        string
	Devminor        string
	Paxrecords      string
	Format          string
}{
	Record:          "record",
	Lastknownrecord: "lastknownrecord",
	Block:           "block",
	Lastknownblock:  "lastknownblock",
	Deleted:         "deleted",
	Typeflag:        "typeflag",
	Name:            "name",
	Linkname:        "linkname",
	Size:            "size",
	Mode:            "mode",
	UID:             "uid",
	Gid:             "gid",
	Uname:           "uname",
	Gname:           "gname",
	Modtime:         "modtime",
	Accesstime:      "accesstime",
	Changetime:      "changetime",
	Devmajor:        "devmajor",
	Devminor:        "devminor",
	Paxrecords:      "paxrecords",
	Format:          "format",
}

var HeaderTableColumns = struct {
	Record          string
	Lastknownrecord string
	Block           string
	Lastknownblock  string
	Deleted         string
	Typeflag        string
	Name            string
	Linkname        string
	Size            string
	Mode            string
	UID             string
	Gid             string
	Uname           string
	Gname           string
	Modtime         string
	Accesstime      string
	Changetime      string
	Devmajor        string
	Devminor        string
	Paxrecords      string
	Format          string
}{
	Record:          "headers.record",
	Lastknownrecord: "headers.lastknownrecord",
	Block:           "headers.block",
	Lastknownblock:  "headers.lastknownblock",
	Deleted:         "headers.deleted",
	Typeflag:        "headers.typeflag",
	Name:            "headers.name",
	Linkname:        "headers.linkname",
	Size:            "headers.size",
	Mode:            "headers.mode",
	UID:             "headers.uid",
	Gid:             "headers.gid",
	Uname:           "headers.uname",
	Gname:           "headers.gname",
	Modtime:         "headers.modtime",
	Accesstime:      "headers.accesstime",
	Changetime:      "headers.changetime",
	Devmajor:        "headers.devmajor",
	Devminor:        "headers.devminor",
	Paxrecords:      "headers.paxrecords",
	Format:          "headers.format",
}

// Generated where

type whereHelperint64 struct{ field string }

func (w whereHelperint64) EQ(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.EQ, x) }
func (w whereHelperint64) NEQ(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.NEQ, x) }
func (w whereHelperint64) LT(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.LT, x) }
func (w whereHelperint64) LTE(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.LTE, x) }
func (w whereHelperint64) GT(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.GT, x) }
func (w whereHelperint64) GTE(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.GTE, x) }
func (w whereHelperint64) IN(slice []int64) qm.QueryMod {
	values := make([]interface{}, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return qm.WhereIn(fmt.Sprintf("%s IN ?", w.field), values...)
}
func (w whereHelperint64) NIN(slice []int64) qm.QueryMod {
	values := make([]interface{}, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return qm.WhereNotIn(fmt.Sprintf("%s NOT IN ?", w.field), values...)
}

type whereHelpertime_Time struct{ field string }

func (w whereHelpertime_Time) EQ(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.EQ, x)
}
func (w whereHelpertime_Time) NEQ(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.NEQ, x)
}
func (w whereHelpertime_Time) LT(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpertime_Time) LTE(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpertime_Time) GT(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpertime_Time) GTE(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

var HeaderWhere = struct {
	Record          whereHelperint64
	Lastknownrecord whereHelperint64
	Block           whereHelperint64
	Lastknownblock  whereHelperint64
	Deleted         whereHelperint64
	Typeflag        whereHelperint64
	Name            whereHelperstring
	Linkname        whereHelperstring
	Size            whereHelperint64
	Mode            whereHelperint64
	UID             whereHelperint64
	Gid             whereHelperint64
	Uname           whereHelperstring
	Gname           whereHelperstring
	Modtime         whereHelpertime_Time
	Accesstime      whereHelpertime_Time
	Changetime      whereHelpertime_Time
	Devmajor        whereHelperint64
	Devminor        whereHelperint64
	Paxrecords      whereHelperstring
	Format          whereHelperint64
}{
	Record:          whereHelperint64{field: "\"headers\".\"record\""},
	Lastknownrecord: whereHelperint64{field: "\"headers\".\"lastknownrecord\""},
	Block:           whereHelperint64{field: "\"headers\".\"block\""},
	Lastknownblock:  whereHelperint64{field: "\"headers\".\"lastknownblock\""},
	Deleted:         whereHelperint64{field: "\"headers\".\"deleted\""},
	Typeflag:        whereHelperint64{field: "\"headers\".\"typeflag\""},
	Name:            whereHelperstring{field: "\"headers\".\"name\""},
	Linkname:        whereHelperstring{field: "\"headers\".\"linkname\""},
	Size:            whereHelperint64{field: "\"headers\".\"size\""},
	Mode:            whereHelperint64{field: "\"headers\".\"mode\""},
	UID:             whereHelperint64{field: "\"headers\".\"uid\""},
	Gid:             whereHelperint64{field: "\"headers\".\"gid\""},
	Uname:           whereHelperstring{field: "\"headers\".\"uname\""},
	Gname:           whereHelperstring{field: "\"headers\".\"gname\""},
	Modtime:         whereHelpertime_Time{field: "\"headers\".\"modtime\""},
	Accesstime:      whereHelpertime_Time{field: "\"headers\".\"accesstime\""},
	Changetime:      whereHelpertime_Time{field: "\"headers\".\"changetime\""},
	Devmajor:        whereHelperint64{field: "\"headers\".\"devmajor\""},
	Devminor:        whereHelperint64{field: "\"headers\".\"devminor\""},
	Paxrecords:      whereHelperstring{field: "\"headers\".\"paxrecords\""},
	Format:          whereHelperint64{field: "\"headers\".\"format\""},
}

// HeaderRels is where relationship names are stored.
var HeaderRels = struct {
}{}

// headerR is where relationships are stored.
type headerR struct {
}

// NewStruct creates a new relationship struct
func (*headerR) NewStruct() *headerR {
	return &headerR{}
}

// headerL is where Load methods for each relationship are stored.
type headerL struct{}

var (
	headerAllColumns            = []string{"record", "lastknownrecord", "block", "lastknownblock", "deleted", "typeflag", "name", "linkname", "size", "mode", "uid", "gid", "uname", "gname", "modtime", "accesstime", "changetime", "devmajor", "devminor", "paxrecords", "format"}
	headerColumnsWithoutDefault = []string{"record", "lastknownrecord", "block", "lastknownblock", "deleted", "typeflag", "name", "linkname", "size", "mode", "uid", "gid", "uname", "gname", "modtime", "accesstime", "changetime", "devmajor", "devminor", "paxrecords", "format"}
	headerColumnsWithDefault    = []string{}
	headerPrimaryKeyColumns     = []string{"name", "linkname"}
	headerGeneratedColumns      = []string{}
)

type (
	// HeaderSlice is an alias for a slice of pointers to Header.
	// This should almost always be used instead of []Header.
	HeaderSlice []*Header
	// HeaderHook is the signature for custom Header hook methods
	HeaderHook func(context.Context, boil.ContextExecutor, *Header) error

	headerQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	headerType                 = reflect.TypeOf(&Header{})
	headerMapping              = queries.MakeStructMapping(headerType)
	headerPrimaryKeyMapping, _ = queries.BindMapping(headerType, headerMapping, headerPrimaryKeyColumns)
	headerInsertCacheMut       sync.RWMutex
	headerInsertCache          = make(map[string]insertCache)
	headerUpdateCacheMut       sync.RWMutex
	headerUpdateCache          = make(map[string]updateCache)
	headerUpsertCacheMut       sync.RWMutex
	headerUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

var headerAfterSelectMu sync.Mutex
var headerAfterSelectHooks []HeaderHook

var headerBeforeInsertMu sync.Mutex
var headerBeforeInsertHooks []HeaderHook
var headerAfterInsertMu sync.Mutex
var headerAfterInsertHooks []HeaderHook

var headerBeforeUpdateMu sync.Mutex
var headerBeforeUpdateHooks []HeaderHook
var headerAfterUpdateMu sync.Mutex
var headerAfterUpdateHooks []HeaderHook

var headerBeforeDeleteMu sync.Mutex
var headerBeforeDeleteHooks []HeaderHook
var headerAfterDeleteMu sync.Mutex
var headerAfterDeleteHooks []HeaderHook

var headerBeforeUpsertMu sync.Mutex
var headerBeforeUpsertHooks []HeaderHook
var headerAfterUpsertMu sync.Mutex
var headerAfterUpsertHooks []HeaderHook

// doAfterSelectHooks executes all "after Select" hooks.
func (o *Header) doAfterSelectHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerAfterSelectHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeInsertHooks executes all "before insert" hooks.
func (o *Header) doBeforeInsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerBeforeInsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterInsertHooks executes all "after Insert" hooks.
func (o *Header) doAfterInsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerAfterInsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeUpdateHooks executes all "before Update" hooks.
func (o *Header) doBeforeUpdateHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerBeforeUpdateHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterUpdateHooks executes all "after Update" hooks.
func (o *Header) doAfterUpdateHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerAfterUpdateHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeDeleteHooks executes all "before Delete" hooks.
func (o *Header) doBeforeDeleteHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerBeforeDeleteHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterDeleteHooks executes all "after Delete" hooks.
func (o *Header) doAfterDeleteHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerAfterDeleteHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeUpsertHooks executes all "before Upsert" hooks.
func (o *Header) doBeforeUpsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerBeforeUpsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterUpsertHooks executes all "after Upsert" hooks.
func (o *Header) doAfterUpsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range headerAfterUpsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// AddHeaderHook registers your hook function for all future operations.
func AddHeaderHook(hookPoint boil.HookPoint, headerHook HeaderHook) {
	switch hookPoint {
	case boil.AfterSelectHook:
		headerAfterSelectMu.Lock()
		headerAfterSelectHooks = append(headerAfterSelectHooks, headerHook)
		headerAfterSelectMu.Unlock()
	case boil.BeforeInsertHook:
		headerBeforeInsertMu.Lock()
		headerBeforeInsertHooks = append(headerBeforeInsertHooks, headerHook)
		headerBeforeInsertMu.Unlock()
	case boil.AfterInsertHook:
		headerAfterInsertMu.Lock()
		headerAfterInsertHooks = append(headerAfterInsertHooks, headerHook)
		headerAfterInsertMu.Unlock()
	case boil.BeforeUpdateHook:
		headerBeforeUpdateMu.Lock()
		headerBeforeUpdateHooks = append(headerBeforeUpdateHooks, headerHook)
		headerBeforeUpdateMu.Unlock()
	case boil.AfterUpdateHook:
		headerAfterUpdateMu.Lock()
		headerAfterUpdateHooks = append(headerAfterUpdateHooks, headerHook)
		headerAfterUpdateMu.Unlock()
	case boil.BeforeDeleteHook:
		headerBeforeDeleteMu.Lock()
		headerBeforeDeleteHooks = append(headerBeforeDeleteHooks, headerHook)
		headerBeforeDeleteMu.Unlock()
	case boil.AfterDeleteHook:
		headerAfterDeleteMu.Lock()
		headerAfterDeleteHooks = append(headerAfterDeleteHooks, headerHook)
		headerAfterDeleteMu.Unlock()
	case boil.BeforeUpsertHook:
		headerBeforeUpsertMu.Lock()
		headerBeforeUpsertHooks = append(headerBeforeUpsertHooks, headerHook)
		headerBeforeUpsertMu.Unlock()
	case boil.AfterUpsertHook:
		headerAfterUpsertMu.Lock()
		headerAfterUpsertHooks = append(headerAfterUpsertHooks, headerHook)
		headerAfterUpsertMu.Unlock()
	}
}

// One returns a single header record from the query.
func (q headerQuery) One(ctx context.Context, exec boil.ContextExecutor) (*Header, error) {
	o := &Header{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(ctx, exec, o)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for headers")
	}

	if err := o.doAfterSelectHooks(ctx, exec); err != nil {
		return o, err
	}

	return o, nil
}

// All returns all Header records from the query.
func (q headerQuery) All(ctx context.Context, exec boil.ContextExecutor) (HeaderSlice, error) {
	var o []*Header

	err := q.Bind(ctx, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to Header slice")
	}

	if len(headerAfterSelectHooks) != 0 {
		for _, obj := range o {
			if err := obj.doAfterSelectHooks(ctx, exec); err != nil {
				return o, err
			}
		}
	}

	return o, nil
}

// Count returns the count of all Header records in the query.
func (q headerQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count headers rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q headerQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if headers exists")
	}

	return count > 0, nil
}

// Headers retrieves all the records using an executor.
func Headers(mods ...qm.QueryMod) headerQuery {
	mods = append(mods, qm.From("\"headers\""))
	q := NewQuery(mods...)
	if len(queries.GetSelect(q)) == 0 {
		queries.SetSelect(q, []string{"\"headers\".*"})
	}

	return headerQuery{q}
}

// FindHeader retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindHeader(ctx context.Context, exec boil.ContextExecutor, name string, linkname string, selectCols ...string) (*Header, error) {
	headerObj := &Header{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"headers\" where \"name\"=? AND \"linkname\"=?", sel,
	)

	q := queries.Raw(query, name, linkname)

	err := q.Bind(ctx, exec, headerObj)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from headers")
	}

	if err = headerObj.doAfterSelectHooks(ctx, exec); err != nil {
		return headerObj, err
	}

	return headerObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *Header) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if o == nil {
		return errors.New("models: no headers provided for insertion")
	}

	var err error

	if err := o.doBeforeInsertHooks(ctx, exec); err != nil {
		return err
	}

	nzDefaults := queries.NonZeroDefaultSet(headerColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	headerInsertCacheMut.RLock()
	cache, cached := headerInsertCache[key]
	headerInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			headerAllColumns,
			headerColumnsWithDefault,
			headerColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(headerType, headerMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(headerType, headerMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"headers\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"headers\" %sDEFAULT VALUES%s"
		}

		var queryOutput, queryReturning string

		if len(cache.retMapping) != 0 {
			queryReturning = fmt.Sprintf(" RETURNING \"%s\"", strings.Join(returnColumns, "\",\""))
		}

		cache.query = fmt.Sprintf(cache.query, queryOutput, queryReturning)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}

	if err != nil {
		return errors.Wrap(err, "models: unable to insert into headers")
	}

	if !cached {
		headerInsertCacheMut.Lock()
		headerInsertCache[key] = cache
		headerInsertCacheMut.Unlock()
	}

	return o.doAfterInsertHooks(ctx, exec)
}

// Update uses an executor to update the Header.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *Header) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) (int64, error) {
	var err error
	if err = o.doBeforeUpdateHooks(ctx, exec); err != nil {
		return 0, err
	}
	key := makeCacheKey(columns, nil)
	headerUpdateCacheMut.RLock()
	cache, cached := headerUpdateCache[key]
	headerUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			headerAllColumns,
			headerPrimaryKeyColumns,
		)

		if !columns.IsWhitelist() {
			wl = strmangle.SetComplement(wl, []string{"created_at"})
		}
		if len(wl) == 0 {
			return 0, errors.New("models: unable to update headers, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"headers\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 0, wl),
			strmangle.WhereClause("\"", "\"", 0, headerPrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(headerType, headerMapping, append(wl, headerPrimaryKeyColumns...))
		if err != nil {
			return 0, err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, values)
	}
	var result sql.Result
	result, err = exec.ExecContext(ctx, cache.query, values...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update headers row")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by update for headers")
	}

	if !cached {
		headerUpdateCacheMut.Lock()
		headerUpdateCache[key] = cache
		headerUpdateCacheMut.Unlock()
	}

	return rowsAff, o.doAfterUpdateHooks(ctx, exec)
}

// UpdateAll updates all rows with the specified column values.
func (q headerQuery) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) (int64, error) {
	queries.SetUpdate(q.Query, cols)

	result, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update all for headers")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to retrieve rows affected for headers")
	}

	return rowsAff, nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o HeaderSlice) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) (int64, error) {
	ln := int64(len(o))
	if ln == 0 {
		return 0, nil
	}

	if len(cols) == 0 {
		return 0, errors.New("models: update all requires at least one column argument")
	}

	colNames := make([]string, len(cols))
	args := make([]interface{}, len(cols))

	i := 0
	for name, value := range cols {
		colNames[i] = name
		args[i] = value
		i++
	}

	// Append all of the primary key values for each column
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), headerPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"headers\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 0, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 0, headerPrimaryKeyColumns, len(o)))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update all in header slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to retrieve rows affected all in update all header")
	}
	return rowsAff, nil
}

// Delete deletes a single Header record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *Header) Delete(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if o == nil {
		return 0, errors.New("models: no Header provided for delete")
	}

	if err := o.doBeforeDeleteHooks(ctx, exec); err != nil {
		return 0, err
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), headerPrimaryKeyMapping)
	sql := "DELETE FROM \"headers\" WHERE \"name\"=? AND \"linkname\"=?"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete from headers")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by delete for headers")
	}

	if err := o.doAfterDeleteHooks(ctx, exec); err != nil {
		return 0, err
	}

	return rowsAff, nil
}

// DeleteAll deletes all matching rows.
func (q headerQuery) DeleteAll(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if q.Query == nil {
		return 0, errors.New("models: no headerQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	result, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete all from headers")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by deleteall for headers")
	}

	return rowsAff, nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o HeaderSlice) DeleteAll(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if len(o) == 0 {
		return 0, nil
	}

	if len(headerBeforeDeleteHooks) != 0 {
		for _, obj := range o {
			if err := obj.doBeforeDeleteHooks(ctx, exec); err != nil {
				return 0, err
			}
		}
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), headerPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"headers\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 0, headerPrimaryKeyColumns, len(o))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete all from header slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by deleteall for headers")
	}

	if len(headerAfterDeleteHooks) != 0 {
		for _, obj := range o {
			if err := obj.doAfterDeleteHooks(ctx, exec); err != nil {
				return 0, err
			}
		}
	}

	return rowsAff, nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *Header) Reload(ctx context.Context, exec boil.ContextExecutor) error {
	ret, err := FindHeader(ctx, exec, o.Name, o.Linkname)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *HeaderSlice) ReloadAll(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := HeaderSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), headerPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"headers\".* FROM \"headers\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 0, headerPrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(ctx, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in HeaderSlice")
	}

	*o = slice

	return nil
}

// HeaderExists checks if the Header row exists.
func HeaderExists(ctx context.Context, exec boil.ContextExecutor, name string, linkname string) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"headers\" where \"name\"=? AND \"linkname\"=? limit 1)"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, name, linkname)
	}
	row := exec.QueryRowContext(ctx, sql, name, linkname)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if headers exists")
	}

	return exists, nil
}

// Exists checks if the Header row exists.
func (o *Header) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	return HeaderExists(ctx, exec, o.Name, o.Linkname)
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *Header) Upsert(ctx context.Context, exec boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("models: no headers provided for upsert")
	}

	if err := o.doBeforeUpsertHooks(ctx, exec); err != nil {
		return err
	}

	nzDefaults := queries.NonZeroDefaultSet(headerColumnsWithDefault, o)

	// Build cache key in-line uglily - mysql vs psql problems
	buf := strmangle.GetBuffer()
	if updateOnConflict {
		buf.WriteByte('t')
	} else {
		buf.WriteByte('f')
	}
	buf.WriteByte('.')
	for _, c := range conflictColumns {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(updateColumns.Kind))
	for _, c := range updateColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(insertColumns.Kind))
	for _, c := range insertColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzDefaults {
		buf.WriteString(c)
	}
	key := buf.String()
	strmangle.PutBuffer(buf)

	headerUpsertCacheMut.RLock()
	cache, cached := headerUpsertCache[key]
	headerUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			headerAllColumns,
			headerColumnsWithDefault,
			headerColumnsWithoutDefault,
			nzDefaults,
		)
		update := updateColumns.UpdateColumnSet(
			headerAllColumns,
			headerPrimaryKeyColumns,
		)

		if updateOnConflict && len(update) == 0 {
			return errors.New("models: unable to upsert headers, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(headerPrimaryKeyColumns))
			copy(conflict, headerPrimaryKeyColumns)
		}
		cache.query = buildUpsertQuerySQLite(dialect, "\"headers\"", updateOnConflict, ret, update, conflict, insert)

		cache.valueMapping, err = queries.BindMapping(headerType, headerMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(headerType, headerMapping, ret)
			if err != nil {
				return err
			}
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)
	var returns []interface{}
	if len(cache.retMapping) != 0 {
		returns = queries.PtrsFromMapping(value, cache.retMapping)
	}

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, vals)
	}
	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(returns...)
		if err == sql.ErrNoRows {
			err = nil // Postgres doesn't return anything when there's no update
		}
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}
	if err != nil {
		return errors.Wrap(err, "models: unable to upsert headers")
	}

	if !cached {
		headerUpsertCacheMut.Lock()
		headerUpsertCache[key] = cache
		headerUpsertCacheMut.Unlock()
	}

	return o.doAfterUpsertHooks(ctx, exec)
}
