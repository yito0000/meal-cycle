package sqlite

import (
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM schema_meta`).Scan(&n); err != nil {
		t.Fatalf("query schema_meta: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row in schema_meta, got %d", n)
	}

	var v int
	if err := db.QueryRow(`SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1`).Scan(&v); err != nil {
		t.Fatalf("query goose_db_version: %v", err)
	}
	if v != 6 {
		t.Fatalf("expected goose version 6, got %d", v)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate should be a no-op: %v", err)
	}
}
