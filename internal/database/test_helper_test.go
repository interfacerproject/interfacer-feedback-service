package database

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("pragma foreign_keys: %v", err)
	}

	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func strPtr(s string) *string {
	return &s
}

// ensureDataDir exists for production code compatibility
func ensureDataDir() {
	os.MkdirAll("./data", 0755)
}
