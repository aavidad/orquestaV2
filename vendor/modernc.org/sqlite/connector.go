package sqlite

import (
	"context"
	"database/sql/driver"
	"fmt"
)

// NewConnector returns a database/sql connector backed by the package's
// canonical driver. It preserves globally registered functions, collations,
// hooks and virtual-table modules while allowing callers to validate each
// physical connection before database/sql admits it to a pool.
func NewConnector(name string) driver.Connector {
	return &connector{name: name, driver: d}
}

// NewValidatedConnector returns a connector that validates the physical main
// database connection before any DSN _pragma or globally registered connection
// hook can run. The validator receives the same connection that is admitted to
// database/sql; returning an error closes it.
//
// Validation necessarily happens after sqlite3_open_v2, because SQLite exposes
// the active sqlite3_file only through that open connection. It happens before
// SQL is prepared or executed on the connection.
func NewValidatedConnector(name string, validate func(FileControl) error) driver.Connector {
	return &connector{name: name, driver: d, validate: validate}
}

type connector struct {
	name     string
	driver   *Driver
	validate func(FileControl) error
}

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.validate == nil {
		return c.driver.Open(c.name)
	}
	return c.openValidated(ctx)
}

func (c *connector) Driver() driver.Driver {
	return c.driver
}

func (c *connector) openValidated(ctx context.Context) (driver.Conn, error) {
	connection, err := newValidatedConn(c.name, c.validate)
	if err != nil {
		return nil, err
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = connection.Close()
		}
	}()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, udf := range c.driver.udfs {
		if err := connection.createFunctionInternal(udf); err != nil {
			return nil, err
		}
	}
	for _, collation := range c.driver.collations {
		if err := connection.createCollationInternal(collation); err != nil {
			return nil, err
		}
	}
	for _, hook := range c.driver.connectionHooks {
		if err := hook(connection, c.name); err != nil {
			return nil, fmt.Errorf("connection hook: %w", err)
		}
	}
	if err := connection.registerModules(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	closeOnError = false
	return connection, nil
}
