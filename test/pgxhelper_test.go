package pgxhelper

import (
	"context"
	"embed"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/theprogrammer67/pgxhelper"
	"github.com/theprogrammer67/pgxhelper/sqlsetpgxhelper"
	"github.com/theprogrammer67/sqlset"
)

var (
	_ suite.SetupAllSuite    = (*DBHelperSuite)(nil)
	_ suite.TearDownAllSuite = (*DBHelperSuite)(nil)
)

type DBHelperSuite struct {
	suite.Suite
	pg      *postgres.PostgresContainer
	connStr string
}

const (
	connTimeout = 10 * time.Second
	testTimeout = 30 * time.Second
)

func TestDBHelper(t *testing.T) {
	suite.Run(t, &DBHelperSuite{})
}

func (s *DBHelperSuite) SetupSuite() {
	const timeout = 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var err error
	s.pg, err = postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(timeout),
		),
	)

	s.Require().NoError(err, "failed to start postgres container")

	s.connStr, err = s.pg.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err, "failed to get connection string")
}

func (s *DBHelperSuite) TearDownSuite() {
	if s.pg != nil {
		_ = s.pg.Terminate(context.Background())
	}
}

type User struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	EMail string `db:"email"`
}

func (s *DBHelperSuite) TestDBHelper() {
	db := pgxhelper.New()
	err := db.Connect(s.connStr, connTimeout)
	s.Require().NoError(err, "failed to connect to test database")
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	_, err = db.Exec(ctx, `
		CREATE TABLE  IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE
		)`)
	s.Require().NoError(err, "failed to create table users")

	s.Run("should insert users in transaction", func() {
		err = db.WithinTransaction(ctx, func(txCtx context.Context) error {
			r, err := db.Exec(txCtx, `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`, "111", "Alice", "alice@example.com")
			if err != nil {
				return err
			}
			if r != 1 {
				return errors.New("expected 1 row affected")
			}

			r, err = db.Exec(txCtx, `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`, "222", "Bob", "bob@example.com")
			if err != nil {
				return err
			}
			if r != 1 {
				return errors.New("expected 1 row affected")
			}

			return nil
		})

		s.Require().NoError(err)

		var users []User

		err = db.Select(ctx, &users, `
			SELECT * 
			FROM users 
			WHERE id = ANY($1::text[])
		`, []string{"111", "222"})
		s.Require().NoError(err)
		s.Equal(2, len(users))
	})

	s.Run("should insert user", func() {
		r, err := db.Exec(ctx, `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`, "333", "John", "john@example.com")
		s.Require().NoError(err)
		s.Equal(int64(1), r, "expected 1 row affected")

		var customer Customer
		err = db.Get(ctx, &customer, `SELECT * FROM users WHERE id = $1`, "333")
		s.Require().NoError(err)
		s.Equal("john@example.com", customer.EMail)
	})
}

type Customer struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	EMail string `db:"email"`
}

//go:embed queries
var queriesFS embed.FS

func (s *DBHelperSuite) TestDBHelperWithSQLSet() {
	queries, err := sqlset.New(queriesFS)
	s.Require().NoError(err, "failed to create sqlset")

	db := sqlsetpgxhelper.New(queries)
	err = db.Connect(s.connStr, connTimeout)
	s.Require().NoError(err, "failed to connect to test database")
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	_, err = db.Exec(ctx, "test", "CreateTableCustomers")
	s.Require().NoError(err, "failed to create table customers")

	s.Run("should insert customers in transaction", func() {
		err = db.WithinTransaction(ctx, func(txCtx context.Context) error {
			r, err := db.Exec(txCtx, "test", "InsertCustomer", "111", "Alice", "alice@example.com")
			if err != nil {
				return err
			}
			if r != 1 {
				return errors.New("expected 1 row affected")
			}

			r, err = db.Exec(txCtx, "test", "InsertCustomer", "222", "Bob", "bob@example.com")
			if err != nil {
				return err
			}
			if r != 1 {
				return errors.New("expected 1 row affected")
			}

			return nil
		})

		s.Require().NoError(err)

		var customers []Customer

		err = db.Select(ctx, &customers, "test", "GetCustomers", []string{"111", "222"})
		s.Require().NoError(err)
		s.Equal(2, len(customers))
	})

	s.Run("should insert customer", func() {
		r, err := db.Exec(ctx, "test", "InsertCustomer", "333", "John", "john@example.com")
		s.Require().NoError(err)
		s.Equal(int64(1), r, "expected 1 row affected")

		var customer Customer
		err = db.Get(ctx, &customer, "test", "GetCustomer", "333")
		s.Require().NoError(err)
		s.Equal("John", customer.Name)
	})

	s.Run("should failed find query", func() {
		r, err := db.Exec(ctx, "test", "Unknown")
		s.ErrorIs(err, sqlset.ErrQueryNotFound)
		s.Equal(int64(0), r, "expected 0 row affected")
	})

}
