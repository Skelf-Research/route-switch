package dataset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using a dedicated SQLite DB per prompt
type SQLiteStore struct {
	basePath   string
	maxRecords int
	mu         sync.Mutex
	dbs        map[string]*sql.DB
}

// NewSQLiteStore creates the store rooted at basePath
func NewSQLiteStore(basePath string, maxRecords int) (*SQLiteStore, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("create dataset dir: %w", err)
	}
	return &SQLiteStore{
		basePath:   basePath,
		maxRecords: maxRecords,
		dbs:        make(map[string]*sql.DB),
	}, nil
}

func (s *SQLiteStore) dbPath(promptID string) string {
	return filepath.Join(s.basePath, fmt.Sprintf("%s.db", promptID))
}

func (s *SQLiteStore) getDB(promptID string) (*sql.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if db, ok := s.dbs[promptID]; ok {
		return db, nil
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", s.dbPath(promptID))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := s.migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	s.dbs[promptID] = db
	return db, nil
}

func (s *SQLiteStore) migrate(db *sql.DB) error {
	ddl := `CREATE TABLE IF NOT EXISTS examples (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		prompt_id TEXT NOT NULL,
		model TEXT NOT NULL,
		input TEXT NOT NULL,
		output TEXT,
		variables TEXT,
		success INTEGER,
		cost REAL,
		metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	return nil
}

// AddRecord inserts a record and enforces retention
func (s *SQLiteStore) AddRecord(ctx context.Context, promptID string, record *Record) error {
	db, err := s.getDB(promptID)
	if err != nil {
		return err
	}

	varsJSON, err := json.Marshal(record.Variables)
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO examples (prompt_id, model, input, output, variables, success, cost, metadata, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		promptID,
		record.Model,
		record.Input,
		record.Output,
		string(varsJSON),
		boolToInt(record.Success),
		record.Cost,
		string(metadataJSON),
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert record: %w", err)
	}

	return s.prune(ctx, db, promptID)
}

// ListRecent returns most recent records up to limit
func (s *SQLiteStore) ListRecent(ctx context.Context, promptID string, limit int) ([]*Record, error) {
	db, err := s.getDB(promptID)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `SELECT id, prompt_id, model, input, output, variables, success, cost, metadata, created_at
		FROM examples WHERE prompt_id = ? ORDER BY created_at DESC LIMIT ?`, promptID, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent records: %w", err)
	}
	defer rows.Close()

	var out []*Record
	for rows.Next() {
		var r Record
		var varsJSON, metadataJSON sql.NullString
		var success int
		if err := rows.Scan(&r.ID, &r.PromptID, &r.Model, &r.Input, &r.Output, &varsJSON, &success, &r.Cost, &metadataJSON, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}
		r.Success = success == 1
		if varsJSON.Valid && varsJSON.String != "" {
			_ = json.Unmarshal([]byte(varsJSON.String), &r.Variables)
		} else {
			r.Variables = map[string]interface{}{}
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			_ = json.Unmarshal([]byte(metadataJSON.String), &r.Metadata)
		} else {
			r.Metadata = map[string]interface{}{}
		}
		out = append(out, &r)
	}

	return out, rows.Err()
}

// TotalCount returns total records stored for prompt
func (s *SQLiteStore) TotalCount(ctx context.Context, promptID string) (int64, error) {
	db, err := s.getDB(promptID)
	if err != nil {
		return 0, err
	}
	var count int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM examples WHERE prompt_id = ?`, promptID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count records: %w", err)
	}
	return count, nil
}

func (s *SQLiteStore) prune(ctx context.Context, db *sql.DB, promptID string) error {
	if s.maxRecords <= 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, `DELETE FROM examples WHERE id IN (
		SELECT id FROM examples WHERE prompt_id = ? ORDER BY created_at DESC LIMIT -1 OFFSET ?)`, promptID, s.maxRecords)
	if err != nil {
		return fmt.Errorf("prune records: %w", err)
	}
	return nil
}

// Close closes all open DB handles
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var firstErr error
	for id, db := range s.dbs {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close db %s: %w", id, err)
		}
		delete(s.dbs, id)
	}
	return firstErr
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
