# pgxhelper

`pgxhelper` is a lightweight, opinionated wrapper around [pgx/v5](https://github.com/jackc/pgx) and [scany/v2](https://github.com/georgysavva/scany) designed to simplify common database operations in Go applications. It provides a thin abstraction layer over `pgxpool` that reduces boilerplate for querying, scanning, and executing commands, including a straightforward way to handle transactions.

## Features

- **Simplified CRUD:** Easy-to-use `Get`, `Select`, and `Exec` methods for common database interactions.
- **Automatic Scanning:** Leverages `scany` to automatically scan database rows into Go structs.
- **Transaction Management:** A simple `WithinTransaction` helper to run operations within a database transaction without manual boilerplate.
- **Connection Pooling:** Built on top of `pgxpool` for efficient database connection management.
- **Context-Aware:** All database operations are `context.Context` aware.

## Installation

```sh
go get github.com/theprogrammer67/pgxhelper
```

## Quick Start

Here is a complete example demonstrating how to connect to a database, create a table, insert data, and query it back.

### 1. Define Your Model

First, define a Go struct that corresponds to your database table schema.

```go
// user.go
package main

type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}
```

### 2. Use the DBHelper

```go
// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/theprogrammer67/pgxhelper"
)

func main() {
	// Database connection string
	connStr := "postgres://user:password@localhost:5432/database?sslmode=disable"

	// 1. Initialize and connect
	db := &pgxhelper.DBHelper{}
	if err := db.Connect(connStr, 5*time.Second); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to the database!")

	ctx := context.Background()

	// 2. Use a transaction to set up the schema and insert data
	err := db.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Create table
		_, err := db.Exec(txCtx, `
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL,
				email TEXT NOT NULL UNIQUE
			)`)
		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		// Insert a new user
		_, err = db.Exec(txCtx,
			"INSERT INTO users (name, email) VALUES ($1, $2)",
			"John Doe", "john.doe@example.com",
		)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}

	log.Println("Table created and user inserted successfully.")

	// 3. Select multiple users
	var users []*User
	err = db.Select(ctx, &users, "SELECT id, name, email FROM users ORDER BY id")
	if err != nil {
		log.Fatalf("Failed to select users: %v", err)
	}
	fmt.Printf("All users: %+v\n", users)
	for _, u := range users {
		fmt.Printf("- ID: %d, Name: %s, Email: %s\n", u.ID, u.Name, u.Email)
	}

	// 4. Get a single user
	var singleUser User
	err = db.Get(ctx, &singleUser, "SELECT id, name, email FROM users WHERE name = $1", "John Doe")
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("Single user fetched: %+v\n", singleUser)
}
```

## API Overview

### `pgxhelper.DBHelper`

The main struct that holds the database connection pool.

- **`NewDBHelper(pool *pgxpool.Pool, opts ...Option) *DBHelper`**: Creates a new `DBHelper` with an existing `pgxpool.Pool`.
- **`Connect(connStr string, timeout time.Duration) error`**: A convenience method to create a `pgxpool.Pool` and connect to the database.
- **`Close()`**: Closes the underlying connection pool.

### Operations

- **`Get(ctx context.Context, dest any, query string, args ...any) error`**: Queries for a single row and scans the result into `dest`. `dest` must be a pointer to a struct.
- **`Select(ctx context.Context, dest any, query string, args ...any) error`**: Queries for multiple rows and scans the results into `dest`. `dest` must be a pointer to a slice of structs.
- **`Exec(ctx context.Context, query string, args ...any) (int64, error)`**: Executes a command (e.g., `INSERT`, `UPDATE`, `DELETE`) and returns the number of rows affected.
- **`WithinTransaction(ctx context.Context, fn func(ctx context.Context) error, opt ...pgx.TxOptions) error`**: Executes the function `fn` within a database transaction. The transaction is automatically committed if `fn` returns `nil`, or rolled back if it returns an error. The context passed to `fn` carries the transaction.
- **`Querier(ctx context.Context) Querier`**: Returns the underlying `pgx.Tx` if the context is transactional, otherwise returns the `pgxpool.Pool`. This is useful for interoperability with other libraries.
