// Package pgtest starts an ephemeral Postgres for tests via Testcontainers.
// Import only from _test.go files so production binaries do not link the Docker SDK.
package pgtest

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/cotherapist-ru/go-kit/envconfig"
	"github.com/cotherapist-ru/go-kit/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	defaultImage    = "postgres:17"
	defaultUser     = "test"
	defaultPassword = "test"
	defaultDatabase = "postgres"
)

// Container is a running Postgres (Testcontainers or DATABASE_* from the environment).
type Container struct {
	cfg   postgres.Config
	owned bool
}

type options struct {
	image string
}

// Option configures Start / StartOrEnv.
type Option func(*options)

// WithImage overrides the Postgres image (default postgres:17).
func WithImage(image string) Option {
	return func(o *options) {
		if image != "" {
			o.image = image
		}
	}
}

func applyOptions(opts []Option) options {
	o := options{image: defaultImage}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

// Config returns connection settings for the test Postgres.
func (c *Container) Config() postgres.Config {
	if c == nil {
		return postgres.Config{}
	}
	return c.cfg
}

// Start launches postgres:17 and registers container termination on tb.Cleanup.
// Without Docker the test is skipped, unless PGTEST_REQUIRE is truthy (then Fatal).
func Start(tb testing.TB, opts ...Option) *Container {
	tb.Helper()
	return startContainer(tb, applyOptions(opts))
}

// StartOrEnv uses DATABASE_* when DATABASE_HOST is set; otherwise Start.
func StartOrEnv(tb testing.TB, opts ...Option) *Container {
	tb.Helper()
	if strings.TrimSpace(os.Getenv("DATABASE_HOST")) == "" {
		return Start(tb, opts...)
	}
	cfg, err := postgres.Load(defaultUser)
	if err != nil {
		failOrSkip(tb, err)
		return nil
	}
	return &Container{cfg: cfg, owned: false}
}

func startContainer(tb testing.TB, o options) *Container {
	tb.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		o.image,
		tcpostgres.WithDatabase(defaultDatabase),
		tcpostgres.WithUsername(defaultUser),
		tcpostgres.WithPassword(defaultPassword),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(tb, ctr)
	if err != nil {
		failOrSkip(tb, err)
		return nil
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		failOrSkip(tb, fmt.Errorf("host: %w", err))
		return nil
	}
	mapped, err := ctr.MappedPort(ctx, "5432/tcp")
	if err != nil {
		failOrSkip(tb, fmt.Errorf("mapped port: %w", err))
		return nil
	}

	return &Container{
		cfg: postgres.Config{
			Host:               host,
			Port:               int(mapped.Num()),
			User:               defaultUser,
			Password:           defaultPassword,
			PreparedStatements: true,
		},
		owned: true,
	}
}

// Connect bootstraps database (CREATE DATABASE + *.up.sql) via postgres.Connect.
// An empty database name becomes DatabaseName(tb). The pool is closed on tb.Cleanup.
func (c *Container) Connect(tb testing.TB, database string, migrations fs.FS) *pgxpool.Pool {
	tb.Helper()
	if c == nil {
		tb.Fatal("pgtest: nil container")
	}
	if database == "" {
		database = DatabaseName(tb)
	}
	pool, err := postgres.Connect(context.Background(), c.cfg, database, migrations)
	if err != nil {
		tb.Fatalf("pgtest connect %s: %v", database, err)
	}
	tb.Cleanup(pool.Close)
	return pool
}

// DatabaseName turns tb.Name() into a Postgres identifier (≤63 chars).
func DatabaseName(tb testing.TB) string {
	tb.Helper()
	var b strings.Builder
	b.WriteString("t_")
	for _, r := range strings.ToLower(tb.Name()) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := b.String()
	if len(s) > 63 {
		s = s[:63]
	}
	return strings.TrimRight(s, "_")
}

func failOrSkip(tb testing.TB, err error) {
	tb.Helper()
	msg := fmt.Sprintf("postgres testcontainer: %v", err)
	if envconfig.Truthy("PGTEST_REQUIRE") {
		tb.Fatal(msg)
	}
	tb.Skip(msg)
}
