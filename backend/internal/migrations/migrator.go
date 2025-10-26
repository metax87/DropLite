package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	dbmigrations "droplite/db/migrations"
)

// Apply 执行 embed 的全部 up 迁移脚本。
func Apply(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("nil database connection")
	}

	if err := ensureSchemaMigrations(ctx, db); err != nil {
		return err
	}

	applied, err := fetchApplied(ctx, db)
	if err != nil {
		return err
	}

	files, err := loadMigrationFiles()
	if err != nil {
		return err
	}

	for _, mig := range files {
		if applied[mig.Name] {
			continue
		}

		if err := applyOne(ctx, db, mig); err != nil {
			return err
		}
	}

	return nil
}

type migrationFile struct {
	Name string
	SQL  string
}

func loadMigrationFiles() ([]migrationFile, error) {
	entries, err := fs.ReadDir(dbmigrations.UpFiles, ".")
	if err != nil {
		return nil, fmt.Errorf("read migration files: %w", err)
	}

	var files []migrationFile
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		data, err := dbmigrations.UpFiles.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		files = append(files, migrationFile{Name: name, SQL: string(data)})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

func ensureSchemaMigrations(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func fetchApplied(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("select schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func applyOne(ctx context.Context, db *sql.DB, mig migrationFile) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}

	if _, err := tx.ExecContext(ctx, mig.SQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("apply migration %s: %w", mig.Name, err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, mig.Name); err != nil {
		tx.Rollback()
		return fmt.Errorf("record migration %s: %w", mig.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", mig.Name, err)
	}

	return nil
}
