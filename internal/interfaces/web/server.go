package web

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
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

const (
	controlTaskMetadataJobIDKey       = "job_id"
	controlTaskMetadataJobNameKey     = "job_name"
	controlTaskMetadataFiredAtKey     = "fired_at"
	controlTaskTerminalParentIDKey    = "alter0.task.terminal_parent_id"
	controlTaskTerminalSessionIDKey   = "alter0.task.terminal_session_id"
	controlTaskTerminalInteractiveKey = "alter0.task.terminal_interactive"
	codexSandboxMetadataKey           = "codex_sandbox"
	codexWorkspaceModeMetadataKey     = "codex_workspace_mode"
	codexWorkspaceModeRepoRoot        = "repo-root"
	codexSandboxDangerFullAccess      = "danger-full-access"
	defaultControlTaskChannelID       = "web-default"
	maxTaskArtifactCount              = 128
	maxTaskArtifactSizeBytes          = 8 * 1024 * 1024
	taskArtifactReadTimeout           = 3 * time.Second
	webLoginCookieName                = "alter0_web_session"
	webLoginCookieTTL                 = 24 * time.Hour
)

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type StreamOrchestrator interface {
	HandleStream(
		ctx context.Context,
		msg shareddomain.UnifiedMessage,
		onDelta func(string) error,
	) (shareddomain.OrchestrationResult, error)
}

type Server struct {
	addr             string
	orchestrator     Orchestrator
	telemetry        *observability.Telemetry
	idGenerator      sharedapp.IDGenerator
	control          *controlapp.Service
	scheduler        *schedulerapp.Manager
	sessions         sessionHistoryService
	tasks            taskService
	memory           *agentMemoryService
	logger           *slog.Logger
	webLoginPassword string
	webSessionToken  string
	webLoginEnabled  bool
	webBindLocalhost bool
}

type sessionHistoryService interface {
	ListSessions(query sessionapp.SessionQuery) sessionapp.SessionPage
	ListMessages(query sessionapp.MessageQuery) sessionapp.MessagePage
}

type taskService interface {
	AssessComplexity(msg shareddomain.UnifiedMessage) taskapp.ComplexityAssessment
	ShouldRunAsync(msg shareddomain.UnifiedMessage) bool
	Submit(msg shareddomain.UnifiedMessage) (taskdomain.Task, error)
	List(query taskapp.ListQuery) taskapp.TaskPage
	Get(taskID string) (taskdomain.Task, bool)
	ListBySession(sessionID string) []taskdomain.Task
	ListLogs(taskID string, cursor int, limit int) (taskapp.TaskLogPage, error)
	ListArtifacts(taskID string) ([]taskdomain.TaskArtifact, error)
	ReadArtifact(ctx context.Context, taskID string, artifactID string) (taskdomain.TaskArtifact, []byte, error)
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
	Result                   shareddomain.OrchestrationResult `json:"result"`
	TaskID                   string                           `json:"task_id,omitempty"`
	TaskStatus               string                           `json:"task_status,omitempty"`
	ExecutionMode            string                           `json:"execution_mode,omitempty"`
	EstimatedDurationSeconds int                              `json:"estimated_duration_seconds,omitempty"`
	ComplexityLevel          string                           `json:"complexity_level,omitempty"`
	TaskCard                 *taskCardResponse                `json:"task_card,omitempty"`
	Error                    string                           `json:"error,omitempty"`
}

type taskCreateResponse struct {
	TaskID        string `json:"task_id"`
	Status        string `json:"status"`
	QueuePosition int    `json:"queue_position,omitempty"`
	AcceptedAt    string `json:"accepted_at"`
}

type controlTaskTerminalInputRequest struct {
	Input        string `json:"input"`
	ReuseTask    bool   `json:"reuse_task,omitempty"`
	AnchorTaskID string `json:"anchor_task_id,omitempty"`
}

type controlTaskTerminalInputResponse struct {
	TaskID            string `json:"task_id"`
	AnchorTaskID      string `json:"anchor_task_id,omitempty"`
	Status            string `json:"status"`
	SessionID         string `json:"session_id"`
	TerminalSessionID string `json:"terminal_session_id"`
	MaxSessions       int    `json:"max_sessions"`
	ActiveSessions    int    `json:"active_sessions"`
	TaskDetailPath    string `json:"task_detail_path"`
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
	Result                   shareddomain.OrchestrationResult `json:"result"`
	TaskID                   string                           `json:"task_id,omitempty"`
	TaskStatus               string                           `json:"task_status,omitempty"`
	ExecutionMode            string                           `json:"execution_mode,omitempty"`
	EstimatedDurationSeconds int                              `json:"estimated_duration_seconds,omitempty"`
	ComplexityLevel          string                           `json:"complexity_level,omitempty"`
	TaskCard                 *taskCardResponse                `json:"task_card,omitempty"`
}

type taskCardResponse struct {
	Notice        string `json:"notice"`
	TaskID        string `json:"task_id"`
	TaskSummary   string `json:"task_summary"`
	TaskDetailURL string `json:"task_detail_url"`
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

type taskArtifactResponse struct {
	ArtifactID  string    `json:"artifact_id"`
	Name        string    `json:"name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Summary     string    `json:"summary,omitempty"`
	DownloadURL string    `json:"download_url"`
	PreviewURL  string    `json:"preview_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
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

type environmentUpsertRequest struct {
	Operator string         `json:"operator,omitempty"`
	Values   map[string]any `json:"values"`
}

type cronTaskConfigRequest struct {
	Input      string `json:"input,omitempty"`
	RetryLimit *int   `json:"retry_limit,omitempty"`
}

type cronTaskConfigResponse struct {
	Input      string `json:"input"`
	RetryLimit int    `json:"retry_limit,omitempty"`
}

type cronJobUpsertRequest struct {
	Name           string                `json:"name,omitempty"`
	Enabled        *bool                 `json:"enabled,omitempty"`
	Timezone       string                `json:"timezone,omitempty"`
	ScheduleMode   string                `json:"schedule_mode,omitempty"`
	CronExpression string                `json:"cron_expression,omitempty"`
	TaskConfig     cronTaskConfigRequest `json:"task_config,omitempty"`
	UserID         string                `json:"user_id,omitempty"`
	ChannelID      string                `json:"channel_id,omitempty"`
	Metadata       map[string]string     `json:"metadata,omitempty"`
	Interval       string                `json:"interval,omitempty"`
	SessionID      string                `json:"session_id,omitempty"`
	Content        string                `json:"content,omitempty"`
}

type cronJobResponse struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Enabled        bool                   `json:"enabled"`
	Timezone       string                 `json:"timezone"`
	ScheduleMode   string                 `json:"schedule_mode"`
	CronExpression string                 `json:"cron_expression"`
	TaskConfig     cronTaskConfigResponse `json:"task_config"`
	UserID         string                 `json:"user_id,omitempty"`
	ChannelID      string                 `json:"channel_id,omitempty"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
	Interval       string                 `json:"interval,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	Content        string                 `json:"content,omitempty"`
}

type cronJobRunResponse struct {
	RunID     string    `json:"run_id"`
	JobID     string    `json:"job_id"`
	FiredAt   time.Time `json:"fired_at"`
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"`
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

type controlTaskListQuery struct {
	SessionID       string
	Status          taskdomain.TaskStatus
	TriggerType     shareddomain.TriggerType
	ChannelType     shareddomain.ChannelType
	ChannelID       string
	MessageID       string
	SourceMessageID string
	StartAt         time.Time
	EndAt           time.Time
	Page            int
	PageSize        int
}

type controlTaskSource struct {
	TriggerType   shareddomain.TriggerType `json:"trigger_type"`
	ChannelType   shareddomain.ChannelType `json:"channel_type"`
	ChannelID     string                   `json:"channel_id"`
	CorrelationID string                   `json:"correlation_id,omitempty"`
	JobID         string                   `json:"job_id,omitempty"`
	JobName       string                   `json:"job_name,omitempty"`
	FiredAt       time.Time                `json:"fired_at,omitempty"`
}

type controlTaskListItem struct {
	TaskID          string                   `json:"task_id"`
	SessionID       string                   `json:"session_id"`
	SourceMessageID string                   `json:"source_message_id,omitempty"`
	Status          taskdomain.TaskStatus    `json:"status"`
	Phase           string                   `json:"phase,omitempty"`
	Progress        int                      `json:"progress"`
	QueuePosition   int                      `json:"queue_position,omitempty"`
	QueueWaitMS     int64                    `json:"queue_wait_ms,omitempty"`
	RetryCount      int                      `json:"retry_count"`
	TriggerType     shareddomain.TriggerType `json:"trigger_type"`
	ChannelType     shareddomain.ChannelType `json:"channel_type"`
	ChannelID       string                   `json:"channel_id"`
	CorrelationID   string                   `json:"correlation_id,omitempty"`
	JobID           string                   `json:"job_id,omitempty"`
	JobName         string                   `json:"job_name,omitempty"`
	FiredAt         time.Time                `json:"fired_at,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	StartedAt       time.Time                `json:"started_at,omitempty"`
	UpdatedAt       time.Time                `json:"updated_at"`
	FinishedAt      time.Time                `json:"finished_at,omitempty"`
	Error           string                   `json:"error,omitempty"`
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
	TerminalSessionID   string `json:"terminal_session_id,omitempty"`
	TerminalMaxSessions int    `json:"terminal_max_sessions,omitempty"`
	RequestMessageID    string `json:"request_message_id,omitempty"`
	ResultMessageID     string `json:"result_message_id,omitempty"`
	TaskDetailPath      string `json:"task_detail_path"`
	SessionTasksPath    string `json:"session_tasks_path,omitempty"`
	SessionMessagesPath string `json:"session_messages_path,omitempty"`
}

type taskControlView struct {
	Task    taskdomain.Task    `json:"task"`
	Source  controlTaskSource  `json:"source"`
	Actions taskControlActions `json:"actions"`
	Link    taskSessionLink    `json:"link"`
}

type WebSecurityOptions struct {
	LoginPassword string
	BindLocalhost bool
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
	securityOptions WebSecurityOptions,
	logger *slog.Logger,
) *Server {
	resolvedPassword := strings.TrimSpace(securityOptions.LoginPassword)
	resolvedBindLocalhost := securityOptions.BindLocalhost
	webSessionToken := ""
	if resolvedPassword != "" {
		if idGenerator != nil {
			webSessionToken = strings.TrimSpace(idGenerator.NewID())
		}
		if webSessionToken == "" {
			webSessionToken = strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
		}
	}
	return &Server{
		addr:             addr,
		orchestrator:     orchestrator,
		telemetry:        telemetry,
		idGenerator:      idGenerator,
		control:          control,
		scheduler:        scheduler,
		sessions:         sessions,
		tasks:            tasks,
		memory:           newAgentMemoryService(memoryOptions),
		logger:           logger,
		webLoginPassword: resolvedPassword,
		webSessionToken:  webSessionToken,
		webLoginEnabled:  resolvedPassword != "",
		webBindLocalhost: resolvedBindLocalhost,
	}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", s.telemetry.MetricsHandler())
	mux.HandleFunc("/healthz", s.healthHandler)
	mux.HandleFunc("/readyz", s.readyHandler)
	mux.HandleFunc("/login", s.loginHandler)
	mux.HandleFunc("/logout", s.logoutHandler)
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
	mux.HandleFunc("/api/control/tasks", s.controlTaskCollectionHandler)
	mux.HandleFunc("/api/control/tasks/", s.controlTaskItemHandler)
	mux.HandleFunc("/api/control/environments", s.environmentConfigHandler)
	mux.HandleFunc("/api/control/environments/audits", s.environmentAuditListHandler)
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

	handler := http.Handler(mux)
	if s.webLoginEnabled {
		handler = s.authMiddleware(handler)
	}

	server := &http.Server{
		Addr:    s.addr,
		Handler: handler,
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

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if !s.webLoginEnabled {
		http.Redirect(w, r, "/chat", http.StatusTemporaryRedirect)
		return
	}
	if r.URL.Path != "/login" {
		http.NotFound(w, r)
		return
	}

	nextPath := normalizeLoginNext(r.URL.Query().Get("next"))
	switch r.Method {
	case http.MethodGet:
		s.renderLoginPage(w, "", nextPath)
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			s.renderLoginPage(w, "请求格式错误", nextPath)
			return
		}
		password := r.FormValue("password")
		nextFromForm := normalizeLoginNext(r.FormValue("next"))
		if nextFromForm != "" {
			nextPath = nextFromForm
		}
		if !secureStringEqual(strings.TrimSpace(password), strings.TrimSpace(s.webLoginPassword)) {
			w.WriteHeader(http.StatusUnauthorized)
			s.renderLoginPage(w, "密码错误，请重试。", nextPath)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     webLoginCookieName,
			Value:    s.webSessionToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   requestUsesHTTPS(r),
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(webLoginCookieTTL.Seconds()),
		})
		http.Redirect(w, r, nextPath, http.StatusSeeOther)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/logout" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     webLoginCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   requestUsesHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) renderLoginPage(w http.ResponseWriter, errorMessage string, nextPath string) {
	if nextPath == "" {
		nextPath = "/chat"
	}
	alert := ""
	if strings.TrimSpace(errorMessage) != "" {
		alert = `<p style="margin:0;color:#b91c1c;background:#fef2f2;border:1px solid #fecaca;border-radius:8px;padding:10px 12px;">` + html.EscapeString(strings.TrimSpace(errorMessage)) + `</p>`
	}
	page := `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>alter0 登录</title>
  <style>
    body{margin:0;background:#f8fafc;color:#0f172a;font:14px/1.45 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif}
    .wrap{min-height:100vh;display:flex;align-items:center;justify-content:center;padding:20px}
    .card{width:100%;max-width:360px;background:#fff;border:1px solid #e2e8f0;border-radius:12px;padding:20px;display:grid;gap:12px;box-shadow:0 10px 24px -20px rgba(15,23,42,.55)}
    h1{margin:0;font-size:18px}
    p{margin:0;color:#475569}
    label{font-weight:600;color:#334155}
    input{height:40px;border:1px solid #cbd5e1;border-radius:10px;padding:0 12px;font:inherit}
    button{height:40px;border:1px solid #1d4ed8;background:#1d4ed8;color:#fff;border-radius:10px;font:inherit;font-weight:700;cursor:pointer}
  </style>
</head>
<body>
  <main class="wrap">
    <form class="card" method="post" action="/login">
      <h1>alter0 控制台登录</h1>
      <p>请输入访问密码后继续。</p>
      ` + alert + `
      <input type="hidden" name="next" value="` + html.EscapeString(nextPath) + `">
      <label for="password">密码</label>
      <input id="password" name="password" type="password" autocomplete="current-password" required>
      <button type="submit">登录</button>
    </form>
  </main>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.webLoginEnabled {
			next.ServeHTTP(w, r)
			return
		}
		if isAuthExemptPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if s.isAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		if shouldRedirectToLogin(r) {
			nextPath := normalizeLoginNext(r.URL.RequestURI())
			http.Redirect(w, r, "/login?next="+url.QueryEscape(nextPath), http.StatusTemporaryRedirect)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	})
}

func (s *Server) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(webLoginCookieName)
	if err != nil {
		return false
	}
	return secureStringEqual(strings.TrimSpace(cookie.Value), strings.TrimSpace(s.webSessionToken))
}

func expectsHTMLNavigation(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept")))
	if strings.Contains(accept, "text/html") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("Sec-Fetch-Mode")), "navigate")
}

func shouldRedirectToLogin(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if isInteractivePagePath(r.URL.Path) {
		return true
	}
	return expectsHTMLNavigation(r)
}

func isInteractivePagePath(path string) bool {
	normalized := strings.TrimSpace(path)
	return normalized == "/" || normalized == "/chat"
}

func isAuthExemptPath(path string) bool {
	normalized := strings.TrimSpace(path)
	if normalized == "/healthz" || normalized == "/readyz" || normalized == "/login" || normalized == "/favicon.ico" {
		return true
	}
	return false
}

func normalizeLoginNext(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return "/chat"
	}
	if !strings.HasPrefix(candidate, "/") || strings.HasPrefix(candidate, "//") || strings.HasPrefix(candidate, "/login") {
		return "/chat"
	}
	return candidate
}

func secureStringEqual(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func requestUsesHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	return proto == "https"
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
	assessment := s.assessComplexity(msg)
	if task, accepted, submitErr := s.submitAsyncTask(msg, assessment); accepted {
		taskCard := buildTaskCard(msg, task)
		if submitErr != nil {
			s.logWebMessageFailure(msg, submitErr)
			writeJSON(w, http.StatusInternalServerError, messageResponse{
				Result:                   asyncAcceptedResult(msg, task, assessment, taskCard),
				ExecutionMode:            assessment.ExecutionMode,
				EstimatedDurationSeconds: assessment.EstimatedDurationSeconds,
				ComplexityLevel:          assessment.ComplexityLevel,
				TaskCard:                 taskCard,
				Error:                    submitErr.Error(),
			})
			return
		}
		writeJSON(w, http.StatusAccepted, messageResponse{
			Result:                   asyncAcceptedResult(msg, task, assessment, taskCard),
			TaskID:                   task.ID,
			TaskStatus:               string(task.Status),
			ExecutionMode:            assessment.ExecutionMode,
			EstimatedDurationSeconds: assessment.EstimatedDurationSeconds,
			ComplexityLevel:          assessment.ComplexityLevel,
			TaskCard:                 taskCard,
		})
		return
	}

	result, err := s.orchestrator.Handle(r.Context(), msg)
	result = attachComplexityMetadata(result, assessment, nil)
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
			Result:                   result,
			ExecutionMode:            assessment.ExecutionMode,
			EstimatedDurationSeconds: assessment.EstimatedDurationSeconds,
			ComplexityLevel:          assessment.ComplexityLevel,
			Error:                    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{
		Result:                   result,
		ExecutionMode:            assessment.ExecutionMode,
		EstimatedDurationSeconds: assessment.EstimatedDurationSeconds,
		ComplexityLevel:          assessment.ComplexityLevel,
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

	assessment := s.assessComplexity(msg)
	if task, accepted, submitErr := s.submitAsyncTask(msg, assessment); accepted {
		taskCard := buildTaskCard(msg, task)
		if submitErr != nil {
			s.logWebMessageFailure(msg, submitErr)
			_ = writeSSE(w, "error", streamErrorResponse{
				Error:  submitErr.Error(),
				Result: asyncAcceptedResult(msg, task, assessment, taskCard),
			})
			flusher.Flush()
			return
		}
		_ = writeSSE(w, "done", streamDoneResponse{
			Result:                   asyncAcceptedResult(msg, task, assessment, taskCard),
			TaskID:                   task.ID,
			TaskStatus:               string(task.Status),
			ExecutionMode:            assessment.ExecutionMode,
			EstimatedDurationSeconds: assessment.EstimatedDurationSeconds,
			ComplexityLevel:          assessment.ComplexityLevel,
			TaskCard:                 taskCard,
		})
		flusher.Flush()
		return
	}

	handleStream := func(
		callback func(delta string) error,
	) (shareddomain.OrchestrationResult, error) {
		if orchestrator, ok := s.orchestrator.(StreamOrchestrator); ok {
			return orchestrator.HandleStream(r.Context(), msg, callback)
		}
		return s.orchestrator.Handle(r.Context(), msg)
	}

	result, handleErr := handleStream(func(delta string) error {
		if strings.TrimSpace(delta) == "" {
			return nil
		}
		if err := writeSSE(w, "delta", streamDeltaResponse{
			Delta: delta,
		}); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})
	result = attachComplexityMetadata(result, assessment, nil)
	if handleErr != nil {
		s.logWebMessageFailure(msg, handleErr)
		_ = writeSSE(w, "error", streamErrorResponse{
			Error:  handleErr.Error(),
			Result: result,
		})
		flusher.Flush()
		return
	}

	if _, ok := s.orchestrator.(StreamOrchestrator); !ok {
		for _, chunk := range chunkText(result.Output, 24) {
			if err := writeSSE(w, "delta", streamDeltaResponse{
				Delta: chunk,
				Route: result.Route,
			}); err != nil {
				return
			}
			flusher.Flush()
		}
	}

	_ = writeSSE(w, "done", streamDoneResponse{
		Result:                   result,
		ExecutionMode:            assessment.ExecutionMode,
		EstimatedDurationSeconds: assessment.EstimatedDurationSeconds,
		ComplexityLevel:          assessment.ComplexityLevel,
	})
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
		s.applyTerminalExecutionDefaults(metadata)

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

		acceptedAt := task.AcceptedAt
		if acceptedAt.IsZero() {
			acceptedAt = task.CreatedAt
		}
		writeJSON(w, http.StatusAccepted, taskCreateResponse{
			TaskID:        task.ID,
			Status:        string(task.Status),
			QueuePosition: task.QueuePosition,
			AcceptedAt:    acceptedAt.Format(time.RFC3339Nano),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) applyTerminalExecutionDefaults(metadata map[string]string) {
	if len(metadata) == 0 {
		return
	}
	taskType := strings.ToLower(strings.TrimSpace(metadata[taskapp.MetadataTaskTypeKey]))
	interactive := strings.ToLower(strings.TrimSpace(metadata[taskapp.MetadataTaskTerminalFlagKey]))
	if taskType != "terminal" || interactive != "true" {
		return
	}
	if strings.TrimSpace(metadata[codexSandboxMetadataKey]) == "" {
		metadata[codexSandboxMetadataKey] = codexSandboxDangerFullAccess
	}
	if strings.TrimSpace(metadata[codexWorkspaceModeMetadataKey]) == "" {
		metadata[codexWorkspaceModeMetadataKey] = codexWorkspaceModeRepoRoot
	}
}

func (s *Server) taskItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task service unavailable"})
		return
	}

	taskID, action, artifactID, subAction, ok := taskResourceID(r.URL.Path)
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
	case action == "artifacts" && artifactID == "" && subAction == "" && r.Method == http.MethodGet:
		items, err := s.tasks.ListArtifacts(taskID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		artifactItems, errCode, errMessage, statusCode := mapTaskArtifacts(items)
		if statusCode != 0 {
			writeJSON(w, statusCode, map[string]string{"error": errMessage, "error_code": errCode})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": artifactItems})
	case action == "artifacts" && artifactID != "" && subAction == "download" && r.Method == http.MethodGet:
		s.taskArtifactDeliveryHandler(w, r, taskID, artifactID, false)
	case action == "artifacts" && artifactID != "" && subAction == "preview" && r.Method == http.MethodGet:
		s.taskArtifactDeliveryHandler(w, r, taskID, artifactID, true)
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

func (s *Server) taskArtifactDeliveryHandler(
	w http.ResponseWriter,
	r *http.Request,
	taskID string,
	artifactID string,
	preview bool,
) {
	ctx, cancel := context.WithTimeout(r.Context(), taskArtifactReadTimeout)
	defer cancel()

	artifact, raw, err := s.tasks.ReadArtifact(ctx, taskID, artifactID)
	if err != nil {
		switch {
		case errors.Is(err, taskapp.ErrTaskNotFound), errors.Is(err, taskapp.ErrArtifactNotFound), errors.Is(err, taskapp.ErrArtifactContentAbsent):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":      "artifact not found",
				"error_code": "artifact_not_found",
			})
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			writeJSON(w, http.StatusGatewayTimeout, map[string]string{
				"error":      "artifact read timeout",
				"error_code": "artifact_read_timeout",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":      "artifact read failed",
				"error_code": "artifact_read_failed",
			})
		}
		return
	}

	artifactSize := artifact.Size
	if artifactSize <= 0 {
		artifactSize = int64(len(raw))
	}
	if artifactSize > maxTaskArtifactSizeBytes || int64(len(raw)) > maxTaskArtifactSizeBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
			"error":      "artifact is too large",
			"error_code": "artifact_too_large",
		})
		return
	}

	contentType := strings.TrimSpace(artifact.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if preview {
		if !supportsArtifactPreviewContentType(contentType) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":      "artifact preview is not supported",
				"error_code": "artifact_preview_not_supported",
			})
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
		return
	}

	filename := sanitizeArtifactFilename(artifact.Name, artifact.ArtifactID)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
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
		writeJSON(w, http.StatusOK, s.toTaskControlView(item))
	case action == "retry" && subAction == "" && r.Method == http.MethodPost:
		item, err := s.tasks.Retry(taskID)
		if err != nil {
			s.writeTaskControlError(w, taskID, err)
			return
		}
		writeJSON(w, http.StatusAccepted, s.toTaskControlView(item))
	case action == "cancel" && subAction == "" && r.Method == http.MethodPost:
		item, err := s.tasks.Cancel(taskID)
		if err != nil {
			s.writeTaskControlError(w, taskID, err)
			return
		}
		writeJSON(w, http.StatusOK, s.toTaskControlView(item))
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
	case action == "terminal" && subAction == "input" && r.Method == http.MethodPost:
		s.controlTaskTerminalInputHandler(w, r, taskID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) controlTaskTerminalInputHandler(w http.ResponseWriter, r *http.Request, taskID string) {
	defer r.Body.Close()

	baseTask, exists := s.tasks.Get(taskID)
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	var req controlTaskTerminalInputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	input := strings.TrimSpace(req.Input)
	if input == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "input is required"})
		return
	}

	terminalSessionID := strings.TrimSpace(baseTask.RequestMetadata[controlTaskTerminalSessionIDKey])
	if terminalSessionID == "" {
		terminalSessionID = strings.TrimSpace(baseTask.SessionID)
	}
	if terminalSessionID == "" {
		terminalSessionID = strings.TrimSpace(baseTask.ID)
	}

	maxSessions := s.resolveTerminalSessionLimit()
	activeSessions := s.collectActiveTerminalSessions()
	if _, exists := activeSessions[terminalSessionID]; !exists && len(activeSessions) >= maxSessions {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":               fmt.Sprintf("terminal session limit reached (%d)", maxSessions),
			"error_code":          "terminal_session_limit_reached",
			"max_sessions":        maxSessions,
			"active_sessions":     len(activeSessions),
			"terminal_session_id": terminalSessionID,
		})
		return
	}

	source := resolveControlTaskSource(baseTask)
	channelID := strings.TrimSpace(source.ChannelID)
	if channelID == "" {
		channelID = defaultControlTaskChannelID
	}
	channelType := source.ChannelType
	if channelType == "" {
		channelType = shareddomain.ChannelTypeWeb
	}

	sourceMessageID := s.idGenerator.NewID()
	metadata := cloneStringMap(baseTask.RequestMetadata)
	metadata[taskapp.MetadataTaskAsyncMode] = "force"
	metadata[taskapp.MetadataTaskSourceMessageID] = sourceMessageID
	metadata[taskapp.MetadataTaskChannelIDKey] = channelID
	metadata[taskapp.MetadataTaskChannelTypeKey] = string(channelType)
	metadata[taskapp.MetadataTaskTriggerTypeKey] = string(shareddomain.TriggerTypeUser)
	metadata[taskapp.MetadataTaskTraceIDKey] = s.idGenerator.NewID()
	metadata[controlTaskTerminalSessionIDKey] = terminalSessionID
	metadata[controlTaskTerminalParentIDKey] = strings.TrimSpace(baseTask.ID)
	metadata[controlTaskTerminalInteractiveKey] = "true"

	msg := shareddomain.UnifiedMessage{
		MessageID:     sourceMessageID,
		SessionID:     strings.TrimSpace(baseTask.SessionID),
		UserID:        strings.TrimSpace(metadata[taskapp.MetadataTaskUserIDKey]),
		ChannelID:     channelID,
		ChannelType:   channelType,
		TriggerType:   shareddomain.TriggerTypeUser,
		Content:       input,
		Metadata:      metadata,
		TraceID:       metadata[taskapp.MetadataTaskTraceIDKey],
		CorrelationID: strings.TrimSpace(metadata[taskapp.MetadataTaskCorrelationKey]),
		ReceivedAt:    time.Now().UTC(),
	}

	task, err := s.tasks.Submit(msg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	anchorTaskID := ""
	if req.ReuseTask {
		anchorTaskID = strings.TrimSpace(req.AnchorTaskID)
		if anchorTaskID == "" {
			anchorTaskID = s.resolveTerminalAnchorTaskID(baseTask)
		}
		if anchorTaskID == "" {
			anchorTaskID = strings.TrimSpace(baseTask.ID)
		}
	}

	writeJSON(w, http.StatusAccepted, controlTaskTerminalInputResponse{
		TaskID:            task.ID,
		AnchorTaskID:      anchorTaskID,
		Status:            string(task.Status),
		SessionID:         task.SessionID,
		TerminalSessionID: terminalSessionID,
		MaxSessions:       maxSessions,
		ActiveSessions:    len(activeSessions),
		TaskDetailPath:    "/api/control/tasks/" + strings.TrimSpace(task.ID),
	})
}

func (s *Server) resolveTerminalSessionLimit() int {
	limit := 5
	if s.control != nil {
		configured := s.control.ResolveEnvironmentInt("task_terminal_max_sessions", 0)
		if configured > 0 {
			limit = configured
		} else {
			limit = s.control.ResolveEnvironmentInt("async_task_workers", limit)
		}
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 5 {
		limit = 5
	}
	return limit
}

func (s *Server) collectActiveTerminalSessions() map[string]struct{} {
	sessions := map[string]struct{}{}
	statuses := []taskdomain.TaskStatus{taskdomain.TaskStatusQueued, taskdomain.TaskStatusRunning}
	for _, status := range statuses {
		for page := 1; page <= 16; page++ {
			result := s.tasks.List(taskapp.ListQuery{Status: status, Page: page, PageSize: 200})
			for _, item := range result.Items {
				sessionID := strings.TrimSpace(item.RequestMetadata[controlTaskTerminalSessionIDKey])
				if sessionID == "" {
					sessionID = strings.TrimSpace(item.SessionID)
				}
				if sessionID == "" {
					continue
				}
				sessions[sessionID] = struct{}{}
			}
			if !result.Pagination.HasNext {
				break
			}
		}
	}
	return sessions
}

func (s *Server) resolveTerminalAnchorTaskID(task taskdomain.Task) string {
	current := task
	visited := map[string]struct{}{}
	for hops := 0; hops < 32; hops++ {
		currentID := strings.TrimSpace(current.ID)
		if currentID == "" {
			break
		}
		if _, exists := visited[currentID]; exists {
			break
		}
		visited[currentID] = struct{}{}

		parentID := strings.TrimSpace(current.RequestMetadata[controlTaskTerminalParentIDKey])
		if parentID == "" || parentID == currentID {
			return currentID
		}
		parent, exists := s.tasks.Get(parentID)
		if !exists {
			return currentID
		}
		current = parent
	}
	return strings.TrimSpace(task.ID)
}

func (s *Server) controlTaskCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query, statusCode, err := parseControlTaskListQuery(r)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	tasks := s.collectTasksForControl(query)
	items := make([]controlTaskListItem, 0, len(tasks))
	for _, item := range tasks {
		items = append(items, toControlTaskListItem(item))
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].TaskID > items[j].TaskID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	from, to := memoryTaskPageBounds(len(items), query.Page, query.PageSize)
	pageItems := make([]controlTaskListItem, 0, to-from)
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

func (s *Server) writeTaskControlError(w http.ResponseWriter, taskID string, err error) {
	if errors.Is(err, taskapp.ErrTaskNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	if errors.Is(err, taskapp.ErrTaskConflict) {
		if item, exists := s.tasks.Get(taskID); exists {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": err.Error(),
				"view":  s.toTaskControlView(item),
			})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func (s *Server) toTaskControlView(task taskdomain.Task) taskControlView {
	return taskControlView{
		Task:    task,
		Source:  resolveControlTaskSource(task),
		Actions: resolveTaskControlActions(task),
		Link:    s.resolveTaskSessionLink(task),
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

func (s *Server) resolveTaskSessionLink(task taskdomain.Task) taskSessionLink {
	sessionID := strings.TrimSpace(task.SessionID)
	taskID := strings.TrimSpace(task.ID)
	terminalSessionID := strings.TrimSpace(task.RequestMetadata[controlTaskTerminalSessionIDKey])
	if terminalSessionID == "" {
		terminalSessionID = sessionID
	}
	link := taskSessionLink{
		TaskID:              taskID,
		SessionID:           sessionID,
		TerminalSessionID:   terminalSessionID,
		TerminalMaxSessions: s.resolveTerminalSessionLimit(),
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
		artifactItems, errCode, errMessage, statusCode := mapTaskArtifacts(items)
		if statusCode != 0 {
			writeJSON(w, statusCode, map[string]string{"error": errMessage, "error_code": errCode})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": artifactItems})
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

func (s *Server) collectTasksForControl(query controlTaskListQuery) []taskdomain.Task {
	items := make([]taskdomain.Task, 0, 64)
	page := 1
	for {
		result := s.tasks.List(taskapp.ListQuery{
			SessionID: query.SessionID,
			Status:    query.Status,
			Page:      page,
			PageSize:  200,
		})
		if len(result.Items) == 0 {
			break
		}
		for _, item := range result.Items {
			if !matchControlTaskFilters(item, query) {
				continue
			}
			items = append(items, item)
		}
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

func toControlTaskListItem(task taskdomain.Task) controlTaskListItem {
	errorText := strings.TrimSpace(task.ErrorMessage)
	if errorText == "" {
		errorText = strings.TrimSpace(task.ErrorCode)
	}
	source := resolveControlTaskSource(task)
	return controlTaskListItem{
		TaskID:          strings.TrimSpace(task.ID),
		SessionID:       strings.TrimSpace(task.SessionID),
		SourceMessageID: resolveTaskSourceMessageID(task),
		Status:          task.Status,
		Phase:           strings.TrimSpace(task.Phase),
		Progress:        task.Progress,
		QueuePosition:   task.QueuePosition,
		QueueWaitMS:     task.QueueWaitMS,
		RetryCount:      task.RetryCount,
		TriggerType:     source.TriggerType,
		ChannelType:     source.ChannelType,
		ChannelID:       source.ChannelID,
		CorrelationID:   source.CorrelationID,
		JobID:           source.JobID,
		JobName:         source.JobName,
		FiredAt:         source.FiredAt,
		CreatedAt:       task.CreatedAt.UTC(),
		StartedAt:       task.StartedAt.UTC(),
		UpdatedAt:       resolveControlTaskUpdatedAt(task),
		FinishedAt:      task.FinishedAt.UTC(),
		Error:           errorText,
	}
}

func resolveControlTaskUpdatedAt(task taskdomain.Task) time.Time {
	if !task.UpdatedAt.IsZero() {
		return task.UpdatedAt.UTC()
	}
	if !task.FinishedAt.IsZero() {
		return task.FinishedAt.UTC()
	}
	if !task.CreatedAt.IsZero() {
		return task.CreatedAt.UTC()
	}
	return time.Now().UTC()
}

func matchControlTaskFilters(task taskdomain.Task, query controlTaskListQuery) bool {
	source := resolveControlTaskSource(task)
	if strings.TrimSpace(string(query.TriggerType)) != "" && source.TriggerType != query.TriggerType {
		return false
	}
	if strings.TrimSpace(string(query.ChannelType)) != "" && source.ChannelType != query.ChannelType {
		return false
	}
	if query.ChannelID != "" && !strings.EqualFold(source.ChannelID, query.ChannelID) {
		return false
	}
	if query.SourceMessageID != "" && !strings.EqualFold(resolveTaskSourceMessageID(task), query.SourceMessageID) {
		return false
	}
	if query.MessageID != "" && !matchTaskMessageID(task, query.MessageID) {
		return false
	}
	at := resolveControlTaskUpdatedAt(task)
	if !query.StartAt.IsZero() && at.Before(query.StartAt) {
		return false
	}
	if !query.EndAt.IsZero() && at.After(query.EndAt) {
		return false
	}
	return true
}

func resolveTaskSourceMessageID(task taskdomain.Task) string {
	if sourceMessageID := strings.TrimSpace(task.SourceMessageID); sourceMessageID != "" {
		return sourceMessageID
	}
	if requestMessageID := strings.TrimSpace(task.MessageLink.RequestMessageID); requestMessageID != "" {
		return requestMessageID
	}
	if messageID := strings.TrimSpace(task.MessageID); messageID != "" {
		return messageID
	}
	return ""
}

func matchTaskMessageID(task taskdomain.Task, messageID string) bool {
	target := strings.TrimSpace(messageID)
	if target == "" {
		return true
	}
	candidates := []string{
		strings.TrimSpace(task.SourceMessageID),
		strings.TrimSpace(task.MessageID),
		strings.TrimSpace(task.MessageLink.RequestMessageID),
		strings.TrimSpace(task.MessageLink.ResultMessageID),
	}
	for _, candidate := range candidates {
		if strings.EqualFold(candidate, target) {
			return true
		}
	}
	return false
}

func resolveControlTaskSource(task taskdomain.Task) controlTaskSource {
	metadata := task.RequestMetadata
	triggerType := parseControlTriggerType(metadata[taskapp.MetadataTaskTriggerTypeKey])
	channelType := parseControlChannelType(metadata[taskapp.MetadataTaskChannelTypeKey])
	channelID := strings.TrimSpace(metadata[taskapp.MetadataTaskChannelIDKey])
	if channelID == "" {
		channelID = defaultControlTaskChannelID
	}
	source := controlTaskSource{
		TriggerType:   triggerType,
		ChannelType:   channelType,
		ChannelID:     channelID,
		CorrelationID: strings.TrimSpace(metadata[taskapp.MetadataTaskCorrelationKey]),
	}
	if source.TriggerType == shareddomain.TriggerTypeCron {
		source.JobID = strings.TrimSpace(metadata[controlTaskMetadataJobIDKey])
		source.JobName = strings.TrimSpace(metadata[controlTaskMetadataJobNameKey])
		if source.CorrelationID == "" {
			source.CorrelationID = source.JobID
		}
		if firedAt, err := parseRFC3339Time(metadata[controlTaskMetadataFiredAtKey]); err == nil {
			source.FiredAt = firedAt
		}
	}
	return source
}

func parseControlTriggerType(raw string) shareddomain.TriggerType {
	switch shareddomain.TriggerType(strings.ToLower(strings.TrimSpace(raw))) {
	case shareddomain.TriggerTypeUser:
		return shareddomain.TriggerTypeUser
	case shareddomain.TriggerTypeCron:
		return shareddomain.TriggerTypeCron
	case shareddomain.TriggerTypeSystem:
		return shareddomain.TriggerTypeSystem
	default:
		return shareddomain.TriggerTypeUser
	}
}

func parseControlChannelType(raw string) shareddomain.ChannelType {
	switch shareddomain.ChannelType(strings.ToLower(strings.TrimSpace(raw))) {
	case shareddomain.ChannelTypeCLI:
		return shareddomain.ChannelTypeCLI
	case shareddomain.ChannelTypeWeb:
		return shareddomain.ChannelTypeWeb
	case shareddomain.ChannelTypeScheduler:
		return shareddomain.ChannelTypeScheduler
	default:
		return shareddomain.ChannelTypeWeb
	}
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

func (s *Server) environmentConfigHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		revealSensitive := parseBoolFlag(r.URL.Query().Get("reveal_sensitive"))
		items := s.control.ListEnvironmentConfigs(revealSensitive)
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPut:
		defer r.Body.Close()
		var req environmentUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		values := normalizeEnvironmentValues(req.Values)
		if len(values) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "values are required"})
			return
		}
		result, err := s.control.UpdateEnvironmentConfigs(values, req.Operator)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) environmentAuditListHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	revealSensitive := parseBoolFlag(r.URL.Query().Get("reveal_sensitive"))
	writeJSON(w, http.StatusOK, map[string]any{"items": s.control.ListEnvironmentAudits(revealSensitive)})
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

	jobID, subResource, ok := cronJobResourceID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid cron job path"})
		return
	}
	if subResource == "runs" {
		s.cronJobRunsHandler(w, r, jobID)
		return
	}
	if subResource != "" {
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

		interval := time.Duration(0)
		intervalRaw := strings.TrimSpace(req.Interval)
		if intervalRaw != "" {
			parsed, err := time.ParseDuration(intervalRaw)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid interval, e.g. 5m or 1h"})
				return
			}
			interval = parsed
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		retryLimit := 0
		if req.TaskConfig.RetryLimit != nil {
			retryLimit = *req.TaskConfig.RetryLimit
		}
		job := schedulerdomain.Job{
			ID:             jobID,
			Name:           strings.TrimSpace(req.Name),
			Interval:       interval,
			Enabled:        enabled,
			SessionID:      strings.TrimSpace(req.SessionID),
			UserID:         strings.TrimSpace(req.UserID),
			ChannelID:      strings.TrimSpace(req.ChannelID),
			Content:        strings.TrimSpace(req.Content),
			Metadata:       cloneStringMap(req.Metadata),
			ScheduleMode:   schedulerdomain.ScheduleMode(strings.ToLower(strings.TrimSpace(req.ScheduleMode))),
			Timezone:       strings.TrimSpace(req.Timezone),
			CronExpression: strings.TrimSpace(req.CronExpression),
			TaskConfig: schedulerdomain.TaskConfig{
				Input:      strings.TrimSpace(req.TaskConfig.Input),
				RetryLimit: retryLimit,
			},
		}
		normalized, err := job.Normalize()
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := s.scheduler.Upsert(normalized); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, toCronJobResponse(normalized))
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

func (s *Server) cronJobRunsHandler(w http.ResponseWriter, r *http.Request, jobID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.sessions == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session history unavailable"})
		return
	}

	query, statusCode, err := parseSessionQuery(r)
	if err != nil {
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}
	query.TriggerType = shareddomain.TriggerTypeCron
	query.JobID = strings.TrimSpace(jobID)

	page := s.sessions.ListSessions(query)
	items := make([]cronJobRunResponse, 0, len(page.Items))
	for _, session := range page.Items {
		status := "success"
		if strings.TrimSpace(session.LastErrorCode) != "" {
			status = "failed"
		}
		firedAt := session.FiredAt
		if firedAt.IsZero() {
			firedAt = session.StartedAt
		}
		items = append(items, cronJobRunResponse{
			RunID:     strings.TrimSpace(session.SessionID),
			JobID:     strings.TrimSpace(jobID),
			FiredAt:   firedAt.UTC(),
			SessionID: strings.TrimSpace(session.SessionID),
			Status:    status,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"pagination": page.Pagination,
	})
}

func toCronJobResponse(job schedulerdomain.Job) cronJobResponse {
	interval := ""
	if job.Interval > 0 {
		interval = job.Interval.String()
	}
	return cronJobResponse{
		ID:             job.ID,
		Name:           job.Name,
		Enabled:        job.Enabled,
		Timezone:       job.Timezone,
		ScheduleMode:   string(job.ScheduleMode),
		CronExpression: job.CronExpression,
		TaskConfig: cronTaskConfigResponse{
			Input:      job.TaskConfig.Input,
			RetryLimit: job.TaskConfig.RetryLimit,
		},
		UserID:    job.UserID,
		ChannelID: job.ChannelID,
		Metadata:  job.Metadata,
		Interval:  interval,
		SessionID: job.SessionID,
		Content:   job.Content,
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

func cronJobResourceID(path string) (string, string, bool) {
	const prefix = "/api/control/cron/jobs/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		id := strings.TrimSpace(parts[0])
		if id == "" {
			return "", "", false
		}
		return id, "", true
	}
	if len(parts) == 2 {
		id := strings.TrimSpace(parts[0])
		subResource := strings.TrimSpace(parts[1])
		if id == "" || subResource == "" {
			return "", "", false
		}
		return id, subResource, true
	}
	return "", "", false
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

func taskResourceID(path string) (string, string, string, string, bool) {
	const prefix = "/api/tasks/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", "", false
	}

	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		taskID := strings.TrimSpace(parts[0])
		if taskID == "" {
			return "", "", "", "", false
		}
		return taskID, "", "", "", true
	}
	if len(parts) == 2 {
		taskID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		if taskID == "" || action == "" {
			return "", "", "", "", false
		}
		return taskID, action, "", "", true
	}
	if len(parts) == 4 {
		taskID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		artifactID := strings.TrimSpace(parts[2])
		subAction := strings.TrimSpace(parts[3])
		if taskID == "" || action != "artifacts" || artifactID == "" || subAction == "" {
			return "", "", "", "", false
		}
		return taskID, action, artifactID, subAction, true
	}
	return "", "", "", "", false
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

	query := sessionapp.SessionQuery{
		StartAt:   startAt,
		EndAt:     endAt,
		Page:      page,
		PageSize:  pageSize,
		ChannelID: strings.TrimSpace(r.URL.Query().Get("channel_id")),
		MessageID: strings.TrimSpace(r.URL.Query().Get("message_id")),
		JobID:     strings.TrimSpace(r.URL.Query().Get("job_id")),
	}
	rawTriggerType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("trigger_type")))
	if rawTriggerType != "" {
		triggerType := shareddomain.TriggerType(rawTriggerType)
		switch triggerType {
		case shareddomain.TriggerTypeUser, shareddomain.TriggerTypeCron, shareddomain.TriggerTypeSystem:
			query.TriggerType = triggerType
		default:
			return sessionapp.SessionQuery{}, http.StatusBadRequest, errors.New("trigger_type must be user/cron/system")
		}
	}
	if query.JobID != "" && query.TriggerType == "" {
		query.TriggerType = shareddomain.TriggerTypeCron
	}
	rawChannelType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("channel_type")))
	if rawChannelType != "" {
		channelType := shareddomain.ChannelType(rawChannelType)
		switch channelType {
		case shareddomain.ChannelTypeCLI, shareddomain.ChannelTypeWeb, shareddomain.ChannelTypeScheduler:
			query.ChannelType = channelType
		default:
			return sessionapp.SessionQuery{}, http.StatusBadRequest, errors.New("channel_type must be cli/web/scheduler")
		}
	}

	return query, http.StatusOK, nil
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

func parseControlTaskListQuery(r *http.Request) (controlTaskListQuery, int, error) {
	page, err := parsePositiveInt(r.URL.Query().Get("page"))
	if err != nil {
		return controlTaskListQuery{}, http.StatusBadRequest, errors.New("page must be a positive integer")
	}
	pageSize, err := parsePositiveInt(r.URL.Query().Get("page_size"))
	if err != nil {
		return controlTaskListQuery{}, http.StatusBadRequest, errors.New("page_size must be a positive integer")
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
		return controlTaskListQuery{}, statusCode, err
	}

	rawTimeRange := strings.TrimSpace(r.URL.Query().Get("time_range"))
	if rawTimeRange != "" {
		parsedStart, parsedEnd, parseErr := parseControlTimeRange(rawTimeRange)
		if parseErr != nil {
			return controlTaskListQuery{}, http.StatusBadRequest, parseErr
		}
		if !parsedStart.IsZero() {
			startAt = parsedStart
		}
		if !parsedEnd.IsZero() {
			endAt = parsedEnd
		}
	}

	if !startAt.IsZero() && !endAt.IsZero() && endAt.Before(startAt) {
		return controlTaskListQuery{}, http.StatusBadRequest, errors.New("time range is invalid")
	}

	query := controlTaskListQuery{
		SessionID:       strings.TrimSpace(r.URL.Query().Get("session_id")),
		ChannelID:       strings.TrimSpace(r.URL.Query().Get("channel_id")),
		MessageID:       strings.TrimSpace(r.URL.Query().Get("message_id")),
		SourceMessageID: strings.TrimSpace(r.URL.Query().Get("source_message_id")),
		StartAt:         startAt,
		EndAt:           endAt,
		Page:            page,
		PageSize:        pageSize,
	}
	rawStatus := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if rawStatus != "" {
		status := taskdomain.TaskStatus(rawStatus)
		if !status.IsValid() {
			return controlTaskListQuery{}, http.StatusBadRequest, errors.New("status must be queued/running/success/failed/canceled")
		}
		query.Status = status
	}
	rawTriggerType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("trigger_type")))
	if rawTriggerType != "" {
		triggerType := shareddomain.TriggerType(rawTriggerType)
		switch triggerType {
		case shareddomain.TriggerTypeUser, shareddomain.TriggerTypeCron, shareddomain.TriggerTypeSystem:
			query.TriggerType = triggerType
		default:
			return controlTaskListQuery{}, http.StatusBadRequest, errors.New("trigger_type must be user/cron/system")
		}
	}
	rawChannelType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("channel_type")))
	if rawChannelType != "" {
		channelType := shareddomain.ChannelType(rawChannelType)
		switch channelType {
		case shareddomain.ChannelTypeCLI, shareddomain.ChannelTypeWeb, shareddomain.ChannelTypeScheduler:
			query.ChannelType = channelType
		default:
			return controlTaskListQuery{}, http.StatusBadRequest, errors.New("channel_type must be cli/web/scheduler")
		}
	}
	return query, http.StatusOK, nil
}

func parseControlTimeRange(raw string) (time.Time, time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	now := time.Now().UTC()
	switch strings.ToLower(trimmed) {
	case "last_1h":
		return now.Add(-1 * time.Hour), now, nil
	case "last_24h":
		return now.Add(-24 * time.Hour), now, nil
	case "last_7d":
		return now.Add(-7 * 24 * time.Hour), now, nil
	}

	parts := strings.Split(trimmed, ",")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, errors.New("time_range must be last_1h/last_24h/last_7d or start,end in RFC3339")
	}
	startAt, err := parseRFC3339Time(parts[0])
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("time_range start must be RFC3339 format")
	}
	endAt, err := parseRFC3339Time(parts[1])
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("time_range end must be RFC3339 format")
	}
	return startAt, endAt, nil
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

func (s *Server) submitAsyncTask(msg shareddomain.UnifiedMessage, assessment taskapp.ComplexityAssessment) (taskdomain.Task, bool, error) {
	if s.tasks == nil {
		return taskdomain.Task{}, false, nil
	}
	if strings.ToLower(strings.TrimSpace(assessment.ExecutionMode)) != taskapp.ExecutionModeAsync {
		return taskdomain.Task{}, false, nil
	}
	item, err := s.tasks.Submit(msg)
	if err != nil {
		return taskdomain.Task{}, true, err
	}
	return item, true, nil
}

func (s *Server) assessComplexity(msg shareddomain.UnifiedMessage) taskapp.ComplexityAssessment {
	if s.tasks == nil {
		return taskapp.ComplexityAssessment{
			EstimatedDurationSeconds: 10,
			ComplexityLevel:          taskapp.ComplexityLevelLow,
			ExecutionMode:            taskapp.ExecutionModeStreaming,
		}
	}
	return s.tasks.AssessComplexity(msg)
}

func buildTaskCard(msg shareddomain.UnifiedMessage, task taskdomain.Task) *taskCardResponse {
	taskID := strings.TrimSpace(task.ID)
	if taskID == "" {
		return nil
	}
	taskSummary := summaryText(strings.TrimSpace(msg.Content), 120)
	if taskSummary == "" {
		taskSummary = "任务摘要待补充"
	}
	return &taskCardResponse{
		Notice:        "当前任务较复杂，已创建后台任务执行，请稍后",
		TaskID:        taskID,
		TaskSummary:   taskSummary,
		TaskDetailURL: "/api/control/tasks/" + taskID,
	}
}

func asyncAcceptedResult(
	msg shareddomain.UnifiedMessage,
	task taskdomain.Task,
	assessment taskapp.ComplexityAssessment,
	taskCard *taskCardResponse,
) shareddomain.OrchestrationResult {
	metadata := map[string]string{
		taskapp.MetadataTaskIDKey: task.ID,
	}
	if strings.TrimSpace(string(task.Status)) != "" {
		metadata[taskapp.MetadataTaskStatusKey] = string(task.Status)
	}
	metadata[taskapp.MetadataExecutionMode] = taskapp.ExecutionModeAsync
	metadata[taskapp.MetadataComplexityLevel] = strings.TrimSpace(assessment.ComplexityLevel)
	metadata[taskapp.MetadataEstimatedDurationSeconds] = strconv.Itoa(assessment.EstimatedDurationSeconds)
	if assessment.Fallback {
		metadata[taskapp.MetadataComplexityFallback] = "true"
	}
	output := ""
	if taskCard != nil {
		output = taskCard.Notice + "\n"
		output += "task_id: " + taskCard.TaskID + "\n"
		output += "task_summary: " + taskCard.TaskSummary + "\n"
		output += "task_detail_url: " + taskCard.TaskDetailURL
		metadata["task_summary"] = taskCard.TaskSummary
		metadata["task_detail_url"] = taskCard.TaskDetailURL
	}
	return shareddomain.OrchestrationResult{
		MessageID: msg.MessageID,
		SessionID: msg.SessionID,
		ErrorCode: "task_accepted",
		Output:    output,
		Metadata:  metadata,
	}
}

func attachComplexityMetadata(
	result shareddomain.OrchestrationResult,
	assessment taskapp.ComplexityAssessment,
	taskCard *taskCardResponse,
) shareddomain.OrchestrationResult {
	metadata := cloneStringMap(result.Metadata)
	metadata[taskapp.MetadataExecutionMode] = strings.TrimSpace(assessment.ExecutionMode)
	metadata[taskapp.MetadataComplexityLevel] = strings.TrimSpace(assessment.ComplexityLevel)
	metadata[taskapp.MetadataEstimatedDurationSeconds] = strconv.Itoa(assessment.EstimatedDurationSeconds)
	if assessment.Fallback {
		metadata[taskapp.MetadataComplexityFallback] = "true"
	}
	if taskCard != nil {
		metadata["task_summary"] = strings.TrimSpace(taskCard.TaskSummary)
		metadata["task_detail_url"] = strings.TrimSpace(taskCard.TaskDetailURL)
	}
	result.Metadata = metadata
	return result
}

func summaryText(content string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 120
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes]) + "..."
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

func mapTaskArtifacts(items []taskdomain.TaskArtifact) ([]taskArtifactResponse, string, string, int) {
	if len(items) == 0 {
		return []taskArtifactResponse{}, "", "", 0
	}
	if len(items) > maxTaskArtifactCount {
		return nil, "artifact_count_exceeded", "artifact count exceeded", http.StatusRequestEntityTooLarge
	}
	out := make([]taskArtifactResponse, 0, len(items))
	for _, item := range items {
		downloadURL := strings.TrimSpace(item.DownloadURL)
		if downloadURL == "" {
			continue
		}
		artifact := taskArtifactResponse{
			ArtifactID:  strings.TrimSpace(item.ArtifactID),
			Name:        strings.TrimSpace(item.Name),
			ContentType: strings.TrimSpace(item.ContentType),
			Size:        item.Size,
			Summary:     strings.TrimSpace(item.Summary),
			DownloadURL: downloadURL,
			PreviewURL:  strings.TrimSpace(item.PreviewURL),
			CreatedAt:   item.CreatedAt,
		}
		if artifact.ContentType == "" {
			artifact.ContentType = "application/octet-stream"
		}
		if artifact.Name == "" {
			artifact.Name = artifact.ArtifactID
		}
		out = append(out, artifact)
	}
	return out, "", "", 0
}

func supportsArtifactPreviewContentType(contentType string) bool {
	lower := strings.ToLower(strings.TrimSpace(contentType))
	if lower == "" {
		return false
	}
	if strings.HasPrefix(lower, "text/") || strings.HasPrefix(lower, "image/") {
		return true
	}
	switch lower {
	case "application/json", "application/xml", "application/yaml", "application/x-yaml", "application/javascript", "application/pdf", "application/xhtml+xml":
		return true
	default:
		return false
	}
}

func sanitizeArtifactFilename(name string, fallback string) string {
	value := strings.TrimSpace(name)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		value = "artifact.bin"
	}
	value = strings.ReplaceAll(value, "\"", "_")
	value = strings.ReplaceAll(value, "\n", "_")
	value = strings.ReplaceAll(value, "\r", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	return value
}

func parseBoolFlag(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func normalizeEnvironmentValues(input map[string]any) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	values := make(map[string]string, len(input))
	for key, raw := range input {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		switch value := raw.(type) {
		case string:
			values[trimmedKey] = value
		case float64:
			if value == float64(int64(value)) {
				values[trimmedKey] = strconv.FormatInt(int64(value), 10)
			} else {
				values[trimmedKey] = strconv.FormatFloat(value, 'f', -1, 64)
			}
		case bool:
			values[trimmedKey] = strconv.FormatBool(value)
		case nil:
			values[trimmedKey] = ""
		default:
			values[trimmedKey] = strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	}
	return values
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
