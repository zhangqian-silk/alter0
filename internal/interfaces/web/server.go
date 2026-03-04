package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	schedulerapp "alter0/internal/scheduler/application"
	schedulerdomain "alter0/internal/scheduler/domain"
	sessionapp "alter0/internal/session/application"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
)

//go:embed static/*
var webStaticFS embed.FS

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type Server struct {
	addr         string
	orchestrator Orchestrator
	telemetry    *observability.Telemetry
	idGenerator  sharedapp.IDGenerator
	control      *controlapp.Service
	scheduler    *schedulerapp.Manager
	sessions     sessionHistoryService
	tasks        taskService
	memory       *agentMemoryService
	logger       *slog.Logger
}

type sessionHistoryService interface {
	ListSessions(query sessionapp.SessionQuery) sessionapp.SessionPage
	ListMessages(query sessionapp.MessageQuery) sessionapp.MessagePage
}

type taskService interface {
	ShouldRunAsync(msg shareddomain.UnifiedMessage) bool
	Submit(msg shareddomain.UnifiedMessage) (taskdomain.Task, error)
	List(query taskapp.ListQuery) taskapp.TaskPage
	Get(taskID string) (taskdomain.Task, bool)
	ListBySession(sessionID string) []taskdomain.Task
	ListLogs(taskID string, cursor int, limit int) (taskapp.TaskLogPage, error)
	ListArtifacts(taskID string) ([]taskdomain.TaskArtifact, error)
	Cancel(taskID string) (taskdomain.Task, error)
	Retry(taskID string) (taskdomain.Task, error)
}

type messageRequest struct {
	SessionID     string            `json:"session_id"`
	UserID        string            `json:"user_id,omitempty"`
	ChannelID     string            `json:"channel_id,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Content       string            `json:"content"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type taskCreateRequest struct {
	SessionID       string            `json:"session_id"`
	SourceMessageID string            `json:"source_message_id,omitempty"`
	TaskType        string            `json:"task_type,omitempty"`
	Input           string            `json:"input"`
	IdempotencyKey  string            `json:"idempotency_key,omitempty"`
	AsyncHint       string            `json:"async_hint,omitempty"`
	UserID          string            `json:"user_id,omitempty"`
	ChannelID       string            `json:"channel_id,omitempty"`
	CorrelationID   string            `json:"correlation_id,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type messageResponse struct {
	Result     shareddomain.OrchestrationResult `json:"result"`
	TaskID     string                           `json:"task_id,omitempty"`
	TaskStatus string                           `json:"task_status,omitempty"`
	Error      string                           `json:"error,omitempty"`
}

type taskCreateResponse struct {
	TaskID          string `json:"task_id"`
	Status          string `json:"status"`
	AcceptedAt      string `json:"accepted_at"`
	EstimatedWaitMS int64  `json:"estimated_wait_ms"`
}

type streamStartResponse struct {
	MessageID string `json:"message_id"`
	SessionID string `json:"session_id"`
	ChannelID string `json:"channel_id"`
	TraceID   string `json:"trace_id"`
}

type streamDeltaResponse struct {
	Delta string             `json:"delta"`
	Route shareddomain.Route `json:"route,omitempty"`
}

type streamDoneResponse struct {
	Result     shareddomain.OrchestrationResult `json:"result"`
	TaskID     string                           `json:"task_id,omitempty"`
	TaskStatus string                           `json:"task_status,omitempty"`
}

type streamErrorResponse struct {
	Error  string                           `json:"error"`
	Result shareddomain.OrchestrationResult `json:"result,omitempty"`
}

type taskLogStreamStartResponse struct {
	TaskID string `json:"task_id"`
	Cursor int    `json:"cursor"`
}

type taskLogStreamEvent struct {
	TaskID     string             `json:"task_id"`
	Cursor     int                `json:"cursor"`
	NextCursor int                `json:"next_cursor"`
	Log        taskdomain.TaskLog `json:"log"`
}

type taskLogStreamDoneResponse struct {
	TaskID     string                `json:"task_id"`
	Status     taskdomain.TaskStatus `json:"status"`
	NextCursor int                   `json:"next_cursor"`
}

type taskLogStreamErrorResponse struct {
	Error      string `json:"error"`
	NextCursor int    `json:"next_cursor"`
}

type channelUpsertRequest struct {
	Type        string            `json:"type"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type skillUpsertRequest struct {
	Name     string            `json:"name"`
	Type     string            `json:"type,omitempty"`
	Enabled  *bool             `json:"enabled,omitempty"`
	Scope    string            `json:"scope,omitempty"`
	Version  string            `json:"version,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type capabilityLifecycleRequest struct {
	Action string `json:"action"`
}

type cronJobUpsertRequest struct {
	Name      string            `json:"name,omitempty"`
	Interval  string            `json:"interval"`
	Enabled   *bool             `json:"enabled,omitempty"`
	SessionID string            `json:"session_id"`
	UserID    string            `json:"user_id,omitempty"`
	ChannelID string            `json:"channel_id,omitempty"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type cronJobResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Interval  string            `json:"interval"`
	Enabled   bool              `json:"enabled"`
	SessionID string            `json:"session_id"`
	UserID    string            `json:"user_id,omitempty"`
	ChannelID string            `json:"channel_id,omitempty"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type memoryTaskSummaryItem struct {
	TaskID     string                `json:"task_id"`
	TaskType   string                `json:"task_type"`
	Goal       string                `json:"goal"`
	Result     string                `json:"result"`
	Status     taskdomain.TaskStatus `json:"status"`
	FinishedAt time.Time             `json:"finished_at"`
	Tags       []string              `json:"tags,omitempty"`
}

type memoryTaskMeta struct {
	TaskID          string                `json:"task_id"`
	SessionID       string                `json:"session_id"`
	SourceMessageID string                `json:"source_message_id"`
	Status          taskdomain.TaskStatus `json:"status"`
	Progress        int                   `json:"progress"`
	CreatedAt       time.Time             `json:"created_at"`
	FinishedAt      time.Time             `json:"finished_at,omitempty"`
	RetryCount      int                   `json:"retry_count"`
	TaskType        string                `json:"task_type"`
}

type memoryTaskListQuery struct {
	Status   taskdomain.TaskStatus
	TaskType string
	StartAt  time.Time
	EndAt    time.Time
	Page     int
	PageSize int
}

type taskControlActionState struct {
	AllowedStatuses []taskdomain.TaskStatus `json:"allowed_statuses"`
	Enabled         bool                    `json:"enabled"`
	Reason          string                  `json:"reason,omitempty"`
}

type taskControlActions struct {
	Retry  taskControlActionState `json:"retry"`
	Cancel taskControlActionState `json:"cancel"`
}

type taskSessionLink struct {
	TaskID              string `json:"task_id"`
	SessionID           string `json:"session_id"`
	RequestMessageID    string `json:"request_message_id,omitempty"`
	ResultMessageID     string `json:"result_message_id,omitempty"`
	TaskDetailPath      string `json:"task_detail_path"`
	SessionTasksPath    string `json:"session_tasks_path,omitempty"`
	SessionMessagesPath string `json:"session_messages_path,omitempty"`
}

type taskControlView struct {
	Task    taskdomain.Task    `json:"task"`
	Actions taskControlActions `json:"actions"`
	Link    taskSessionLink    `json:"link"`
}

func NewServer(
	addr string,
	orchestrator Orchestrator,
	telemetry *observability.Telemetry,
	idGenerator sharedapp.IDGenerator,
	control *controlapp.Service,
	scheduler *schedulerapp.Manager,
	sessions sessionHistoryService,
	tasks taskService,
	memoryOptions AgentMemoryOptions,
	logger *slog.Logger,
) *Server {
	return &Server{
		addr:         addr,
		orchestrator: orchestrator,
		telemetry:    telemetry,
		idGenerator:  idGenerator,
		control:      control,
		scheduler:    scheduler,
		sessions:     sessions,
		tasks:        tasks,
		memory:       newAgentMemoryService(memoryOptions),
		logger:       logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", s.telemetry.MetricsHandler())
	mux.HandleFunc("/healthz", s.healthHandler)
	mux.HandleFunc("/readyz", s.readyHandler)
	mux.HandleFunc("/", s.rootHandler)
	mux.HandleFunc("/chat", s.chatPageHandler)
	mux.HandleFunc("/api/messages", s.messageHandler)
	mux.HandleFunc("/api/messages/stream", s.messageStreamHandler)
	mux.HandleFunc("/api/sessions", s.sessionListHandler)
	mux.HandleFunc("/api/sessions/", s.sessionMessageListHandler)
	mux.HandleFunc("/api/tasks", s.taskCollectionHandler)
	mux.HandleFunc("/api/tasks/", s.taskItemHandler)
	mux.HandleFunc("/api/agent/memory", s.agentMemoryHandler)
	mux.HandleFunc("/api/memory/tasks", s.memoryTaskCollectionHandler)
	mux.HandleFunc("/api/memory/tasks/", s.memoryTaskItemHandler)
	mux.HandleFunc("/api/control/tasks/", s.controlTaskItemHandler)
	mux.HandleFunc("/api/control/channels", s.channelListHandler)
	mux.HandleFunc("/api/control/channels/", s.channelItemHandler)
	mux.HandleFunc("/api/control/capabilities", s.capabilityListHandler)
	mux.HandleFunc("/api/control/capabilities/audit", s.capabilityAuditListHandler)
	mux.HandleFunc("/api/control/capabilities/", s.capabilityItemHandler)
	mux.HandleFunc("/api/control/skills", s.skillListHandler)
	mux.HandleFunc("/api/control/skills/", s.skillItemHandler)
	mux.HandleFunc("/api/control/mcps", s.mcpListHandler)
	mux.HandleFunc("/api/control/mcps/", s.mcpItemHandler)
	mux.HandleFunc("/api/control/cron/jobs", s.cronJobListHandler)
	mux.HandleFunc("/api/control/cron/jobs/", s.cronJobItemHandler)

	assetsFS, err := fs.Sub(webStaticFS, "static/assets")
	if err != nil {
		return err
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	server := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/chat", http.StatusTemporaryRedirect)
}

func (s *Server) chatPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/chat" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	content, err := webStaticFS.ReadFile("static/chat.html")
	if err != nil {
		s.logger.Error("chat page unavailable", slog.String("error", err.Error()))
		http.Error(w, "chat page unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) readyHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func (s *Server) messageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	msg, statusCode, err := s.prepareMessage(r)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	s.countGateway(string(msg.ChannelType))
	if task, accepted, submitErr := s.submitAsyncTask(msg); accepted {
		if submitErr != nil {
			s.logWebMessageFailure(msg, submitErr)
			writeJSON(w, http.StatusInternalServerError, messageResponse{
				Result: asyncAcceptedResult(msg, task),
				Error:  submitErr.Error(),
			})
			return
		}
		writeJSON(w, http.StatusAccepted, messageResponse{
			Result:     asyncAcceptedResult(msg, task),
			TaskID:     task.ID,
			TaskStatus: string(task.Status),
		})
		return
	}

	result, err := s.orchestrator.Handle(r.Context(), msg)
	if err != nil {
		statusCode := http.StatusBadRequest
		switch result.ErrorCode {
		case "command_failed", "nl_execution_failed":
			statusCode = http.StatusInternalServerError
		case "queue_timeout":
			statusCode = http.StatusGatewayTimeout
		case "rate_limited":
			statusCode = http.StatusTooManyRequests
		case "queue_canceled":
			statusCode = http.StatusRequestTimeout
		}
		s.logWebMessageFailure(msg, err)
		writeJSON(w, statusCode, messageResponse{
			Result: result,
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{
		Result: result,
	})
}

func (s *Server) messageStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	msg, statusCode, err := s.prepareMessage(r)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	s.countGateway(string(msg.ChannelType))
	if err := writeSSE(w, "start", streamStartResponse{
		MessageID: msg.MessageID,
		SessionID: msg.SessionID,
		ChannelID: msg.ChannelID,
		TraceID:   msg.TraceID,
	}); err != nil {
		return
	}
	flusher.Flush()

	if task, accepted, submitErr := s.submitAsyncTask(msg); accepted {
		if submitErr != nil {
			s.logWebMessageFailure(msg, submitErr)
			_ = writeSSE(w, "error", streamErrorResponse{
				Error:  submitErr.Error(),
				Result: asyncAcceptedResult(msg, task),
			})
			flusher.Flush()
			return
		}
		_ = writeSSE(w, "done", streamDoneResponse{
			Result:     asyncAcceptedResult(msg, task),
			TaskID:     task.ID,
			TaskStatus: string(task.Status),
		})
		flusher.Flush()
		return
	}

	result, handleErr := s.orchestrator.Handle(r.Context(), msg)
	if handleErr != nil {
		s.logWebMessageFailure(msg, handleErr)
		_ = writeSSE(w, "error", streamErrorResponse{
			Error:  handleErr.Error(),
			Result: result,
		})
		flusher.Flush()
		return
	}

	for _, chunk := range chunkText(result.Output, 24) {
		if err := writeSSE(w, "delta", streamDeltaResponse{
			Delta: chunk,
			Route: result.Route,
		}); err != nil {
			return
		}
		flusher.Flush()
	}

	_ = writeSSE(w, "done", streamDoneResponse{Result: result})
	flusher.Flush()
}

func (s *Server) sessionListHandler(w http.ResponseWriter, r *http.Request) {
	if s.sessions == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session history unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query, statusCode, err := parseSessionQuery(r)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s.sessions.ListSessions(query))
}

func (s *Server) sessionMessageListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	sessionID, resource, ok := sessionResourceID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session path"})
		return
	}

	switch resource {
	case "messages":
		if s.sessions == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session history unavailable"})
			return
		}
		query, statusCode, err := parseMessageQuery(r, sessionID)
		if err != nil {
			writeJSON(w, statusCode, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, s.sessions.ListMessages(query))
	case "tasks":
		if s.tasks == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task service unavailable"})
			return
		}
		items := s.tasks.ListBySession(sessionID)
		messageID := strings.TrimSpace(r.URL.Query().Get("message_id"))
		if messageID != "" {
			filtered := make([]taskdomain.Task, 0, len(items))
			for _, item := range items {
				if strings.TrimSpace(item.SourceMessageID) == messageID ||
					strings.TrimSpace(item.MessageLink.RequestMessageID) == messageID ||
					strings.TrimSpace(item.MessageLink.ResultMessageID) == messageID {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		latestRaw := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("latest")))
		if (latestRaw == "true" || latestRaw == "1" || latestRaw == "yes") && len(items) > 1 {
			items = items[:1]
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session path"})
		return
	}
}

func (s *Server) taskCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task service unavailable"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		query, statusCode, err := parseTaskListQuery(r)
		if err != nil {
			writeJSON(w, statusCode, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, s.tasks.List(query))
	case http.MethodPost:
		defer r.Body.Close()
		var req taskCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		sessionID := strings.TrimSpace(req.SessionID)
		if sessionID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id is required"})
			return
		}
		input := strings.TrimSpace(req.Input)
		if input == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "input is required"})
			return
		}
		channelID := strings.TrimSpace(req.ChannelID)
		if channelID == "" {
			channelID = "web-default"
		}

		metadata := cloneStringMap(req.Metadata)
		metadata[taskapp.MetadataTaskTypeKey] = strings.TrimSpace(req.TaskType)
		metadata[taskapp.MetadataTaskIdempotencyKey] = strings.TrimSpace(req.IdempotencyKey)
		metadata[taskapp.MetadataTaskAsyncMode] = strings.TrimSpace(req.AsyncHint)
		if strings.TrimSpace(metadata[taskapp.MetadataTaskAsyncMode]) == "" {
			metadata[taskapp.MetadataTaskAsyncMode] = "force"
		}

		sourceMessageID := strings.TrimSpace(req.SourceMessageID)
		if sourceMessageID == "" {
			sourceMessageID = s.idGenerator.NewID()
		}
		metadata[taskapp.MetadataTaskSourceMessageID] = sourceMessageID

		msg := shareddomain.UnifiedMessage{
			MessageID:     sourceMessageID,
			SessionID:     sessionID,
			UserID:        strings.TrimSpace(req.UserID),
			ChannelID:     channelID,
			ChannelType:   shareddomain.ChannelTypeWeb,
			TriggerType:   shareddomain.TriggerTypeUser,
			Content:       input,
			Metadata:      metadata,
			TraceID:       s.idGenerator.NewID(),
			CorrelationID: strings.TrimSpace(req.CorrelationID),
			ReceivedAt:    time.Now().UTC(),
		}
		task, err := s.tasks.Submit(msg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		estimatedWaitMS := int64(0)
		if task.TimeoutMS > 0 {
			estimatedWaitMS = task.TimeoutMS
		}
		writeJSON(w, http.StatusAccepted, taskCreateResponse{
			TaskID:          task.ID,
			Status:          string(task.Status),
			AcceptedAt:      task.CreatedAt.Format(time.RFC3339Nano),
			EstimatedWaitMS: estimatedWaitMS,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) taskItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task service unavailable"})
		return
	}

	taskID, action, ok := taskResourceID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid task path"})
		return
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		item, exists := s.tasks.Get(taskID)
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusOK, item)
	case action == "logs" && r.Method == http.MethodGet:
		cursor, limit, statusCode, err := parseTaskLogQuery(r)
		if err != nil {
			writeJSON(w, statusCode, map[string]string{"error": err.Error()})
			return
		}
		page, err := s.tasks.ListLogs(taskID, cursor, limit)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusOK, page)
	case action == "artifacts" && r.Method == http.MethodGet:
		items, err := s.tasks.ListArtifacts(taskID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case action == "cancel" && r.Method == http.MethodPost:
		item, err := s.tasks.Cancel(taskID)
		if err != nil {
			if errors.Is(err, taskapp.ErrTaskNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
				return
			}
			if errors.Is(err, taskapp.ErrTaskConflict) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, item)
	case action == "retry" && r.Method == http.MethodPost:
		item, err := s.tasks.Retry(taskID)
		if err != nil {
			if errors.Is(err, taskapp.ErrTaskNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
				return
			}
			if errors.Is(err, taskapp.ErrTaskConflict) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, item)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) controlTaskItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task service unavailable"})
		return
	}

	taskID, action, subAction, ok := controlTaskResourceID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid control task path"})
		return
	}

	switch {
	case action == "" && subAction == "" && r.Method == http.MethodGet:
		item, exists := s.tasks.Get(taskID)
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusOK, toTaskControlView(item))
	case action == "retry" && subAction == "" && r.Method == http.MethodPost:
		item, err := s.tasks.Retry(taskID)
		if err != nil {
			s.writeTaskControlError(w, taskID, err)
			return
		}
		writeJSON(w, http.StatusAccepted, toTaskControlView(item))
	case action == "cancel" && subAction == "" && r.Method == http.MethodPost:
		item, err := s.tasks.Cancel(taskID)
		if err != nil {
			s.writeTaskControlError(w, taskID, err)
			return
		}
		writeJSON(w, http.StatusOK, toTaskControlView(item))
	case action == "logs" && subAction == "" && r.Method == http.MethodGet:
		cursor, limit, statusCode, err := parseTaskLogQuery(r)
		if err != nil {
			writeJSON(w, statusCode, map[string]string{"error": err.Error()})
			return
		}
		page, err := s.tasks.ListLogs(taskID, cursor, limit)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusOK, page)
	case action == "logs" && subAction == "stream" && r.Method == http.MethodGet:
		s.streamTaskLogs(w, r, taskID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) writeTaskControlError(w http.ResponseWriter, taskID string, err error) {
	if errors.Is(err, taskapp.ErrTaskNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	if errors.Is(err, taskapp.ErrTaskConflict) {
		if item, exists := s.tasks.Get(taskID); exists {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": err.Error(),
				"view":  toTaskControlView(item),
			})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func toTaskControlView(task taskdomain.Task) taskControlView {
	return taskControlView{
		Task:    task,
		Actions: resolveTaskControlActions(task),
		Link:    resolveTaskSessionLink(task),
	}
}

func resolveTaskControlActions(task taskdomain.Task) taskControlActions {
	allowedRetry := []taskdomain.TaskStatus{taskdomain.TaskStatusFailed, taskdomain.TaskStatusCanceled}
	allowedCancel := []taskdomain.TaskStatus{taskdomain.TaskStatusQueued, taskdomain.TaskStatusRunning}
	retryEnabled := task.Status == taskdomain.TaskStatusFailed || task.Status == taskdomain.TaskStatusCanceled
	cancelEnabled := task.Status == taskdomain.TaskStatusQueued || task.Status == taskdomain.TaskStatusRunning

	retryReason := ""
	if !retryEnabled {
		retryReason = "retry is allowed only when task status is failed or canceled"
	}
	cancelReason := ""
	if !cancelEnabled {
		cancelReason = "cancel is allowed only when task status is queued or running"
	}

	return taskControlActions{
		Retry: taskControlActionState{
			AllowedStatuses: allowedRetry,
			Enabled:         retryEnabled,
			Reason:          retryReason,
		},
		Cancel: taskControlActionState{
			AllowedStatuses: allowedCancel,
			Enabled:         cancelEnabled,
			Reason:          cancelReason,
		},
	}
}

func resolveTaskSessionLink(task taskdomain.Task) taskSessionLink {
	sessionID := strings.TrimSpace(task.SessionID)
	taskID := strings.TrimSpace(task.ID)
	link := taskSessionLink{
		TaskID:              taskID,
		SessionID:           sessionID,
		RequestMessageID:    strings.TrimSpace(task.MessageLink.RequestMessageID),
		ResultMessageID:     strings.TrimSpace(task.MessageLink.ResultMessageID),
		TaskDetailPath:      "/api/control/tasks/" + taskID,
		SessionTasksPath:    "/api/sessions/" + sessionID + "/tasks",
		SessionMessagesPath: "/api/sessions/" + sessionID + "/messages",
	}
	if link.RequestMessageID == "" {
		link.RequestMessageID = strings.TrimSpace(task.SourceMessageID)
	}
	if sessionID == "" {
		link.SessionTasksPath = ""
		link.SessionMessagesPath = ""
	}
	return link
}

func (s *Server) streamTaskLogs(w http.ResponseWriter, r *http.Request, taskID string) {
	cursor, err := parseNonNegativeInt(r.URL.Query().Get("cursor"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cursor must be a non-negative integer"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	if _, exists := s.tasks.Get(taskID); !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if err := writeSSE(w, "start", taskLogStreamStartResponse{TaskID: taskID, Cursor: cursor}); err != nil {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()

	for {
		page, listErr := s.tasks.ListLogs(taskID, cursor, 200)
		if listErr != nil {
			_ = writeSSE(w, "error", taskLogStreamErrorResponse{Error: "task not found", NextCursor: cursor})
			flusher.Flush()
			return
		}
		for _, item := range page.Items {
			nextCursor := cursor + 1
			if err := writeSSE(w, "log", taskLogStreamEvent{
				TaskID:     taskID,
				Cursor:     cursor,
				NextCursor: nextCursor,
				Log:        item,
			}); err != nil {
				return
			}
			cursor = nextCursor
			flusher.Flush()
		}

		if page.NextCursor > cursor {
			cursor = page.NextCursor
		}
		if page.HasMore {
			continue
		}

		task, exists := s.tasks.Get(taskID)
		if !exists {
			_ = writeSSE(w, "error", taskLogStreamErrorResponse{Error: "task not found", NextCursor: cursor})
			flusher.Flush()
			return
		}
		if task.Status.IsTerminal() {
			_ = writeSSE(w, "done", taskLogStreamDoneResponse{TaskID: taskID, Status: task.Status, NextCursor: cursor})
			flusher.Flush()
			return
		}

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) agentMemoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.memory == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent memory unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, s.memory.Snapshot())
}

func (s *Server) memoryTaskCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query, statusCode, err := parseMemoryTaskListQuery(r)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	tasks := s.collectTasksForMemory(query.Status)
	items := make([]memoryTaskSummaryItem, 0, len(tasks))
	for _, item := range tasks {
		if !matchMemoryTaskFilters(item, query) {
			continue
		}
		items = append(items, resolveMemoryTaskSummary(item))
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].FinishedAt.Equal(items[j].FinishedAt) {
			return items[i].TaskID > items[j].TaskID
		}
		return items[i].FinishedAt.After(items[j].FinishedAt)
	})

	from, to := memoryTaskPageBounds(len(items), query.Page, query.PageSize)
	pageItems := make([]memoryTaskSummaryItem, 0, to-from)
	pageItems = append(pageItems, items[from:to]...)
	writeJSON(w, http.StatusOK, map[string]any{
		"items": pageItems,
		"pagination": taskapp.Pagination{
			Page:     query.Page,
			PageSize: query.PageSize,
			Total:    len(items),
			HasNext:  to < len(items),
		},
	})
}

func (s *Server) memoryTaskItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task service unavailable"})
		return
	}
	taskID, action, ok := memoryTaskResourceID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid memory task path"})
		return
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		item, exists := s.tasks.Get(taskID)
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		refs := []any{}
		if s.memory != nil {
			runtimeRefs := s.memory.TaskSummaryRefs(taskID)
			refs = make([]any, 0, len(runtimeRefs))
			for _, ref := range runtimeRefs {
				refs = append(refs, ref)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"meta":         resolveMemoryTaskMeta(item),
			"summary_refs": refs,
		})
	case action == "logs" && r.Method == http.MethodGet:
		cursor, limit, statusCode, err := parseTaskLogQuery(r)
		if err != nil {
			writeJSON(w, statusCode, map[string]string{"error": err.Error()})
			return
		}
		page, err := s.tasks.ListLogs(taskID, cursor, limit)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"items":        []taskdomain.TaskLog{},
				"cursor":       cursor,
				"next_cursor":  cursor,
				"has_more":     false,
				"error_code":   "task_logs_unavailable",
				"rebuild_hint": "日志缺失或文件损坏，可执行摘要重建",
			})
			return
		}
		writeJSON(w, http.StatusOK, page)
	case action == "artifacts" && r.Method == http.MethodGet:
		items, err := s.tasks.ListArtifacts(taskID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case action == "rebuild-summary" && r.Method == http.MethodPost:
		if s.memory == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent memory unavailable"})
			return
		}
		item, exists := s.tasks.Get(taskID)
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		refs, err := s.memory.RebuildTaskSummary(item)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"task_id":      taskID,
			"status":       "rebuilt",
			"summary_refs": refs,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) collectTasksForMemory(status taskdomain.TaskStatus) []taskdomain.Task {
	items := make([]taskdomain.Task, 0, 64)
	page := 1
	for {
		result := s.tasks.List(taskapp.ListQuery{
			Status:   status,
			Page:     page,
			PageSize: 200,
		})
		if len(result.Items) == 0 {
			break
		}
		items = append(items, result.Items...)
		if !result.Pagination.HasNext {
			break
		}
		page++
		if page > 10000 {
			break
		}
	}
	return items
}

func matchMemoryTaskFilters(task taskdomain.Task, query memoryTaskListQuery) bool {
	if strings.TrimSpace(string(query.Status)) != "" && task.Status != query.Status {
		return false
	}
	if strings.TrimSpace(query.TaskType) != "" {
		if !strings.EqualFold(resolveMemoryTaskType(task), strings.TrimSpace(query.TaskType)) {
			return false
		}
	}
	at := resolveMemoryTaskTime(task)
	if !query.StartAt.IsZero() && at.Before(query.StartAt) {
		return false
	}
	if !query.EndAt.IsZero() && at.After(query.EndAt) {
		return false
	}
	return true
}

func resolveMemoryTaskSummary(task taskdomain.Task) memoryTaskSummaryItem {
	summary := task.TaskSummary
	if summary.IsZero() {
		finished := task.FinishedAt
		if finished.IsZero() {
			finished = task.UpdatedAt
		}
		if finished.IsZero() {
			finished = task.CreatedAt
		}
		if finished.IsZero() {
			finished = time.Now().UTC()
		}
		summary = taskdomain.TaskSummary{
			TaskID:     strings.TrimSpace(task.ID),
			TaskType:   resolveMemoryTaskType(task),
			Goal:       strings.TrimSpace(task.RequestContent),
			Result:     strings.TrimSpace(task.Summary),
			Status:     task.Status,
			FinishedAt: finished,
			Tags:       []string{"task", strings.ToLower(strings.TrimSpace(string(task.Status))), resolveMemoryTaskType(task)},
		}
	}
	if summary.FinishedAt.IsZero() {
		summary.FinishedAt = resolveMemoryTaskTime(task)
	}
	if strings.TrimSpace(summary.Result) == "" {
		summary.Result = strings.TrimSpace(task.Summary)
	}
	if strings.TrimSpace(summary.Result) == "" {
		summary.Result = strings.TrimSpace(task.Result.Output)
	}
	if strings.TrimSpace(summary.Result) == "" {
		summary.Result = strings.TrimSpace(task.ErrorMessage)
	}
	if strings.TrimSpace(summary.Result) == "" {
		summary.Result = "-"
	}
	if strings.TrimSpace(summary.Goal) == "" {
		summary.Goal = strings.TrimSpace(task.RequestContent)
	}
	if strings.TrimSpace(summary.Goal) == "" {
		summary.Goal = "-"
	}
	if !summary.Status.IsValid() {
		summary.Status = task.Status
	}
	return memoryTaskSummaryItem{
		TaskID:     strings.TrimSpace(summary.TaskID),
		TaskType:   strings.TrimSpace(summary.TaskType),
		Goal:       strings.TrimSpace(summary.Goal),
		Result:     strings.TrimSpace(summary.Result),
		Status:     summary.Status,
		FinishedAt: summary.FinishedAt.UTC(),
		Tags:       append([]string(nil), summary.Tags...),
	}
}

func resolveMemoryTaskMeta(task taskdomain.Task) memoryTaskMeta {
	return memoryTaskMeta{
		TaskID:          strings.TrimSpace(task.ID),
		SessionID:       strings.TrimSpace(task.SessionID),
		SourceMessageID: strings.TrimSpace(task.SourceMessageID),
		Status:          task.Status,
		Progress:        task.Progress,
		CreatedAt:       task.CreatedAt.UTC(),
		FinishedAt:      task.FinishedAt.UTC(),
		RetryCount:      task.RetryCount,
		TaskType:        resolveMemoryTaskType(task),
	}
}

func resolveMemoryTaskType(task taskdomain.Task) string {
	if value := strings.TrimSpace(task.TaskSummary.TaskType); value != "" {
		return strings.ToLower(value)
	}
	if value := strings.TrimSpace(task.TaskType); value != "" {
		return strings.ToLower(value)
	}
	if value := strings.TrimSpace(task.RequestMetadata[taskapp.MetadataTaskTypeKey]); value != "" {
		return strings.ToLower(value)
	}
	if value := strings.TrimSpace(string(task.Result.Route)); value != "" {
		return strings.ToLower(value)
	}
	return "task"
}

func resolveMemoryTaskTime(task taskdomain.Task) time.Time {
	if !task.TaskSummary.FinishedAt.IsZero() {
		return task.TaskSummary.FinishedAt.UTC()
	}
	if !task.FinishedAt.IsZero() {
		return task.FinishedAt.UTC()
	}
	if !task.UpdatedAt.IsZero() {
		return task.UpdatedAt.UTC()
	}
	if !task.CreatedAt.IsZero() {
		return task.CreatedAt.UTC()
	}
	return time.Now().UTC()
}

func memoryTaskPageBounds(total int, page int, pageSize int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if offset >= total {
		return total, total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return offset, end
}

func (s *Server) channelListHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.control.ListChannels()})
}

func (s *Server) channelItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}

	channelID, ok := resourceID(r.URL.Path, "/api/control/channels/")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid channel path"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		defer r.Body.Close()
		var req channelUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		channel := controldomain.Channel{
			ID:          channelID,
			Type:        shareddomain.ChannelType(strings.TrimSpace(req.Type)),
			Enabled:     enabled,
			Description: strings.TrimSpace(req.Description),
			Metadata:    req.Metadata,
		}
		if err := s.control.UpsertChannel(channel); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, channel)
	case http.MethodDelete:
		if !s.control.DeleteChannel(channelID) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "channel not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) capabilityListHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filterType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	if filterType == "" {
		writeJSON(w, http.StatusOK, map[string]any{"items": s.control.ListCapabilities()})
		return
	}
	capabilityType := controldomain.CapabilityType(filterType)
	if !capabilityType.IsSupported() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type must be skill or mcp"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.control.ListCapabilitiesByType(capabilityType)})
}

func (s *Server) capabilityAuditListHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filterType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	items := s.control.ListCapabilityAudits()
	if filterType != "" {
		capabilityType := controldomain.CapabilityType(filterType)
		if !capabilityType.IsSupported() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type must be skill or mcp"})
			return
		}
		filtered := make([]controldomain.CapabilityAudit, 0, len(items))
		for _, item := range items {
			if item.CapabilityType == capabilityType {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) capabilityItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}

	capabilityType, capabilityID, ok := typedResourceID(r.URL.Path, "/api/control/capabilities/")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid capability path"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.upsertTypedCapability(w, r, capabilityID, capabilityType)
	case http.MethodPost:
		s.applyCapabilityLifecycle(w, r, capabilityID, capabilityType)
	case http.MethodDelete:
		if !s.control.DeleteCapability(capabilityType, capabilityID) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "capability not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) skillListHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.control.ListCapabilitiesByType(controldomain.CapabilityTypeSkill)})
}

func (s *Server) skillItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}

	skillID, ok := resourceID(r.URL.Path, "/api/control/skills/")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid skill path"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.upsertTypedCapability(w, r, skillID, controldomain.CapabilityTypeSkill)
	case http.MethodPost:
		s.applyCapabilityLifecycle(w, r, skillID, controldomain.CapabilityTypeSkill)
	case http.MethodDelete:
		if !s.control.DeleteSkill(skillID) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "skill not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) mcpListHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.control.ListMCPs()})
}

func (s *Server) mcpItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}

	mcpID, ok := resourceID(r.URL.Path, "/api/control/mcps/")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mcp path"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.upsertTypedCapability(w, r, mcpID, controldomain.CapabilityTypeMCP)
	case http.MethodPost:
		s.applyCapabilityLifecycle(w, r, mcpID, controldomain.CapabilityTypeMCP)
	case http.MethodDelete:
		if !s.control.DeleteMCP(mcpID) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "mcp not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) upsertTypedCapability(w http.ResponseWriter, r *http.Request, capabilityID string, forcedType controldomain.CapabilityType) {
	defer r.Body.Close()
	var req skillUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	capabilityType := forcedType
	if capabilityType == "" {
		capabilityType = controldomain.CapabilityType(strings.ToLower(strings.TrimSpace(req.Type)))
	}
	if req.Type != "" && forcedType != "" && strings.ToLower(strings.TrimSpace(req.Type)) != string(forcedType) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "capability type mismatch"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	capability := controldomain.Capability{
		ID:       capabilityID,
		Name:     strings.TrimSpace(req.Name),
		Type:     capabilityType,
		Enabled:  enabled,
		Scope:    controldomain.CapabilityScope(strings.ToLower(strings.TrimSpace(req.Scope))),
		Version:  strings.TrimSpace(req.Version),
		Metadata: req.Metadata,
	}
	if err := s.control.UpsertCapability(capability); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, capability.Normalized())
}

func (s *Server) applyCapabilityLifecycle(w http.ResponseWriter, r *http.Request, capabilityID string, capabilityType controldomain.CapabilityType) {
	defer r.Body.Close()
	var req capabilityLifecycleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case string(controldomain.CapabilityLifecycleEnable):
		capability, err := s.control.SetCapabilityEnabled(capabilityType, capabilityID, true)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "capability not found"})
			return
		}
		writeJSON(w, http.StatusOK, capability)
	case string(controldomain.CapabilityLifecycleDisable):
		capability, err := s.control.SetCapabilityEnabled(capabilityType, capabilityID, false)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "capability not found"})
			return
		}
		writeJSON(w, http.StatusOK, capability)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action must be enable or disable"})
	}
}

func (s *Server) cronJobListHandler(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "scheduler unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	jobs := s.scheduler.List()
	items := make([]cronJobResponse, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, toCronJobResponse(job))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) cronJobItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "scheduler unavailable"})
		return
	}

	jobID, ok := resourceID(r.URL.Path, "/api/control/cron/jobs/")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid cron job path"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		defer r.Body.Close()
		var req cronJobUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		interval, err := time.ParseDuration(strings.TrimSpace(req.Interval))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid interval, e.g. 30s or 5m"})
			return
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		job := schedulerdomain.Job{
			ID:        jobID,
			Name:      strings.TrimSpace(req.Name),
			Interval:  interval,
			Enabled:   enabled,
			SessionID: strings.TrimSpace(req.SessionID),
			UserID:    strings.TrimSpace(req.UserID),
			ChannelID: strings.TrimSpace(req.ChannelID),
			Content:   req.Content,
			Metadata:  req.Metadata,
		}
		if err := s.scheduler.Upsert(job); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, toCronJobResponse(job))
	case http.MethodDelete:
		if !s.scheduler.Delete(jobID) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "cron job not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func toCronJobResponse(job schedulerdomain.Job) cronJobResponse {
	return cronJobResponse{
		ID:        job.ID,
		Name:      job.Name,
		Interval:  job.Interval.String(),
		Enabled:   job.Enabled,
		SessionID: job.SessionID,
		UserID:    job.UserID,
		ChannelID: job.ChannelID,
		Content:   job.Content,
		Metadata:  job.Metadata,
	}
}

func resourceID(path, prefix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

func sessionResourceID(path string) (string, string, bool) {
	const prefix = "/api/sessions/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	resource := strings.TrimSpace(parts[1])
	if resource != "messages" && resource != "tasks" {
		return "", "", false
	}
	sessionID := strings.TrimSpace(parts[0])
	if sessionID == "" {
		return "", "", false
	}
	return sessionID, resource, true
}

func taskResourceID(path string) (string, string, bool) {
	const prefix = "/api/tasks/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		taskID := strings.TrimSpace(parts[0])
		if taskID == "" {
			return "", "", false
		}
		return taskID, "", true
	}
	if len(parts) == 2 {
		taskID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		if taskID == "" || action == "" {
			return "", "", false
		}
		return taskID, action, true
	}
	return "", "", false
}

func controlTaskResourceID(path string) (string, string, string, bool) {
	const prefix = "/api/control/tasks/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}

	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		taskID := strings.TrimSpace(parts[0])
		if taskID == "" {
			return "", "", "", false
		}
		return taskID, "", "", true
	}
	if len(parts) == 2 {
		taskID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		if taskID == "" || action == "" {
			return "", "", "", false
		}
		return taskID, action, "", true
	}
	if len(parts) == 3 {
		taskID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		subAction := strings.TrimSpace(parts[2])
		if taskID == "" || action == "" || subAction == "" {
			return "", "", "", false
		}
		return taskID, action, subAction, true
	}
	return "", "", "", false
}

func memoryTaskResourceID(path string) (string, string, bool) {
	const prefix = "/api/memory/tasks/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		taskID := strings.TrimSpace(parts[0])
		if taskID == "" {
			return "", "", false
		}
		return taskID, "", true
	}
	if len(parts) == 2 {
		taskID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		if taskID == "" || action == "" {
			return "", "", false
		}
		return taskID, action, true
	}
	return "", "", false
}

func typedResourceID(path, prefix string) (controldomain.CapabilityType, string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	capabilityType := controldomain.CapabilityType(strings.ToLower(strings.TrimSpace(parts[0])))
	if !capabilityType.IsSupported() {
		return "", "", false
	}
	id := strings.TrimSpace(parts[1])
	if id == "" {
		return "", "", false
	}
	return capabilityType, id, true
}

func parseSessionQuery(r *http.Request) (sessionapp.SessionQuery, int, error) {
	page, pageSize, statusCode, err := parsePaginationQuery(r)
	if err != nil {
		return sessionapp.SessionQuery{}, statusCode, err
	}
	startAt, endAt, statusCode, err := parseTimeRangeQuery(r)
	if err != nil {
		return sessionapp.SessionQuery{}, statusCode, err
	}

	return sessionapp.SessionQuery{
		StartAt:  startAt,
		EndAt:    endAt,
		Page:     page,
		PageSize: pageSize,
	}, http.StatusOK, nil
}

func parseMessageQuery(r *http.Request, sessionID string) (sessionapp.MessageQuery, int, error) {
	page, pageSize, statusCode, err := parsePaginationQuery(r)
	if err != nil {
		return sessionapp.MessageQuery{}, statusCode, err
	}
	startAt, endAt, statusCode, err := parseTimeRangeQuery(r)
	if err != nil {
		return sessionapp.MessageQuery{}, statusCode, err
	}

	return sessionapp.MessageQuery{
		SessionID: sessionID,
		StartAt:   startAt,
		EndAt:     endAt,
		Page:      page,
		PageSize:  pageSize,
	}, http.StatusOK, nil
}

func parseTaskListQuery(r *http.Request) (taskapp.ListQuery, int, error) {
	page, pageSize, statusCode, err := parsePaginationQuery(r)
	if err != nil {
		return taskapp.ListQuery{}, statusCode, err
	}
	query := taskapp.ListQuery{
		SessionID: strings.TrimSpace(r.URL.Query().Get("session_id")),
		Page:      page,
		PageSize:  pageSize,
	}
	rawStatus := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if rawStatus != "" {
		status := taskdomain.TaskStatus(rawStatus)
		if !status.IsValid() {
			return taskapp.ListQuery{}, http.StatusBadRequest, errors.New("status must be queued/running/success/failed/canceled")
		}
		query.Status = status
	}
	return query, http.StatusOK, nil
}

func parseMemoryTaskListQuery(r *http.Request) (memoryTaskListQuery, int, error) {
	page, err := parsePositiveInt(r.URL.Query().Get("page"))
	if err != nil {
		return memoryTaskListQuery{}, http.StatusBadRequest, errors.New("page must be a positive integer")
	}
	pageSize, err := parsePositiveInt(r.URL.Query().Get("page_size"))
	if err != nil {
		return memoryTaskListQuery{}, http.StatusBadRequest, errors.New("page_size must be a positive integer")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	startAt, endAt, statusCode, err := parseTimeRangeQuery(r)
	if err != nil {
		return memoryTaskListQuery{}, statusCode, err
	}
	query := memoryTaskListQuery{
		TaskType: strings.ToLower(strings.TrimSpace(r.URL.Query().Get("task_type"))),
		StartAt:  startAt,
		EndAt:    endAt,
		Page:     page,
		PageSize: pageSize,
	}
	rawStatus := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if rawStatus != "" {
		status := taskdomain.TaskStatus(rawStatus)
		if !status.IsValid() {
			return memoryTaskListQuery{}, http.StatusBadRequest, errors.New("status must be queued/running/success/failed/canceled")
		}
		query.Status = status
	}
	return query, http.StatusOK, nil
}

func parseTaskLogQuery(r *http.Request) (int, int, int, error) {
	cursor, err := parseNonNegativeInt(r.URL.Query().Get("cursor"))
	if err != nil {
		return 0, 0, http.StatusBadRequest, errors.New("cursor must be a non-negative integer")
	}
	limit, err := parsePositiveInt(r.URL.Query().Get("limit"))
	if err != nil {
		return 0, 0, http.StatusBadRequest, errors.New("limit must be a positive integer")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return cursor, limit, http.StatusOK, nil
}

func parsePaginationQuery(r *http.Request) (int, int, int, error) {
	page, err := parsePositiveInt(r.URL.Query().Get("page"))
	if err != nil {
		return 0, 0, http.StatusBadRequest, errors.New("page must be a positive integer")
	}
	pageSize, err := parsePositiveInt(r.URL.Query().Get("page_size"))
	if err != nil {
		return 0, 0, http.StatusBadRequest, errors.New("page_size must be a positive integer")
	}
	return page, pageSize, http.StatusOK, nil
}

func parseTimeRangeQuery(r *http.Request) (time.Time, time.Time, int, error) {
	startAt, err := parseRFC3339Time(r.URL.Query().Get("start_at"))
	if err != nil {
		return time.Time{}, time.Time{}, http.StatusBadRequest, errors.New("start_at must be RFC3339 format")
	}
	endAt, err := parseRFC3339Time(r.URL.Query().Get("end_at"))
	if err != nil {
		return time.Time{}, time.Time{}, http.StatusBadRequest, errors.New("end_at must be RFC3339 format")
	}
	return startAt, endAt, http.StatusOK, nil
}

func parsePositiveInt(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid positive integer")
	}
	return value, nil
}

func parseNonNegativeInt(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 0 {
		return 0, errors.New("invalid non-negative integer")
	}
	return value, nil
}

func parseRFC3339Time(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func (s *Server) prepareMessage(r *http.Request) (shareddomain.UnifiedMessage, int, error) {
	defer r.Body.Close()

	var req messageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return shareddomain.UnifiedMessage{}, http.StatusBadRequest, errors.New("invalid json body")
	}
	if strings.TrimSpace(req.Content) == "" {
		return shareddomain.UnifiedMessage{}, http.StatusBadRequest, errors.New("content is required")
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = s.idGenerator.NewID()
	}

	channelID := strings.TrimSpace(req.ChannelID)
	if channelID == "" {
		channelID = "web-default"
	}

	channelType := shareddomain.ChannelTypeWeb
	if s.control != nil {
		channel, ok := s.control.ResolveChannel(channelID)
		if !ok {
			return shareddomain.UnifiedMessage{}, http.StatusBadRequest, errors.New("channel not found")
		}
		if !channel.Enabled {
			return shareddomain.UnifiedMessage{}, http.StatusBadRequest, errors.New("channel is disabled")
		}
		channelType = channel.Type
	}

	return shareddomain.UnifiedMessage{
		MessageID:     s.idGenerator.NewID(),
		SessionID:     sessionID,
		UserID:        req.UserID,
		ChannelID:     channelID,
		ChannelType:   channelType,
		TriggerType:   shareddomain.TriggerTypeUser,
		Content:       req.Content,
		Metadata:      req.Metadata,
		TraceID:       s.idGenerator.NewID(),
		CorrelationID: strings.TrimSpace(req.CorrelationID),
		ReceivedAt:    time.Now().UTC(),
	}, http.StatusOK, nil
}

func (s *Server) submitAsyncTask(msg shareddomain.UnifiedMessage) (taskdomain.Task, bool, error) {
	if s.tasks == nil {
		return taskdomain.Task{}, false, nil
	}
	if !s.tasks.ShouldRunAsync(msg) {
		return taskdomain.Task{}, false, nil
	}
	item, err := s.tasks.Submit(msg)
	if err != nil {
		return taskdomain.Task{}, true, err
	}
	return item, true, nil
}

func asyncAcceptedResult(msg shareddomain.UnifiedMessage, task taskdomain.Task) shareddomain.OrchestrationResult {
	metadata := map[string]string{
		taskapp.MetadataTaskIDKey: task.ID,
	}
	if strings.TrimSpace(string(task.Status)) != "" {
		metadata[taskapp.MetadataTaskStatusKey] = string(task.Status)
	}
	return shareddomain.OrchestrationResult{
		MessageID: msg.MessageID,
		SessionID: msg.SessionID,
		ErrorCode: "task_accepted",
		Metadata:  metadata,
	}
}

func (s *Server) countGateway(channelType string) {
	if s.telemetry == nil {
		return
	}
	s.telemetry.CountGateway(channelType)
}

func (s *Server) logWebMessageFailure(msg shareddomain.UnifiedMessage, err error) {
	if s.logger == nil {
		return
	}
	s.logger.Error("web message failed",
		slog.String("trace_id", msg.TraceID),
		slog.String("session_id", msg.SessionID),
		slog.String("message_id", msg.MessageID),
		slog.String("channel_id", msg.ChannelID),
		slog.String("channel_type", string(msg.ChannelType)),
		slog.String("error", err.Error()),
	)
}

func writeSSE(w http.ResponseWriter, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte("event: " + event + "\n")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
		return err
	}
	return nil
}

func chunkText(content string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 1
	}
	runes := []rune(content)
	if len(runes) == 0 {
		return nil
	}

	chunks := make([]string, 0, (len(runes)+maxRunes-1)/maxRunes)
	for start := 0; start < len(runes); start += maxRunes {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}
