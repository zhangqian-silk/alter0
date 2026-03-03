package localfile

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	controldomain "alter0/internal/control/domain"
	shareddomain "alter0/internal/shared/domain"
)

func TestControlStoreJSONRoundTrip(t *testing.T) {
	store := NewControlStore(t.TempDir(), FormatJSON)
	err := store.Save(context.Background(),
		[]controldomain.Channel{
			{ID: "web-default", Type: shareddomain.ChannelTypeWeb, Enabled: true},
		},
		[]controldomain.Capability{
			{ID: "summary", Name: "Summary", Type: controldomain.CapabilityTypeSkill, Enabled: true, Scope: controldomain.CapabilityScopeGlobal, Version: "v1.0.0"},
			{ID: "github-mcp", Name: "GitHub MCP", Type: controldomain.CapabilityTypeMCP, Enabled: true, Scope: controldomain.CapabilityScopeSession, Version: "v1.1.0"},
		},
		[]controldomain.CapabilityAudit{
			{CapabilityID: "summary", CapabilityType: controldomain.CapabilityTypeSkill, Action: controldomain.CapabilityLifecycleUpdate, Version: "v1.0.0", Scope: controldomain.CapabilityScopeGlobal},
		},
	)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	channels, capabilities, audits, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != "web-default" {
		t.Fatalf("unexpected channels: %+v", channels)
	}
	if len(capabilities) != 2 {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	if len(audits) != 1 || audits[0].CapabilityID != "summary" {
		t.Fatalf("unexpected audits: %+v", audits)
	}
}

func TestControlStoreMarkdownRoundTrip(t *testing.T) {
	store := NewControlStore(t.TempDir(), FormatMarkdown)
	err := store.Save(context.Background(),
		[]controldomain.Channel{
			{ID: "cli-default", Type: shareddomain.ChannelTypeCLI, Enabled: true},
		},
		[]controldomain.Capability{
			{ID: "default-nl", Name: "Default NL", Type: controldomain.CapabilityTypeSkill, Enabled: true, Scope: controldomain.CapabilityScopeGlobal, Version: "v1.0.0"},
		},
		[]controldomain.CapabilityAudit{},
	)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	channels, capabilities, audits, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != "cli-default" {
		t.Fatalf("unexpected channels: %+v", channels)
	}
	if len(capabilities) != 1 || capabilities[0].ID != "default-nl" {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	if len(audits) != 0 {
		t.Fatalf("unexpected audits: %+v", audits)
	}
}

func TestControlStoreLoadLegacySkills(t *testing.T) {
	baseDir := t.TempDir()
	store := NewControlStore(baseDir, FormatJSON)
	legacy := map[string]any{
		"channels": []map[string]any{
			{"id": "web-default", "type": "web", "enabled": true},
		},
		"skills": []map[string]any{
			{"id": "summary", "name": "Summary", "enabled": true},
		},
	}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy failed: %v", err)
	}
	if err := writeFile(filepath.Join(baseDir, "control.json"), append(raw, '\n')); err != nil {
		t.Fatalf("write legacy failed: %v", err)
	}

	_, capabilities, _, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(capabilities) != 1 {
		t.Fatalf("expected 1 capability, got %d", len(capabilities))
	}
	if capabilities[0].Type != controldomain.CapabilityTypeSkill {
		t.Fatalf("expected migrated type skill, got %s", capabilities[0].Type)
	}
	if capabilities[0].Scope != controldomain.CapabilityScopeGlobal {
		t.Fatalf("expected migrated scope global, got %s", capabilities[0].Scope)
	}
	if capabilities[0].Version != controldomain.DefaultCapabilityVersion {
		t.Fatalf("expected migrated default version, got %s", capabilities[0].Version)
	}
}
