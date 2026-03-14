package dataset

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStore_AddAndList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(dir, 5)
	if err != nil {
		t.Fatalf("NewSQLiteStore error: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	promptID := "prompt-123"

	for i := 0; i < 7; i++ {
		rec := &Record{
			PromptID: promptID,
			Model:    "model-a",
			Input:    "input",
			Output:   "output",
			Variables: map[string]interface{}{
				"city": "city" + string(rune('A'+i)),
			},
			Success:   true,
			Cost:      float64(i),
			Metadata:  map[string]interface{}{"trial": i},
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := store.AddRecord(ctx, promptID, rec); err != nil {
			t.Fatalf("AddRecord error: %v", err)
		}
	}

	count, err := store.TotalCount(ctx, promptID)
	if err != nil {
		t.Fatalf("TotalCount error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 retained records, got %d", count)
	}

	records, err := store.ListRecent(ctx, promptID, 3)
	if err != nil {
		t.Fatalf("ListRecent error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
}

func TestSQLiteStore_PersistsFiles(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(dir, 2)
	if err != nil {
		t.Fatalf("NewSQLiteStore error: %v", err)
	}
	promptID := "alpha"
	ctx := context.Background()
	if err := store.AddRecord(ctx, promptID, &Record{PromptID: promptID, Model: "x", Input: "in", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("AddRecord error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
	path := filepath.Join(dir, promptID+".db")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("expected db file to exist at %s", path)
		}
		t.Fatalf("stat db file error: %v", err)
	}
}
