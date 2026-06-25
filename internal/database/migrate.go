package database

import (
	"fmt"
	"log"
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

	// Ledger table records which migration files have been applied so each runs
	// exactly once. Created here (not as a migration file) so it bootstraps itself
	// and works on both fresh and existing databases. Existing DBs (which already
	// have 001-025 applied but no ledger) re-run those files once — they are all
	// idempotent — and record them; subsequent boots skip everything recorded.
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename   TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var appliedRows []struct {
		Filename string `gorm:"column:filename"`
	}
	if err := db.Raw(`SELECT filename FROM schema_migrations`).Scan(&appliedRows).Error; err != nil {
		return fmt.Errorf("read schema_migrations: %w", err)
	}
	applied := make(map[string]bool, len(appliedRows))
	for _, r := range appliedRows {
		applied[r.Filename] = true
	}

	for _, path := range entries {
		name := filepath.Base(path)
		if applied[name] {
			continue
		}

		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		log.Printf("migration: applying %s", name)
		// Run the migration and record it in ONE transaction: a failed migration
		// rolls back cleanly and is retried (never half-applied) on the next boot.
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sqlBytes)).Error; err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
			return tx.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, name).Error
		}); err != nil {
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
