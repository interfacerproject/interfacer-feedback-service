package database

import (
	"database/sql"
	"log"
	"os"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	dbInstance *sql.DB
	dbOnce     sync.Once
	dbErr      error
)

// ConnectDB returns a singleton SQLite connection with WAL mode and foreign keys enabled.
// Matching the singleton pattern from interfacer-dpp's database.go.
func ConnectDB() (*sql.DB, error) {
	dbOnce.Do(func() {
		path := os.Getenv("SQLITE_PATH")
		if path == "" {
			path = "./data/feedback.db"
		}

		db, err := sql.Open("sqlite", path)
		if err != nil {
			dbErr = err
			return
		}

		// Enable WAL mode for better concurrent read performance
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			dbErr = err
			return
		}

		// Enable foreign key enforcement
		if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
			dbErr = err
			return
		}

		// Run migrations
		if err := migrate(db); err != nil {
			dbErr = err
			return
		}

		log.Println("Successfully connected to SQLite!")
		dbInstance = db
	})
	return dbInstance, dbErr
}

// migrate creates tables and indexes if they don't exist.
func migrate(db *sql.DB) error {
	// Reviews table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reviews (
			id TEXT PRIMARY KEY,
			project_ulid TEXT NOT NULL,
			user_ulid TEXT NOT NULL,
			rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
			content TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Unique constraint: one review per user per project
	_, err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_project_user
		ON reviews(project_ulid, user_ulid)
	`)
	if err != nil {
		return err
	}

	// Comments table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id TEXT PRIMARY KEY,
			project_ulid TEXT NOT NULL,
			user_ulid TEXT NOT NULL,
			parent_id TEXT,
			content TEXT NOT NULL,
			attachments TEXT,
			status TEXT DEFAULT 'active',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Indexes for comments
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_comments_project ON comments(project_ulid)",
		"CREATE INDEX IF NOT EXISTS idx_comments_user ON comments(user_ulid)",
		"CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id)",
		"CREATE INDEX IF NOT EXISTS idx_comments_created ON comments(created_at)",
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	log.Println("Database migrations complete")
	return nil
}
