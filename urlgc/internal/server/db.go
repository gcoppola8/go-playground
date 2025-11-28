package server

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database operations
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// NewDB initializes a new SQLite database connection and creates tables if they don't exist
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize tables
	if err := db.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return db, nil
}

// initTables creates the necessary tables if they don't exist
func (db *DB) initTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS short_urls (
		code TEXT PRIMARY KEY,
		target_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_created_at ON short_urls(created_at);
	`
	_, err := db.conn.Exec(query)
	return err
}

// Put stores a code-to-URL mapping in the database
func (db *DB) Put(code, targetURL string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `INSERT INTO short_urls (code, target_url) VALUES (?, ?)`
	_, err := db.conn.Exec(query, code, targetURL)
	if err != nil {
		return fmt.Errorf("failed to insert into database: %w", err)
	}
	return nil
}

// Get retrieves a target URL by code from the database
func (db *DB) Get(code string) (string, bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var targetURL string
	query := `SELECT target_url FROM short_urls WHERE code = ?`
	err := db.conn.QueryRow(query, code).Scan(&targetURL)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to query database: %w", err)
	}
	return targetURL, true, nil
}

// GetAll retrieves all code-to-URL mappings from the database
func (db *DB) GetAll() (map[string]string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `SELECT code, target_url FROM short_urls`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all rows: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var code, targetURL string
		if err := rows.Scan(&code, &targetURL); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result[code] = targetURL
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// BulkPut stores multiple code-to-URL mappings in a single transaction
func (db *DB) BulkPut(data map[string]string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO short_urls (code, target_url) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for code, targetURL := range data {
		if _, err := stmt.Exec(code, targetURL); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete removes a code-to-URL mapping from the database
func (db *DB) Delete(code string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `DELETE FROM short_urls WHERE code = ?`
	_, err := db.conn.Exec(query, code)
	if err != nil {
		return fmt.Errorf("failed to delete from database: %w", err)
	}
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// PeriodicSync periodically syncs the in-memory store to the database
func (db *DB) PeriodicSync(store *Store, interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := db.syncFromStore(store); err != nil {
				log.Printf("Error syncing to database: %v", err)
			} else {
				log.Println("Successfully synced in-memory store to database")
			}
		case <-stopCh:
			log.Println("Stopping periodic database sync")
			return
		}
	}
}

// syncFromStore syncs all data from the in-memory store to the database
func (db *DB) syncFromStore(store *Store) error {
	dataCopy := store.GetAll()
	return db.BulkPut(dataCopy)
}

// LoadIntoStore loads all data from the database into the in-memory store
func (db *DB) LoadIntoStore(store *Store) error {
	data, err := db.GetAll()
	if err != nil {
		return fmt.Errorf("failed to load data from database: %w", err)
	}

	store.LoadAll(data)
	log.Printf("Loaded %d URL mappings from database into memory", len(data))
	return nil
}
