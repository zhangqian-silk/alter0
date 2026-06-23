package web

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	codexapp "alter0/internal/codex/application"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	llmdomain "alter0/internal/llm/domain"
	orchdomain "alter0/internal/orchestration/domain"
	schedulerapp "alter0/internal/scheduler/application"
	schedulerdomain "alter0/internal/scheduler/domain"
	sessionapp "alter0/internal/session/application"
	sessiondomain "alter0/internal/session/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
	taskapp "alter0/internal/task/application"
	taskdomain "alter0/internal/task/domain"
	terminalapp "alter0/internal/terminal/application"
	terminaldomain "alter0/internal/terminal/domain"
)

//go:embed static/dist
var webStaticFS embed.FS

const (
	canonicalChatSessionID                        = sessiondomain.CanonicalChatSessionID
	codexSandboxMetadataKey                       = "codex_sandbox"
	codexSandboxDangerFullAccess                  = "danger-full-access"
	maxTaskArtifactCount                          = 128
	webLoginCookieName                            = "alter0_web_session_host"
	legacySharedWebLoginCookieName                = "alter0_web_session"
	webLoginCookieTTL                             = 24 * time.Hour
	webPageCacheControl                           = "no-cache"
	bridgeStaticAssetCacheControl                 = "no-cache"
	immutableStaticAssetCacheControl              = "public, max-age=31536000, immutable"
	frontendDevOriginEnvKey                       = "ALTER0_WEB_FRONTEND_DEV_ORIGIN"
	conversationRuntimeExecutionEngineMetadataKey = "alter0.execution.engine"
	conversationRuntimeExecutionEngineCodex       = "codex"
	conversationRuntimeLLMProviderMetadataKey     = "alter0.llm.provider_id"
	conversationRuntimeLLMModelMetadataKey        = "alter0.llm.model"
	conversationRuntimeCodexProviderID            = "alter0-codex"
	conversationRuntimeCodexModelID               = "codex"
)

var workbenchPagePaths = map[string]struct{}{
	"/chat":     {},
	"/terminal": {},
	"/settings": {},
}

type Orchestrator interface {
	Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error)
}

type intentInspector interface {
	Classify(content string) orchdomain.Intent
}

type Server struct {
	addr                        string
	orchestrator                Orchestrator
	telemetry                   *observability.Telemetry
	idGenerator                 sharedapp.IDGenerator
	control                     *controlapp.Service
	scheduler                   *schedulerapp.Manager
	sessions                    sessionHistoryService
	tasks                       taskService
	terminals                   terminalService
	runtime                     runtimeRestarter
	runtimeInfo                 runtimeInfoProvider
	memory                      *memoryContextService
	llm                         llmService
	logger                      *slog.Logger
	webLoginPassword            string
	webSessionToken             string
	webLoginEnabled             bool
	webBindLocalhost            bool
	workspaceRoot               string
	frontendDevOrigin           string
	frontendDevProxy            http.Handler
	workspaceService            *workspaceServiceRegistry
	conversationRuntimeSessions *conversationRuntimeSessionRegistry
	workspaceRuntime            workspaceServiceRuntime
	codexAccounts               codexAccountService
	maintenance                 *maintenanceService
}

type llmService interface {
	GetConfig(ctx context.Context) (*llmdomain.ModelConfig, error)
	GetProvider(ctx context.Context, providerID string) (*llmdomain.ModelProvider, error)
	GetDefaultProvider(ctx context.Context) (*llmdomain.ModelProvider, error)
	GetEnabledProviders(ctx context.Context) ([]llmdomain.ModelProvider, error)
	AddProvider(ctx context.Context, provider llmdomain.ModelProvider) error
	UpdateProvider(ctx context.Context, currentProviderID string, provider llmdomain.ModelProvider) error
	RemoveProvider(ctx context.Context, providerID string) error
	SetDefaultProvider(ctx context.Context, providerID string) error
	EnableProvider(ctx context.Context, providerID string, enabled bool) error
}

type sessionHistoryService interface {
	ListSessions(query sessionapp.SessionQuery) sessionapp.SessionPage
	ListMessages(query sessionapp.MessageQuery) sessionapp.MessagePage
	DeleteSession(sessionID string) error
}

type taskService interface {
	List(query taskapp.ListQuery) taskapp.TaskPage
	Get(taskID string) (taskdomain.Task, bool)
	ListBySession(sessionID string) []taskdomain.Task
	ListLogs(taskID string, cursor int, limit int) (taskapp.TaskLogPage, error)
	ListArtifacts(taskID string) ([]taskdomain.TaskArtifact, error)
	DeleteBySession(sessionID string) error
}

type terminalService interface {
	Create(req terminalapp.CreateRequest) (terminaldomain.Session, error)
	Recover(req terminalapp.RecoverRequest) (terminaldomain.Session, error)
	List(ownerID string) []terminaldomain.Session
	Get(ownerID string, sessionID string) (terminaldomain.Session, bool)
	ListTurns(ownerID string, sessionID string) ([]terminalapp.TurnSummary, error)
	GetStepDetail(ownerID string, sessionID string, turnID string, stepID string) (terminalapp.StepDetail, error)
	ListEntries(ownerID string, sessionID string, cursor int, limit int) (terminalapp.EntryPage, error)
	Input(ownerID string, sessionID string, input string) (terminaldomain.Session, error)
	InputWithAttachments(req terminalapp.InputRequest) (terminaldomain.Session, error)
	SetPinned(ownerID string, sessionID string, pinned bool) (terminaldomain.Session, error)
	Delete(ownerID string, sessionID string) (terminaldomain.Session, error)
}

type runtimeRestarter interface {
	RequestRestart(options RuntimeRestartOptions) (bool, error)
	GetRestartStatus() RuntimeRestartStatus
}

type runtimeInfoProvider interface {
	GetRuntimeInfo() RuntimeInfo
}

type codexAccountService interface {
	ListStatuses(ctx context.Context) ([]codexapp.AccountStatus, *codexapp.CurrentStatus, error)
	AddFromRaw(name string, raw []byte, overwrite bool) (*codexapp.Record, error)
	Switch(name string) (*codexapp.Record, string, error)
	StartLoginSession(ctx context.Context, name string, overwrite bool) (codexapp.LoginSession, error)
	GetLoginSession(id string) (codexapp.LoginSession, bool)
	RuntimeStatus() (*codexapp.RuntimeStatus, error)
	UpdateRuntimeSettings(model string, reasoningEffort string) (*codexapp.RuntimeStatus, error)
}

type RuntimeRestartOptions struct {
	SyncRemoteMaster             bool `json:"sync_remote_master"`
	ConfirmDiscardTrackedChanges bool `json:"confirm_discard_tracked_changes"`
}

type RuntimeRestartStatus struct {
	Status                       string    `json:"status"`
	Error                        string    `json:"error,omitempty"`
	SyncRemoteMaster             bool      `json:"sync_remote_master"`
	ConfirmDiscardTrackedChanges bool      `json:"confirm_discard_tracked_changes"`
	StartedAt                    time.Time `json:"started_at,omitempty"`
	UpdatedAt                    time.Time `json:"updated_at,omitempty"`
}

type RuntimeInfo struct {
	StartedAt  time.Time `json:"started_at,omitempty"`
	CommitHash string    `json:"commit_hash,omitempty"`
}

type codexAccountCreateRequest struct {
	Name            string `json:"name"`
	Overwrite       bool   `json:"overwrite"`
	AuthFileContent string `json:"auth_file_content"`
}

type codexAccountLoginSessionCreateRequest struct {
	Name      string `json:"name"`
	Overwrite bool   `json:"overwrite"`
}

type codexRuntimeUpdateRequest struct {
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort"`
}

type messageRequest struct {
	SessionID     string                     `json:"session_id"`
	UserID        string                     `json:"user_id,omitempty"`
	ChannelID     string                     `json:"channel_id,omitempty"`
	CorrelationID string                     `json:"correlation_id,omitempty"`
	Content       string                     `json:"content"`
	Attachments   []messageAttachmentRequest `json:"attachments,omitempty"`
	Metadata      map[string]string          `json:"metadata,omitempty"`
}

type messageAttachmentRequest struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name"`
	ContentType    string `json:"content_type"`
	DataURL        string `json:"data_url,omitempty"`
	PreviewDataURL string `json:"preview_data_url,omitempty"`
	AssetURL       string `json:"asset_url,omitempty"`
	PreviewURL     string `json:"preview_url,omitempty"`
}

type messageResponse struct {
	Result shareddomain.OrchestrationResult `json:"result"`
	Error  string                           `json:"error,omitempty"`
}

type conversationRuntimeRoute string

const (
	conversationRuntimeRouteChat       conversationRuntimeRoute = "chat"
	conversationRuntimeTitleMaxRunes                            = 32
	conversationRuntimeSessionPageSize                          = 200
)

type conversationRuntimeSessionCollectionResponse struct {
	Items []conversationRuntimeSessionResponse `json:"items"`
}

type conversationRuntimeSessionItemResponse struct {
	Session conversationRuntimeSessionResponse `json:"session"`
}

type conversationRuntimeSessionPatchRequest struct {
	Title           *string   `json:"title"`
	ModelProviderID *string   `json:"model_provider_id"`
	ModelID         *string   `json:"model_id"`
	ToolIDs         *[]string `json:"tool_ids"`
	SkillIDs        *[]string `json:"skill_ids"`
	MCPIDs          *[]string `json:"mcp_ids"`
}

type conversationRuntimeSessionResponse struct {
	ID              string                               `json:"id"`
	Status          string                               `json:"status,omitempty"`
	Title           string                               `json:"title"`
	TitleAuto       bool                                 `json:"title_auto"`
	TitleScore      int                                  `json:"title_score"`
	Pinned          bool                                 `json:"pinned,omitempty"`
	CreatedAt       time.Time                            `json:"created_at"`
	TargetType      string                               `json:"target_type"`
	TargetID        string                               `json:"target_id,omitempty"`
	TargetName      string                               `json:"target_name,omitempty"`
	ModelProviderID string                               `json:"model_provider_id,omitempty"`
	ModelID         string                               `json:"model_id,omitempty"`
	ToolIDs         []string                             `json:"tool_ids,omitempty"`
	SkillIDs        []string                             `json:"skill_ids,omitempty"`
	MCPIDs          []string                             `json:"mcp_ids,omitempty"`
	Messages        []conversationRuntimeMessageResponse `json:"messages,omitempty"`
}

type conversationRuntimeMessageResponse struct {
	ID           string                          `json:"id"`
	Role         string                          `json:"role"`
	Text         string                          `json:"text"`
	Attachments  []conversationRuntimeAttachment `json:"attachments,omitempty"`
	Route        string                          `json:"route,omitempty"`
	Source       string                          `json:"source,omitempty"`
	Error        bool                            `json:"error"`
	Status       string                          `json:"status"`
	At           time.Time                       `json:"at"`
	ProcessSteps []shareddomain.ProcessStep      `json:"process_steps,omitempty"`
	TaskID       string                          `json:"task_id,omitempty"`
	TaskStatus   string                          `json:"task_status,omitempty"`
}

type conversationRuntimeAttachment struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	AssetURL    string `json:"asset_url,omitempty"`
	PreviewURL  string `json:"preview_url,omitempty"`
}

type taskArtifactResponse struct {
	ArtifactID  string    `json:"artifact_id"`
	Name        string    `json:"name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Summary     string    `json:"summary,omitempty"`
	DownloadURL string    `json:"download_url,omitempty"`
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
	Builtin        bool                   `json:"builtin,omitempty"`
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
	UpdatedAt  time.Time             `json:"updated_at"`
	Tags       []string              `json:"tags,omitempty"`
}

type memoryTaskMeta struct {
	TaskID          string                `json:"task_id"`
	SessionID       string                `json:"session_id"`
	SourceMessageID string                `json:"source_message_id"`
	Status          taskdomain.TaskStatus `json:"status"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	FinishedAt      time.Time             `json:"finished_at,omitempty"`
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
	terminals terminalService,
	memoryOptions MemoryContextOptions,
	securityOptions WebSecurityOptions,
	llm llmService,
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
	frontendDevOrigin := resolveFrontendDevOrigin()
	workspaceRoot := resolveServerWorkspaceRoot()
	workspaceServiceRegistryPath := filepath.Join(workspaceRoot, ".alter0", workspaceServiceRegistryFilename)
	workspaceServiceRegistry, err := newFileWorkspaceServiceRegistry(workspaceServiceRegistryPath, "alter0.cn")
	if err != nil && logger != nil {
		logger.Error("failed to initialize workspace service registry", slog.String("error", err.Error()))
	}
	conversationRuntimeRegistryPath := filepath.Join(workspaceRoot, ".alter0", conversationRuntimeSessionRegistryFilename)
	conversationRuntimeRegistry, err := newFileConversationRuntimeSessionRegistry(conversationRuntimeRegistryPath)
	if err != nil && logger != nil {
		logger.Error("failed to initialize conversation runtime session registry", slog.String("error", err.Error()))
	}
	server := &Server{
		addr:                        addr,
		orchestrator:                orchestrator,
		telemetry:                   telemetry,
		idGenerator:                 idGenerator,
		control:                     control,
		scheduler:                   scheduler,
		sessions:                    sessions,
		tasks:                       tasks,
		terminals:                   terminals,
		memory:                      newMemoryContextService(memoryOptions),
		llm:                         llm,
		logger:                      logger,
		webLoginPassword:            resolvedPassword,
		webSessionToken:             webSessionToken,
		webLoginEnabled:             resolvedPassword != "",
		webBindLocalhost:            resolvedBindLocalhost,
		workspaceRoot:               workspaceRoot,
		frontendDevOrigin:           frontendDevOrigin,
		frontendDevProxy:            newFrontendDevProxy(frontendDevOrigin, logger),
		workspaceService:            workspaceServiceRegistry,
		conversationRuntimeSessions: conversationRuntimeRegistry,
		workspaceRuntime:            newWorkspaceServiceRuntime(logger),
	}
	server.ensureMaintenanceService()
	server.registerMaintenanceSchedulerJobs()
	return server
}

func (s *Server) ensureMaintenanceService() {
	if s == nil || s.maintenance != nil {
		return
	}
	s.maintenance = newMaintenanceService(s, s.idGenerator, s.logger)
}

func (s *Server) SetRuntimeRestarter(restarter runtimeRestarter) {
	if s == nil {
		return
	}
	s.runtime = restarter
}

func (s *Server) SetRuntimeInfoProvider(provider runtimeInfoProvider) {
	if s == nil {
		return
	}
	s.runtimeInfo = provider
}

func (s *Server) SetCodexAccountService(service codexAccountService) {
	if s == nil {
		return
	}
	s.codexAccounts = service
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", s.telemetry.MetricsHandler())
	mux.HandleFunc("/healthz", s.healthHandler)
	mux.HandleFunc("/readyz", s.readyHandler)
	mux.HandleFunc("/login", s.loginHandler)
	mux.HandleFunc("/logout", s.logoutHandler)
	mux.HandleFunc("/", s.rootHandler)
	for path := range workbenchPagePaths {
		mux.HandleFunc(path, s.chatPageHandler)
	}
	mux.HandleFunc("/api/messages", s.messageHandler)
	mux.HandleFunc("/api/conversation-runtime/sessions", s.conversationRuntimeSessionCollectionHandler)
	mux.HandleFunc("/api/conversation-runtime/sessions/", s.conversationRuntimeSessionItemHandler)
	mux.HandleFunc("/api/sessions", s.sessionListHandler)
	mux.HandleFunc("/api/sessions/", s.sessionMessageListHandler)
	mux.HandleFunc("/api/memory/context", s.memoryContextHandler)
	mux.HandleFunc("/api/memory/tasks", s.memoryTaskCollectionHandler)
	mux.HandleFunc("/api/memory/tasks/", s.memoryTaskItemHandler)
	mux.HandleFunc("/api/control/workspace-services", s.workspaceServiceCollectionHandler)
	mux.HandleFunc("/api/control/workspace-services/", s.workspaceServiceItemHandler)
	mux.HandleFunc("/api/control/runtime", s.runtimeInfoHandler)
	mux.HandleFunc("/api/control/runtime/restart", s.runtimeRestartHandler)
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
	mux.HandleFunc("/api/control/codex/accounts", s.codexAccountCollectionHandler)
	mux.HandleFunc("/api/control/codex/accounts/login-sessions", s.codexAccountLoginSessionCollectionHandler)
	mux.HandleFunc("/api/control/codex/accounts/", s.codexAccountItemHandler)
	mux.HandleFunc("/api/control/codex/runtime", s.codexRuntimeHandler)
	mux.HandleFunc("/api/control/llm/providers", s.llmProviderListHandler)
	mux.HandleFunc("/api/control/llm/providers/", s.llmProviderItemHandler)
	mux.HandleFunc("/api/terminal/sessions", s.terminalSessionCollectionHandler)
	mux.HandleFunc("/api/terminal/sessions/recover", s.terminalSessionRecoverHandler)
	mux.HandleFunc("/api/terminal/sessions/", s.terminalSessionItemHandler)

	assetsFS, err := webAssetFS("assets")
	if err != nil {
		return err
	}
	mux.Handle("/assets/", cacheControlledFileServer("/assets/", assetsFS, immutableStaticAssetCacheControl))
	if legacyFS, legacyErr := webAssetFS("legacy"); legacyErr == nil {
		mux.Handle("/legacy/", cacheControlledFileServer("/legacy/", legacyFS, bridgeStaticAssetCacheControl))
	}

	handler := s.withWorkspaceServiceGateway(http.Handler(mux))
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
	if s.shouldProxyFrontendDevRequest(r.URL.Path) {
		s.serveFrontendDevProxy(w, r)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/chat", http.StatusTemporaryRedirect)
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	nextPath := normalizeLoginNext(r.URL.Query().Get("next"))
	if !s.webLoginEnabled {
		http.Redirect(w, r, nextPath, http.StatusTemporaryRedirect)
		return
	}
	if r.URL.Path != "/login" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.renderLoginPage(w, "", nextPath)
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			s.renderLoginPage(w, "Invalid request payload.", nextPath)
			return
		}
		password := r.FormValue("password")
		nextFromForm := normalizeLoginNext(r.FormValue("next"))
		if nextFromForm != "" {
			nextPath = nextFromForm
		}
		if !secureStringEqual(strings.TrimSpace(password), strings.TrimSpace(s.webLoginPassword)) {
			w.WriteHeader(http.StatusUnauthorized)
			s.renderLoginPage(w, "Incorrect password. Please try again.", nextPath)
			return
		}
		cookie := &http.Cookie{
			Name:     webLoginCookieName,
			Value:    s.webSessionToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   requestUsesHTTPS(r),
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(webLoginCookieTTL.Seconds()),
		}
		http.SetCookie(w, cookie)
		s.clearLegacySharedLoginCookie(w, r)
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
	cookie := &http.Cookie{
		Name:     webLoginCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   requestUsesHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
	s.clearLegacySharedLoginCookie(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) renderLoginPage(w http.ResponseWriter, errorMessage string, nextPath string) {
	if nextPath == "" {
		nextPath = "/chat"
	}
	alert := ""
	if strings.TrimSpace(errorMessage) != "" {
		alert = `<p class="alert">` + html.EscapeString(strings.TrimSpace(errorMessage)) + `</p>`
	}
	page := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
  <title>Alter0 Login</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=Sora:wght@600;700&display=swap" rel="stylesheet">
  <style>
    :root{color-scheme:light}
    *{box-sizing:border-box}
    html{height:100%;background:#f6f7f9}
    body{margin:0;min-height:100vh;min-height:100dvh;overflow:hidden;overscroll-behavior:none;background:#f6f7f9;background-image:linear-gradient(180deg,#fff 0%,#f6f7f9 48%,#eef2f5 100%);color:#101828;font:15px/1.6 "IBM Plex Sans","Segoe UI",sans-serif}
    .wrap{min-height:100vh;min-height:100dvh;display:grid;place-items:center;padding:max(18px,env(safe-area-inset-top)) 24px;padding-bottom:max(18px,env(safe-area-inset-bottom))}
    .card{width:min(100%,420px);max-height:100%;display:grid;gap:18px;padding:28px;border:1px solid rgba(15,23,42,.1);border-radius:8px;background:#fff;box-shadow:0 20px 46px -38px rgba(15,23,42,.28);overflow:auto}
    .eyebrow{display:inline-flex;align-items:center;min-height:26px;width:max-content;padding:0 9px;border-radius:8px;border:1px solid rgba(15,159,143,.18);background:rgba(15,159,143,.08);color:#087f73;font-size:12px;font-weight:700;letter-spacing:.02em;text-transform:uppercase}
    .copy{display:grid;gap:8px}
    h1{margin:0;font:700 30px/1.12 "Sora","IBM Plex Sans","Segoe UI",sans-serif;letter-spacing:0}
    p{margin:0;color:#667085}
    .lede{color:#344054}
    .field{display:grid;gap:8px}
    label{font-size:12px;font-weight:700;letter-spacing:.02em;color:#667085}
    .password-control{position:relative}
    input{width:100%;height:46px;border:1px solid rgba(15,23,42,.12);border-radius:8px;padding:0 46px 0 13px;background:#fff;color:#101828;font:inherit}
    input:focus{outline:2px solid rgba(15,159,143,.32);outline-offset:2px;border-color:rgba(15,159,143,.34)}
    .password-toggle{position:absolute;top:5px;right:5px;width:36px;height:36px;border:0;border-radius:8px;background:transparent;color:#667085;font:inherit;font-size:12px;font-weight:700;cursor:pointer}
    .password-toggle:hover{background:#f2f4f7;color:#101828}
    button[type=submit]{height:46px;border:1px solid #101828;background:#101828;color:#fff;border-radius:8px;font:inherit;font-weight:700;cursor:pointer}
    button[type=submit]:hover{background:#1d2939;border-color:#1d2939}
    .alert{min-height:44px;margin:0;color:#b42318;background:#fef3f2;border:1px solid #fecdca;border-radius:8px;padding:10px 12px}
    .status{display:flex;align-items:center;gap:8px;min-height:28px;color:#087f73;font-size:13px;font-weight:600}
    .status-dot{width:8px;height:8px;border-radius:999px;background:#0f9f8f;box-shadow:0 0 0 3px rgba(15,159,143,.12)}
    .meta{padding-top:14px;border-top:1px solid rgba(15,23,42,.08);font-size:13px;color:#667085}
    @media (max-width: 640px){
      body{font-size:14px}
      .wrap{padding:max(14px,env(safe-area-inset-top)) 14px;padding-bottom:max(14px,env(safe-area-inset-bottom))}
      .card{gap:16px;padding:20px;border-radius:8px}
      h1{font-size:26px}
      p{line-height:1.55}
    }
  </style>
</head>
<body>
  <main class="wrap">
    <form class="card" method="post" action="/login">
      <span class="eyebrow">Alter0 workspace</span>
      <div class="copy">
        <h1>Alter0 Console Login</h1>
        <p class="lede">Start in a secure Alter0 workspace.</p>
        <p>Enter the access password to continue into chat, terminal, and settings from one private workbench.</p>
      </div>
      ` + alert + `
      <input type="hidden" name="next" value="` + html.EscapeString(nextPath) + `">
      <div class="field">
        <label for="password">Password</label>
        <div class="password-control">
          <input id="password" name="password" type="password" autocomplete="current-password" required>
          <button class="password-toggle" type="button" aria-label="Show password" data-password-toggle>Show</button>
        </div>
      </div>
      <p class="status"><span class="status-dot" aria-hidden="true"></span>Ready.</p>
      <button type="submit">Sign in</button>
      <p class="meta">After sign-in, Alter0 returns to ` + html.EscapeString(nextPath) + `.</p>
    </form>
  </main>
  <script>
    const toggle = document.querySelector("[data-password-toggle]");
    const input = document.querySelector("#password");
    if (toggle && input) {
      toggle.addEventListener("click", () => {
        const hidden = input.type === "password";
        input.type = hidden ? "text" : "password";
        toggle.textContent = hidden ? "Hide" : "Show";
        toggle.setAttribute("aria-label", hidden ? "Hide password" : "Show password");
        input.focus();
      });
    }
  </script>
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
		if s.isPublicWorkspaceReadOnlyHost(r.Host) {
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
			nextPath := loginNextForRequest(r)
			http.Redirect(w, r, "/login?next="+url.QueryEscape(nextPath), http.StatusTemporaryRedirect)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	})
}

func (s *Server) isPublicWorkspaceReadOnlyHost(host string) bool {
	if s == nil || s.workspaceService == nil {
		return false
	}
	entry, ok := s.workspaceService.ResolveHost(host)
	return ok && entry.PublicReadOnly
}

func (s *Server) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(webLoginCookieName)
	if err != nil {
		return false
	}
	return secureStringEqual(strings.TrimSpace(cookie.Value), strings.TrimSpace(s.webSessionToken))
}

func (s *Server) clearLegacySharedLoginCookie(w http.ResponseWriter, r *http.Request) {
	domain := s.resolveLegacySharedLoginCookieDomain(r.Host)
	if domain == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     legacySharedWebLoginCookieName,
		Value:    "",
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   requestUsesHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (s *Server) resolveLegacySharedLoginCookieDomain(host string) string {
	if s == nil || s.workspaceService == nil {
		return ""
	}
	baseDomain := normalizePreviewBaseDomain(s.workspaceService.baseDomain)
	normalizedHost := normalizePreviewHost(host)
	if baseDomain == "" || normalizedHost == "" {
		return ""
	}
	if normalizedHost == baseDomain || strings.HasSuffix(normalizedHost, "."+baseDomain) {
		return baseDomain
	}
	return ""
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
	if normalized == "/" {
		return true
	}
	_, ok := workbenchPagePaths[normalized]
	return ok
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
	pathOnly := candidate
	if index := strings.IndexAny(pathOnly, "?#"); index >= 0 {
		pathOnly = pathOnly[:index]
	}
	if _, ok := workbenchPagePaths[pathOnly]; ok {
		return pathOnly
	}
	return candidate
}

func loginNextForRequest(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "/chat"
	}
	if isInteractivePagePath(r.URL.Path) {
		return normalizeLoginNext(r.URL.Path)
	}
	return normalizeLoginNext(r.URL.RequestURI())
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
	if _, ok := workbenchPagePaths[r.URL.Path]; !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.frontendDevProxy != nil {
		s.serveFrontendDevProxy(w, r)
		return
	}

	content, err := readWebShellPage()
	if err != nil {
		s.logger.Error("chat page unavailable", slog.String("error", err.Error()))
		http.Error(w, "chat page unavailable", http.StatusInternalServerError)
		return
	}
	content = versionWebShellAssetReferences(content, readEmbeddedWebDistAsset)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", webPageCacheControl)
	_, _ = w.Write(content)
}

func readWebShellPage() ([]byte, error) {
	return webStaticFS.ReadFile("static/dist/index.html")
}

func readEmbeddedWebDistAsset(assetPath string) ([]byte, error) {
	return webStaticFS.ReadFile(filepath.ToSlash(filepath.Join("static", "dist", assetPath)))
}

var webShellAssetReferencePattern = regexp.MustCompile(`((?:src|href)="\/)(assets\/index-[^"?]+\.(?:js|css))(?:\?v=[^"]*)?(")`)

func versionWebShellAssetReferences(content []byte, readAsset func(string) ([]byte, error)) []byte {
	if len(content) == 0 || readAsset == nil {
		return content
	}
	html := string(content)
	versioned := webShellAssetReferencePattern.ReplaceAllStringFunc(html, func(match string) string {
		parts := webShellAssetReferencePattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		assetContent, err := readAsset(parts[2])
		if err != nil || len(assetContent) == 0 {
			return match
		}
		return parts[1] + parts[2] + "?v=" + shortAssetContentHash(assetContent) + parts[3]
	})
	return []byte(versioned)
}

func shortAssetContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:12]
}

func webAssetFS(name string) (fs.FS, error) {
	distPath := filepath.ToSlash(filepath.Join("static", "dist", name))
	return fs.Sub(webStaticFS, distPath)
}

func cacheControlledFileServer(prefix string, assets fs.FS, cacheControl string) http.Handler {
	fileServer := http.StripPrefix(prefix, http.FileServer(http.FS(assets)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cacheControl != "" {
			w.Header().Set("Cache-Control", cacheControl)
		}
		fileServer.ServeHTTP(w, r)
	})
}

func resolveFrontendDevOrigin() string {
	raw := strings.TrimSpace(os.Getenv(frontendDevOriginEnvKey))
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func newFrontendDevProxy(origin string, logger *slog.Logger) http.Handler {
	trimmedOrigin := strings.TrimSpace(origin)
	if trimmedOrigin == "" {
		return nil
	}
	target, err := url.Parse(trimmedOrigin)
	if err != nil {
		return nil
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		if logger != nil {
			logger.Error("frontend dev proxy failed",
				slog.String("origin", trimmedOrigin),
				slog.String("path", r.URL.Path),
				slog.String("error", proxyErr.Error()),
			)
		}
		http.Error(w, "frontend dev server unavailable", http.StatusBadGateway)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", webPageCacheControl)
		proxy.ServeHTTP(w, r)
	})
}

func (s *Server) shouldProxyFrontendDevRequest(requestPath string) bool {
	if s == nil || s.frontendDevProxy == nil {
		return false
	}
	return isFrontendDevAssetPath(requestPath)
}

func (s *Server) serveFrontendDevProxy(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.frontendDevProxy == nil {
		http.NotFound(w, r)
		return
	}
	s.frontendDevProxy.ServeHTTP(w, r)
}

func isFrontendDevAssetPath(requestPath string) bool {
	switch {
	case requestPath == "/@react-refresh":
		return true
	case requestPath == "/index.html":
		return true
	case requestPath == "/vite.svg":
		return true
	case strings.HasPrefix(requestPath, "/@vite/"):
		return true
	case strings.HasPrefix(requestPath, "/@fs/"):
		return true
	case strings.HasPrefix(requestPath, "/@id/"):
		return true
	case strings.HasPrefix(requestPath, "/src/"):
		return true
	case strings.HasPrefix(requestPath, "/node_modules/"):
		return true
	default:
		return false
	}
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
	s.markConversationRuntimeSessionStarted(conversationRuntimeRouteChat, msg)
	s.touchSessionActivityAt(msg.SessionID, msg.ReceivedAt)

	result, err := s.orchestrator.Handle(r.Context(), msg)
	if err != nil {
		s.markConversationRuntimeSessionFinished(conversationRuntimeRouteChat, msg, conversationRuntimeSessionStatusFailed)
		s.touchSessionActivity(msg.SessionID)
		statusCode := http.StatusBadRequest
		switch result.ErrorCode {
		case "command_failed", "agent_execution_failed":
			statusCode = http.StatusInternalServerError
		}
		writeJSON(w, statusCode, messageResponse{
			Result: result,
			Error:  err.Error(),
		})
		return
	}
	s.markConversationRuntimeSessionFinished(conversationRuntimeRouteChat, msg, conversationRuntimeSessionStatusReady)
	s.touchSessionActivity(msg.SessionID)

	writeJSON(w, http.StatusOK, messageResponse{
		Result: result,
	})
}

func (s *Server) messageStreamHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusGone, map[string]string{"error": "chat stream endpoint removed; use POST /api/messages"})
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

func (s *Server) conversationRuntimeSessionCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.sessions == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session history unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	route, ok := parseConversationRuntimeRoute(r.URL.Query().Get("route"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid runtime route"})
		return
	}

	page := s.sessions.ListSessions(sessionapp.SessionQuery{
		TriggerType: shareddomain.TriggerTypeUser,
		ChannelType: shareddomain.ChannelTypeWeb,
		Page:        1,
		PageSize:    conversationRuntimeSessionPageSize,
	})
	itemsByID := make(map[string]conversationRuntimeSessionResponse, len(page.Items))
	for _, summary := range page.Items {
		detail, ok, err := s.loadConversationRuntimeSession(route, summary.SessionID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if ok {
			detail.Pinned = summary.Pinned
			itemsByID[detail.ID] = s.mergeConversationRuntimeSessionWithRegistry(route, detail)
		}
	}
	for _, entry := range s.listConversationRuntimeRegistryEntries(route) {
		if _, ok := itemsByID[entry.SessionID]; ok {
			continue
		}
		itemsByID[entry.SessionID] = s.conversationRuntimeSessionResponseFromRegistryEntry(entry)
	}
	items := make([]conversationRuntimeSessionResponse, 0, len(itemsByID))
	for _, item := range itemsByID {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		left := conversationRuntimeSessionSortTime(items[i])
		right := conversationRuntimeSessionSortTime(items[j])
		if left.Equal(right) {
			return items[i].ID < items[j].ID
		}
		return left.After(right)
	})
	writeJSON(w, http.StatusOK, conversationRuntimeSessionCollectionResponse{Items: items})
}

func (s *Server) conversationRuntimeSessionItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.sessions == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session history unavailable"})
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	sessionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/conversation-runtime/sessions/"))
	if sessionID == "" || strings.Contains(sessionID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid runtime session path"})
		return
	}
	route, ok := parseConversationRuntimeRoute(r.URL.Query().Get("route"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid runtime route"})
		return
	}
	if r.Method == http.MethodPatch {
		s.patchConversationRuntimeSession(w, r, route, sessionID)
		return
	}

	session, matched, err := s.loadConversationRuntimeSession(route, sessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !matched {
		registryEntry, ok := s.resolveConversationRuntimeRegistryEntry(route, sessionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "runtime session not found"})
			return
		}
		s.touchSessionActivity(sessionID)
		writeJSON(w, http.StatusOK, conversationRuntimeSessionItemResponse{
			Session: s.conversationRuntimeSessionResponseFromRegistryEntry(registryEntry),
		})
		return
	}
	s.touchSessionActivity(sessionID)
	writeJSON(w, http.StatusOK, conversationRuntimeSessionItemResponse{
		Session: s.mergeConversationRuntimeSessionWithRegistry(route, session),
	})
}

func (s *Server) patchConversationRuntimeSession(w http.ResponseWriter, r *http.Request, route conversationRuntimeRoute, sessionID string) {
	if s.conversationRuntimeSessions == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "runtime session registry unavailable"})
		return
	}
	defer r.Body.Close()

	var req conversationRuntimeSessionPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	entry := conversationRuntimeSessionRegistryEntry{
		SessionID:  sessionID,
		Route:      route,
		Status:     conversationRuntimeSessionStatusReady,
		Title:      "New",
		TitleAuto:  true,
		CreatedAt:  time.Now().UTC(),
		TargetType: "model",
		TargetID:   "raw-model",
		TargetName: "Raw Model",
	}
	if current, ok := s.resolveConversationRuntimeRegistryEntry(route, sessionID); ok {
		entry = current
	} else if session, matched, err := s.loadConversationRuntimeSession(route, sessionID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	} else if matched {
		entry = conversationRuntimeRegistryEntryFromSessionResponse(route, session)
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			title = "New"
		}
		entry.Title = title
		entry.TitleAuto = false
		entry.TitleScore = 1
	}
	if req.ModelProviderID != nil {
		entry.ModelProviderID = strings.TrimSpace(*req.ModelProviderID)
	}
	if req.ModelID != nil {
		entry.ModelID = strings.TrimSpace(*req.ModelID)
	}
	if req.ToolIDs != nil {
		entry.ToolIDs = normalizeConversationRuntimeRegistryList(*req.ToolIDs)
		if entry.ToolIDs == nil {
			entry.ToolIDs = []string{}
		}
	}
	if req.SkillIDs != nil {
		entry.SkillIDs = normalizeConversationRuntimeRegistryList(*req.SkillIDs)
		if entry.SkillIDs == nil {
			entry.SkillIDs = []string{}
		}
	}
	if req.MCPIDs != nil {
		entry.MCPIDs = normalizeConversationRuntimeRegistryList(*req.MCPIDs)
		if entry.MCPIDs == nil {
			entry.MCPIDs = []string{}
		}
	}
	entry.UpdatedAt = time.Now().UTC()

	patched, err := s.conversationRuntimeSessions.Upsert(entry)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, conversationRuntimeSessionItemResponse{
		Session: s.conversationRuntimeSessionResponseFromRegistryEntry(patched),
	})
}

func (s *Server) touchSessionActivity(sessionID string) {
	s.touchSessionActivityAt(sessionID, time.Now().UTC())
}

func (s *Server) touchSessionActivityAt(sessionID string, at time.Time) {
	touchService, ok := s.sessions.(sessionTouchService)
	if !ok {
		return
	}
	_ = touchService.TouchSession(sessionID, at)
}

func (s *Server) sessionMessageListHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, resource, resourceID, action, ok := sessionResourceID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session path"})
		return
	}

	switch resource {
	case "":
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		if s.sessions == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session history unavailable"})
			return
		}
		if s.tasks != nil {
			if err := s.tasks.DeleteBySession(sessionID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		if err := s.sessions.DeleteSession(sessionID); err != nil && !errors.Is(err, sessionapp.ErrSessionNotFound) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.deleteConversationRuntimeSessionRegistryEntry(sessionID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := removeConversationSessionWorkspace(s.workspaceRoot, sessionID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	case "messages":
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
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
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
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
	case "pin":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		pinService, ok := s.sessions.(sessionPinService)
		if !ok {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session pinning unavailable"})
			return
		}
		var request struct {
			Pinned *bool `json:"pinned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pin request"})
			return
		}
		if request.Pinned == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pinned is required"})
			return
		}
		if err := pinService.SetSessionPinned(sessionID, *request.Pinned); err != nil {
			if !errors.Is(err, sessionapp.ErrSessionNotFound) || !s.setConversationRuntimeSessionPinned(sessionID, *request.Pinned) {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"session_id": sessionID,
			"pinned":     *request.Pinned,
		})
	case "attachments":
		switch {
		case resourceID == "" && action == "":
			s.handleSessionAttachmentUpload(w, r, sessionID)
		case resourceID != "" && (action == "original" || action == "preview"):
			s.handleSessionAttachmentRead(w, r, sessionID, resourceID, action)
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session path"})
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session path"})
		return
	}
}

func (s *Server) loadConversationRuntimeSession(
	route conversationRuntimeRoute,
	sessionID string,
) (conversationRuntimeSessionResponse, bool, error) {
	page := s.sessions.ListMessages(sessionapp.MessageQuery{
		SessionID: strings.TrimSpace(sessionID),
		Page:      1,
		PageSize:  conversationRuntimeSessionPageSize,
	})
	if len(page.Items) == 0 {
		return conversationRuntimeSessionResponse{}, false, nil
	}
	session, ok := buildConversationRuntimeSession(route, page.Items)
	if !ok {
		return conversationRuntimeSessionResponse{}, false, nil
	}
	return session, true, nil
}

func (s *Server) listConversationRuntimeRegistryEntries(route conversationRuntimeRoute) []conversationRuntimeSessionRegistryEntry {
	if s == nil || s.conversationRuntimeSessions == nil {
		return nil
	}
	return s.conversationRuntimeSessions.List(route)
}

func (s *Server) resolveConversationRuntimeRegistryEntry(route conversationRuntimeRoute, sessionID string) (conversationRuntimeSessionRegistryEntry, bool) {
	if s == nil || s.conversationRuntimeSessions == nil {
		return conversationRuntimeSessionRegistryEntry{}, false
	}
	return s.conversationRuntimeSessions.Resolve(route, sessionID)
}

func (s *Server) setConversationRuntimeSessionPinned(sessionID string, pinned bool) bool {
	if s == nil || s.conversationRuntimeSessions == nil {
		return false
	}
	_, err := s.conversationRuntimeSessions.SetPinned(conversationRuntimeRouteChat, sessionID, pinned)
	return err == nil
}

func (s *Server) conversationRuntimeSessionResponseFromRegistryEntry(entry conversationRuntimeSessionRegistryEntry) conversationRuntimeSessionResponse {
	return conversationRuntimeSessionResponse{
		ID:              entry.SessionID,
		Status:          normalizeConversationRuntimeSessionStatus(entry.Status),
		Title:           entry.Title,
		TitleAuto:       entry.TitleAuto,
		TitleScore:      entry.TitleScore,
		Pinned:          entry.Pinned,
		CreatedAt:       entry.CreatedAt.UTC(),
		TargetType:      entry.TargetType,
		TargetID:        entry.TargetID,
		TargetName:      entry.TargetName,
		ModelProviderID: entry.ModelProviderID,
		ModelID:         entry.ModelID,
		ToolIDs:         append([]string(nil), entry.ToolIDs...),
		SkillIDs:        append([]string(nil), entry.SkillIDs...),
		MCPIDs:          append([]string(nil), entry.MCPIDs...),
	}
}

func conversationRuntimeRegistryEntryFromSessionResponse(route conversationRuntimeRoute, session conversationRuntimeSessionResponse) conversationRuntimeSessionRegistryEntry {
	return conversationRuntimeSessionRegistryEntry{
		SessionID:       strings.TrimSpace(session.ID),
		Route:           route,
		Status:          normalizeConversationRuntimeSessionStatus(session.Status),
		Title:           strings.TrimSpace(session.Title),
		TitleAuto:       session.TitleAuto,
		TitleScore:      session.TitleScore,
		Pinned:          session.Pinned,
		CreatedAt:       session.CreatedAt.UTC(),
		UpdatedAt:       time.Now().UTC(),
		TargetType:      session.TargetType,
		TargetID:        session.TargetID,
		TargetName:      session.TargetName,
		ModelProviderID: session.ModelProviderID,
		ModelID:         session.ModelID,
		ToolIDs:         append([]string(nil), session.ToolIDs...),
		SkillIDs:        append([]string(nil), session.SkillIDs...),
		MCPIDs:          append([]string(nil), session.MCPIDs...),
	}
}

func (s *Server) mergeConversationRuntimeSessionWithRegistry(route conversationRuntimeRoute, session conversationRuntimeSessionResponse) conversationRuntimeSessionResponse {
	entry, ok := s.resolveConversationRuntimeRegistryEntry(route, session.ID)
	if !ok {
		return session
	}
	merged := s.conversationRuntimeSessionResponseFromRegistryEntry(entry)
	merged.Messages = session.Messages
	if merged.Title == "" {
		merged.Title = session.Title
		merged.TitleAuto = session.TitleAuto
		merged.TitleScore = session.TitleScore
	}
	if merged.CreatedAt.IsZero() {
		merged.CreatedAt = session.CreatedAt
	}
	if merged.TargetType == "" {
		merged.TargetType = session.TargetType
		merged.TargetID = session.TargetID
		merged.TargetName = session.TargetName
	}
	if merged.ModelProviderID == "" {
		merged.ModelProviderID = session.ModelProviderID
	}
	if merged.ModelID == "" {
		merged.ModelID = session.ModelID
	}
	if len(merged.ToolIDs) == 0 {
		merged.ToolIDs = append([]string(nil), session.ToolIDs...)
	}
	if len(merged.SkillIDs) == 0 {
		merged.SkillIDs = append([]string(nil), session.SkillIDs...)
	}
	if len(merged.MCPIDs) == 0 {
		merged.MCPIDs = append([]string(nil), session.MCPIDs...)
	}
	if merged.Status == "" {
		merged.Status = session.Status
	}
	merged.Pinned = session.Pinned
	return merged
}

func (s *Server) markConversationRuntimeSessionStarted(route conversationRuntimeRoute, msg shareddomain.UnifiedMessage) {
	if s == nil || s.conversationRuntimeSessions == nil {
		return
	}
	entry := buildConversationRuntimeRegistryEntryFromMessage(route, msg, conversationRuntimeSessionStatusBusy)
	if _, err := s.conversationRuntimeSessions.Upsert(entry); err != nil && s.logger != nil {
		s.logger.Error("failed to persist conversation runtime session start",
			slog.String("session_id", msg.SessionID),
			slog.String("route", string(route)),
			slog.String("error", err.Error()),
		)
	}
}

func (s *Server) markConversationRuntimeSessionFinished(route conversationRuntimeRoute, msg shareddomain.UnifiedMessage, status string) {
	if s == nil || s.conversationRuntimeSessions == nil {
		return
	}
	entry := buildConversationRuntimeRegistryEntryFromMessage(route, msg, status)
	if _, err := s.conversationRuntimeSessions.Upsert(entry); err != nil && s.logger != nil {
		s.logger.Error("failed to persist conversation runtime session finish",
			slog.String("session_id", msg.SessionID),
			slog.String("route", string(route)),
			slog.String("status", normalizeConversationRuntimeSessionStatus(status)),
			slog.String("error", err.Error()),
		)
	}
}

func (s *Server) deleteConversationRuntimeSessionRegistryEntry(sessionID string) error {
	if s == nil || s.conversationRuntimeSessions == nil {
		return nil
	}
	return s.conversationRuntimeSessions.Delete(sessionID)
}

func conversationRuntimeSessionSortTime(item conversationRuntimeSessionResponse) time.Time {
	if !item.CreatedAt.IsZero() {
		return item.CreatedAt.UTC()
	}
	return time.Time{}
}

func parseConversationRuntimeRoute(raw string) (conversationRuntimeRoute, bool) {
	switch conversationRuntimeRoute(strings.TrimSpace(raw)) {
	case conversationRuntimeRouteChat:
		return conversationRuntimeRouteChat, true
	default:
		return "", false
	}
}

func parseStoredConversationRuntimeRoute(raw string) (conversationRuntimeRoute, bool, bool) {
	switch strings.TrimSpace(raw) {
	case string(conversationRuntimeRouteChat):
		return conversationRuntimeRouteChat, false, true
	default:
		return "", false, false
	}
}

func buildConversationRuntimeSession(
	route conversationRuntimeRoute,
	records []sessiondomain.MessageRecord,
) (conversationRuntimeSessionResponse, bool) {
	if len(records) == 0 {
		return conversationRuntimeSessionResponse{}, false
	}

	sessionID := strings.TrimSpace(records[0].SessionID)
	targetType, targetID, targetName := resolveConversationRuntimeTarget(records)

	title := deriveConversationRuntimeTitle(records)
	modelProviderID, modelID := resolveConversationRuntimeModel(records)
	toolIDs, skillIDs, mcpIDs := resolveConversationRuntimeCapabilities(records)
	messages := make([]conversationRuntimeMessageResponse, 0, len(records))
	for _, record := range records {
		messages = append(messages, buildConversationRuntimeMessage(record))
	}

	return conversationRuntimeSessionResponse{
		ID:              sessionID,
		Status:          deriveConversationRuntimeSessionStatus(records),
		Title:           title,
		TitleAuto:       false,
		TitleScore:      1,
		CreatedAt:       records[0].Timestamp.UTC(),
		TargetType:      targetType,
		TargetID:        targetID,
		TargetName:      targetName,
		ModelProviderID: modelProviderID,
		ModelID:         modelID,
		ToolIDs:         toolIDs,
		SkillIDs:        skillIDs,
		MCPIDs:          mcpIDs,
		Pinned:          conversationRuntimeSessionPinned(records),
		Messages:        messages,
	}, true
}

func conversationRuntimeSessionPinned(records []sessiondomain.MessageRecord) bool {
	for _, record := range records {
		switch strings.ToLower(strings.TrimSpace(record.Metadata["alter0.session.pinned"])) {
		case "true", "1", "yes", "on":
			return true
		}
	}
	return false
}

func deriveConversationRuntimeSessionStatus(records []sessiondomain.MessageRecord) string {
	for idx := len(records) - 1; idx >= 0; idx-- {
		record := records[idx]
		if record.Role != sessiondomain.MessageRoleAssistant {
			continue
		}
		if strings.TrimSpace(record.RouteResult.ErrorCode) != "" {
			return conversationRuntimeSessionStatusFailed
		}
		return conversationRuntimeSessionStatusReady
	}
	return conversationRuntimeSessionStatusReady
}

func buildConversationRuntimeMessage(record sessiondomain.MessageRecord) conversationRuntimeMessageResponse {
	status := "done"
	isError := strings.TrimSpace(record.RouteResult.ErrorCode) != ""
	if isError {
		status = "error"
	}
	attachments := execdomain.DecodeUserAttachments(record.Metadata)
	items := make([]conversationRuntimeAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		items = append(items, conversationRuntimeAttachment{
			ID:          strings.TrimSpace(attachment.ID),
			Name:        strings.TrimSpace(attachment.Name),
			ContentType: strings.TrimSpace(attachment.ContentType),
			AssetURL:    strings.TrimSpace(attachment.AssetURL),
			PreviewURL:  strings.TrimSpace(attachment.PreviewURL),
		})
	}
	return conversationRuntimeMessageResponse{
		ID:           strings.TrimSpace(record.MessageID),
		Role:         string(record.Role),
		Text:         strings.TrimSpace(record.Content),
		Attachments:  items,
		Route:        string(record.RouteResult.Route),
		Source:       strings.TrimSpace(record.Source.JobName),
		Error:        isError,
		Status:       status,
		At:           record.Timestamp.UTC(),
		ProcessSteps: append([]shareddomain.ProcessStep(nil), record.RouteResult.ProcessSteps...),
		TaskID:       strings.TrimSpace(record.RouteResult.TaskID),
		TaskStatus:   status,
	}
}

func resolveConversationRuntimeTarget(records []sessiondomain.MessageRecord) (string, string, string) {
	return "model", "raw-model", "Raw Model"
}

func resolveConversationRuntimeModel(records []sessiondomain.MessageRecord) (string, string) {
	for idx := len(records) - 1; idx >= 0; idx-- {
		metadata := records[idx].Metadata
		if strings.TrimSpace(metadata[conversationRuntimeExecutionEngineMetadataKey]) == conversationRuntimeExecutionEngineCodex {
			return conversationRuntimeCodexProviderID, conversationRuntimeCodexModelID
		}
		providerID := strings.TrimSpace(metadata[conversationRuntimeLLMProviderMetadataKey])
		modelID := strings.TrimSpace(metadata[conversationRuntimeLLMModelMetadataKey])
		if providerID != "" || modelID != "" {
			return providerID, modelID
		}
	}
	return "", ""
}

func resolveConversationRuntimeCapabilities(records []sessiondomain.MessageRecord) ([]string, []string, []string) {
	for idx := len(records) - 1; idx >= 0; idx-- {
		metadata := records[idx].Metadata
		if len(metadata) == 0 {
			continue
		}
		return nil,
			normalizeConversationRuntimeIDs(metadata["alter0.skills.include"]),
			normalizeConversationRuntimeIDs(metadata["alter0.mcp.request.enable"])
	}
	return nil, nil, nil
}

func normalizeConversationRuntimeIDs(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		return nil
	}
	deduped := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	if len(deduped) == 0 {
		return nil
	}
	return deduped
}

func deriveConversationRuntimeTitle(records []sessiondomain.MessageRecord) string {
	for _, record := range records {
		if record.Role != sessiondomain.MessageRoleUser {
			continue
		}
		title := strings.TrimSpace(record.Content)
		if title == "" {
			continue
		}
		runes := []rune(title)
		if len(runes) > conversationRuntimeTitleMaxRunes {
			runes = runes[:conversationRuntimeTitleMaxRunes]
		}
		return string(runes)
	}
	return "New"
}

func (s *Server) memoryContextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.memory == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "memory context unavailable"})
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
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "memory context unavailable"})
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
		UpdatedAt:  task.UpdatedAt.UTC(),
		Tags:       append([]string(nil), summary.Tags...),
	}
}

func resolveMemoryTaskMeta(task taskdomain.Task) memoryTaskMeta {
	return memoryTaskMeta{
		TaskID:          strings.TrimSpace(task.ID),
		SessionID:       strings.TrimSpace(task.SessionID),
		SourceMessageID: strings.TrimSpace(task.SourceMessageID),
		Status:          task.Status,
		CreatedAt:       task.CreatedAt.UTC(),
		UpdatedAt:       task.UpdatedAt.UTC(),
		FinishedAt:      task.FinishedAt.UTC(),
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

func (s *Server) runtimeInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.runtimeInfo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "runtime info unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, s.runtimeInfo.GetRuntimeInfo())
}

func (s *Server) runtimeRestartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.runtime == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "runtime restart unavailable"})
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, s.runtime.GetRestartStatus())
		return
	}

	var req RuntimeRestartOptions
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
	}

	accepted, err := s.runtime.RequestRestart(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !accepted {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "runtime restart already in progress"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"accepted":                        true,
		"status":                          "restarting",
		"sync_remote_master":              req.SyncRemoteMaster,
		"confirm_discard_tracked_changes": req.ConfirmDiscardTrackedChanges,
	})
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
		if existing, ok := s.findCronJob(jobID); ok && existing.Builtin {
			if req.Enabled == nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "builtin scheduler job can only be enabled or disabled"})
				return
			}
			updated, found, err := s.scheduler.SetEnabled(jobID, *req.Enabled)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			if !found {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "cron job not found"})
				return
			}
			writeJSON(w, http.StatusOK, toCronJobResponse(updated))
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
		if existing, ok := s.findCronJob(jobID); ok && existing.Builtin {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "builtin scheduler job cannot be deleted"})
			return
		}
		if !s.scheduler.Delete(jobID) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "cron job not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) findCronJob(jobID string) (schedulerdomain.Job, bool) {
	if s == nil || s.scheduler == nil {
		return schedulerdomain.Job{}, false
	}
	normalized := strings.ToLower(strings.TrimSpace(jobID))
	for _, job := range s.scheduler.List() {
		if strings.ToLower(strings.TrimSpace(job.ID)) == normalized {
			return job, true
		}
	}
	return schedulerdomain.Job{}, false
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
		Builtin:        job.Builtin,
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

// LLM Provider handlers

func (s *Server) llmProviderListHandler(w http.ResponseWriter, r *http.Request) {
	if s.llm == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "llm service unavailable"})
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		config, err := s.llm.GetConfig(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		providers := []llmdomain.ModelProvider{}
		if config != nil {
			providers = config.Providers
		}

		// Mask API keys
		items := make([]llmProviderResponse, 0, len(providers))
		for _, p := range providers {
			items = append(items, toLLMProviderResponse(p, true))
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": items})

	case http.MethodPost:
		// Create new provider
		var req llmProviderCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		apiKey := normalizeLLMProviderAPIKey(req.APIKey)
		if apiKey == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api_key is required"})
			return
		}

		provider := llmdomain.ModelProvider{
			ID:           strings.TrimSpace(req.ID),
			Name:         req.Name,
			ProviderType: req.ProviderType,
			APIType:      req.APIType,
			BaseURL:      req.BaseURL,
			APIKey:       apiKey,
			OpenRouter:   req.OpenRouter,
			DefaultModel: req.DefaultModel,
			Models:       req.Models,
			IsEnabled:    req.IsEnabled,
		}
		if provider.ID == "" {
			provider.ID = s.newLLMProviderID()
		}

		if err := s.llm.AddProvider(ctx, provider); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		created, _ := s.llm.GetProvider(ctx, provider.ID)
		writeJSON(w, http.StatusCreated, toLLMProviderResponse(*created, true))

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) codexAccountCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.codexAccounts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "codex account service unavailable"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, active, err := s.codexAccounts.ListStatuses(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		runtimeStatus, err := s.codexAccounts.RuntimeStatus()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":   items,
			"active":  active,
			"runtime": runtimeStatus,
		})
	case http.MethodPost:
		var req codexAccountCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if strings.TrimSpace(req.AuthFileContent) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "auth_file_content is required"})
			return
		}
		record, err := s.codexAccounts.AddFromRaw(strings.TrimSpace(req.Name), []byte(req.AuthFileContent), req.Overwrite)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, record)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) codexAccountLoginSessionCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.codexAccounts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "codex account service unavailable"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req codexAccountLoginSessionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	session, err := s.codexAccounts.StartLoginSession(r.Context(), strings.TrimSpace(req.Name), req.Overwrite)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, session)
}

func (s *Server) codexRuntimeHandler(w http.ResponseWriter, r *http.Request) {
	if s.codexAccounts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "codex account service unavailable"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		runtimeStatus, err := s.codexAccounts.RuntimeStatus()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, runtimeStatus)
	case http.MethodPut:
		var req codexRuntimeUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		runtimeStatus, err := s.codexAccounts.UpdateRuntimeSettings(strings.TrimSpace(req.Model), strings.TrimSpace(req.ReasoningEffort))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, runtimeStatus)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) codexAccountItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.codexAccounts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "codex account service unavailable"})
		return
	}

	resourceType, resourceID, action, ok := codexAccountResourcePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid codex account path"})
		return
	}

	switch {
	case resourceType == "login-session":
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		session, found := s.codexAccounts.GetLoginSession(resourceID)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "login session not found"})
			return
		}
		writeJSON(w, http.StatusOK, session)
	case resourceType == "account" && action == "switch":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		record, backupPath, err := s.codexAccounts.Switch(resourceID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		_, active, activeErr := s.codexAccounts.ListStatuses(r.Context())
		if activeErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": activeErr.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"account":     record,
			"backup_path": backupPath,
			"active":      active,
		})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid codex account path"})
	}
}

func codexAccountResourcePath(path string) (resourceType string, resourceID string, action string, ok bool) {
	const prefix = "/api/control/codex/accounts/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if trimmed == "" {
		return "", "", "", false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 2 && parts[0] == "login-sessions" && strings.TrimSpace(parts[1]) != "" {
		return "login-session", parts[1], "", true
	}
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "switch" {
		return "account", parts[0], "switch", true
	}
	return "", "", "", false
}

func (s *Server) llmProviderItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.llm == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "llm service unavailable"})
		return
	}

	providerID, ok := resourceID(r.URL.Path, "/api/control/llm/providers/")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider id"})
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		provider, err := s.llm.GetProvider(ctx, providerID)
		if err != nil || provider == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
			return
		}
		writeJSON(w, http.StatusOK, toLLMProviderResponse(*provider, true))

	case http.MethodPut:
		var req llmProviderUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		existing, err := s.llm.GetProvider(ctx, providerID)
		if err != nil || existing == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
			return
		}
		apiKey := normalizeLLMProviderAPIKey(req.APIKey)
		if apiKey == "" {
			apiKey = existing.APIKey
		}
		if req.IsEnabled && strings.TrimSpace(apiKey) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api_key is required"})
			return
		}

		provider := llmdomain.ModelProvider{
			ID:           strings.TrimSpace(req.ID),
			Name:         req.Name,
			ProviderType: req.ProviderType,
			APIType:      req.APIType,
			BaseURL:      req.BaseURL,
			APIKey:       apiKey,
			OpenRouter:   req.OpenRouter,
			DefaultModel: req.DefaultModel,
			Models:       req.Models,
			IsEnabled:    req.IsEnabled,
		}
		if provider.ID == "" {
			provider.ID = providerID
		}

		if err := s.llm.UpdateProvider(ctx, providerID, provider); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		updated, _ := s.llm.GetProvider(ctx, provider.ID)
		writeJSON(w, http.StatusOK, toLLMProviderResponse(*updated, true))

	case http.MethodDelete:
		if err := s.llm.RemoveProvider(ctx, providerID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	case http.MethodPost:
		// Handle sub-actions: set-default, enable, disable
		var req llmProviderActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		switch req.Action {
		case "set-default":
			if err := s.llm.SetDefaultProvider(ctx, providerID); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		case "enable":
			if err := s.llm.EnableProvider(ctx, providerID, true); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		case "disable":
			if err := s.llm.EnableProvider(ctx, providerID, false); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown action"})
			return
		}

		updated, _ := s.llm.GetProvider(ctx, providerID)
		writeJSON(w, http.StatusOK, toLLMProviderResponse(*updated, true))

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

type llmProviderResponse struct {
	ID           string                      `json:"id"`
	Name         string                      `json:"name"`
	ProviderType string                      `json:"provider_type,omitempty"`
	APIType      string                      `json:"api_type"`
	BaseURL      string                      `json:"base_url"`
	APIKey       string                      `json:"api_key"` // Masked
	OpenRouter   *llmdomain.OpenRouterConfig `json:"openrouter,omitempty"`
	DefaultModel string                      `json:"default_model"`
	Models       []llmdomain.ModelInfo       `json:"models"`
	IsEnabled    bool                        `json:"is_enabled"`
	IsDefault    bool                        `json:"is_default"`
}

type llmProviderUpdateRequest struct {
	ID           string                      `json:"id"`
	Name         string                      `json:"name"`
	ProviderType string                      `json:"provider_type,omitempty"`
	APIType      string                      `json:"api_type"`
	BaseURL      string                      `json:"base_url"`
	APIKey       string                      `json:"api_key"`
	OpenRouter   *llmdomain.OpenRouterConfig `json:"openrouter,omitempty"`
	DefaultModel string                      `json:"default_model"`
	Models       []llmdomain.ModelInfo       `json:"models"`
	IsEnabled    bool                        `json:"is_enabled"`
}

type llmProviderCreateRequest struct {
	ID           string                      `json:"id"`
	Name         string                      `json:"name"`
	ProviderType string                      `json:"provider_type,omitempty"`
	APIType      string                      `json:"api_type"`
	BaseURL      string                      `json:"base_url"`
	APIKey       string                      `json:"api_key"`
	OpenRouter   *llmdomain.OpenRouterConfig `json:"openrouter,omitempty"`
	DefaultModel string                      `json:"default_model"`
	Models       []llmdomain.ModelInfo       `json:"models"`
	IsEnabled    bool                        `json:"is_enabled"`
}

type llmProviderActionRequest struct {
	Action string `json:"action"` // set-default, enable, disable
}

func (s *Server) newLLMProviderID() string {
	seed := time.Now().UTC().Format(time.RFC3339Nano)
	if s.idGenerator != nil {
		seed = s.idGenerator.NewID()
	}
	sum := sha256.Sum256([]byte(seed))
	return "prov_" + hex.EncodeToString(sum[:10])
}

func toLLMProviderResponse(p llmdomain.ModelProvider, maskKey bool) llmProviderResponse {
	apiKey := p.APIKey
	if maskKey && strings.TrimSpace(apiKey) == "" {
		apiKey = ""
	} else if maskKey && len(apiKey) > 8 {
		apiKey = apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
	} else if maskKey {
		apiKey = "****"
	}
	return llmProviderResponse{
		ID:           p.ID,
		Name:         p.Name,
		ProviderType: p.ProviderType,
		APIType:      p.APIType,
		BaseURL:      p.BaseURL,
		APIKey:       apiKey,
		OpenRouter:   p.OpenRouter,
		DefaultModel: p.DefaultModel,
		Models:       p.Models,
		IsEnabled:    p.IsEnabled,
		IsDefault:    p.IsDefault,
	}
}

func normalizeLLMProviderAPIKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "-" {
		return ""
	}
	return trimmed
}

func sessionResourceID(path string) (string, string, string, string, bool) {
	const prefix = "/api/sessions/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", "", false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if trimmed == "" {
		return "", "", "", "", false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		sessionID := strings.TrimSpace(parts[0])
		if sessionID == "" {
			return "", "", "", "", false
		}
		return sessionID, "", "", "", true
	}
	sessionID := strings.TrimSpace(parts[0])
	if sessionID == "" {
		return "", "", "", "", false
	}
	resource := strings.TrimSpace(parts[1])
	switch {
	case len(parts) == 2 && (resource == "messages" || resource == "tasks" || resource == "attachments" || resource == "pin"):
		return sessionID, resource, "", "", true
	case len(parts) == 4 && resource == "attachments":
		resourceID := strings.TrimSpace(parts[2])
		action := strings.TrimSpace(parts[3])
		if resourceID == "" || (action != "original" && action != "preview") {
			return "", "", "", "", false
		}
		return sessionID, resource, resourceID, action, true
	default:
		return "", "", "", "", false
	}
}

func resolveServerWorkspaceRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	absolute, err := filepath.Abs(wd)
	if err != nil {
		return wd
	}
	return absolute
}

func removeConversationSessionWorkspace(baseDir string, sessionID string) error {
	root := strings.TrimSpace(baseDir)
	if root == "" {
		root = "."
	}
	segment := sanitizeWorkspaceSegment(sessionID)
	if segment == "" {
		return fmt.Errorf("invalid session id")
	}
	if err := os.RemoveAll(filepath.Join(root, ".alter0", "workspaces", "sessions", segment)); err != nil {
		return fmt.Errorf("remove session workspace: %w", err)
	}
	return nil
}

func sanitizeWorkspaceSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	hyphenPending := false
	for _, ch := range trimmed {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.' {
			builder.WriteRune(ch)
			hyphenPending = false
			continue
		}
		if hyphenPending {
			continue
		}
		builder.WriteByte('-')
		hyphenPending = true
	}
	sanitized := strings.Trim(builder.String(), "-._")
	if sanitized == "" {
		return ""
	}
	if len(sanitized) > 96 {
		sanitized = sanitized[:96]
	}
	return strings.ToLower(sanitized)
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
		if status != taskdomain.TaskStatusSuccess && status != taskdomain.TaskStatusFailed && status != taskdomain.TaskStatusCanceled {
			return memoryTaskListQuery{}, http.StatusBadRequest, errors.New("status must be success/failed/canceled")
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
	return s.prepareMessageFromRequest(req, canonicalChatSessionID)
}

func (s *Server) prepareMessageFromRequest(req messageRequest, defaultSessionID string) (shareddomain.UnifiedMessage, int, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(defaultSessionID)
	}
	if sessionID == "" {
		sessionID = s.idGenerator.NewID()
	}
	attachments, err := s.normalizeConversationMessageAttachments(sessionID, req.Attachments)
	if err != nil {
		return shareddomain.UnifiedMessage{}, http.StatusBadRequest, err
	}
	content := strings.TrimSpace(req.Content)
	if content == "" && len(attachments) == 0 {
		return shareddomain.UnifiedMessage{}, http.StatusBadRequest, errors.New("content or attachments are required")
	}
	if content == "" && len(attachments) > 0 {
		content = defaultAttachmentContent(attachments)
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
	metadata := cloneStringMap(req.Metadata)
	if rawAttachments, err := execdomain.EncodeUserAttachments(attachments); err != nil {
		return shareddomain.UnifiedMessage{}, http.StatusBadRequest, errors.New("invalid attachments")
	} else if rawAttachments != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata[execdomain.UserAttachmentsMetadataKey] = rawAttachments
		if rawImages, imageErr := execdomain.EncodeUserImageAttachments(execdomain.FilterUserImageAttachments(attachments)); imageErr == nil && rawImages != "" {
			metadata[execdomain.UserImageAttachmentsMetadataKey] = rawImages
		}
	}

	return shareddomain.UnifiedMessage{
		MessageID:     s.idGenerator.NewID(),
		SessionID:     sessionID,
		UserID:        req.UserID,
		ChannelID:     channelID,
		ChannelType:   channelType,
		TriggerType:   shareddomain.TriggerTypeUser,
		Content:       content,
		Metadata:      metadata,
		TraceID:       s.idGenerator.NewID(),
		CorrelationID: strings.TrimSpace(req.CorrelationID),
		ReceivedAt:    time.Now().UTC(),
	}, http.StatusOK, nil
}

func defaultAttachmentContent(attachments []execdomain.UserAttachment) string {
	count := len(attachments)
	if count <= 0 {
		return ""
	}
	imageCount := 0
	for _, attachment := range attachments {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(attachment.ContentType)), "image/") {
			imageCount += 1
		}
	}
	if imageCount == count {
		if count <= 1 {
			return "Attached image."
		}
		return fmt.Sprintf("Attached %d images.", count)
	}
	if count <= 1 {
		return "Attached file."
	}
	if imageCount == 0 {
		return fmt.Sprintf("Attached %d files.", count)
	}
	return fmt.Sprintf("Attached %d files, including %d images.", count, imageCount)
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
