package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	agentapp "alter0/internal/agent/application"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
	shareddomain "alter0/internal/shared/domain"
	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
)

type stubSessionHistory struct {
	sessionPage      sessionapp.SessionPage
	messagePage      sessionapp.MessagePage
	lastSessionQuery sessionapp.SessionQuery
	lastMessageQuery sessionapp.MessageQuery
	deleteErr        error
	lastDeletedID    string
}

func (s *stubSessionHistory) ListSessions(query sessionapp.SessionQuery) sessionapp.SessionPage {
	s.lastSessionQuery = query
	return s.sessionPage
}

func (s *stubSessionHistory) ListMessages(query sessionapp.MessageQuery) sessionapp.MessagePage {
	s.lastMessageQuery = query
	return s.messagePage
}

func (s *stubSessionHistory) DeleteSession(sessionID string) error {
	s.lastDeletedID = sessionID
	return s.deleteErr
}

type stubSessionTaskService struct {
	lastDeletedSessionID string
	deleteErr            error
}

func (s *stubSessionTaskService) AssessComplexity(shareddomain.UnifiedMessage) taskapp.ComplexityAssessment {
	return taskapp.ComplexityAssessment{}
}

func (s *stubSessionTaskService) AssessComplexityWithContext(context.Context, shareddomain.UnifiedMessage) taskapp.ComplexityAssessment {
	return taskapp.ComplexityAssessment{}
}

func (s *stubSessionTaskService) ShouldRunAsync(shareddomain.UnifiedMessage) bool {
	return false
}

func (s *stubSessionTaskService) Submit(shareddomain.UnifiedMessage) (taskdomain.Task, error) {
	return taskdomain.Task{}, nil
}

func (s *stubSessionTaskService) List(taskapp.ListQuery) taskapp.TaskPage {
	return taskapp.TaskPage{}
}

func (s *stubSessionTaskService) Get(string) (taskdomain.Task, bool) {
	return taskdomain.Task{}, false
}

func (s *stubSessionTaskService) ListBySession(string) []taskdomain.Task {
	return []taskdomain.Task{}
}

func (s *stubSessionTaskService) ListLogs(string, int, int) (taskapp.TaskLogPage, error) {
	return taskapp.TaskLogPage{}, nil
}

func (s *stubSessionTaskService) ListArtifacts(string) ([]taskdomain.TaskArtifact, error) {
	return []taskdomain.TaskArtifact{}, nil
}

func (s *stubSessionTaskService) ReadArtifact(context.Context, string, string) (taskdomain.TaskArtifact, []byte, error) {
	return taskdomain.TaskArtifact{}, nil, nil
}

func (s *stubSessionTaskService) Cancel(string) (taskdomain.Task, error) {
	return taskdomain.Task{}, nil
}

func (s *stubSessionTaskService) Retry(string) (taskdomain.Task, error) {
	return taskdomain.Task{}, nil
}

func (s *stubSessionTaskService) DeleteBySession(sessionID string) error {
	s.lastDeletedSessionID = sessionID
	return s.deleteErr
}

func TestSessionListHandlerReturnsPagedData(t *testing.T) {
	history := &stubSessionHistory{
		sessionPage: sessionapp.SessionPage{
			Items: []sessiondomain.SessionSummary{
				{
					SessionID:    "s-1",
					MessageCount: 2,
				},
			},
			Pagination: sessionapp.Pagination{
				Page:     2,
				PageSize: 10,
				Total:    21,
				HasNext:  true,
			},
		},
	}
	server := &Server{
		sessions: history,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions?page=2&page_size=10&trigger_type=cron&channel_type=scheduler&channel_id=scheduler-default&message_id=msg-1&agent_id=researcher&job_id=job-daily",
		nil,
	)
	rec := httptest.NewRecorder()
	server.sessionListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if history.lastSessionQuery.Page != 2 || history.lastSessionQuery.PageSize != 10 {
		t.Fatalf("unexpected query %+v", history.lastSessionQuery)
	}
	if history.lastSessionQuery.TriggerType != shareddomain.TriggerTypeCron {
		t.Fatalf("expected trigger_type cron, got %s", history.lastSessionQuery.TriggerType)
	}
	if history.lastSessionQuery.ChannelType != shareddomain.ChannelTypeScheduler {
		t.Fatalf("expected channel_type scheduler, got %s", history.lastSessionQuery.ChannelType)
	}
	if history.lastSessionQuery.ChannelID != "scheduler-default" {
		t.Fatalf("expected channel_id scheduler-default, got %s", history.lastSessionQuery.ChannelID)
	}
	if history.lastSessionQuery.MessageID != "msg-1" {
		t.Fatalf("expected message_id msg-1, got %s", history.lastSessionQuery.MessageID)
	}
	if history.lastSessionQuery.AgentID != "researcher" {
		t.Fatalf("expected agent_id researcher, got %s", history.lastSessionQuery.AgentID)
	}
	if history.lastSessionQuery.JobID != "job-daily" {
		t.Fatalf("expected job_id job-daily, got %s", history.lastSessionQuery.JobID)
	}

	var body sessionapp.SessionPage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body failed: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].SessionID != "s-1" {
		t.Fatalf("unexpected body %+v", body)
	}
}

func TestSessionMessageListHandlerSupportsTimeRange(t *testing.T) {
	history := &stubSessionHistory{
		messagePage: sessionapp.MessagePage{
			Items: []sessiondomain.MessageRecord{
				{
					MessageID: "m-1",
					SessionID: "s-1",
					Role:      sessiondomain.MessageRoleAssistant,
					Content:   "answer",
					Timestamp: time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
					RouteResult: sessiondomain.RouteResult{
						Route: shareddomain.RouteNL,
					},
				},
			},
			Pagination: sessionapp.Pagination{
				Page:     1,
				PageSize: 20,
				Total:    1,
				HasNext:  false,
			},
		},
	}
	server := &Server{
		sessions: history,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions/s-1/messages?page=1&page_size=20&start_at=2026-03-03T00:00:00Z&end_at=2026-03-03T23:59:59Z",
		nil,
	)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if history.lastMessageQuery.SessionID != "s-1" {
		t.Fatalf("expected session id s-1, got %q", history.lastMessageQuery.SessionID)
	}
	if history.lastMessageQuery.StartAt.IsZero() || history.lastMessageQuery.EndAt.IsZero() {
		t.Fatalf("expected non-zero time range, got %+v", history.lastMessageQuery)
	}
}

func TestSessionHandlersValidateInputs(t *testing.T) {
	history := &stubSessionHistory{}
	server := &Server{
		sessions: history,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	invalidPageReq := httptest.NewRequest(http.MethodGet, "/api/sessions?page=0", nil)
	invalidPageRec := httptest.NewRecorder()
	server.sessionListHandler(invalidPageRec, invalidPageReq)
	if invalidPageRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidPageRec.Code)
	}

	invalidPathReq := httptest.NewRequest(http.MethodGet, "/api/sessions/s-1/records", nil)
	invalidPathRec := httptest.NewRecorder()
	server.sessionMessageListHandler(invalidPathRec, invalidPathReq)
	if invalidPathRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidPathRec.Code)
	}

	invalidTriggerReq := httptest.NewRequest(http.MethodGet, "/api/sessions?trigger_type=timer", nil)
	invalidTriggerRec := httptest.NewRecorder()
	server.sessionListHandler(invalidTriggerRec, invalidTriggerReq)
	if invalidTriggerRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidTriggerRec.Code)
	}

	invalidChannelReq := httptest.NewRequest(http.MethodGet, "/api/sessions?channel_type=mobile", nil)
	invalidChannelRec := httptest.NewRecorder()
	server.sessionListHandler(invalidChannelRec, invalidChannelReq)
	if invalidChannelRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidChannelRec.Code)
	}
}

func TestSessionDeleteHandlerRemovesHistoryTasksAndWorkspace(t *testing.T) {
	baseDir := t.TempDir()
	workspaceDir := filepath.Join(baseDir, ".alter0", "workspaces", "sessions", "session-delete")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "artifact.txt"), []byte("payload"), 0o644); err != nil {
		t.Fatalf("write workspace file: %v", err)
	}

	history := &stubSessionHistory{}
	tasks := &stubSessionTaskService{}
	server := &Server{
		sessions:      history,
		tasks:         tasks,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: baseDir,
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/session-delete", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if history.lastDeletedID != "session-delete" {
		t.Fatalf("expected history delete for session-delete, got %q", history.lastDeletedID)
	}
	if tasks.lastDeletedSessionID != "session-delete" {
		t.Fatalf("expected task delete for session-delete, got %q", tasks.lastDeletedSessionID)
	}
	if _, err := os.Stat(workspaceDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected workspace removed, got %v", err)
	}
}

func TestConversationRuntimeSessionCollectionHandlerFiltersByRoute(t *testing.T) {
	service := sessionapp.NewService()
	base := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	if err := service.Append(
		sessiondomain.MessageRecord{
			MessageID: "chat-user",
			SessionID: "chat-session",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "Inspect this repository",
			Timestamp: base,
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			Metadata: map[string]string{
				"alter0.llm.provider_id":    "openai",
				"alter0.llm.model":          "gpt-5.4",
				"alter0.agent.tools":        `["memory"]`,
				"alter0.skills.include":     `["frontend-design"]`,
				"alter0.mcp.request.enable": `["filesystem"]`,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: "chat-assistant",
			SessionID: "chat-session",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "Repository loaded.",
			Timestamp: base.Add(time.Minute),
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			RouteResult: sessiondomain.RouteResult{
				Route: shareddomain.RouteNL,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: "agent-user",
			SessionID: "agent-session",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "Ship the bug fix",
			Timestamp: base.Add(2 * time.Minute),
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
				AgentID:     "coding",
				AgentName:   "Coding Agent",
			},
			Metadata: map[string]string{
				"alter0.execution.engine": "codex",
				"alter0.skills.include":   `["deploy-test-service"]`,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: "agent-assistant",
			SessionID: "agent-session",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "Patch applied.",
			Timestamp: base.Add(3 * time.Minute),
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
				AgentID:     "coding",
				AgentName:   "Coding Agent",
			},
			RouteResult: sessiondomain.RouteResult{
				Route: shareddomain.RouteCommand,
			},
		},
	); err != nil {
		t.Fatalf("append records: %v", err)
	}

	server := &Server{
		sessions: service,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/conversation-runtime/sessions?route=chat", nil)
	server.conversationRuntimeSessionCollectionHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload struct {
		Items []struct {
			ID              string   `json:"id"`
			Title           string   `json:"title"`
			TargetType      string   `json:"target_type"`
			TargetID        string   `json:"target_id"`
			ModelProviderID string   `json:"model_provider_id"`
			ModelID         string   `json:"model_id"`
			ToolIDs         []string `json:"tool_ids"`
			SkillIDs        []string `json:"skill_ids"`
			MCPIDs          []string `json:"mcp_ids"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 chat runtime session, got %d", len(payload.Items))
	}
	if payload.Items[0].ID != "chat-session" || payload.Items[0].TargetType != "model" {
		t.Fatalf("unexpected chat item %+v", payload.Items[0])
	}
	if payload.Items[0].Title != "Inspect this repository" {
		t.Fatalf("expected title from first user message, got %+v", payload.Items[0])
	}
	if payload.Items[0].ModelProviderID != "openai" || payload.Items[0].ModelID != "gpt-5.4" {
		t.Fatalf("expected provider/model from metadata, got %+v", payload.Items[0])
	}
	if len(payload.Items[0].ToolIDs) != 1 || payload.Items[0].ToolIDs[0] != "memory" {
		t.Fatalf("expected tool ids from metadata, got %+v", payload.Items[0].ToolIDs)
	}
	if len(payload.Items[0].SkillIDs) != 1 || payload.Items[0].SkillIDs[0] != "frontend-design" {
		t.Fatalf("expected skill ids from metadata, got %+v", payload.Items[0].SkillIDs)
	}
	if len(payload.Items[0].MCPIDs) != 1 || payload.Items[0].MCPIDs[0] != "filesystem" {
		t.Fatalf("expected mcp ids from metadata, got %+v", payload.Items[0].MCPIDs)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/conversation-runtime/sessions?route=agent-runtime", nil)
	server.conversationRuntimeSessionCollectionHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 agent runtime session, got %d", len(payload.Items))
	}
	if payload.Items[0].ID != "agent-session" || payload.Items[0].TargetType != "agent" || payload.Items[0].TargetID != "coding" {
		t.Fatalf("unexpected agent item %+v", payload.Items[0])
	}
	if payload.Items[0].ModelProviderID != "alter0-codex" || payload.Items[0].ModelID != "codex" {
		t.Fatalf("expected codex runtime model selection, got %+v", payload.Items[0])
	}
}

func TestConversationRuntimeSessionItemHandlerReturnsMessagesAndAttachments(t *testing.T) {
	service := sessionapp.NewService()
	base := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	rawAttachments, err := execdomain.EncodeUserAttachments([]execdomain.UserAttachment{
		{
			ID:          "asset-1",
			Kind:        execdomain.UserAttachmentKindImage,
			Name:        "diagram.png",
			ContentType: "image/png",
			AssetURL:    "/api/sessions/chat-session/attachments/asset-1/original",
			PreviewURL:  "/api/sessions/chat-session/attachments/asset-1/preview",
		},
	})
	if err != nil {
		t.Fatalf("encode attachments: %v", err)
	}
	if err := service.Append(
		sessiondomain.MessageRecord{
			MessageID: "chat-user",
			SessionID: "chat-session",
			Role:      sessiondomain.MessageRoleUser,
			Content:   "Inspect this repository",
			Timestamp: base,
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			Metadata: map[string]string{
				execdomain.UserAttachmentsMetadataKey: rawAttachments,
			},
		},
		sessiondomain.MessageRecord{
			MessageID: "chat-assistant",
			SessionID: "chat-session",
			Role:      sessiondomain.MessageRoleAssistant,
			Content:   "Repository loaded.",
			Timestamp: base.Add(time.Minute),
			Source: sessiondomain.MessageSource{
				TriggerType: shareddomain.TriggerTypeUser,
				ChannelType: shareddomain.ChannelTypeWeb,
				ChannelID:   "web-default",
			},
			RouteResult: sessiondomain.RouteResult{
				Route:     shareddomain.RouteNL,
				ErrorCode: "",
				ProcessSteps: []shareddomain.ProcessStep{
					{
						ID:     "step-1",
						Kind:   "action",
						Title:  "codex_exec",
						Detail: "Checked git status",
						Status: "completed",
					},
				},
			},
		},
	); err != nil {
		t.Fatalf("append records: %v", err)
	}

	server := &Server{
		sessions: service,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/conversation-runtime/sessions/chat-session?route=chat", nil)
	server.conversationRuntimeSessionItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload struct {
		Session struct {
			ID       string `json:"id"`
			Messages []struct {
				ID           string `json:"id"`
				Role         string `json:"role"`
				Text         string `json:"text"`
				Status       string `json:"status"`
				Error        bool   `json:"error"`
				ProcessSteps []struct {
					Title string `json:"title"`
				} `json:"process_steps"`
				Attachments []struct {
					ID         string `json:"id"`
					AssetURL   string `json:"asset_url"`
					PreviewURL string `json:"preview_url"`
				} `json:"attachments"`
			} `json:"messages"`
		} `json:"session"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Session.ID != "chat-session" {
		t.Fatalf("unexpected session id %q", payload.Session.ID)
	}
	if len(payload.Session.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(payload.Session.Messages))
	}
	if payload.Session.Messages[0].Attachments[0].PreviewURL != "/api/sessions/chat-session/attachments/asset-1/preview" {
		t.Fatalf("expected user attachment preview url, got %+v", payload.Session.Messages[0].Attachments)
	}
	if payload.Session.Messages[1].Status != "done" || payload.Session.Messages[1].Error {
		t.Fatalf("expected assistant message to restore as done, got %+v", payload.Session.Messages[1])
	}
	if len(payload.Session.Messages[1].ProcessSteps) != 1 || payload.Session.Messages[1].ProcessSteps[0].Title != "codex_exec" {
		t.Fatalf("expected process steps, got %+v", payload.Session.Messages[1].ProcessSteps)
	}
}

func TestSessionDeleteHandlerAllowsMissingHistory(t *testing.T) {
	history := &stubSessionHistory{deleteErr: sessionapp.ErrSessionNotFound}
	tasks := &stubSessionTaskService{}
	server := &Server{
		sessions:      history,
		tasks:         tasks,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: t.TempDir(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/session-missing", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSessionDeleteHandlerReturnsTaskDeleteFailure(t *testing.T) {
	history := &stubSessionHistory{}
	tasks := &stubSessionTaskService{deleteErr: errors.New("task delete failed")}
	server := &Server{
		sessions:      history,
		tasks:         tasks,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		workspaceRoot: t.TempDir(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/session-delete", nil)
	rec := httptest.NewRecorder()
	server.sessionMessageListHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestAgentSessionProfileHandlerReturnsSchemaAndAttributes(t *testing.T) {
	baseDir := t.TempDir()
	profilePath := filepath.Join(baseDir, ".alter0", "agents", "coding", "sessions", "session-coding.md")
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		t.Fatalf("prepare session profile dir: %v", err)
	}
	content := `# Agent Session Profile

<!-- alter0:agent-session:auto:start -->
## Session Identity
- agent_id: coding
- session_id: session-coding

## Instance Attributes
- repository_path: /workspace/alter0-remote
- branch: feature/session-profile
- preview_subdomain: coding-run-42
<!-- alter0:agent-session:auto:end -->

## Notes
<!-- alter0:agent-session:notes:start -->
<!-- alter0:agent-session:notes:end -->
`
	if err := os.WriteFile(profilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write session profile: %v", err)
	}

	control := controlapp.NewService()
	server := &Server{
		control:       control,
		agents:        agentapp.NewCatalog(control),
		workspaceRoot: baseDir,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/session-profile?agent_id=coding&session_id=session-coding", nil)
	rec := httptest.NewRecorder()
	server.agentSessionProfileHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		AgentID    string                                   `json:"agent_id"`
		SessionID  string                                   `json:"session_id"`
		Path       string                                   `json:"path"`
		Exists     bool                                     `json:"exists"`
		Fields     []controldomain.AgentSessionProfileField `json:"fields"`
		Attributes map[string]string                        `json:"attributes"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if response.AgentID != "coding" || response.SessionID != "session-coding" || !response.Exists {
		t.Fatalf("unexpected response envelope: %+v", response)
	}
	if len(response.Fields) < 4 {
		t.Fatalf("expected builtin coding fields, got %+v", response.Fields)
	}
	if response.Attributes["repository_path"] != "/workspace/alter0-remote" {
		t.Fatalf("expected repository_path attribute, got %+v", response.Attributes)
	}
	if response.Attributes["preview_subdomain"] != "coding-run-42" {
		t.Fatalf("expected preview_subdomain attribute, got %+v", response.Attributes)
	}
}
