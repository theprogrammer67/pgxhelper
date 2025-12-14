// Package pgxhelper provides a thin wrapper around pgxpool to simplify common database operations.
package pgxhelper

import (
	"context"
	"fmt"
)

// Get queries for a single row and scans it into dest.
func (d *DBHelper) Get(ctx context.Context, dest any, query string, args ...any) error {
	return d.scanAPI.Get(ctx, d.Querier(ctx), dest, query, args...)
}

// Select queries for multiple rows and scans them into a slice.
func (d *DBHelper) Select(ctx context.Context, dest any, query string, args ...any) error {
	return d.scanAPI.Select(ctx, d.Querier(ctx), dest, query, args...)
}

// Exec executes a query that doesn't return rows, such as INSERT, UPDATE, or DELETE.
// It returns the number of rows affected.
func (d *DBHelper) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	tag, err := d.Querier(ctx).Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("exec failed: %w", err)
	}

	return tag.RowsAffected(), nil
}
