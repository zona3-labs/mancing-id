package infrastructure

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"github.com/zone3-labs/mancing-id/internal/config"
)

func NewPostgresDB(config *config.Config) (*sql.DB, error) {
	dsn := config.PostgresDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(time.Duration(config.Postgres.MaxIdleConn))
	db.SetMaxOpenConns(config.Postgres.MaxOpenConn)
	db.SetConnMaxLifetime(time.Duration(config.Postgres.MaxConnLifetime))

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
