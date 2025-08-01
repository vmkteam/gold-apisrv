package test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	"apisrv/pkg/db"

	"github.com/go-pg/pg/v10"
	"github.com/vmkteam/embedlog"
)

var logger embedlog.Logger

func getenv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

type Cleaner func()

func Setup(t *testing.T) (db.DB, embedlog.Logger) {
	// Create db connection
	conn, err := setup()
	if err != nil {
		if t == nil {
			panic(err)
		}
		t.Fatal(err)
	}

	// Cleanup after tests.
	if t != nil {
		t.Cleanup(func() {
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}

	logger = embedlog.NewLogger(true, true)
	return db.New(conn), logger
}

func setup() (*pg.DB, error) {
	var (
		pghost = getenv("PGHOST", "localhost")
		pgport = getenv("PGPORT", "5432")
		pgdb   = getenv("PGDATABASE", "test-apisrv")
		pguser = getenv("PGUSER", "postgres")
		pgpass = getenv("PGPASSWORD", "postgres")
	)

	url := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable", pguser, pgpass, net.JoinHostPort(pghost, pgport), pgdb)

	cfg, err := pg.ParseURL(url)
	if err != nil {
		return nil, err
	}
	conn := pg.Connect(cfg)

	if r := getenv("DB_LOG_QUERY", "false"); r == "true" {
		conn.AddQueryHook(testDBLogQuery{})
	}

	return conn, nil
}

type testDBLogQuery struct{}

func (d testDBLogQuery) BeforeQuery(ctx context.Context, _ *pg.QueryEvent) (context.Context, error) {
	return ctx, nil
}

func (d testDBLogQuery) AfterQuery(ctx context.Context, q *pg.QueryEvent) error {
	if fm, err := q.FormattedQuery(); err == nil {
		logger.Print(ctx, string(fm))
	}
	return nil
}

func Ptr[T any](v T) *T {
	return &v
}
