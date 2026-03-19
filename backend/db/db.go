package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	conn *sql.DB
}

// New initializes the database connection and runs schema migrations
func New(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &Database{conn: conn}

	// Run schema migration
	if err := db.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Init runs the database schema initialization
func (db *Database) Init() error {
	schemaPath := "db/schema.sql"
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = db.conn.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Load seed data
	if err := db.loadSeedData(); err != nil {
		return fmt.Errorf("failed to load seed data: %w", err)
	}

	return nil
}

// loadSeedData loads initial quiz questions
func (db *Database) loadSeedData() error {
	seedPath := "db/seed.sql"
	seed, err := os.ReadFile(seedPath)
	if err != nil {
		return fmt.Errorf("failed to read seed file: %w", err)
	}

	_, err = db.conn.Exec(string(seed))
	if err != nil {
		return fmt.Errorf("failed to execute seed data: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying SQL connection
func (db *Database) Conn() *sql.DB {
	return db.conn
}
