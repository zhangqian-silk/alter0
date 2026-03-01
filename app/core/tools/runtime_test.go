package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeToolName(t *testing.T) {
	if got := NormalizeToolName(" Web-Search "); got != "web_search" {
		t.Fatalf("expected web_search, got %q", got)
	}
}

func TestPolicyGateRequiresConfirmationByConfig(t *testing.T) {
	gate := NewPolicyGate(Config{RequireConfirm: []string{"tts"}})
	decision := gate.Evaluate(Request{Tool: "tts", AgentID: "default"})
	if decision.Allowed {
		t.Fatal("expected tool to be blocked without confirmation")
	}
	if decision.Code != ErrorCodeConfirmRequired {
		t.Fatalf("expected confirm required code, got %s", decision.Code)
	}

	confirmed := gate.Evaluate(Request{Tool: "tts", AgentID: "default", Confirmed: true})
	if !confirmed.Allowed {
		t.Fatalf("expected confirmed request to pass, got %+v", confirmed)
	}
}

func TestPolicyGateRequiresConfirmationForHighRiskToolByDefault(t *testing.T) {
	gate := NewPolicyGate(Config{})

	decision := gate.Evaluate(Request{Tool: "message", AgentID: "default"})
	if decision.Allowed {
		t.Fatal("expected high-risk tool to require confirmation")
	}
	if decision.Code != ErrorCodeConfirmRequired {
		t.Fatalf("expected confirm required code, got %+v", decision)
	}
	if decision.Reason != "high-risk tool requires confirmation" {
		t.Fatalf("unexpected decision reason: %+v", decision)
	}

	confirmed := gate.Evaluate(Request{Tool: "message", AgentID: "default", Confirmed: true})
	if !confirmed.Allowed {
		t.Fatalf("expected confirmed request to pass, got %+v", confirmed)
	}
}

func TestPolicyGateAppliesAgentAllowlist(t *testing.T) {
	gate := NewPolicyGate(Config{
		Agent: map[string]AgentPolicy{
			"alpha": {Allow: []string{"web_search"}},
		},
	})
	allowed := gate.Evaluate(Request{Tool: "web_search", AgentID: "alpha"})
	if !allowed.Allowed {
		t.Fatalf("expected web_search allowed, got %+v", allowed)
	}
	blocked := gate.Evaluate(Request{Tool: "web_fetch", AgentID: "alpha"})
	if blocked.Allowed || blocked.Code != ErrorCodePolicyDenied {
		t.Fatalf("expected allowlist deny, got %+v", blocked)
	}
}

func TestPolicyGateBlocksSensitiveMemoryPathOnSharedSurface(t *testing.T) {
	gate := NewPolicyGate(Config{})

	blocked := gate.Evaluate(Request{
		Tool:      "memory_get",
		ChannelID: "telegram",
		Surface:   "shared",
		Args: map[string]interface{}{
			"path": "MEMORY.md",
		},
	})
	if blocked.Allowed {
		t.Fatalf("expected memory_get blocked on shared surface, got %+v", blocked)
	}
	if blocked.Code != ErrorCodeContextRestricted {
		t.Fatalf("expected context restriction code, got %+v", blocked)
	}
}

func TestPolicyGateAllowsDailyMemoryPathOnSharedSurface(t *testing.T) {
	gate := NewPolicyGate(Config{})

	decision := gate.Evaluate(Request{
		Tool:      "memory_get",
		ChannelID: "telegram",
		Surface:   "shared",
		Args: map[string]interface{}{
			"path": "memory/2026-03-01.md",
		},
	})
	if !decision.Allowed {
		t.Fatalf("expected memory_get allowed for daily memory path, got %+v", decision)
	}
}

func TestPolicyGateBlocksLongTermSearchOnSharedSurface(t *testing.T) {
	gate := NewPolicyGate(Config{})

	decision := gate.Evaluate(Request{
		Tool:      "memory_search",
		ChannelID: "slack",
		Surface:   "group",
		Args: map[string]interface{}{
			"query":             "roadmap",
			"include_long_term": true,
		},
	})
	if decision.Allowed {
		t.Fatalf("expected long-term memory search denied on shared surface, got %+v", decision)
	}
	if decision.Code != ErrorCodeContextRestricted {
		t.Fatalf("expected context restriction code, got %+v", decision)
	}
}

func TestPolicyGateAllowsMemoryGetOnTrustedMainChannel(t *testing.T) {
	gate := NewPolicyGate(Config{})

	decision := gate.Evaluate(Request{
		Tool:      "memory_get",
		ChannelID: "cli",
		Args: map[string]interface{}{
			"path": "MEMORY.md",
		},
	})
	if !decision.Allowed {
		t.Fatalf("expected memory_get allowed on trusted channel, got %+v", decision)
	}
}

func TestPolicyGatePostureSnapshotIncludesConflicts(t *testing.T) {
	gate := NewPolicyGate(Config{
		GlobalAllow: []string{"web_search", "message"},
		GlobalDeny:  []string{"message"},
	})

	raw := gate.PostureSnapshot()
	if ok, _ := raw["ok"].(bool); ok {
		t.Fatalf("expected posture snapshot to report issues, got %+v", raw)
	}
	issues, ok := raw["issues"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected typed issues, got %T", raw["issues"])
	}
	if len(issues) == 0 {
		t.Fatalf("expected at least one issue, got %+v", raw)
	}

	foundConflict := false
	for _, issue := range issues {
		if issue["id"] == "allow_deny_conflict" {
			foundConflict = true
			break
		}
	}
	if !foundConflict {
		t.Fatalf("expected allow_deny_conflict issue, got %+v", issues)
	}
}

func TestRuntimeInvokeWritesAuditAndNormalizesResult(t *testing.T) {
	auditDir := t.TempDir()
	r := NewRuntime(Config{RequireConfirm: []string{"message"}}, auditDir)

	blocked := r.Invoke(context.Background(), Request{Tool: "message", AgentID: "default"}, nil)
	if blocked.Status != ResultStatusBlocked {
		t.Fatalf("expected blocked status, got %+v", blocked)
	}

	success := r.Invoke(context.Background(), Request{Tool: "web_search", AgentID: "default", Args: map[string]interface{}{"query": "alter0"}}, func(context.Context, Request) (interface{}, error) {
		return map[string]interface{}{"ok": true}, nil
	})
	if success.Status != ResultStatusSuccess {
		t.Fatalf("expected success, got %+v", success)
	}

	retryable := r.Invoke(context.Background(), Request{Tool: "web_fetch", AgentID: "default"}, func(context.Context, Request) (interface{}, error) {
		return nil, &ToolError{Code: "UPSTREAM_TIMEOUT", Message: "timeout", Retryable: true}
	})
	if retryable.Status != ResultStatusRetryable {
		t.Fatalf("expected retryable status, got %+v", retryable)
	}

	failed := r.Invoke(context.Background(), Request{
		Tool:      "browser",
		AgentID:   "default",
		Confirmed: true,
		Args: map[string]interface{}{
			"action": "status",
		},
	}, func(context.Context, Request) (interface{}, error) {
		return nil, errors.New("boom")
	})
	if failed.Status != ResultStatusFailed || failed.Code != ErrorCodeExecutionFailed {
		t.Fatalf("expected normalized failure, got %+v", failed)
	}

	files, err := os.ReadDir(filepath.Join(auditDir, mustSingleDayDir(t, auditDir)))
	if err != nil {
		t.Fatalf("read audit dir failed: %v", err)
	}
	if len(files) != 1 || files[0].Name() != "tool_execution.jsonl" {
		t.Fatalf("expected tool_execution.jsonl, got %#v", files)
	}

	content, err := os.ReadFile(filepath.Join(auditDir, mustSingleDayDir(t, auditDir), "tool_execution.jsonl"))
	if err != nil {
		t.Fatalf("read audit file failed: %v", err)
	}
	lines := splitNonEmptyLines(string(content))
	if len(lines) != 4 {
		t.Fatalf("expected 4 audit lines, got %d", len(lines))
	}

	var entry AuditEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("unmarshal audit entry failed: %v", err)
	}
	if entry.Tool != "message" || entry.Decision != "deny" {
		t.Fatalf("unexpected first audit entry: %+v", entry)
	}
}

func TestRuntimeInvokeValidatesStructuredToolArgs(t *testing.T) {
	r := NewRuntime(Config{}, t.TempDir())

	invalidBrowser := r.Invoke(context.Background(), Request{
		Tool:      "browser",
		AgentID:   "default",
		Confirmed: true,
		Args: map[string]interface{}{
			"action": "act",
			"request": map[string]interface{}{
				"kind": "tap",
			},
		},
	}, func(context.Context, Request) (interface{}, error) {
		t.Fatal("execute should not be called when args are invalid")
		return nil, nil
	})
	if invalidBrowser.Status != ResultStatusFailed {
		t.Fatalf("expected failed status, got %+v", invalidBrowser)
	}
	if invalidBrowser.Code != ErrorCodeInvalidArgs {
		t.Fatalf("expected invalid args code, got %+v", invalidBrowser)
	}

	invalidNodes := r.Invoke(context.Background(), Request{
		Tool:      "nodes",
		AgentID:   "default",
		Confirmed: true,
		Args: map[string]interface{}{
			"action": "run",
		},
	}, func(context.Context, Request) (interface{}, error) {
		t.Fatal("execute should not be called when args are invalid")
		return nil, nil
	})
	if invalidNodes.Code != ErrorCodeInvalidArgs {
		t.Fatalf("expected invalid args code, got %+v", invalidNodes)
	}

	validNodes := r.Invoke(context.Background(), Request{
		Tool:      "nodes",
		AgentID:   "default",
		Confirmed: true,
		Args: map[string]interface{}{
			"action":  "run",
			"command": []interface{}{"uname", "-a"},
		},
	}, func(context.Context, Request) (interface{}, error) {
		return map[string]interface{}{"ok": true}, nil
	})
	if validNodes.Status != ResultStatusSuccess {
		t.Fatalf("expected success status, got %+v", validNodes)
	}
}

func TestRuntimeStatusSnapshotIncludesToolchainSchema(t *testing.T) {
	r := NewRuntime(Config{}, t.TempDir())
	snapshot := r.StatusSnapshot()

	protocol, ok := snapshot["protocol"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected protocol section, got %T", snapshot["protocol"])
	}
	toolchain, ok := protocol["toolchain"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected toolchain section, got %T", protocol["toolchain"])
	}
	browser, ok := toolchain["browser"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected browser schema, got %T", toolchain["browser"])
	}
	if _, exists := browser["act_kinds"]; !exists {
		t.Fatalf("expected browser act_kinds in schema, got %+v", browser)
	}
}

func mustSingleDayDir(t *testing.T, base string) string {
	t.Helper()
	items, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read base dir failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one day dir, got %d", len(items))
	}
	return items[0].Name()
}

func splitNonEmptyLines(raw string) []string {
	lines := make([]string, 0)
	current := ""
	for _, r := range raw {
		if r == '\n' {
			if current != "" {
				lines = append(lines, current)
			}
			current = ""
			continue
		}
		current += string(r)
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
