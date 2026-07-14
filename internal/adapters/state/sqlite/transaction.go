package sqlite

import (
	"context"
	"database/sql"
)

type queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func beginTransaction(ctx context.Context, repository *Repository) (*sql.Tx, error) {
	database, err := repository.database()
	if err != nil {
		return nil, err
	}
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	return transaction, nil
}

func beginReadTransaction(ctx context.Context, repository *Repository) (*sql.Tx, error) {
	database, err := repository.database()
	if err != nil {
		return nil, err
	}
	transaction, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	return transaction, nil
}

func commit(transaction *sql.Tx) error {
	if err := transaction.Commit(); err != nil {
		return mapDatabaseError(err)
	}
	return nil
}
