package postgres

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/cotherapist-ru/go-kit/envconfig"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is PostgreSQL connection settings shared by Cotherapist Go services.
type Config struct {
	Host               string
	Port               int
	User               string
	Password           string
	PreparedStatements bool
}

// Load reads DATABASE_* env vars. defaultUser is used when DATABASE_USER is empty.
func Load(defaultUser string) (Config, error) {
	host := envconfig.Or("DATABASE_HOST", "localhost")

	port, err := envconfig.Int("DATABASE_PORT", 5432)
	if err != nil {
		return Config{}, err
	}

	user := envconfig.Or("DATABASE_USER", defaultUser)
	password := os.Getenv("DATABASE_PASSWORD")
	if password == "" {
		return Config{}, fmt.Errorf("DATABASE_PASSWORD is required")
	}

	return Config{
		Host:               host,
		Port:               port,
		User:               user,
		Password:           password,
		PreparedStatements: os.Getenv("DATABASE_PREPARED_STATEMENTS") != "false",
	}, nil
}

// ConnectionURL builds a postgres:// DSN for the given database name.
func (d Config) ConnectionURL(database string) string {
	u := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:   "/" + database,
	}
	u.User = url.UserPassword(d.User, d.Password)
	q := u.Query()
	q.Set("sslmode", "prefer")
	u.RawQuery = q.Encode()
	return u.String()
}

// PoolConfig parses a pgx pool config, optionally disabling prepared statements for PgBouncer.
func (d Config) PoolConfig(database string) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(d.ConnectionURL(database))
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	if !d.PreparedStatements {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
		cfg.ConnConfig.StatementCacheCapacity = 0
		cfg.ConnConfig.DescriptionCacheCapacity = 0
	}
	return cfg, nil
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
