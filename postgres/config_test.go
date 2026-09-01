package postgres

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestConfigConnectionURL(t *testing.T) {
	cfg := Config{
		Host:     "db.example.com",
		Port:     5433,
		User:     "promo_user",
		Password: "s3cret",
	}

	got := cfg.ConnectionURL("cotherapist_promo")
	if !strings.Contains(got, "db.example.com:5433") {
		t.Fatalf("host/port missing: %s", got)
	}
	if !strings.Contains(got, "/cotherapist_promo") {
		t.Fatalf("database name missing: %s", got)
	}
	if !strings.Contains(got, "promo_user") {
		t.Fatalf("user missing: %s", got)
	}
}

func TestLoadRequiresPassword(t *testing.T) {
	t.Setenv("DATABASE_PASSWORD", "")
	t.Setenv("DATABASE_HOST", "localhost")

	_, err := Load("cotherapist_promo")
	if err == nil {
		t.Fatal("expected error when DATABASE_PASSWORD is empty")
	}
}

func TestLoadPreparedStatementsFalse(t *testing.T) {
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("DATABASE_PREPARED_STATEMENTS", "false")

	cfg, err := Load("cotherapist_promo")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PreparedStatements {
		t.Fatal("expected prepared statements disabled")
	}
	if cfg.User != "cotherapist_promo" {
		t.Fatalf("user=%q", cfg.User)
	}
}

func TestPoolConfigDisablesPreparedStatements(t *testing.T) {
	cfg := Config{
		Host:               "localhost",
		Port:               5432,
		User:               "promo",
		Password:           "secret",
		PreparedStatements: false,
	}

	poolCfg, err := cfg.PoolConfig("cotherapist_promo")
	if err != nil {
		t.Fatal(err)
	}
	if poolCfg.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Fatalf("expected exec mode, got %v", poolCfg.ConnConfig.DefaultQueryExecMode)
	}
	if poolCfg.ConnConfig.StatementCacheCapacity != 0 {
		t.Fatalf("StatementCacheCapacity = %d, want 0", poolCfg.ConnConfig.StatementCacheCapacity)
	}
	if poolCfg.ConnConfig.DescriptionCacheCapacity != 0 {
		t.Fatalf("DescriptionCacheCapacity = %d, want 0", poolCfg.ConnConfig.DescriptionCacheCapacity)
	}

	maintenanceCfg, err := cfg.PoolConfig("postgres")
	if err != nil {
		t.Fatal(err)
	}
	if maintenanceCfg.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Fatalf("maintenance pool expected exec mode, got %v", maintenanceCfg.ConnConfig.DefaultQueryExecMode)
	}
}

func TestQuoteIdentifier(t *testing.T) {
	if got := quoteIdentifier(`foo`); got != `"foo"` {
		t.Fatalf("got=%s", got)
	}
	if got := quoteIdentifier(`a"b`); got != `"a""b"` {
		t.Fatalf("got=%s", got)
	}
}
