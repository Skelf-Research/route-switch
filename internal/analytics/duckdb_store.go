package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
)

// DuckDBStore implements Store using DuckDB on disk.
type DuckDBStore struct {
	db   *sql.DB
	path string
	mu   sync.Mutex
}

// NewDuckDBStore opens/initializes a DuckDB database at path.
func NewDuckDBStore(path string) (*DuckDBStore, error) {
	if path == "" {
		return nil, fmt.Errorf("duckdb path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create analytics dir: %w", err)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}

	store := &DuckDBStore{db: db, path: path}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *DuckDBStore) migrate() error {
	ddl := `
CREATE SEQUENCE IF NOT EXISTS invocation_seq START 1;
CREATE TABLE IF NOT EXISTS invocations (
	id BIGINT PRIMARY KEY DEFAULT nextval('invocation_seq'),
	prompt_id TEXT,
	template_id TEXT,
	combination_id TEXT,
	provider TEXT,
	model TEXT,
	duration_ms DOUBLE,
	success BOOLEAN,
	cost DOUBLE,
	input_tokens INTEGER,
	output_tokens INTEGER,
	metadata JSON,
	created_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_inv_prompt_id ON invocations(prompt_id);
CREATE INDEX IF NOT EXISTS idx_inv_template_id ON invocations(template_id);
`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("apply duckdb migrations: %w", err)
	}
	return nil
}

// RecordInvocation inserts a record.
func (s *DuckDBStore) RecordInvocation(ctx context.Context, record *InvocationRecord) error {
	if record == nil {
		return fmt.Errorf("record is nil")
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	metaJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO invocations
			(prompt_id, template_id, combination_id, provider, model, duration_ms, success, cost, input_tokens, output_tokens, metadata, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.PromptID,
		record.TemplateID,
		record.CombinationID,
		record.Provider,
		record.Model,
		float64(record.Duration.Microseconds())/1000.0,
		record.Success,
		record.Cost,
		record.InputTokens,
		record.OutputTokens,
		string(metaJSON),
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert invocation: %w", err)
	}
	return nil
}

// QueryPromptStats returns aggregated stats for the filter.
func (s *DuckDBStore) QueryPromptStats(ctx context.Context, filter StatsFilter) (*PromptStats, error) {
	where, args := buildWhere(filter)
	query := `
SELECT 
	COALESCE(prompt_id, template_id) as prompt_id,
	template_id,
	COUNT(*) as total_requests,
	SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count,
	SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as error_count,
	COALESCE(AVG(duration_ms), 0) as avg_latency,
	COALESCE(AVG(cost), 0) as avg_cost,
	MIN(created_at) as first_seen,
	MAX(created_at) as last_seen
FROM invocations`

	if where != "" {
		query += " WHERE " + where
	}
	query += " GROUP BY prompt_id, template_id"

	row := s.db.QueryRowContext(ctx, query, args...)
	var stats PromptStats
	var successCount sql.NullFloat64
	var errorCount sql.NullFloat64
	if err := row.Scan(
		&stats.PromptID,
		&stats.TemplateID,
		&stats.TotalRequests,
		&successCount,
		&errorCount,
		&stats.AvgLatencyMS,
		&stats.AvgCost,
		&stats.FirstSeen,
		&stats.LastSeen,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query prompt stats: %w", err)
	}

	if stats.TotalRequests > 0 && successCount.Valid {
		stats.SuccessRate = successCount.Float64 / float64(stats.TotalRequests)
	}
	if errorCount.Valid {
		stats.ErrorCount = int64(errorCount.Float64)
	}
	return &stats, nil
}

// QuerySystemStats returns aggregates across all prompts.
func (s *DuckDBStore) QuerySystemStats(ctx context.Context) (*SystemStats, error) {
	query := `
SELECT 
	COUNT(DISTINCT COALESCE(template_id, prompt_id)) as total_prompts,
	COUNT(*) as total_requests,
	SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count,
	COALESCE(AVG(duration_ms), 0) as avg_latency,
	COALESCE(AVG(cost), 0) as avg_cost
FROM invocations`

	row := s.db.QueryRowContext(ctx, query)
	var stats SystemStats
	var successCount sql.NullFloat64
	if err := row.Scan(
		&stats.TotalPrompts,
		&stats.TotalRequests,
		&successCount,
		&stats.AvgLatencyMS,
		&stats.AvgCost,
	); err != nil {
		if err == sql.ErrNoRows {
			return &SystemStats{}, nil
		}
		return nil, fmt.Errorf("query system stats: %w", err)
	}

	if stats.TotalRequests > 0 && successCount.Valid {
		stats.SuccessRate = successCount.Float64 / float64(stats.TotalRequests)
	}
	return &stats, nil
}

// Close closes the underlying DB.
func (s *DuckDBStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		err := s.db.Close()
		s.db = nil
		return err
	}
	return nil
}

func buildWhere(filter StatsFilter) (string, []interface{}) {
	var parts []string
	var args []interface{}
	if filter.PromptID != "" {
		parts = append(parts, "prompt_id = ?")
		args = append(args, filter.PromptID)
	}
	if filter.TemplateID != "" {
		parts = append(parts, "template_id = ?")
		args = append(args, filter.TemplateID)
	}
	return strings.Join(parts, " AND "), args
}
