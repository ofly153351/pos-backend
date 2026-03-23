package database

import (
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB, migrationsDir string) error {
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

		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return err
		}
	}

	return nil
}
