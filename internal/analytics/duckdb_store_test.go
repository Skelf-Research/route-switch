package analytics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDuckDBStore_RecordAndQuery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.duckdb")

	store, err := NewDuckDBStore(path)
	if err != nil {
		t.Fatalf("NewDuckDBStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	record := &InvocationRecord{
		PromptID:      "combo-1",
		TemplateID:    "template-1",
		CombinationID: "combo-1",
		Provider:      "mock",
		Model:         "gpt-4",
		Duration:      150 * time.Millisecond,
		Success:       true,
		Cost:          0.001,
		InputTokens:   100,
		OutputTokens:  50,
		Metadata: map[string]interface{}{
			"request_id": "req-1",
		},
		CreatedAt: time.Now().Add(-1 * time.Minute),
	}

	if err := store.RecordInvocation(ctx, record); err != nil {
		t.Fatalf("RecordInvocation: %v", err)
	}

	stats, err := store.QueryPromptStats(ctx, StatsFilter{TemplateID: "template-1"})
	if err != nil {
		t.Fatalf("QueryPromptStats: %v", err)
	}

	if stats == nil {
		t.Fatalf("expected stats, got nil")
	}

	if stats.TotalRequests != 1 {
		t.Fatalf("expected 1 total request, got %d", stats.TotalRequests)
	}

	if stats.SuccessRate != 1 {
		t.Fatalf("expected success rate 1, got %f", stats.SuccessRate)
	}

	systemStats, err := store.QuerySystemStats(ctx)
	if err != nil {
		t.Fatalf("QuerySystemStats: %v", err)
	}

	if systemStats.TotalPrompts != 1 {
		t.Fatalf("expected 1 prompt, got %d", systemStats.TotalPrompts)
	}
}

func TestDuckDBStore_NoStats(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.duckdb")

	store, err := NewDuckDBStore(path)
	if err != nil {
		t.Fatalf("NewDuckDBStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	stats, err := store.QueryPromptStats(ctx, StatsFilter{PromptID: "missing"})
	if err != nil {
		t.Fatalf("QueryPromptStats: %v", err)
	}
	if stats != nil {
		t.Fatalf("expected nil stats, got %+v", stats)
	}

	systemStats, err := store.QuerySystemStats(ctx)
	if err != nil {
		t.Fatalf("QuerySystemStats: %v", err)
	}
	if systemStats.TotalRequests != 0 {
		t.Fatalf("expected 0 requests, got %d", systemStats.TotalRequests)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("duckdb path missing: %v", err)
	}
}
