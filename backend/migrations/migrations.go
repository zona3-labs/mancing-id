package migrations

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

// Files contains the clean pre-production PostgreSQL baseline.
//
//go:embed *.sql
var Files embed.FS

func Apply(db *sql.DB) error {
	goose.SetBaseFS(Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, ".")
}

func Reset(db *sql.DB) error {
	goose.SetBaseFS(Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.DownTo(db, ".", 0); err != nil {
		return err
	}
	return goose.Up(db, ".")
}
