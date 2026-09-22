package pgtest

import (
	"context"
	"os"
	"strconv"
	"testing"
)

func TestStartConnectPing(t *testing.T) {
	ctr := Start(t)
	pool := ctr.Connect(t, DatabaseName(t), os.DirFS("testdata"))

	var n int
	if err := pool.QueryRow(context.Background(), "SELECT 1").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("SELECT 1 = %d", n)
	}

	if _, err := pool.Exec(context.Background(), "INSERT INTO widgets (name) VALUES ($1)", "w"); err != nil {
		t.Fatalf("migrations not applied: %v", err)
	}
}

func TestStartFollowsDATABASEHost(t *testing.T) {
	owned := Start(t)
	cfg := owned.Config()
	t.Setenv("DATABASE_HOST", cfg.Host)
	t.Setenv("DATABASE_PORT", strconv.Itoa(cfg.Port))
	t.Setenv("DATABASE_USER", cfg.User)
	t.Setenv("DATABASE_PASSWORD", cfg.Password)

	again := Start(t)
	if again.owned {
		t.Fatal("Start must reuse DATABASE_HOST instead of launching Testcontainers")
	}
	if again.Config().Host != cfg.Host || again.Config().Port != cfg.Port {
		t.Fatalf("got %+v want %+v", again.Config(), cfg)
	}
}

func TestStartOrEnvReusesDATABASEHost(t *testing.T) {
	owned := Start(t)
	cfg := owned.Config()
	t.Setenv("DATABASE_HOST", cfg.Host)
	t.Setenv("DATABASE_PORT", strconv.Itoa(cfg.Port))
	t.Setenv("DATABASE_USER", cfg.User)
	t.Setenv("DATABASE_PASSWORD", cfg.Password)

	reused := StartOrEnv(t)
	if reused.owned {
		t.Fatal("expected env-backed container, not a new Testcontainers instance")
	}
	if reused.Config().Host != cfg.Host || reused.Config().Port != cfg.Port {
		t.Fatalf("got %+v want %+v", reused.Config(), cfg)
	}

	pool := reused.Connect(t, DatabaseName(t), nil)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestConnectDefaultDatabaseName(t *testing.T) {
	ctr := Start(t)
	pool := ctr.Connect(t, "", nil)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}
