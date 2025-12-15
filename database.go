package pgxhelper

import (
	"context"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/dbscan"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBHelper is a wrapper around pgxpool.Pool to simplify common database operations.
type DBHelper struct {
	pool    *pgxpool.Pool
	scanAPI *pgxscan.API
}

// Option is a functional option for configuring a DBHelper.
type Option func(*DBHelper)

// WithScanAPI is a functional option to set the pgxscan.API for the DBHelper.
func WithScanAPI(scanAPI *pgxscan.API) Option {
	return func(h *DBHelper) {
		h.scanAPI = scanAPI
	}
}

// New creates and returns a new DBHelper.
func New(opts ...Option) *DBHelper {
	h := &DBHelper{
		scanAPI: mustNewAPI(mustNewDBScanAPI(dbscan.WithAllowUnknownColumns(true))),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Connect establishes a connection to the database using the provided connection string and timeout.
// It creates a new pgxpool.Pool and pings the database to ensure the connection is live.
func (d *DBHelper) Connect(connStr string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conf, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("parse database config failure: %w", err)
	}

	d.pool, err = pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return fmt.Errorf("database pool creation failure: %w", err)
	}

	return d.Ping(ctx)
}

// Ping checks the connection to the database.
func (d *DBHelper) Ping(ctx context.Context) error {
	err := d.pool.Ping(ctx)
	if err != nil {
		return fmt.Errorf("database ping error: %w", err)
	}

	rows, err := d.pool.Query(ctx, "SELECT 1 AS result FROM pg_database WHERE datname = $1",
		d.pool.Config().ConnConfig.Database)
	if err != nil {
		return fmt.Errorf("database query failure: %w", err)
	}
	defer rows.Close()

	return nil
}

// Close closes all connections in the pool and prevents further use.
func (d *DBHelper) Close() {
	d.pool.Close()
}

// Querier returns the appropriate querier from the context.
// If the context contains a transaction, it returns the transaction.
// Otherwise, it returns the connection pool.
func (d *DBHelper) Querier(ctx context.Context) Querier {
	if tx := ctxGetTx(ctx); tx != nil {
		return tx
	}

	return d.pool
}

// WithinTransaction runs the given function within a transactional context.
func (d *DBHelper) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error, opt ...pgx.TxOptions) error {
	d.requireNoTransaction(ctx)

	var txOpt pgx.TxOptions
	if len(opt) > 0 {
		txOpt = opt[0]
	}

	return pgx.BeginTxFunc(ctx, d.pool, txOpt, func(tx pgx.Tx) error {
		return fn(ctxWithTx(ctx, tx))
	})

}

// private

func mustNewDBScanAPI(opts ...dbscan.APIOption) *dbscan.API {
	api, err := pgxscan.NewDBScanAPI(opts...)
	if err != nil {
		panic(err)
	}

	return api
}

func mustNewAPI(dbscanAPI *dbscan.API) *pgxscan.API {
	api, err := pgxscan.NewAPI(dbscanAPI)
	if err != nil {
		panic(err)
	}

	return api
}

// Querier is an interface that abstracts database operations.
// It is implemented by *pgxpool.Pool, *pgx.Conn, and pgx.Tx.
// This allows functions to accept any of these types as a querier.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

var (
	_ Querier = &pgxpool.Pool{}
	_ Querier = &pgx.Conn{}
	_ Querier = pgx.Tx(nil)
)

type txKey struct{}

// ctxWithTx injects transaction to context
func ctxWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// ctxGetTx extracts transaction from context
func ctxGetTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}

	return nil
}

// requireNoTransaction panics if the given context contains a transaction.
func (d *DBHelper) requireNoTransaction(ctx context.Context) {
	if tx := ctxGetTx(ctx); tx != nil {
		// Assume this is a code design error, not an error value.
		panic("context already contains an unexpected transaction")
	}
}
