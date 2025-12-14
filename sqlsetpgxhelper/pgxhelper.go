// Package sqlset provides a pgxhelper wrapper that works with
// github.com/theprogrammer67/sqlset.
package sqlsetpgxhelper

import (
	"context"

	"github.com/theprogrammer67/pgxhelper"
	"github.com/theprogrammer67/sqlset"
)

// DBHelper is a wrapper around pgxhelper.DBHelper that uses a sqlset.SQLSet
// to retrieve queries.
type DBHelper struct {
	*pgxhelper.DBHelper
	sqlSet *sqlset.SQLSet
}

// New creates and returns a new DBHelper.
func New(sqlSet *sqlset.SQLSet, opts ...pgxhelper.Option) *DBHelper {
	return &DBHelper{
		DBHelper: pgxhelper.New(opts...),
		sqlSet:   sqlSet,
	}
}

// Get retrieves a query from the SQL set using setID and queryID, executes it
// as a single-row query, and scans the result into dest.
func (d *DBHelper) Get(ctx context.Context, dest any, setID, queryID string, args ...any) error {
	query, err := d.sqlSet.Get(setID, queryID)
	if err != nil {
		return err
	}
	return d.DBHelper.Get(ctx, dest, query, args...)
}

// Select retrieves a query from the SQL set using setID and queryID, executes
// it, and scans the resulting rows into the dest slice.
func (d *DBHelper) Select(ctx context.Context, dest any, setID, queryID string, args ...any) error {
	query, err := d.sqlSet.Get(setID, queryID)
	if err != nil {
		return err
	}
	return d.DBHelper.Select(ctx, dest, query, args...)
}

// Exec retrieves a query from the SQL set using setID and queryID and
// executes it, returning the number of affected rows.
func (d *DBHelper) Exec(ctx context.Context, setID, queryID string, args ...any) (int64, error) {
	query, err := d.sqlSet.Get(setID, queryID)
	if err != nil {
		return 0, err
	}

	return d.DBHelper.Exec(ctx, query, args...)
}
