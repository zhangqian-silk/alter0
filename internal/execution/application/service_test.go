package application

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

type stubProcessor struct {
	output       string
	source       string
	lastContent  string
	lastMetadata map[string]string
}

func (s *stubProcessor) Process(_ context.Context, content string, metadata map[string]string) (string, error) {
	s.lastContent = content
	s.lastMetadata = map[string]string{}
	for key, value := range metadata {
		s.lastMetadata[key] = value
	}
	if strings.TrimSpace(s.source) != "" {
		metadata[execdomain.ExecutionSourceMetadataKey] = s.source
	}
	return s.output, nil
}

func TestExecuteNaturalLanguageInjectsRuntimeMetadata(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	service := NewService(processor)

	_, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "msg-runtime",
		SessionID:   "session-runtime",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "runtime metadata",
		TraceID:     "trace-runtime",
		Metadata: map[string]string{
			"task_id": "task-runtime",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := processor.lastMetadata[execdomain.RuntimeSessionIDMetadataKey]; got != "session-runtime" {
		t.Fatalf("runtime session metadata = %q, want session-runtime", got)
	}
	if got := processor.lastMetadata[execdomain.RuntimeMessageIDMetadataKey]; got != "msg-runtime" {
		t.Fatalf("runtime message metadata = %q, want msg-runtime", got)
	}
	if got := processor.lastMetadata[execdomain.RuntimeTraceIDMetadataKey]; got != "trace-runtime" {
		t.Fatalf("runtime trace metadata = %q, want trace-runtime", got)
	}
	if got := processor.lastMetadata["task_id"]; got != "task-runtime" {
		t.Fatalf("task metadata = %q, want task-runtime", got)
	}
}

func TestExecuteNaturalLanguageReturnsExecutionSourceMetadata(t *testing.T) {
	processor := &stubProcessor{output: "ok", source: execdomain.ExecutionSourceModel}
	service := NewService(processor)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "msg-source",
		SessionID:   "session-source",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "source metadata",
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[execdomain.ExecutionSourceMetadataKey]; got != execdomain.ExecutionSourceModel {
		t.Fatalf("execution source metadata = %q, want %q", got, execdomain.ExecutionSourceModel)
	}
}

type stubSkillSource struct {
	items    []controldomain.Capability
	mcpItems []controldomain.Capability
}

func (s *stubSkillSource) ListCapabilitiesByType(capabilityType controldomain.CapabilityType) []controldomain.Capability {
	switch capabilityType {
	case controldomain.CapabilityTypeSkill:
		out := make([]controldomain.Capability, 0, len(s.items))
		for _, item := range s.items {
			out = append(out, item)
		}
		return out
	case controldomain.CapabilityTypeMCP:
		out := make([]controldomain.Capability, 0, len(s.mcpItems))
		for _, item := range s.mcpItems {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func TestExecuteNaturalLanguageInjectsEnabledSkillsByPriority(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	source := &stubSkillSource{items: []controldomain.Capability{
		{
			ID:      "summary",
			Name:    "Summary",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: true,
			Metadata: map[string]string{
				skillPriorityKey:          "200",
				skillDescriptionKey:       "summary documents",
				skillGuideKey:             "read the resolved context before writing",
				skillParameterTemplateKey: `{"lang":"zh-CN"}`,
				skillConstraintsKey:       "max:300, keep-tone",
				skillFilePathKey:          ".alter0/skills/summary.md",
				skillWritableKey:          "true",
			},
		},
		{
			ID:      "rewrite",
			Name:    "Rewrite",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: true,
			Metadata: map[string]string{
				skillPriorityKey: "100",
			},
		},
		{
			ID:      "disabled",
			Name:    "Disabled",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: false,
			Metadata: map[string]string{
				skillPriorityKey: "999",
			},
		},
	}}
	service := NewServiceWithSkills(processor, source, nil)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "write release note",
		TraceID:     "t1",
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if result.Output != "ok" {
		t.Fatalf("ExecuteNaturalLanguage() output = %q, want %q", result.Output, "ok")
	}
	if got := result.Metadata[resultSkillInjectedKey]; got != "2" {
		t.Fatalf("skills injected count = %q, want 2", got)
	}
	if got := result.Metadata[resultSkillInjectedIDsKey]; got != "summary,rewrite" {
		t.Fatalf("skills injected ids = %q, want summary,rewrite", got)
	}
	if got := result.Metadata[resultSkillProtocolKey]; got != execdomain.SkillContextProtocolVersion {
		t.Fatalf("skills protocol = %q, want %q", got, execdomain.SkillContextProtocolVersion)
	}
	if got := result.Metadata[resultSkillConflictKey]; got != "0" {
		t.Fatalf("skills conflict count = %q, want 0", got)
	}

	rawSkillContext := processor.lastMetadata[execdomain.SkillContextMetadataKey]
	if rawSkillContext == "" {
		t.Fatalf("missing %s metadata", execdomain.SkillContextMetadataKey)
	}
	var skillContext execdomain.SkillContext
	if err := json.Unmarshal([]byte(rawSkillContext), &skillContext); err != nil {
		t.Fatalf("unmarshal skill context: %v", err)
	}
	if len(skillContext.Skills) != 2 {
		t.Fatalf("skill context size = %d, want 2", len(skillContext.Skills))
	}
	if skillContext.Skills[0].ID != "summary" || skillContext.Skills[1].ID != "rewrite" {
		t.Fatalf("unexpected skill order: %+v", skillContext.Skills)
	}
	if skillContext.Skills[0].Description != "summary documents" {
		t.Fatalf("skill description = %q, want summary documents", skillContext.Skills[0].Description)
	}
	if skillContext.Skills[0].Guide != "read the resolved context before writing" {
		t.Fatalf("skill guide = %q, want guide preserved", skillContext.Skills[0].Guide)
	}
	if got := skillContext.Skills[0].ParameterTemplate["lang"]; got != "zh-CN" {
		t.Fatalf("unexpected parameter template lang = %q", got)
	}
	if len(skillContext.Skills[0].Constraints) != 2 {
		t.Fatalf("constraints size = %d, want 2", len(skillContext.Skills[0].Constraints))
	}
	if got := skillContext.Skills[0].FilePath; got != ".alter0/skills/summary.md" {
		t.Fatalf("skill file path = %q, want .alter0/skills/summary.md", got)
	}
	if !skillContext.Skills[0].Writable {
		t.Fatalf("expected skill writable flag true")
	}
}

func TestExecuteNaturalLanguageRespectsIncludeExcludeSelection(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	source := &stubSkillSource{items: []controldomain.Capability{
		{ID: "summary", Name: "Summary", Type: controldomain.CapabilityTypeSkill, Enabled: true},
		{ID: "rewrite", Name: "Rewrite", Type: controldomain.CapabilityTypeSkill, Enabled: true},
		{ID: "proofread", Name: "Proofread", Type: controldomain.CapabilityTypeSkill, Enabled: true},
	}}
	service := NewServiceWithSkills(processor, source, nil)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "refine output",
		TraceID:     "t1",
		Metadata: map[string]string{
			skillIncludeFilterKey: "summary,rewrite",
			skillExcludeFilterKey: "rewrite",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[resultSkillInjectedIDsKey]; got != "summary" {
		t.Fatalf("skills injected ids = %q, want summary", got)
	}
	if got := result.Metadata[resultSkillInjectedKey]; got != "1" {
		t.Fatalf("skills injected count = %q, want 1", got)
	}
	if got := result.Metadata[resultSkillConflictKey]; got != "0" {
		t.Fatalf("skills conflict count = %q, want 0", got)
	}
}

func TestExecuteNaturalLanguageInjectsSelectedMemoryFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "USER.md"), []byte("name: alter0"), 0o644); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "SOUL.md"), []byte("tone: concise"), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	processor := &stubProcessor{output: "ok"}
	service := NewServiceWithSkills(processor, nil, nil)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m-memory",
		SessionID:   "s-memory",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "remember this",
		TraceID:     "t-memory",
		Metadata: map[string]string{
			memoryIncludeFilterKey: `["user_md","soul_md"]`,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[resultMemoryInjectedIDsKey]; got != "user_md,soul_md" {
		t.Fatalf("memory injected ids = %q, want user_md,soul_md", got)
	}
	if got := result.Metadata[resultMemoryInjectedKey]; got != "2" {
		t.Fatalf("memory injected count = %q, want 2", got)
	}
	rawMemoryContext := processor.lastMetadata[execdomain.MemoryContextMetadataKey]
	if rawMemoryContext == "" {
		t.Fatalf("missing %s metadata", execdomain.MemoryContextMetadataKey)
	}
	var memoryContext execdomain.MemoryContext
	if err := json.Unmarshal([]byte(rawMemoryContext), &memoryContext); err != nil {
		t.Fatalf("unmarshal memory context: %v", err)
	}
	if len(memoryContext.Files) != 2 {
		t.Fatalf("memory context size = %d, want 2", len(memoryContext.Files))
	}
	if memoryContext.Files[0].Title != "USER.md" || memoryContext.Files[0].Content != "name: alter0" {
		t.Fatalf("unexpected first memory file: %+v", memoryContext.Files[0])
	}
	if memoryContext.Files[1].Title != "SOUL.md" || memoryContext.Files[1].Content != "tone: concise" {
		t.Fatalf("unexpected second memory file: %+v", memoryContext.Files[1])
	}
}

func TestExecuteNaturalLanguageInjectsAgentSpecificAgentsMD(t *testing.T) {
	root := t.TempDir()
	agentPath := filepath.Join(root, ".alter0", "agents", "researcher", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(agentPath), 0o755); err != nil {
		t.Fatalf("create agent memory dir: %v", err)
	}
	if err := os.WriteFile(agentPath, []byte("scope: researcher only"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("shared root should not be injected"), 0o644); err != nil {
		t.Fatalf("write root AGENTS.md: %v", err)
	}
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	processor := &stubProcessor{output: "ok"}
	service := NewServiceWithSkills(processor, nil, nil)

	_, err = service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m-agent-memory",
		SessionID:   "s-agent-memory",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "use agent memory",
		TraceID:     "t-agent-memory",
		Metadata: map[string]string{
			execdomain.AgentIDMetadataKey: "researcher",
			memoryIncludeFilterKey:        `["agents_md"]`,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}

	rawMemoryContext := processor.lastMetadata[execdomain.MemoryContextMetadataKey]
	if rawMemoryContext == "" {
		t.Fatalf("missing %s metadata", execdomain.MemoryContextMetadataKey)
	}
	var memoryContext execdomain.MemoryContext
	if err := json.Unmarshal([]byte(rawMemoryContext), &memoryContext); err != nil {
		t.Fatalf("unmarshal memory context: %v", err)
	}
	if len(memoryContext.Files) != 1 {
		t.Fatalf("memory context size = %d, want 1", len(memoryContext.Files))
	}
	if got := memoryContext.Files[0].Path; !strings.HasSuffix(got, "/.alter0/agents/researcher/AGENTS.md") {
		t.Fatalf("unexpected agent AGENTS path: %q", got)
	}
	if got := memoryContext.Files[0].Content; got != "scope: researcher only" {
		t.Fatalf("unexpected agent AGENTS content: %q", got)
	}
}

func TestExecuteNaturalLanguageReturnsMissingAgentSpecificAgentsMDPath(t *testing.T) {
	root := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	processor := &stubProcessor{output: "ok"}
	service := NewServiceWithSkills(processor, nil, nil)

	_, err = service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m-agent-missing",
		SessionID:   "s-agent-missing",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "create agent memory",
		TraceID:     "t-agent-missing",
		Metadata: map[string]string{
			execdomain.AgentIDMetadataKey: "Writing Agent",
			memoryIncludeFilterKey:        `["agents_md"]`,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}

	rawMemoryContext := processor.lastMetadata[execdomain.MemoryContextMetadataKey]
	if rawMemoryContext == "" {
		t.Fatalf("missing %s metadata", execdomain.MemoryContextMetadataKey)
	}
	var memoryContext execdomain.MemoryContext
	if err := json.Unmarshal([]byte(rawMemoryContext), &memoryContext); err != nil {
		t.Fatalf("unmarshal memory context: %v", err)
	}
	if len(memoryContext.Files) != 1 {
		t.Fatalf("memory context size = %d, want 1", len(memoryContext.Files))
	}
	file := memoryContext.Files[0]
	if file.Exists {
		t.Fatalf("expected missing agent AGENTS.md path, got %+v", file)
	}
	if got := file.Path; !strings.HasSuffix(got, "/.alter0/agents/writing-agent/AGENTS.md") {
		t.Fatalf("unexpected missing agent AGENTS path: %q", got)
	}
}

func TestExecuteNaturalLanguageResolvesSkillConflicts(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	source := &stubSkillSource{items: []controldomain.Capability{
		{
			ID:      "writer-core",
			Name:    "Writer",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: true,
			Metadata: map[string]string{
				skillPriorityKey:          "300",
				skillAbilitiesKey:         "draft, edit",
				skillParameterTemplateKey: `{"tone":"formal","lang":"zh-CN"}`,
			},
		},
		{
			ID:      "writer-legacy",
			Name:    "Writer",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: true,
			Metadata: map[string]string{
				skillPriorityKey:          "200",
				skillAbilitiesKey:         "draft",
				skillParameterTemplateKey: `{"tone":"casual"}`,
			},
		},
		{
			ID:      "reviewer",
			Name:    "Reviewer",
			Type:    controldomain.CapabilityTypeSkill,
			Enabled: true,
			Metadata: map[string]string{
				skillPriorityKey:          "150",
				skillAbilitiesKey:         "edit, qa",
				skillParameterTemplateKey: `{"tone":"strict","check":"true"}`,
			},
		},
	}}
	service := NewServiceWithSkills(processor, source, nil)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "prepare plan",
		TraceID:     "t1",
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[resultSkillInjectedIDsKey]; got != "writer-core,reviewer" {
		t.Fatalf("skills injected ids = %q, want writer-core,reviewer", got)
	}
	if got := result.Metadata[resultSkillConflictKey]; got != "3" {
		t.Fatalf("skills conflict count = %q, want 3", got)
	}
	if got := result.Metadata[resultSkillConflictTypes]; got != "duplicate_name,duplicate_ability,parameter_conflict" {
		t.Fatalf("skills conflict types = %q, want duplicate_name,duplicate_ability,parameter_conflict", got)
	}
	if strings.TrimSpace(result.Metadata[resultSkillConflictDetail]) == "" {
		t.Fatalf("expected skills conflict detail metadata")
	}

	rawSkillContext := processor.lastMetadata[execdomain.SkillContextMetadataKey]
	var skillContext execdomain.SkillContext
	if err := json.Unmarshal([]byte(rawSkillContext), &skillContext); err != nil {
		t.Fatalf("unmarshal skill context: %v", err)
	}
	if len(skillContext.Skills) != 2 {
		t.Fatalf("skill context size = %d, want 2", len(skillContext.Skills))
	}
	if len(skillContext.Conflicts) != 3 {
		t.Fatalf("skill conflicts size = %d, want 3", len(skillContext.Conflicts))
	}
	if got := skillContext.Skills[1].ParameterTemplate["tone"]; got != "" {
		t.Fatalf("expected reviewer tone parameter dropped, got %q", got)
	}
	if len(skillContext.Skills[1].Abilities) != 1 || skillContext.Skills[1].Abilities[0] != "qa" {
		t.Fatalf("expected reviewer only keeps qa ability, got %+v", skillContext.Skills[1].Abilities)
	}
	if got := skillContext.ResolvedParameters["tone"]; got != "formal" {
		t.Fatalf("resolved tone = %q, want formal", got)
	}
}

func TestExecuteNaturalLanguageInjectsMCPContextWithTransportMapping(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	source := &stubSkillSource{
		items: []controldomain.Capability{},
		mcpItems: []controldomain.Capability{
			{
				ID:      "filesystem",
				Name:    "Filesystem",
				Type:    controldomain.CapabilityTypeMCP,
				Enabled: true,
				Scope:   controldomain.CapabilityScopeGlobal,
				Metadata: map[string]string{
					mcpTransportMetadataKey:     mcpTransportStdio,
					mcpCommandMetadataKey:       "npx",
					mcpArgsMetadataKey:          `["-y","@modelcontextprotocol/server-filesystem"]`,
					mcpToolWhitelistMetadataKey: "read_file,list_dir",
					mcpTimeoutMetadataKey:       "9000",
				},
			},
			{
				ID:      "github",
				Name:    "GitHub",
				Type:    controldomain.CapabilityTypeMCP,
				Enabled: true,
				Scope:   controldomain.CapabilityScopeGlobal,
				Metadata: map[string]string{
					mcpTransportMetadataKey:     mcpTransportHTTP,
					mcpURLMetadataKey:           "https://mcp.example.com/github",
					mcpHeadersMetadataKey:       `{"Authorization":"Bearer token"}`,
					mcpToolWhitelistMetadataKey: `["issues.read","pr.read"]`,
				},
			},
		},
	}
	service := NewServiceWithSkills(processor, source, nil)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "mcp-map-1",
		SessionID:   "session-map",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "list workspace",
		TraceID:     "trace-map",
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[resultMCPInjectedKey]; got != "2" {
		t.Fatalf("mcp injected count = %q, want 2", got)
	}
	if got := result.Metadata[resultMCPProtocolKey]; got != execdomain.MCPContextProtocolVersion {
		t.Fatalf("mcp protocol = %q, want %q", got, execdomain.MCPContextProtocolVersion)
	}

	rawMCPContext := processor.lastMetadata[execdomain.MCPContextMetadataKey]
	if strings.TrimSpace(rawMCPContext) == "" {
		t.Fatalf("missing %s metadata", execdomain.MCPContextMetadataKey)
	}
	var mcpContext execdomain.MCPContext
	if err := json.Unmarshal([]byte(rawMCPContext), &mcpContext); err != nil {
		t.Fatalf("unmarshal mcp context: %v", err)
	}
	if len(mcpContext.Servers) != 2 {
		t.Fatalf("mcp servers size = %d, want 2", len(mcpContext.Servers))
	}
	if mcpContext.Servers[0].Transport != mcpTransportStdio || mcpContext.Servers[0].Command != "npx" {
		t.Fatalf("unexpected stdio mapping: %+v", mcpContext.Servers[0])
	}
	if mcpContext.Servers[1].Transport != mcpTransportHTTP || mcpContext.Servers[1].URL != "https://mcp.example.com/github" {
		t.Fatalf("unexpected http mapping: %+v", mcpContext.Servers[1])
	}
}

func TestExecuteNaturalLanguageSupportsSessionAndRequestScopedMCP(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	source := &stubSkillSource{
		mcpItems: []controldomain.Capability{
			{
				ID:      "session-fs",
				Name:    "SessionFS",
				Type:    controldomain.CapabilityTypeMCP,
				Enabled: true,
				Scope:   controldomain.CapabilityScopeSession,
				Metadata: map[string]string{
					mcpTransportMetadataKey: mcpTransportStdio,
					mcpCommandMetadataKey:   "npx",
				},
			},
			{
				ID:      "request-github",
				Name:    "RequestGitHub",
				Type:    controldomain.CapabilityTypeMCP,
				Enabled: true,
				Scope:   controldomain.CapabilityScopeRequest,
				Metadata: map[string]string{
					mcpTransportMetadataKey: mcpTransportHTTP,
					mcpURLMetadataKey:       "https://mcp.example.com/github",
				},
			},
		},
	}
	service := NewServiceWithSkills(processor, source, nil)

	firstResult, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "mcp-scope-1",
		SessionID:   "session-scope",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "activate session mcp",
		TraceID:     "trace-scope-1",
		Metadata: map[string]string{
			mcpSessionEnableKey: "session-fs",
		},
	})
	if err != nil {
		t.Fatalf("first ExecuteNaturalLanguage() error = %v", err)
	}
	if got := firstResult.Metadata[resultMCPInjectedIDsKey]; got != "session-fs" {
		t.Fatalf("first injected ids = %q, want session-fs", got)
	}

	secondResult, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "mcp-scope-2",
		SessionID:   "session-scope",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "reuse session mcp and request mcp",
		TraceID:     "trace-scope-2",
		Metadata: map[string]string{
			mcpRequestEnableKey: "request-github",
		},
	})
	if err != nil {
		t.Fatalf("second ExecuteNaturalLanguage() error = %v", err)
	}
	if got := secondResult.Metadata[resultMCPInjectedIDsKey]; got != "request-github,session-fs" {
		t.Fatalf("second injected ids = %q, want request-github,session-fs", got)
	}

	thirdResult, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "mcp-scope-3",
		SessionID:   "another-session",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "isolated session should not inherit",
		TraceID:     "trace-scope-3",
	})
	if err != nil {
		t.Fatalf("third ExecuteNaturalLanguage() error = %v", err)
	}
	if got := thirdResult.Metadata[resultMCPInjectedKey]; got != "0" {
		t.Fatalf("third injected count = %q, want 0", got)
	}
}

func TestExecuteNaturalLanguageEnforcesMCPWhitelistTimeoutIsolationAndAudit(t *testing.T) {
	processor := &stubProcessor{output: "ok"}
	source := &stubSkillSource{
		mcpItems: []controldomain.Capability{
			{
				ID:      "safe-http",
				Name:    "SafeHTTP",
				Type:    controldomain.CapabilityTypeMCP,
				Enabled: true,
				Scope:   controldomain.CapabilityScopeRequest,
				Metadata: map[string]string{
					mcpTransportMetadataKey:        mcpTransportHTTP,
					mcpURLMetadataKey:              "https://mcp.example.com/safe",
					mcpToolWhitelistMetadataKey:    `["repo.read","issue.read"]`,
					mcpTimeoutMetadataKey:          "8000",
					mcpFailureIsolationMetadataKey: "true",
				},
			},
			{
				ID:      "bad-timeout",
				Name:    "BadTimeout",
				Type:    controldomain.CapabilityTypeMCP,
				Enabled: true,
				Scope:   controldomain.CapabilityScopeRequest,
				Metadata: map[string]string{
					mcpTransportMetadataKey: mcpTransportHTTP,
					mcpURLMetadataKey:       "https://mcp.example.com/bad",
					mcpTimeoutMetadataKey:   "-1",
				},
			},
		},
	}

	logBuffer := bytes.NewBuffer(nil)
	logger := slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{Level: slog.LevelInfo}))
	service := NewServiceWithSkills(processor, source, logger)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "mcp-sec-1",
		SessionID:   "session-sec",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "call mcp with controls",
		TraceID:     "trace-sec-1",
		Metadata: map[string]string{
			mcpRequestEnableKey: "safe-http,bad-timeout",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[resultMCPInjectedIDsKey]; got != "safe-http" {
		t.Fatalf("injected ids = %q, want safe-http", got)
	}
	if got := result.Metadata[resultMCPInjectedKey]; got != "1" {
		t.Fatalf("injected count = %q, want 1", got)
	}
	if got := result.Metadata[resultMCPAuditCountKey]; got != "2" {
		t.Fatalf("audit count = %q, want 2", got)
	}

	rawMCPContext := processor.lastMetadata[execdomain.MCPContextMetadataKey]
	var mcpContext execdomain.MCPContext
	if err := json.Unmarshal([]byte(rawMCPContext), &mcpContext); err != nil {
		t.Fatalf("unmarshal mcp context: %v", err)
	}
	if len(mcpContext.Servers) != 1 {
		t.Fatalf("mcp servers size = %d, want 1", len(mcpContext.Servers))
	}
	if len(mcpContext.Servers[0].ToolWhitelist) != 2 {
		t.Fatalf("tool whitelist size = %d, want 2", len(mcpContext.Servers[0].ToolWhitelist))
	}
	if mcpContext.Servers[0].TimeoutMS != 8000 {
		t.Fatalf("timeout = %d, want 8000", mcpContext.Servers[0].TimeoutMS)
	}

	rawAudit := result.Metadata[resultMCPAuditDetailKey]
	if strings.TrimSpace(rawAudit) == "" {
		t.Fatalf("expected mcp audit metadata")
	}
	if !strings.Contains(rawAudit, "bad-timeout") {
		t.Fatalf("expected blocked server in audit metadata, got %s", rawAudit)
	}
	if !strings.Contains(logBuffer.String(), "mcp resolved") {
		t.Fatalf("expected mcp audit log output, got %s", logBuffer.String())
	}
}
