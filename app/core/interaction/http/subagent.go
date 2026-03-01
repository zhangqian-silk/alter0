package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"alter0/app/core/interaction/http/httpstate"
)

const (
	subagentModeRun     = "run"
	subagentModeSession = "session"
)

type SubagentAnnouncement struct {
	ChannelID  string
	To         string
	Message    string
	SubagentID string
	Mode       string
	Turn       int
	Status     string
}

type subagentAnnounceTarget struct {
	ChannelID string `json:"channel_id"`
	To        string `json:"to"`
}

type subagentCreateRequest struct {
	Mode       string                  `json:"mode"`
	Content    string                  `json:"content,omitempty"`
	UserID     string                  `json:"user_id,omitempty"`
	AgentID    string                  `json:"agent_id,omitempty"`
	TimeoutSec int                     `json:"timeout_sec,omitempty"`
	Announce   *subagentAnnounceTarget `json:"announce,omitempty"`
}

type subagentTurnRequest struct {
	Content    string `json:"content"`
	TimeoutSec int    `json:"timeout_sec,omitempty"`
}

type subagentTurnResponse struct {
	SubagentID string            `json:"subagent_id"`
	Mode       string            `json:"mode"`
	Status     string            `json:"status"`
	Turn       int               `json:"turn"`
	Result     *outgoingResponse `json:"result,omitempty"`
	Error      string            `json:"error,omitempty"`
}

type subagentStatusResponse struct {
	SubagentID string                  `json:"subagent_id"`
	Mode       string                  `json:"mode"`
	Status     string                  `json:"status"`
	UserID     string                  `json:"user_id"`
	AgentID    string                  `json:"agent_id,omitempty"`
	CreatedAt  string                  `json:"created_at"`
	UpdatedAt  string                  `json:"updated_at"`
	Result     *outgoingResponse       `json:"result,omitempty"`
	Error      string                  `json:"error,omitempty"`
	Announce   *subagentAnnounceTarget `json:"announce,omitempty"`
	Turns      []subagentTurnResponse  `json:"turns,omitempty"`
}

type subagentListResponse struct {
	Subagents []subagentStatusResponse `json:"subagents"`
}

func (c *HTTPChannel) SetSubagentAnnouncer(announcer func(context.Context, SubagentAnnouncement) error) {
	c.subagentAnnouncer = announcer
}

func (c *HTTPChannel) handleSubagents(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/subagents" {
		switch r.Method {
		case http.MethodPost:
			c.handleSubagentCreate(w, r)
		case http.MethodGet:
			c.handleSubagentList(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id, action, ok := parseSubagentPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.handleSubagentGet(w, r, id)
	case "messages":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.handleSubagentMessage(w, r, id)
	case "close":
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.handleSubagentClose(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (c *HTTPChannel) handleSubagentCreate(w http.ResponseWriter, r *http.Request) {
	if c.handler == nil {
		http.Error(w, "handler not ready", http.StatusServiceUnavailable)
		return
	}

	var req subagentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	mode := normalizeSubagentMode(req.Mode)
	if mode == "" {
		http.Error(w, "mode must be run or session", http.StatusBadRequest)
		return
	}
	if mode == subagentModeRun && strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required for run mode", http.StatusBadRequest)
		return
	}
	if req.Announce != nil {
		if strings.TrimSpace(req.Announce.ChannelID) == "" || strings.TrimSpace(req.Announce.To) == "" {
			http.Error(w, "announce.channel_id and announce.to are required", http.StatusBadRequest)
			return
		}
	}

	userID := strings.TrimSpace(req.UserID)
	now := time.Now().UTC()
	record, err := c.subagentStore.Create(c.newID("subagent"), mode, userID, strings.TrimSpace(req.AgentID), toServiceAnnounceTarget(req.Announce), now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if mode == subagentModeRun {
		timeout := c.subagentTurnTimeout(req.TimeoutSec)
		go c.executeSubagentTurn(context.Background(), record.ID, req.Content, timeout, true)
	}

	payload := c.snapshotSubagent(record.ID)
	if payload == nil {
		http.Error(w, "failed to create subagent", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *HTTPChannel) handleSubagentList(w http.ResponseWriter, r *http.Request) {
	limit := parseListLimit(r.URL.Query().Get("limit"))
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	modeFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))

	items := c.subagentStore.List(statusFilter, modeFilter, limit)

	resp := subagentListResponse{Subagents: make([]subagentStatusResponse, 0, len(items))}
	for _, record := range items {
		recordCopy := record
		resp.Subagents = append(resp.Subagents, toSubagentStatusResponse(&recordCopy))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *HTTPChannel) handleSubagentGet(w http.ResponseWriter, r *http.Request, id string) {
	payload := c.snapshotSubagent(id)
	if payload == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *HTTPChannel) handleSubagentMessage(w http.ResponseWriter, r *http.Request, id string) {
	var req subagentTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	timeout := c.subagentTurnTimeout(req.TimeoutSec)
	turn, statusCode := c.executeSubagentTurn(r.Context(), id, req.Content, timeout, false)
	if turn == nil {
		http.NotFound(w, r)
		return
	}

	payload := subagentTurnResponse{
		SubagentID: id,
		Mode:       turn.Mode,
		Status:     turn.Status,
		Turn:       turn.Turn,
		Result:     turn.Result,
		Error:      turn.Error,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *HTTPChannel) handleSubagentClose(w http.ResponseWriter, r *http.Request, id string) {
	record, err := c.subagentStore.CloseSession(id, time.Now().UTC())
	if err != nil {
		if errors.Is(err, httpstate.ErrSubagentNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "only session mode supports close", http.StatusBadRequest)
		return
	}
	payload := toSubagentStatusResponse(&record)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *HTTPChannel) executeSubagentTurn(ctx context.Context, id string, content string, timeout time.Duration, autoClose bool) (*subagentTurnResponse, int) {
	_ = autoClose
	if c.handler == nil {
		return &subagentTurnResponse{SubagentID: id, Status: "failed", Error: "handler not ready"}, http.StatusServiceUnavailable
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return &subagentTurnResponse{SubagentID: id, Status: "failed", Error: "content is required"}, http.StatusBadRequest
	}

	start, err := c.subagentStore.BeginTurn(id, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, httpstate.ErrSubagentNotFound):
			return nil, http.StatusNotFound
		case errors.Is(err, httpstate.ErrSubagentSessionClosed):
			record, ok := c.subagentStore.Get(id)
			status := "closed"
			if ok {
				status = record.Status
			}
			return &subagentTurnResponse{SubagentID: id, Mode: subagentModeSession, Status: status, Error: "session is not active"}, http.StatusConflict
		case errors.Is(err, httpstate.ErrSubagentRunAlreadyDone):
			record, ok := c.subagentStore.Get(id)
			status := "completed"
			if ok {
				status = record.Status
			}
			return &subagentTurnResponse{SubagentID: id, Mode: subagentModeRun, Status: status, Error: "run mode already finished"}, http.StatusConflict
		default:
			return &subagentTurnResponse{SubagentID: id, Status: "failed", Error: err.Error()}, http.StatusBadRequest
		}
	}

	incoming := incomingRequest{Content: content, UserID: start.UserID, AgentID: start.AgentID}
	msg, respCh := c.prepareMessage(incoming)
	if msg.Meta == nil {
		msg.Meta = map[string]interface{}{}
	}
	msg.Meta["subagent_id"] = id
	msg.Meta["subagent_mode"] = start.Mode
	msg.Meta["subagent_turn"] = start.Turn

	c.handler(msg)

	status := "completed"
	var result *outgoingResponse
	var serviceResult *httpstate.TaskResult
	var errMsg string

	waitTimeout := timeout
	if waitTimeout <= 0 {
		waitTimeout = defaultResponseTimeout
	}

	select {
	case response := <-respCh:
		closed, _ := response.Meta["closed"].(bool)
		decision, _ := response.Meta["decision"].(string)
		result = &outgoingResponse{
			TaskID:   response.TaskID,
			Response: response.Content,
			Closed:   closed,
			Decision: decision,
		}
		serviceResult = &httpstate.TaskResult{
			TaskID:   response.TaskID,
			Response: response.Content,
			Closed:   closed,
			Decision: decision,
		}
	case <-time.After(waitTimeout):
		status = "timeout"
		errMsg = "subagent turn timeout"
	}
	c.removePendingRequest(msg.RequestID)

	_, _ = c.subagentStore.FinishTurn(id, start.Turn, status, serviceResult, errMsg, time.Now().UTC())

	if start.Announce != nil {
		_ = c.emitSubagentAnnouncement(ctx, SubagentAnnouncement{
			ChannelID:  start.Announce.ChannelID,
			To:         start.Announce.To,
			Message:    buildSubagentAnnouncementText(id, start.Mode, start.Turn, status, result, errMsg),
			SubagentID: id,
			Mode:       start.Mode,
			Turn:       start.Turn,
			Status:     status,
		})
	}

	statusCode := http.StatusOK
	if status == "timeout" {
		statusCode = http.StatusGatewayTimeout
	}
	turn := &subagentTurnResponse{
		SubagentID: id,
		Mode:       start.Mode,
		Status:     status,
		Turn:       start.Turn,
		Result:     result,
		Error:      errMsg,
	}
	return turn, statusCode
}

func (c *HTTPChannel) emitSubagentAnnouncement(ctx context.Context, announcement SubagentAnnouncement) error {
	if c.subagentAnnouncer == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.subagentAnnouncer(ctx, announcement)
}

func (c *HTTPChannel) snapshotSubagent(id string) *subagentStatusResponse {
	record, ok := c.subagentStore.Get(id)
	if !ok {
		return nil
	}
	payload := toSubagentStatusResponse(&record)
	return &payload
}

func toSubagentStatusResponse(record *httpstate.SubagentRecord) subagentStatusResponse {
	payload := subagentStatusResponse{
		SubagentID: record.ID,
		Mode:       record.Mode,
		Status:     record.Status,
		UserID:     record.UserID,
		AgentID:    record.AgentID,
		CreatedAt:  record.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  record.UpdatedAt.Format(time.RFC3339),
		Error:      record.Error,
		Announce:   toHTTPAnnounceTarget(record.Announce),
	}
	if record.Result != nil {
		payload.Result = &outgoingResponse{
			TaskID:   record.Result.TaskID,
			Response: record.Result.Response,
			Closed:   record.Result.Closed,
			Decision: record.Result.Decision,
		}
	}
	if len(record.Turns) > 0 {
		payload.Turns = make([]subagentTurnResponse, 0, len(record.Turns))
		for _, turn := range record.Turns {
			entry := subagentTurnResponse{
				SubagentID: record.ID,
				Mode:       record.Mode,
				Status:     turn.Status,
				Turn:       turn.Turn,
				Error:      turn.Error,
			}
			if turn.Result != nil {
				entry.Result = &outgoingResponse{
					TaskID:   turn.Result.TaskID,
					Response: turn.Result.Response,
					Closed:   turn.Result.Closed,
					Decision: turn.Result.Decision,
				}
			}
			payload.Turns = append(payload.Turns, entry)
		}
	}
	return payload
}

func toServiceAnnounceTarget(target *subagentAnnounceTarget) *httpstate.SubagentAnnounceTarget {
	if target == nil {
		return nil
	}
	return &httpstate.SubagentAnnounceTarget{ChannelID: target.ChannelID, To: target.To}
}

func toHTTPAnnounceTarget(target *httpstate.SubagentAnnounceTarget) *subagentAnnounceTarget {
	if target == nil {
		return nil
	}
	return &subagentAnnounceTarget{ChannelID: target.ChannelID, To: target.To}
}

func normalizeSubagentMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", subagentModeRun:
		return subagentModeRun
	case subagentModeSession:
		return subagentModeSession
	default:
		return ""
	}
}

func parseSubagentPath(path string) (id string, action string, ok bool) {
	if !strings.HasPrefix(path, "/api/subagents/") {
		return "", "", false
	}
	tail := strings.Trim(strings.TrimPrefix(path, "/api/subagents/"), "/")
	if tail == "" {
		return "", "", false
	}
	parts := strings.Split(tail, "/")
	if len(parts) == 1 {
		return parts[0], "", true
	}
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func (c *HTTPChannel) subagentTurnTimeout(timeoutSec int) time.Duration {
	if timeoutSec <= 0 {
		return defaultResponseTimeout
	}
	return time.Duration(timeoutSec) * time.Second
}

func buildSubagentAnnouncementText(id string, mode string, turn int, status string, result *outgoingResponse, errMsg string) string {
	prefix := fmt.Sprintf("Subagent %s (%s) turn %d: %s", id, mode, turn, status)
	if strings.TrimSpace(errMsg) != "" {
		return prefix + " - " + errMsg
	}
	if result == nil {
		return prefix
	}
	preview := strings.TrimSpace(result.Response)
	if preview == "" {
		return prefix
	}
	runes := []rune(preview)
	if len(runes) > 160 {
		preview = string(runes[:160]) + "..."
	}
	return prefix + "\n" + preview
}
