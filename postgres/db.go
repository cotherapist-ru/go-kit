package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect bootstraps the database if needed, opens a pool, and applies *.up.sql migrations.
func Connect(ctx context.Context, dbCfg Config, database string, migrations fs.FS) (*pgxpool.Pool, error) {
	if err := bootstrap(ctx, dbCfg, database); err != nil {
		return nil, err
	}

	poolCfg, err := dbCfg.PoolConfig(database)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	if migrations != nil {
		if err := runMigrations(ctx, pool, migrations); err != nil {
			pool.Close()
			return nil, err
		}
	}

	return pool, nil
}

func bootstrap(ctx context.Context, dbCfg Config, database string) error {
	poolCfg, err := dbCfg.PoolConfig("postgres")
	if err != nil {
		return fmt.Errorf("maintenance pool config: %w", err)
	}

	maintenance, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("connect maintenance db: %w", err)
	}
	defer maintenance.Close()

	var exists bool
	err = maintenance.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", database,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}

	if !exists {
		quoted := quoteIdentifier(database)
		if _, err := maintenance.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
			return fmt.Errorf("create database: %w", err)
		}
		slog.Info("database created", "database", database)
	}

	return nil
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".up.sql")
		var applied bool
		if err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}

		body, err := fs.ReadFile(migrations, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", version,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
		slog.Info("migration applied", "version", version)
	}

	return nil
}
