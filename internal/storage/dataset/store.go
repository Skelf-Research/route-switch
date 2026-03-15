package dataset

import (
	"context"
	"time"
)

// Record represents a stored prompt invocation
type Record struct {
	ID        int64
	PromptID  string
	Model     string
	Input     string
	Output    string
	Variables map[string]interface{}
	Success   bool
	Cost      float64
	Metadata  map[string]interface{}
	CreatedAt time.Time
}

// DatasetStore defines persistence for prompt datasets
type DatasetStore interface {
	AddRecord(ctx context.Context, promptID string, record *Record) error
	ListRecent(ctx context.Context, promptID string, limit int) ([]*Record, error)
	TotalCount(ctx context.Context, promptID string) (int64, error)
	Close() error
}
