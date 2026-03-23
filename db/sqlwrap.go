package db

import (
	"context"
	"database/sql"

	"orquesta/storage"
)

type Handle struct {
	*sql.DB
	driver string
}

type Tx struct {
	*sql.Tx
	driver string
}

func newHandle(inner *sql.DB, driver string) *Handle {
	return &Handle{DB: inner, driver: driver}
}

func (h *Handle) Exec(query string, args ...any) (sql.Result, error) {
	return h.DB.Exec(storage.RebindQuery(h.driver, query), args...)
}

func (h *Handle) Query(query string, args ...any) (*sql.Rows, error) {
	return h.DB.Query(storage.RebindQuery(h.driver, query), args...)
}

func (h *Handle) QueryRow(query string, args ...any) *sql.Row {
	return h.DB.QueryRow(storage.RebindQuery(h.driver, query), args...)
}

func (h *Handle) Begin() (*Tx, error) {
	tx, err := h.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, driver: h.driver}, nil
}

func (h *Handle) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := h.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, driver: h.driver}, nil
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.Tx.Exec(storage.RebindQuery(tx.driver, query), args...)
}

func (tx *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.Tx.Query(storage.RebindQuery(tx.driver, query), args...)
}

func (tx *Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.Tx.QueryRow(storage.RebindQuery(tx.driver, query), args...)
}
