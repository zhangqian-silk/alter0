package application

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orchdomain "alter0/internal/orchestration/domain"
	shareddomain "alter0/internal/shared/domain"
	taskdomain "alter0/internal/task/domain"
	tasksummaryapp "alter0/internal/tasksummary/application"
)

type stubClassifier struct {
	intent orchdomain.Intent
}

func (s *stubClassifier) Classify(_ string) orchdomain.Intent {
	return s.intent
}

type stubRegistry struct {
	handlers map[string]orchdomain.CommandHandler
}

func (r *stubRegistry) Register(handler orchdomain.CommandHandler) error {
	if r.handlers == nil {
		r.handlers = map[string]orchdomain.CommandHandler{}
	}
	r.handlers[handler.Name()] = handler
	return nil
}

func (r *stubRegistry) Resolve(name string) (orchdomain.CommandHandler, bool) {
	if r.handlers == nil {
		return nil, false
	}
	handler, ok := r.handlers[name]
	return handler, ok
}

func (r *stubRegistry) Exists(name string) bool {
	_, ok := r.Resolve(name)
	return ok
}

func (r *stubRegistry) List() []string {
	return nil
}

type stubHandler struct {
	name   string
	output string
}

func (h *stubHandler) Name() string {
	return h.name
}

func (h *stubHandler) Aliases() []string {
	return nil
}

func (h *stubHandler) Execute(_ context.Context, _ orchdomain.CommandRequest) (orchdomain.CommandResult, error) {
	return orchdomain.CommandResult{Output: h.output}, nil
}

type stubExecutor struct {
	output      string
	outputs     []string
	meta        map[string]string
	called      int
	lastMessage shareddomain.UnifiedMessage
}

func (e *stubExecutor) ExecuteNaturalLanguage(_ context.Context, msg shareddomain.UnifiedMessage) (shareddomain.ExecutionResult, error) {
	e.called++
	e.lastMessage = msg
	output := e.output
	if len(e.outputs) > 0 {
		output = e.outputs[0]
		e.outputs = e.outputs[1:]
	}
	return shareddomain.ExecutionResult{
		Output:   output,
		Metadata: e.meta,
	}, nil
}

type spyTelemetry struct {
	routeCount map[string]int
	commandCnt map[string]int
	errorCount map[string]int
	memoryEvent map[string]int
}

func newSpyTelemetry() *spyTelemetry {
	return &spyTelemetry{
		routeCount: map[string]int{},
		commandCnt: map[string]int{},
		errorCount: map[string]int{},
		memoryEvent: map[string]int{},
	}
}

func (t *spyTelemetry) CountGateway(_ string) {}

func (t *spyTelemetry) CountRoute(route string) {
	t.routeCount[route]++
}

func (t *spyTelemetry) CountCommand(command string) {
	t.commandCnt[command]++
}

func (t *spyTelemetry) CountError(route string) {
	t.errorCount[route]++
}

func (t *spyTelemetry) ObserveDuration(_ string, _ time.Duration) {}

func (t *spyTelemetry) CountMemoryEvent(event string) {
	t.memoryEvent[event]++
}

func TestHandleCommand(t *testing.T) {
	registry := &stubRegistry{
		handlers: map[string]orchdomain.CommandHandler{
			"echo": &stubHandler{name: "echo", output: "ok"},
		},
	}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{output: "nl"}
	service := NewService(
		&stubClassifier{
			intent: orchdomain.Intent{
				Type:        orchdomain.IntentTypeCommand,
				CommandName: "echo",
			},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.Handle(context.Background(), validMessage("/echo hi"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Route != shareddomain.RouteCommand {
		t.Fatalf("expected command route, got %s", result.Route)
	}
	if result.Output != "ok" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if executor.called != 0 {
		t.Fatalf("executor should not be called for command route")
	}
}

func TestHandleNL(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{output: "nl response"}
	service := NewService(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.Handle(context.Background(), validMessage("hello"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Route != shareddomain.RouteNL {
		t.Fatalf("expected nl route, got %s", result.Route)
	}
	if result.Output != "nl response" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if result.Metadata != nil {
		t.Fatalf("expected nil metadata, got %+v", result.Metadata)
	}
	if executor.called != 1 {
		t.Fatalf("executor should be called exactly once, got %d", executor.called)
	}
}

func TestHandleNLPassesExecutionMetadata(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		output: "nl response",
		meta: map[string]string{
			"skills.injected_count": "1",
		},
	}
	service := NewService(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.Handle(context.Background(), validMessage("hello"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := result.Metadata["skills.injected_count"]; got != "1" {
		t.Fatalf("expected skills.injected_count metadata, got %q", got)
	}
}

func TestHandleUnknownCommand(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	service := NewService(
		&stubClassifier{
			intent: orchdomain.Intent{
				Type:        orchdomain.IntentTypeCommand,
				CommandName: "missing",
			},
		},
		registry,
		&stubExecutor{output: "nl"},
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.Handle(context.Background(), validMessage("/missing"))
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if result.ErrorCode != "command_not_found" {
		t.Fatalf("expected command_not_found, got %q", result.ErrorCode)
	}
}

func TestHandleNLInjectsSessionMemory(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		outputs: []string{
			"plan alpha: blue green rollout",
			"refined execution plan",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithSessionMemoryOptions(SessionMemoryOptions{
			MaxTurns:    4,
			TTL:         30 * time.Minute,
			MaxSnippets: 180,
		}),
	)

	firstMessage := validMessage("Give me a release plan for this sprint")
	firstMessage.SessionID = "session-memory"
	if _, err := service.Handle(context.Background(), firstMessage); err != nil {
		t.Fatalf("first message failed: %v", err)
	}

	secondMessage := validMessage("Please expand that plan with rollback details")
	secondMessage.SessionID = "session-memory"
	result, err := service.Handle(context.Background(), secondMessage)
	if err != nil {
		t.Fatalf("second message failed: %v", err)
	}

	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "Recent turns:") {
		t.Fatalf("expected recent turns in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "release plan for this sprint") {
		t.Fatalf("expected previous user message in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "plan alpha: blue green rollout") {
		t.Fatalf("expected previous assistant output in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Reference resolution:") {
		t.Fatalf("expected reference resolution section, got %q", prompt)
	}
	if result.Metadata["memory_reference_resolved"] != "true" {
		t.Fatalf("expected memory_reference_resolved=true, got %q", result.Metadata["memory_reference_resolved"])
	}
}

func TestHandleNLMemoryKeepsStableReferenceAcrossFollowUps(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		outputs: []string{
			"plan alpha: baseline rollout",
			"plan alpha: add canary validation",
			"plan alpha: include release calendar",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithSessionMemoryOptions(SessionMemoryOptions{
			MaxTurns:    3,
			TTL:         time.Hour,
			MaxSnippets: 180,
		}),
	)

	messages := []string{
		"Draft a release plan",
		"Refine that plan with verification gates",
		"For that plan, add owner and timeline",
	}
	for _, content := range messages {
		msg := validMessage(content)
		msg.SessionID = "session-stable"
		if _, err := service.Handle(context.Background(), msg); err != nil {
			t.Fatalf("handle message %q failed: %v", content, err)
		}
	}

	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "plan alpha: add canary validation") {
		t.Fatalf("expected latest plan context in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Reference resolution:") {
		t.Fatalf("expected reference resolution in prompt, got %q", prompt)
	}
}

func TestHandleNLInjectsLongTermMemoryAcrossSessions(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		outputs: []string{
			"Preference saved.",
			"Here is the plan.",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithSessionMemoryOptions(SessionMemoryOptions{
			MaxTurns:    4,
			TTL:         30 * time.Minute,
			MaxSnippets: 180,
		}),
		WithLongTermMemoryOptions(LongTermMemoryOptions{
			MaxEntriesPerScope: 64,
			MaxHits:            3,
			MaxSnippet:         180,
		}),
	)

	firstMessage := validMessage("remember my response style")
	firstMessage.SessionID = "history-session"
	firstMessage.UserID = "user-a"
	firstMessage.Metadata = map[string]string{
		longTermMemoryTenantMetadataKey:   "tenant-a",
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response-style",
		longTermMemoryValueMetadataKey:    "concise bullet answers",
		longTermMemoryTagsMetadataKey:     "style, concise",
	}
	if _, err := service.Handle(context.Background(), firstMessage); err != nil {
		t.Fatalf("first message failed: %v", err)
	}

	secondMessage := validMessage("Please keep concise style for this new session")
	secondMessage.SessionID = "new-session"
	secondMessage.UserID = "user-a"
	secondMessage.Metadata = map[string]string{
		longTermMemoryTenantMetadataKey: "tenant-a",
	}
	result, err := service.Handle(context.Background(), secondMessage)
	if err != nil {
		t.Fatalf("second message failed: %v", err)
	}

	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "[LONG TERM MEMORY]") {
		t.Fatalf("expected long-term memory section in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "concise bullet answers") {
		t.Fatalf("expected long-term preference in prompt, got %q", prompt)
	}
	if result.Metadata["memory_long_term_injected"] != "true" {
		t.Fatalf("expected memory_long_term_injected=true, got %q", result.Metadata["memory_long_term_injected"])
	}
}

func TestHandleNLLongTermMemoryPreventsCrossUserPollution(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		outputs: []string{
			"saved user a preference",
			"saved user b preference",
			"response for user a",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithLongTermMemoryOptions(LongTermMemoryOptions{
			MaxEntriesPerScope: 64,
			MaxHits:            3,
			MaxSnippet:         180,
		}),
	)

	userA := validMessage("store user a preference")
	userA.SessionID = "session-a1"
	userA.UserID = "user-a"
	userA.Metadata = map[string]string{
		longTermMemoryTenantMetadataKey:   "tenant-a",
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response-style",
		longTermMemoryValueMetadataKey:    "concise bullet answers",
		longTermMemoryTagsMetadataKey:     "style, concise",
	}
	if _, err := service.Handle(context.Background(), userA); err != nil {
		t.Fatalf("user a write failed: %v", err)
	}

	userB := validMessage("store user b preference")
	userB.SessionID = "session-b1"
	userB.UserID = "user-b"
	userB.Metadata = map[string]string{
		longTermMemoryTenantMetadataKey:   "tenant-a",
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response-style",
		longTermMemoryValueMetadataKey:    "verbose narrative",
		longTermMemoryTagsMetadataKey:     "style, verbose",
	}
	if _, err := service.Handle(context.Background(), userB); err != nil {
		t.Fatalf("user b write failed: %v", err)
	}

	query := validMessage("Please use concise style")
	query.SessionID = "session-a2"
	query.UserID = "user-a"
	query.Metadata = map[string]string{
		longTermMemoryTenantMetadataKey: "tenant-a",
	}
	if _, err := service.Handle(context.Background(), query); err != nil {
		t.Fatalf("user a query failed: %v", err)
	}

	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "concise bullet answers") {
		t.Fatalf("expected user a memory in prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "verbose narrative") {
		t.Fatalf("expected user b memory to be isolated, got %q", prompt)
	}
}

func TestHandleNLMandatoryContextOverridesConflictingLongTermMemory(t *testing.T) {
	dir := t.TempDir()
	contextPath := filepath.Join(dir, "SOUL.md")
	if err := os.WriteFile(contextPath, []byte("response_style: concise bullet answers\n"), 0o644); err != nil {
		t.Fatalf("write mandatory context file: %v", err)
	}

	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		outputs: []string{
			"preference stored",
			"response generated",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithLongTermMemoryOptions(LongTermMemoryOptions{
			MaxEntriesPerScope: 32,
			MaxHits:            3,
			MaxSnippet:         180,
		}),
		WithMandatoryContextOptions(MandatoryContextOptions{
			FilePath: contextPath,
		}),
	)

	writeMessage := validMessage("remember my response style")
	writeMessage.SessionID = "session-write"
	writeMessage.UserID = "user-r026"
	writeMessage.Metadata = map[string]string{
		longTermMemoryTenantMetadataKey:   "tenant-r026",
		longTermMemoryStrategyMetadataKey: "add",
		longTermMemoryKindMetadataKey:     "preference",
		longTermMemoryKeyMetadataKey:      "response-style",
		longTermMemoryValueMetadataKey:    "detailed narrative",
	}
	if _, err := service.Handle(context.Background(), writeMessage); err != nil {
		t.Fatalf("write message failed: %v", err)
	}

	queryMessage := validMessage("please keep response style")
	queryMessage.SessionID = "session-query"
	queryMessage.UserID = "user-r026"
	queryMessage.Metadata = map[string]string{
		longTermMemoryTenantMetadataKey: "tenant-r026",
	}
	result, err := service.Handle(context.Background(), queryMessage)
	if err != nil {
		t.Fatalf("query message failed: %v", err)
	}

	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "[MANDATORY CONTEXT]") {
		t.Fatalf("expected mandatory context section in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "response_style: concise bullet answers") {
		t.Fatalf("expected mandatory context content in prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "detailed narrative") {
		t.Fatalf("expected conflicting long-term memory removed from prompt, got %q", prompt)
	}
	if result.Metadata[mandatoryContextConflictCountMetadataKey] != "1" {
		t.Fatalf("expected conflict count metadata 1, got %q", result.Metadata[mandatoryContextConflictCountMetadataKey])
	}
	if result.Metadata[mandatoryContextInjectedMetadataKey] != "true" {
		t.Fatalf("expected mandatory context injected metadata, got %q", result.Metadata[mandatoryContextInjectedMetadataKey])
	}
	if result.Metadata[mandatoryContextVersionMetadataKey] == "" {
		t.Fatalf("expected mandatory context version metadata")
	}
}

func TestHandleNLMandatoryContextHotReloadAcrossSessions(t *testing.T) {
	dir := t.TempDir()
	contextPath := filepath.Join(dir, "SOUL.md")
	if err := os.WriteFile(contextPath, []byte("language: zh-cn\n"), 0o644); err != nil {
		t.Fatalf("write first mandatory context file: %v", err)
	}

	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		outputs: []string{
			"first response",
			"second response",
		},
	}
	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithMandatoryContextOptions(MandatoryContextOptions{
			FilePath: contextPath,
		}),
	)

	firstMessage := validMessage("hello from session one")
	firstMessage.SessionID = "session-1"
	firstResult, err := service.Handle(context.Background(), firstMessage)
	if err != nil {
		t.Fatalf("first message failed: %v", err)
	}
	if !strings.Contains(executor.lastMessage.Content, "language: zh-cn") {
		t.Fatalf("expected first prompt include initial mandatory context, got %q", executor.lastMessage.Content)
	}
	firstVersion := firstResult.Metadata[mandatoryContextVersionMetadataKey]
	if firstVersion == "" {
		t.Fatalf("expected first result has mandatory context version")
	}

	if err := os.WriteFile(contextPath, []byte("language: en-us\n"), 0o644); err != nil {
		t.Fatalf("write second mandatory context file: %v", err)
	}

	secondMessage := validMessage("hello from session two")
	secondMessage.SessionID = "session-2"
	secondResult, err := service.Handle(context.Background(), secondMessage)
	if err != nil {
		t.Fatalf("second message failed: %v", err)
	}
	if !strings.Contains(executor.lastMessage.Content, "language: en-us") {
		t.Fatalf("expected second prompt include reloaded context, got %q", executor.lastMessage.Content)
	}
	if strings.Contains(executor.lastMessage.Content, "language: zh-cn") {
		t.Fatalf("expected old mandatory context removed after reload, got %q", executor.lastMessage.Content)
	}
	if secondResult.Metadata[mandatoryContextVersionMetadataKey] == firstVersion {
		t.Fatalf("expected version change after hot reload")
	}
}

func TestHandleNLInjectsRecentTaskSummariesByDefault(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		output: "response",
	}
	taskMemory := tasksummaryapp.NewStore(tasksummaryapp.Options{
		RecentWindow: 5,
		DeepTopK:     5,
	})
	now := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	taskMemory.Record(buildTaskMemoryTask("task-r031-1", "session-task-1", taskdomain.TaskStatusSuccess, "release", "release finished", now.Add(-5*time.Minute)))
	taskMemory.Record(buildTaskMemoryTask("task-r031-2", "session-task-2", taskdomain.TaskStatusSuccess, "release", "release verified", now.Add(-4*time.Minute)))
	taskMemory.Record(buildTaskMemoryTask("task-r031-3", "session-task-3", taskdomain.TaskStatusFailed, "migration", "rollback executed", now.Add(-3*time.Minute)))
	taskMemory.Record(buildTaskMemoryTask("task-r031-4", "session-task-4", taskdomain.TaskStatusSuccess, "ops", "incident mitigated", now.Add(-2*time.Minute)))
	taskMemory.Record(buildTaskMemoryTask("task-r031-5", "session-task-5", taskdomain.TaskStatusSuccess, "ops", "postmortem generated", now.Add(-time.Minute)))
	taskMemory.Record(buildTaskMemoryTask("task-r031-6", "session-task-6", taskdomain.TaskStatusSuccess, "docs", "spec updated", now))

	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithTaskSummaryMemory(taskMemory),
	)
	msg := validMessage("继续当前任务，不需要历史任务")
	msg.SessionID = "session-live"
	msg.ReceivedAt = now
	result, err := service.Handle(context.Background(), msg)
	if err != nil {
		t.Fatalf("handle message failed: %v", err)
	}
	if got := result.Metadata["task_summary_injected_count"]; got != "5" {
		t.Fatalf("expected 5 injected summaries, got %q", got)
	}
	if got := result.Metadata["task_summary_retrieval_mode"]; got != "recent" {
		t.Fatalf("expected recent mode, got %q", got)
	}
	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "[TASK SUMMARY MEMORY]") {
		t.Fatalf("expected task summary memory prompt section, got %q", prompt)
	}
	if strings.Contains(prompt, "task-r031-1") {
		t.Fatalf("expected only latest 5 summaries injected, got %q", prompt)
	}
	if !strings.Contains(prompt, "task-r031-6") {
		t.Fatalf("expected latest summary injected, got %q", prompt)
	}
}

func TestHandleNLTriggersDeepRetrievalWithHistoryIntent(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		output: "response",
	}
	taskMemory := tasksummaryapp.NewStore(tasksummaryapp.Options{
		RecentWindow: 5,
		DeepTopK:     5,
	})
	now := time.Date(2026, 3, 4, 11, 0, 0, 0, time.UTC)
	taskMemory.Record(buildTaskMemoryTask("task-recent", "session-new", taskdomain.TaskStatusSuccess, "release", "recent release", now.Add(-time.Hour)))
	legacy := buildTaskMemoryTask("task-legacy", "session-old", taskdomain.TaskStatusFailed, "migration", "legacy migration failed", now.Add(-7*24*time.Hour))
	legacy.Logs = append(legacy.Logs,
		taskdomain.TaskLog{Timestamp: now.Add(-7*24*time.Hour + time.Minute), Level: taskdomain.TaskLogLevelError, Message: "schema mismatch"},
		taskdomain.TaskLog{Timestamp: now.Add(-7*24*time.Hour + 2*time.Minute), Level: taskdomain.TaskLogLevelWarn, Message: "rollback executed"},
	)
	legacy.Artifacts = append(legacy.Artifacts, taskdomain.TaskArtifact{
		ArtifactID:  "artifact-legacy",
		Name:        "migration.log",
		ContentType: "text/plain",
		Content:     "legacy stacktrace",
		CreatedAt:   now.Add(-7*24*time.Hour + 3*time.Minute),
	})
	taskMemory.Record(legacy)

	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithTaskSummaryMemory(taskMemory),
	)
	msg := validMessage("回顾上周 migration 任务，给我 task-legacy 的日志细节")
	msg.SessionID = "session-query"
	msg.ReceivedAt = now
	result, err := service.Handle(context.Background(), msg)
	if err != nil {
		t.Fatalf("handle message failed: %v", err)
	}
	if got := result.Metadata["task_summary_retrieval_mode"]; got != "deep" {
		t.Fatalf("expected deep mode, got %q", got)
	}
	if got := result.Metadata["deep_retrieval_triggered"]; got != "true" {
		t.Fatalf("expected deep retrieval triggered metadata, got %q", got)
	}
	if got := telemetry.memoryEvent["deep_retrieval_triggered"]; got != 1 {
		t.Fatalf("expected deep_retrieval_triggered telemetry count 1, got %d", got)
	}
	prompt := executor.lastMessage.Content
	if !strings.Contains(prompt, "task-legacy") {
		t.Fatalf("expected deep retrieval hit in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "logs:") || !strings.Contains(prompt, "artifacts:") {
		t.Fatalf("expected detail drill-down logs and artifacts in prompt, got %q", prompt)
	}
}

func TestHandleNLOverridesDeepRetrievalWithNegation(t *testing.T) {
	registry := &stubRegistry{}
	telemetry := newSpyTelemetry()
	executor := &stubExecutor{
		output: "response",
	}
	taskMemory := tasksummaryapp.NewStore(tasksummaryapp.Options{
		RecentWindow: 5,
		DeepTopK:     5,
	})
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	taskMemory.Record(buildTaskMemoryTask("task-now", "session-now", taskdomain.TaskStatusSuccess, "release", "release done", now.Add(-time.Hour)))

	service := NewServiceWithOptions(
		&stubClassifier{
			intent: orchdomain.Intent{Type: orchdomain.IntentTypeNL},
		},
		registry,
		executor,
		telemetry,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithTaskSummaryMemory(taskMemory),
	)
	msg := validMessage("不要查历史任务，只看当前任务进度")
	msg.SessionID = "session-query"
	msg.ReceivedAt = now
	result, err := service.Handle(context.Background(), msg)
	if err != nil {
		t.Fatalf("handle message failed: %v", err)
	}
	if got := result.Metadata["task_summary_retrieval_mode"]; got != "recent" {
		t.Fatalf("expected recent mode when negated, got %q", got)
	}
	if got := result.Metadata["deep_retrieval_overridden"]; got != "true" {
		t.Fatalf("expected deep retrieval overridden metadata, got %q", got)
	}
	if got := result.Metadata["deep_retrieval_triggered"]; got != "false" {
		t.Fatalf("expected deep retrieval not triggered, got %q", got)
	}
	if got := telemetry.memoryEvent["deep_retrieval_overridden"]; got != 1 {
		t.Fatalf("expected deep_retrieval_overridden telemetry count 1, got %d", got)
	}
}

func buildTaskMemoryTask(taskID string, sessionID string, status taskdomain.TaskStatus, taskType string, result string, finishedAt time.Time) taskdomain.Task {
	return taskdomain.Task{
		ID:             taskID,
		SessionID:      sessionID,
		MessageID:      "msg-" + taskID,
		Status:         status,
		Progress:       100,
		RetryCount:     0,
		MaxRetries:     1,
		TimeoutMS:      90000,
		CreatedAt:      finishedAt.Add(-time.Minute),
		UpdatedAt:      finishedAt,
		FinishedAt:     finishedAt,
		RequestContent: "handle " + taskID,
		RequestMetadata: map[string]string{
			"alter0.task.type": taskType,
		},
		Summary: "summary " + taskID,
		TaskSummary: taskdomain.TaskSummary{
			TaskID:     taskID,
			TaskType:   taskType,
			Goal:       "handle " + taskID,
			Result:     result,
			Status:     status,
			FinishedAt: finishedAt,
			Tags:       []string{"task", string(status), taskType},
		},
	}
}

func validMessage(content string) shareddomain.UnifiedMessage {
	return shareddomain.UnifiedMessage{
		MessageID:   "m1",
		SessionID:   "s1",
		ChannelID:   "cli-default",
		ChannelType: shareddomain.ChannelTypeCLI,
		TriggerType: shareddomain.TriggerTypeUser,
		Content:     content,
		TraceID:     "t1",
		ReceivedAt:  time.Now().UTC(),
	}
}
