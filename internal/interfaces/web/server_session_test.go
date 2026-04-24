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
