package db

import (
	"context"
	"database/sql"
	"time"

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
	var (
		res sql.Result
		err error
	)
	err = ejecutarConReintentos(func() error {
		res, err = h.DB.Exec(storage.RebindQuery(h.driver, query), args...)
		return err
	})
	return res, err
}

func (h *Handle) Query(query string, args ...any) (*sql.Rows, error) {
	var (
		rows *sql.Rows
		err  error
	)
	err = ejecutarConReintentos(func() error {
		rows, err = h.DB.Query(storage.RebindQuery(h.driver, query), args...)
		return err
	})
	return rows, err
}

func (h *Handle) QueryRow(query string, args ...any) *sql.Row {
	return h.DB.QueryRow(storage.RebindQuery(h.driver, query), args...)
}

func (h *Handle) Begin() (*Tx, error) {
	var (
		tx  *sql.Tx
		err error
	)
	err = ejecutarConReintentos(func() error {
		tx, err = h.DB.Begin()
		return err
	})
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, driver: h.driver}, nil
}

func (h *Handle) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	var (
		tx  *sql.Tx
		err error
	)
	for intento := 0; intento < persistenciaBusyMaxIntentos; intento++ {
		tx, err = h.DB.BeginTx(ctx, opts)
		if err == nil || !esErrorPersistenciaBusy(err) {
			break
		}
		time.Sleep(persistenciaBusyBackoff)
	}
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, driver: h.driver}, nil
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	var (
		res sql.Result
		err error
	)
	err = ejecutarConReintentos(func() error {
		res, err = tx.Tx.Exec(storage.RebindQuery(tx.driver, query), args...)
		return err
	})
	return res, err
}

func (tx *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	var (
		rows *sql.Rows
		err  error
	)
	err = ejecutarConReintentos(func() error {
		rows, err = tx.Tx.Query(storage.RebindQuery(tx.driver, query), args...)
		return err
	})
	return rows, err
}

func (tx *Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.Tx.QueryRow(storage.RebindQuery(tx.driver, query), args...)
}
