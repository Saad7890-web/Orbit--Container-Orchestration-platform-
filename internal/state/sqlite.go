package state

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

const migrationsTable = "schema_migrations"

func (s *Store) Migrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store is not initialized")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	applied, err := loadAppliedMigrations(ctx, tx)
	if err != nil {
		return err
	}

	files, err := migrationFiles()
	if err != nil {
		return err
	}

	for _, mf := range files {
		if _, ok := applied[mf.Name]; ok {
			continue
		}

		sqlBytes, err := migrationFileContents(mf.Name)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", mf.Name, err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO schema_migrations(version, applied_at)
			VALUES (?, ?)
		`, mf.Name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("record migration %s: %w", mf.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}

	return nil
}

type migrationFile struct {
	Name string
}

func loadAppliedMigrations(ctx context.Context, tx *sql.Tx) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	out := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		out[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}
	return out, nil
}

func migrationFiles() ([]migrationFile, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	files := make([]migrationFile, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, migrationFile{Name: e.Name()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

func migrationFileContents(name string) ([]byte, error) {
	b, err := migrationFS.ReadFile("migrations/" + name)
	if err != nil {
		return nil, fmt.Errorf("read migration %q: %w", name, err)
	}
	return b, nil
}