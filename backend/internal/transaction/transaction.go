package transaction

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Manager interface {
	WithinTransaction(context.Context, func(context.Context, DBTX) error) error
}

type SQLManager struct {
	db *sql.DB
}

func NewSQLManager(db *sql.DB) *SQLManager {
	return &SQLManager{db: db}
}

func (m *SQLManager) WithinTransaction(ctx context.Context, fn func(context.Context, DBTX) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
