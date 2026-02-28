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

func TestPruneTaskMemoryByClosedAt(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := NewStore(database)
	ctx := context.Background()

	oldClosedTask, err := store.CreateTask(ctx, "u-1", "old-closed", "cli")
	if err != nil {
		t.Fatalf("create oldClosedTask failed: %v", err)
	}
	recentClosedTask, err := store.CreateTask(ctx, "u-1", "recent-closed", "cli")
	if err != nil {
		t.Fatalf("create recentClosedTask failed: %v", err)
	}
	openTask, err := store.CreateTask(ctx, "u-1", "open", "cli")
	if err != nil {
		t.Fatalf("create openTask failed: %v", err)
	}

	if err := store.CloseTask(ctx, oldClosedTask.ID); err != nil {
		t.Fatalf("close old task failed: %v", err)
	}
	if err := store.CloseTask(ctx, recentClosedTask.ID); err != nil {
		t.Fatalf("close recent task failed: %v", err)
	}

	now := time.Now().Unix()
	_, err = database.Conn().ExecContext(ctx, `UPDATE tasks SET closed_at = ?, updated_at = ? WHERE id = ?`, now-86400*30, now-86400*30, oldClosedTask.ID)
	if err != nil {
		t.Fatalf("force old closed_at failed: %v", err)
	}
	_, err = database.Conn().ExecContext(ctx, `UPDATE tasks SET closed_at = ?, updated_at = ? WHERE id = ?`, now-3600, now-3600, recentClosedTask.ID)
	if err != nil {
		t.Fatalf("force recent closed_at failed: %v", err)
	}

	if err := store.UpsertTaskMemory(ctx, oldClosedTask.ID, "old memory"); err != nil {
		t.Fatalf("upsert old memory failed: %v", err)
	}
	if err := store.UpsertTaskMemory(ctx, recentClosedTask.ID, "recent memory"); err != nil {
		t.Fatalf("upsert recent memory failed: %v", err)
	}
	if err := store.UpsertTaskMemory(ctx, openTask.ID, "open memory"); err != nil {
		t.Fatalf("upsert open memory failed: %v", err)
	}

	pruned, err := store.PruneTaskMemoryByClosedAt(ctx, now-86400)
	if err != nil {
		t.Fatalf("prune failed: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("expected pruned=1, got %d", pruned)
	}

	if _, err := store.GetTaskMemory(ctx, oldClosedTask.ID); err != sql.ErrNoRows {
		t.Fatalf("expected old memory to be pruned, got %v", err)
	}
	if _, err := store.GetTaskMemory(ctx, recentClosedTask.ID); err != nil {
		t.Fatalf("expected recent closed memory to stay, got %v", err)
	}
	if _, err := store.GetTaskMemory(ctx, openTask.ID); err != nil {
		t.Fatalf("expected open task memory to stay, got %v", err)
	}
}

func TestPruneTaskMemoryByOpenUpdatedAt(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.NewSQLiteDB(filepath.Join(tempDir, "db"))
	if err != nil {
		t.Fatalf("init sqlite failed: %v", err)
	}
	defer database.Close()

	store := NewStore(database)
	ctx := context.Background()

	oldOpenTask, err := store.CreateTask(ctx, "u-1", "old-open", "cli")
	if err != nil {
		t.Fatalf("create oldOpenTask failed: %v", err)
	}
	recentOpenTask, err := store.CreateTask(ctx, "u-1", "recent-open", "cli")
	if err != nil {
		t.Fatalf("create recentOpenTask failed: %v", err)
	}
	closedTask, err := store.CreateTask(ctx, "u-1", "closed", "cli")
	if err != nil {
		t.Fatalf("create closedTask failed: %v", err)
	}
	if err := store.CloseTask(ctx, closedTask.ID); err != nil {
		t.Fatalf("close task failed: %v", err)
	}

	now := time.Now().Unix()
	if err := store.UpsertTaskMemory(ctx, oldOpenTask.ID, "old open memory"); err != nil {
		t.Fatalf("upsert old open memory failed: %v", err)
	}
	if err := store.UpsertTaskMemory(ctx, recentOpenTask.ID, "recent open memory"); err != nil {
		t.Fatalf("upsert recent open memory failed: %v", err)
	}
	if err := store.UpsertTaskMemory(ctx, closedTask.ID, "closed memory"); err != nil {
		t.Fatalf("upsert closed memory failed: %v", err)
	}

	_, err = database.Conn().ExecContext(ctx, `UPDATE task_memory SET updated_at = ? WHERE task_id = ?`, now-86400*20, oldOpenTask.ID)
	if err != nil {
		t.Fatalf("force old open updated_at failed: %v", err)
	}
	_, err = database.Conn().ExecContext(ctx, `UPDATE task_memory SET updated_at = ? WHERE task_id = ?`, now-3600, recentOpenTask.ID)
	if err != nil {
		t.Fatalf("force recent open updated_at failed: %v", err)
	}
	_, err = database.Conn().ExecContext(ctx, `UPDATE task_memory SET updated_at = ? WHERE task_id = ?`, now-86400*20, closedTask.ID)
	if err != nil {
		t.Fatalf("force closed updated_at failed: %v", err)
	}

	pruned, err := store.PruneTaskMemoryByOpenUpdatedAt(ctx, now-86400*7)
	if err != nil {
		t.Fatalf("prune open memory failed: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("expected pruned=1, got %d", pruned)
	}

	if _, err := store.GetTaskMemory(ctx, oldOpenTask.ID); err != sql.ErrNoRows {
		t.Fatalf("expected old open memory to be pruned, got %v", err)
	}
	if _, err := store.GetTaskMemory(ctx, recentOpenTask.ID); err != nil {
		t.Fatalf("expected recent open memory to stay, got %v", err)
	}
	if _, err := store.GetTaskMemory(ctx, closedTask.ID); err != nil {
		t.Fatalf("expected closed memory to stay for closed-policy prune, got %v", err)
	}
}

func TestTaskMemorySnapshotRestoreIsAtomic(t *testing.T) {
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
	task2, err := store.CreateTask(ctx, "u-2", "task2", "cli")
	if err != nil {
		t.Fatalf("create task2 failed: %v", err)
	}

	snapshots := []TaskMemorySnapshot{
		{TaskID: task1.ID, Summary: "valid summary", UpdatedAt: 100},
		{TaskID: task2.ID, Summary: "wrong owner", UpdatedAt: 101},
	}

	applied, err := store.RestoreTaskMemorySnapshot(ctx, "u-1", snapshots)
	if err == nil {
		t.Fatal("expected restore to fail when one snapshot belongs to another user")
	}
	if applied != 0 {
		t.Fatalf("expected applied=0 on atomic failure, got %d", applied)
	}
	if _, err := store.GetTaskMemory(ctx, task1.ID); err != sql.ErrNoRows {
		t.Fatalf("expected no memory persisted after rollback, got %v", err)
	}
}
