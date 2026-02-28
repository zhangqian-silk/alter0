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

func TestTaskMemorySnapshotExportAndRestore(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := NewStore(database)
	ctx := context.Background()

	task1, err := store.CreateTask(ctx, "u-1", "task1", "cli")
	if err != nil {
		t.Fatalf("create task1 failed: %v", err)
	}
	task2, err := store.CreateTask(ctx, "u-1", "task2", "cli")
	if err != nil {
		t.Fatalf("create task2 failed: %v", err)
	}
	if _, err := store.CreateTask(ctx, "u-2", "other", "cli"); err != nil {
		t.Fatalf("create task for other user failed: %v", err)
	}

	if err := store.UpsertTaskMemory(ctx, task1.ID, "memory one"); err != nil {
		t.Fatalf("upsert task1 memory failed: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	if err := store.UpsertTaskMemory(ctx, task2.ID, "memory two"); err != nil {
		t.Fatalf("upsert task2 memory failed: %v", err)
	}

	snapshots, err := store.ExportTaskMemorySnapshot(ctx, "u-1", 10)
	if err != nil {
		t.Fatalf("export snapshot failed: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots[0].TaskID != task2.ID {
		t.Fatalf("expected latest snapshot first, got %s", snapshots[0].TaskID)
	}

	if err := store.DeleteTaskMemory(ctx, task1.ID); err != nil {
		t.Fatalf("delete task1 memory failed: %v", err)
	}
	if err := store.DeleteTaskMemory(ctx, task2.ID); err != nil {
		t.Fatalf("delete task2 memory failed: %v", err)
	}

	applied, err := store.RestoreTaskMemorySnapshot(ctx, "u-1", snapshots)
	if err != nil {
		t.Fatalf("restore snapshot failed: %v", err)
	}
	if applied != 2 {
		t.Fatalf("expected 2 applied snapshots, got %d", applied)
	}

	restored, err := store.GetTaskMemory(ctx, task1.ID)
	if err != nil {
		t.Fatalf("get restored task1 memory failed: %v", err)
	}
	if restored.Summary != "memory one" {
		t.Fatalf("unexpected restored summary for task1: %s", restored.Summary)
	}
}
