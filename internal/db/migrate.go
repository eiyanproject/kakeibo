package db

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func RunMigrations(dbURL string) error {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	sqlDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return goose.Up(sqlDB, "migrations")
}
