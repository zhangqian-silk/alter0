package task

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"alter0/app/core/orchestrator/db"
)

func TestTaskMemoryUpsertAndGet(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := NewStore(database)
	ctx := context.Background()

	task, err := store.CreateTask(ctx, "u-1", "task", "cli")
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	if err := store.UpsertTaskMemory(ctx, task.ID, "first snapshot"); err != nil {
		t.Fatalf("upsert memory failed: %v", err)
	}

	memory, err := store.GetTaskMemory(ctx, task.ID)
	if err != nil {
		t.Fatalf("get memory failed: %v", err)
	}
	if memory.TaskID != task.ID {
		t.Fatalf("unexpected task id: %s", memory.TaskID)
	}
	if memory.Summary != "first snapshot" {
		t.Fatalf("unexpected summary: %s", memory.Summary)
	}
	firstUpdatedAt := memory.UpdatedAt

	time.Sleep(1100 * time.Millisecond)
	if err := store.UpsertTaskMemory(ctx, task.ID, "second snapshot"); err != nil {
		t.Fatalf("upsert memory update failed: %v", err)
	}
	updated, err := store.GetTaskMemory(ctx, task.ID)
	if err != nil {
		t.Fatalf("get updated memory failed: %v", err)
	}
	if updated.Summary != "second snapshot" {
		t.Fatalf("unexpected updated summary: %s", updated.Summary)
	}
	if updated.UpdatedAt <= firstUpdatedAt {
		t.Fatalf("expected updated timestamp to increase: first=%d second=%d", firstUpdatedAt, updated.UpdatedAt)
	}
}

func TestTaskMemoryGetMissing(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := NewStore(database)
	_, err = store.GetTaskMemory(context.Background(), "task-not-found")
	if err == nil {
		t.Fatal("expected error for missing task memory")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestTaskMemoryDelete(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := NewStore(database)
	ctx := context.Background()
	task, err := store.CreateTask(ctx, "u-1", "task", "cli")
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}
	if err := store.UpsertTaskMemory(ctx, task.ID, "snapshot"); err != nil {
		t.Fatalf("upsert memory failed: %v", err)
	}
	if err := store.DeleteTaskMemory(ctx, task.ID); err != nil {
		t.Fatalf("delete memory failed: %v", err)
	}
	_, err = store.GetTaskMemory(ctx, task.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}
