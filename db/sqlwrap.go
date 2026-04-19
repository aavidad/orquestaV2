package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"
	"time"

	"orquesta/storage"
)

type Handle struct {
	*sql.DB
	driver string
}

type noopResult struct{}

func (noopResult) LastInsertId() (int64, error) { return 0, nil }
func (noopResult) RowsAffected() (int64, error) { return 0, nil }

var (
	noDBFallbackOnce sync.Once
	noDBFallback     *sql.DB
)

func noDBFallbackDB() *sql.DB {
	noDBFallbackOnce.Do(func() {
		db, err := sql.Open(noDBDriverName, "")
		if err != nil {
			return
		}
		noDBFallback = db
	})
	return noDBFallback
}

func noDBFallbackRows() (*sql.Rows, error) {
	db := noDBFallbackDB()
	if db == nil {
		return nil, fmt.Errorf("persistencia no disponible")
	}
	rows, err := db.Query("SELECT 1 WHERE 0")
	return rows, err
}

func noDBFallbackQueryRow() *sql.Row {
	db := noDBFallbackDB()
	if db == nil {
		return &sql.Row{}
	}
	return db.QueryRow("SELECT 1 WHERE 0")
}

type Tx struct {
	*sql.Tx
	driver string
	dirty  bool
}

func newHandle(inner *sql.DB, driver string) *Handle {
	return &Handle{DB: inner, driver: driver}
}

func (h *Handle) Exec(query string, args ...any) (sql.Result, error) {
	if h == nil || h.DB == nil {
		return noopResult{}, nil
	}
	var (
		res sql.Result
		err error
	)
	err = ejecutarConReintentos(func() error {
		res, err = h.DB.Exec(storage.RebindQuery(h.driver, query), args...)
		return err
	})
	if err == nil && runtimeOrdersHotIndexMutatingQuery(query) {
		markRuntimeOrdersHotIndexDirty()
	}
	return res, err
}

func (h *Handle) Query(query string, args ...any) (*sql.Rows, error) {
	if h == nil || h.DB == nil {
		return noDBFallbackRows()
	}
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
	if h == nil || h.DB == nil {
		return noDBFallbackQueryRow()
	}
	return h.DB.QueryRow(storage.RebindQuery(h.driver, query), args...)
}

func (h *Handle) Begin() (*Tx, error) {
	if h == nil || h.DB == nil {
		return nil, driver.ErrBadConn
	}
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
	if h == nil || h.DB == nil {
		return nil, driver.ErrBadConn
	}
	var (
		tx  *sql.Tx
		err error
	)
	inicio := time.Now()
	for intento := 0; intento < persistenciaBusyMaxIntentos; intento++ {
		tx, err = h.DB.BeginTx(ctx, opts)
		if err == nil || !esErrorPersistenciaBusy(err) {
			break
		}
		if !persistenciaBusyDebeReintentar(inicio, intento) {
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
	if err == nil && runtimeOrdersHotIndexMutatingQuery(query) {
		tx.dirty = true
		markRuntimeOrdersHotIndexDirty()
	}
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
