package application

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
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

func decodeMemoryContextFromMetadata(t *testing.T, metadata map[string]string) execdomain.MemoryContext {
	t.Helper()
	rawMemoryContext := metadata[execdomain.MemoryContextMetadataKey]
	if rawMemoryContext == "" {
		t.Fatalf("missing %s metadata", execdomain.MemoryContextMetadataKey)
	}
	var memoryContext execdomain.MemoryContext
	if err := json.Unmarshal([]byte(rawMemoryContext), &memoryContext); err != nil {
		t.Fatalf("unmarshal memory context: %v", err)
	}
	return memoryContext
}

func decodeSkillContextFromMetadata(t *testing.T, metadata map[string]string) execdomain.SkillContext {
	t.Helper()
	rawSkillContext := metadata[execdomain.SkillContextMetadataKey]
	if rawSkillContext == "" {
		t.Fatalf("missing %s metadata", execdomain.SkillContextMetadataKey)
	}
	var skillContext execdomain.SkillContext
	if err := json.Unmarshal([]byte(rawSkillContext), &skillContext); err != nil {
		t.Fatalf("unmarshal skill context: %v", err)
	}
	return skillContext
}

func findMemoryFileBySelection(memoryContext execdomain.MemoryContext, selection string) (execdomain.MemoryFileSpec, bool) {
	for _, file := range memoryContext.Files {
		if file.Selection == selection {
			return file, true
		}
	}
	return execdomain.MemoryFileSpec{}, false
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, strings.TrimSpace(string(output)))
	}
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

func TestExecuteNaturalLanguageInjectsAgentOwnedSkillAndCreatesRulebook(t *testing.T) {
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
	source := &stubSkillSource{items: []controldomain.Capability{
		{ID: "summary", Name: "Summary", Type: controldomain.CapabilityTypeSkill, Enabled: true},
	}}
	service := NewServiceWithSkills(processor, source, nil)

	result, err := service.ExecuteNaturalLanguage(context.Background(), shareddomain.UnifiedMessage{
		MessageID:   "m-agent-skill",
		SessionID:   "s-agent-skill",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "keep this agent's format stable",
		TraceID:     "t-agent-skill",
		Metadata: map[string]string{
			execdomain.AgentIDMetadataKey:   "coding",
			execdomain.AgentNameMetadataKey: "Coding Agent",
			skillIncludeFilterKey:           `["summary"]`,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[resultSkillInjectedIDsKey]; got != "agent-skill-coding,summary" {
		t.Fatalf("skills injected ids = %q, want agent-skill-coding,summary", got)
	}
	if got := result.Metadata[resultSkillInjectedKey]; got != "2" {
		t.Fatalf("skills injected count = %q, want 2", got)
	}

	skillContext := decodeSkillContextFromMetadata(t, processor.lastMetadata)
	if len(skillContext.Skills) != 2 {
		t.Fatalf("skill context size = %d, want 2", len(skillContext.Skills))
	}
	agentSkill := skillContext.Skills[0]
	if agentSkill.ID != "agent-skill-coding" {
		t.Fatalf("agent skill id = %q, want agent-skill-coding", agentSkill.ID)
	}
	if agentSkill.FilePath != ".alter0/agents/coding/SKILL.md" {
		t.Fatalf("agent skill file path = %q, want .alter0/agents/coding/SKILL.md", agentSkill.FilePath)
	}
	if !agentSkill.Writable {
		t.Fatal("expected agent skill writable")
	}
	if !strings.Contains(agentSkill.Guide, "AGENTS.md") {
		t.Fatalf("agent skill guide = %q, want AGENTS.md boundary", agentSkill.Guide)
	}

	skillPath := filepath.Join(root, ".alter0", "agents", "coding", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read agent skill file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Coding Agent Skill") {
		t.Fatalf("unexpected agent skill content: %q", content)
	}
	if !strings.Contains(content, ".alter0/agents/coding/AGENTS.md") {
		t.Fatalf("expected agent skill content to distinguish AGENTS.md, got %q", content)
	}
}

func TestExecuteNaturalLanguageSeedsTravelAgentOwnedSkill(t *testing.T) {
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
		MessageID:   "m-travel-agent-skill",
		SessionID:   "s-travel-agent-skill",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "create a travel guide",
		TraceID:     "t-travel-agent-skill",
		Metadata: map[string]string{
			execdomain.AgentIDMetadataKey:           "travel-master",
			execdomain.AgentNameMetadataKey:         "Travel Master Agent",
			execdomain.AgentCapabilitiesMetadataKey: `["travel","product-master"]`,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}

	skillContext := decodeSkillContextFromMetadata(t, processor.lastMetadata)
	if len(skillContext.Skills) != 1 {
		t.Fatalf("skill context size = %d, want 1", len(skillContext.Skills))
	}
	agentSkill := skillContext.Skills[0]
	if agentSkill.FilePath != ".alter0/agents/travel-master/SKILL.md" {
		t.Fatalf("travel agent skill file path = %q, want .alter0/agents/travel-master/SKILL.md", agentSkill.FilePath)
	}
	if !strings.Contains(agentSkill.Description, "travel agent") {
		t.Fatalf("travel agent skill description = %q, want travel-specific description", agentSkill.Description)
	}
	if !strings.Contains(agentSkill.Guide, "city-page") {
		t.Fatalf("travel agent skill guide = %q, want travel-specific guidance", agentSkill.Guide)
	}

	skillPath := filepath.Join(root, ".alter0", "agents", "travel-master", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read travel agent skill file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "## Stable travel contract") {
		t.Fatalf("unexpected travel agent skill content: %q", content)
	}
	if !strings.Contains(content, "day-by-day route planning") {
		t.Fatalf("expected travel agent skill content to include travel page defaults, got %q", content)
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
	memoryContext := decodeMemoryContextFromMetadata(t, processor.lastMetadata)
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

func TestExecuteNaturalLanguageAutoRecallsSelectedMemorySnippets(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".alter0", "memory", "long-term"), 0o755); err != nil {
		t.Fatalf("mkdir memory dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".alter0", "memory", "long-term", "MEMORY.md"), []byte(strings.Join([]string{
		"# Memory",
		"- response_style: concise Chinese replies",
		"- deployment_target: linux cloud server",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("write MEMORY.md: %v", err)
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
		MessageID:   "m-memory-recall",
		SessionID:   "s-memory-recall",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "use my response style",
		TraceID:     "t-memory-recall",
		Metadata: map[string]string{
			memoryIncludeFilterKey: `["memory_long_term"]`,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}
	if got := result.Metadata[resultMemoryRecallKey]; got != "1" {
		t.Fatalf("memory recall count = %q, want 1", got)
	}
	memoryContext := decodeMemoryContextFromMetadata(t, processor.lastMetadata)
	if len(memoryContext.Recall) != 1 {
		t.Fatalf("memory recall size = %d, want 1: %+v", len(memoryContext.Recall), memoryContext.Recall)
	}
	if !strings.Contains(memoryContext.Recall[0].Snippet, "response_style") {
		t.Fatalf("expected recalled snippet to contain response_style, got %+v", memoryContext.Recall[0])
	}
	if memoryContext.Recall[0].Line != 2 {
		t.Fatalf("expected recalled line 2, got %+v", memoryContext.Recall[0])
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

	memoryContext := decodeMemoryContextFromMetadata(t, processor.lastMetadata)
	agentFile, ok := findMemoryFileBySelection(memoryContext, memorySelectionAgentsMD)
	if !ok {
		t.Fatalf("expected %s in memory context: %+v", memorySelectionAgentsMD, memoryContext.Files)
	}
	if got := agentFile.Path; !strings.HasSuffix(got, "/.alter0/agents/researcher/AGENTS.md") {
		t.Fatalf("unexpected agent AGENTS path: %q", got)
	}
	if got := agentFile.Content; got != "scope: researcher only" {
		t.Fatalf("unexpected agent AGENTS content: %q", got)
	}
	sessionProfile, ok := findMemoryFileBySelection(memoryContext, memorySelectionAgentSession)
	if !ok {
		t.Fatalf("expected %s in memory context: %+v", memorySelectionAgentSession, memoryContext.Files)
	}
	if !sessionProfile.Exists || sessionProfile.Writable {
		t.Fatalf("unexpected session profile flags: %+v", sessionProfile)
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

	memoryContext := decodeMemoryContextFromMetadata(t, processor.lastMetadata)
	file, ok := findMemoryFileBySelection(memoryContext, memorySelectionAgentsMD)
	if !ok {
		t.Fatalf("expected %s in memory context: %+v", memorySelectionAgentsMD, memoryContext.Files)
	}
	if file.Exists {
		t.Fatalf("expected missing agent AGENTS.md path, got %+v", file)
	}
	if got := file.Path; !strings.HasSuffix(got, "/.alter0/agents/writing-agent/AGENTS.md") {
		t.Fatalf("unexpected missing agent AGENTS path: %q", got)
	}
}

func TestExecuteNaturalLanguageAutoInjectsAgentSessionProfile(t *testing.T) {
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
		MessageID:   "m-session-profile",
		SessionID:   "session-profile",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "continue helping me",
		TraceID:     "t-session-profile",
		Metadata: map[string]string{
			execdomain.AgentIDMetadataKey:   "researcher",
			execdomain.AgentNameMetadataKey: "Research Agent",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}

	memoryContext := decodeMemoryContextFromMetadata(t, processor.lastMetadata)
	sessionProfile, ok := findMemoryFileBySelection(memoryContext, memorySelectionAgentSession)
	if !ok {
		t.Fatalf("expected %s in memory context: %+v", memorySelectionAgentSession, memoryContext.Files)
	}
	if !sessionProfile.Exists || sessionProfile.Writable {
		t.Fatalf("unexpected session profile flags: %+v", sessionProfile)
	}
	if got := sessionProfile.Path; !strings.HasSuffix(got, "/.alter0/agents/researcher/sessions/session-profile.md") {
		t.Fatalf("unexpected session profile path: %q", got)
	}
	for _, expected := range []string{
		"# Agent Session Profile",
		"- agent_id: researcher",
		"- agent_name: Research Agent",
		"- session_id: session-profile",
		"- channel_type: web",
		"## Notes",
	} {
		if !strings.Contains(sessionProfile.Content, expected) {
			t.Fatalf("expected session profile to contain %q, got %q", expected, sessionProfile.Content)
		}
	}
}

func TestExecuteNaturalLanguageAgentSessionProfileCapturesCodingRepositoryContext(t *testing.T) {
	root := t.TempDir()
	workspaceDir := filepath.Join(root, ".alter0", "workspaces", "sessions", "coding-session")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	repoDir := filepath.Join(workspaceDir, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo workspace: %v", err)
	}
	runGitCommand(t, repoDir, "init")
	runGitCommand(t, repoDir, "checkout", "-b", "feat/session-memory")
	runGitCommand(t, repoDir, "remote", "add", "origin", "https://example.com/demo/repo.git")

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
		MessageID:   "m-coding-session",
		SessionID:   "coding-session",
		ChannelID:   "web-default",
		ChannelType: shareddomain.ChannelTypeWeb,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     "continue the repository work",
		TraceID:     "t-coding-session",
		Metadata: map[string]string{
			execdomain.AgentIDMetadataKey:   "coding",
			execdomain.AgentNameMetadataKey: "Coding Agent",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteNaturalLanguage() error = %v", err)
	}

	memoryContext := decodeMemoryContextFromMetadata(t, processor.lastMetadata)
	sessionProfile, ok := findMemoryFileBySelection(memoryContext, memorySelectionAgentSession)
	if !ok {
		t.Fatalf("expected %s in memory context: %+v", memorySelectionAgentSession, memoryContext.Files)
	}
	for _, expected := range []string{
		"## Coding Context",
		"- local_repository_path: " + filepath.ToSlash(repoDir),
		"- source_repository_path: " + filepath.ToSlash(root),
		"- remote_repository: https://example.com/demo/repo.git",
		"- active_branch: feat/session-memory",
		"- session_workspace_path: " + filepath.ToSlash(workspaceDir),
		"- preview_url: https://",
	} {
		if !strings.Contains(sessionProfile.Content, expected) {
			t.Fatalf("expected coding session profile to contain %q, got %q", expected, sessionProfile.Content)
		}
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
