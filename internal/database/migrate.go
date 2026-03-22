package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
)

func RunMigrations(db *sql.DB, migrationsDir string) error {
	entries, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}

	sort.Strings(entries)

	for _, path := range entries {
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return err
		}
	}

	return nil
}
