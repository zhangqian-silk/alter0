package task

import (
	"context"
	"path/filepath"
	"testing"

	"alter0/app/core/orchestrator/db"
)

func TestGlobalStats(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := NewStore(database)
	ctx := context.Background()

	openTask, err := store.CreateTask(ctx, "user-a", "open task", "cli")
	if err != nil {
		t.Fatalf("create open task failed: %v", err)
	}
	closedTask, err := store.CreateTask(ctx, "user-a", "closed task", "cli")
	if err != nil {
		t.Fatalf("create closed task failed: %v", err)
	}
	if err := store.CloseTask(ctx, closedTask.ID); err != nil {
		t.Fatalf("close task failed: %v", err)
	}
	if err := store.AppendMessage(ctx, openTask.ID, "user-a", "cli", "user", "hello", nil); err != nil {
		t.Fatalf("append message failed: %v", err)
	}
	if err := store.UpsertTaskMemory(ctx, openTask.ID, "summary"); err != nil {
		t.Fatalf("upsert memory failed: %v", err)
	}

	stats, err := store.GlobalStats(ctx)
	if err != nil {
		t.Fatalf("global stats failed: %v", err)
	}

	if stats.Total != 2 {
		t.Fatalf("expected total=2, got %d", stats.Total)
	}
	if stats.Open != 1 {
		t.Fatalf("expected open=1, got %d", stats.Open)
	}
	if stats.Closed != 1 {
		t.Fatalf("expected closed=1, got %d", stats.Closed)
	}
	if stats.WithMemory != 1 {
		t.Fatalf("expected with_memory=1, got %d", stats.WithMemory)
	}
}

func TestGlobalStatsEmptyStore(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := NewStore(database)
	stats, err := store.GlobalStats(context.Background())
	if err != nil {
		t.Fatalf("global stats failed: %v", err)
	}
	if stats.Total != 0 || stats.Open != 0 || stats.Closed != 0 || stats.WithMemory != 0 {
		t.Fatalf("expected all zeros, got %+v", stats)
	}
}
