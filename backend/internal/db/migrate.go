package db

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(db *sql.DB) error {
	// All migration statements are idempotent. MySQL DDL autocommits; record the version only after success.
	names := []string{"migrations/001_initial.sql"}
	for i, name := range names {
		raw, err := migrations.ReadFile(name)
		if err != nil {
			return err
		}
		var clean []string
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				continue
			}
			clean = append(clean, line)
		}
		sqlText := strings.Join(clean, "\n")
		for _, statement := range strings.Split(sqlText, ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err = db.Exec(statement); err != nil {
				return fmt.Errorf("migration %d failed: %w", i+1, err)
			}
		}
		if _, err = db.Exec("INSERT IGNORE INTO schema_migrations(id) VALUES(?)", i+1); err != nil {
			return err
		}
	}
	return nil
}
