package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
)

const noDBDriverName = "orquesta_nodb"

func init() {
	sql.Register(noDBDriverName, noDBDriver{})
}

type noDBDriver struct{}

func (noDBDriver) Open(string) (driver.Conn, error) {
	return noDBConn{}, nil
}

type noDBConn struct{}

func (noDBConn) Prepare(string) (driver.Stmt, error) { return noDBStmt{}, nil }
func (noDBConn) Close() error                        { return nil }
func (noDBConn) Begin() (driver.Tx, error)           { return noDBTx{}, nil }

func (noDBConn) PrepareContext(context.Context, string) (driver.Stmt, error) {
	return noDBStmt{}, nil
}

func (noDBConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return noDBTx{}, nil
}

func (noDBConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return noDBRows{}, nil
}

func (noDBConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return noopResult{}, nil
}

type noDBStmt struct{}

func (noDBStmt) Close() error                               { return nil }
func (noDBStmt) NumInput() int                              { return -1 }
func (noDBStmt) Exec([]driver.Value) (driver.Result, error) { return noopResult{}, nil }
func (noDBStmt) Query([]driver.Value) (driver.Rows, error)  { return noDBRows{}, nil }
func (noDBStmt) ExecContext(context.Context, []driver.NamedValue) (driver.Result, error) {
	return noopResult{}, nil
}
func (noDBStmt) QueryContext(context.Context, []driver.NamedValue) (driver.Rows, error) {
	return noDBRows{}, nil
}

type noDBRows struct{}

func (noDBRows) Columns() []string         { return []string{"noop"} }
func (noDBRows) Close() error              { return nil }
func (noDBRows) Next([]driver.Value) error { return io.EOF }

type noDBTx struct{}

func (noDBTx) Commit() error   { return nil }
func (noDBTx) Rollback() error { return nil }
