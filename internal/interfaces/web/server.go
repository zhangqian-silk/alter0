package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
	schedulerapp "alter0/internal/scheduler/application"
	schedulerdomain "alter0/internal/scheduler/domain"
	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
	"alter0/internal/shared/infrastructure/observability"
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
	logger       *slog.Logger
}

type messageRequest struct {
	SessionID     string            `json:"session_id"`
	UserID        string            `json:"user_id,omitempty"`
	ChannelID     string            `json:"channel_id,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Content       string            `json:"content"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type messageResponse struct {
	Result shareddomain.OrchestrationResult `json:"result"`
	Error  string                           `json:"error,omitempty"`
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
	Result shareddomain.OrchestrationResult `json:"result"`
}

type streamErrorResponse struct {
	Error  string                           `json:"error"`
	Result shareddomain.OrchestrationResult `json:"result,omitempty"`
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

func NewServer(
	addr string,
	orchestrator Orchestrator,
	telemetry *observability.Telemetry,
	idGenerator sharedapp.IDGenerator,
	control *controlapp.Service,
	scheduler *schedulerapp.Manager,
	logger *slog.Logger,
) *Server {
	return &Server{
		addr:         addr,
		orchestrator: orchestrator,
		telemetry:    telemetry,
		idGenerator:  idGenerator,
		control:      control,
		scheduler:    scheduler,
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
