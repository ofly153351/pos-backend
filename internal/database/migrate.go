package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB, migrationsDir string) error {
	if migrationsDir == "" {
		return fmt.Errorf("migrations dir is empty")
	}

	absDir, info, err := resolveMigrationsDir(migrationsDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("migrations path %q is not a directory", absDir)
	}

	entries, err := filepath.Glob(filepath.Join(absDir, "*.sql"))
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no migration files found in %q", absDir)
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

func resolveMigrationsDir(migrationsDir string) (string, os.FileInfo, error) {
	if filepath.IsAbs(migrationsDir) {
		info, err := os.Stat(migrationsDir)
		if err != nil {
			return "", nil, fmt.Errorf("cannot access migrations dir %q: %w", migrationsDir, err)
		}
		return migrationsDir, info, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}

	current := cwd
	for {
		candidate := filepath.Join(current, migrationsDir)
		info, statErr := os.Stat(candidate)
		if statErr == nil {
			return candidate, info, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			absInput, absErr := filepath.Abs(migrationsDir)
			if absErr != nil {
				return "", nil, absErr
			}
			return "", nil, fmt.Errorf("cannot access migrations dir %q: %w", absInput, os.ErrNotExist)
		}
		current = parent
	}
}
