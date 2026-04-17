package localfile

import (
	"encoding/json"
	"testing"
	"time"

	codexdomain "alter0/internal/codex/domain"
)

func TestStoreSaveLoadAndList(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}

	record := Record{
		Name:     "work",
		SavedAt:  time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC),
		Snapshot: codexdomain.Snapshot{AccountName: "Work Account", AccountID: "acct-work", IdentityKey: "oauth:account:acct-work", AuthHash: "hash-work"},
		RawAuth:  json.RawMessage(`{"tokens":{"account_id":"acct-work"}}`),
	}
	if err := store.Save(record, false); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := store.Load("work")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Name != "work" {
		t.Fatalf("loaded.Name = %q, want work", loaded.Name)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "work" {
		t.Fatalf("List = %+v, want single work record", items)
	}
}
