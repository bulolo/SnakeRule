package repo

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// This integration test is enabled in CI against a disposable PostgreSQL database.
func TestOpenDatabaseAppliesInitialSchema(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL migration integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	if err := OpenDatabase(db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := OpenDatabase(db); err != nil {
		t.Fatalf("reapply migrations: %v", err)
	}

	var organizations bool
	if err := db.QueryRow(`SELECT to_regclass('public.organizations') IS NOT NULL`).Scan(&organizations); err != nil {
		t.Fatalf("check initial schema: %v", err)
	}
	if !organizations {
		t.Fatal("initial migration did not create organizations")
	}

	var initialMigrationApplied bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM snakerule_goose_db_version WHERE version_id = 1 AND is_applied)`).Scan(&initialMigrationApplied); err != nil {
		t.Fatalf("check initial migration version: %v", err)
	}
	if !initialMigrationApplied {
		t.Fatal("initial migration version 1 is not recorded as applied")
	}
}
