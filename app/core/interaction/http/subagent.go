package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
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

type subagentRecord struct {
	ID        string
	Mode      string
	Status    string
	UserID    string
	AgentID   string
	Announce  *subagentAnnounceTarget
	CreatedAt time.Time
	UpdatedAt time.Time
	Result    *outgoingResponse
	Error     string
	Turns     []subagentTurnRecord
}

type subagentTurnRecord struct {
	Turn   int
	Status string
	Result *outgoingResponse
	Error  string
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
	if userID == "" {
		userID = "local_user"
	}
	record := &subagentRecord{
		ID:        c.newID("subagent"),
		Mode:      mode,
		Status:    "pending",
		UserID:    userID,
		AgentID:   strings.TrimSpace(req.AgentID),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Announce:  req.Announce,
	}
	if mode == subagentModeSession {
		record.Status = "active"
	}

	c.subagentMu.Lock()
	c.subagents[record.ID] = record
	c.subagentMu.Unlock()

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

	c.subagentMu.Lock()
	items := make([]*subagentRecord, 0, len(c.subagents))
	for _, record := range c.subagents {
		if statusFilter != "" && statusFilter != "all" && record.Status != statusFilter {
			continue
		}
		if modeFilter != "" && modeFilter != "all" && record.Mode != modeFilter {
			continue
		}
		copyRecord := *record
		items = append(items, &copyRecord)
	}
	c.subagentMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}

	resp := subagentListResponse{Subagents: make([]subagentStatusResponse, 0, len(items))}
	for _, record := range items {
		resp.Subagents = append(resp.Subagents, toSubagentStatusResponse(record))
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
	c.subagentMu.Lock()
	record, ok := c.subagents[id]
	if !ok {
		c.subagentMu.Unlock()
		http.NotFound(w, r)
		return
	}
	if record.Mode != subagentModeSession {
		c.subagentMu.Unlock()
		http.Error(w, "only session mode supports close", http.StatusBadRequest)
		return
	}
	if record.Status == "active" || record.Status == "pending" {
		record.Status = "closed"
		record.UpdatedAt = time.Now().UTC()
	}
	payload := toSubagentStatusResponse(record)
	c.subagentMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *HTTPChannel) executeSubagentTurn(ctx context.Context, id string, content string, timeout time.Duration, autoClose bool) (*subagentTurnResponse, int) {
	if c.handler == nil {
		return &subagentTurnResponse{SubagentID: id, Status: "failed", Error: "handler not ready"}, http.StatusServiceUnavailable
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return &subagentTurnResponse{SubagentID: id, Status: "failed", Error: "content is required"}, http.StatusBadRequest
	}

	c.subagentMu.Lock()
	record, ok := c.subagents[id]
	if !ok {
		c.subagentMu.Unlock()
		return nil, http.StatusNotFound
	}
	if record.Mode == subagentModeSession && record.Status != "active" {
		status := record.Status
		c.subagentMu.Unlock()
		return &subagentTurnResponse{SubagentID: id, Mode: subagentModeSession, Status: status, Error: "session is not active"}, http.StatusConflict
	}
	if record.Mode == subagentModeRun && len(record.Turns) > 0 {
		status := record.Status
		c.subagentMu.Unlock()
		return &subagentTurnResponse{SubagentID: id, Mode: subagentModeRun, Status: status, Error: "run mode already finished"}, http.StatusConflict
	}
	mode := record.Mode
	turnIndex := len(record.Turns) + 1
	record.Status = "running"
	record.UpdatedAt = time.Now().UTC()
	announce := record.Announce
	userID := record.UserID
	agentID := record.AgentID
	c.subagentMu.Unlock()

	incoming := incomingRequest{Content: content, UserID: userID, AgentID: agentID}
	msg, respCh := c.prepareMessage(incoming)
	if msg.Meta == nil {
		msg.Meta = map[string]interface{}{}
	}
	msg.Meta["subagent_id"] = id
	msg.Meta["subagent_mode"] = mode
	msg.Meta["subagent_turn"] = turnIndex

	c.handler(msg)

	status := "completed"
	var result *outgoingResponse
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
	case <-time.After(waitTimeout):
		status = "timeout"
		errMsg = "subagent turn timeout"
	}
	c.removePendingRequest(msg.RequestID)

	c.subagentMu.Lock()
	record, ok = c.subagents[id]
	if ok {
		record.UpdatedAt = time.Now().UTC()
		record.Error = errMsg
		record.Result = result
		record.Turns = append(record.Turns, subagentTurnRecord{
			Turn:   turnIndex,
			Status: status,
			Result: result,
			Error:  errMsg,
		})
		if mode == subagentModeRun {
			record.Status = status
			if autoClose && record.Status == "completed" {
				record.Status = "completed"
			}
		} else {
			if status == "timeout" {
				record.Status = "active"
			} else {
				record.Status = "active"
			}
		}
	}
	c.subagentMu.Unlock()

	if announce != nil {
		_ = c.emitSubagentAnnouncement(ctx, SubagentAnnouncement{
			ChannelID:  announce.ChannelID,
			To:         announce.To,
			Message:    buildSubagentAnnouncementText(id, mode, turnIndex, status, result, errMsg),
			SubagentID: id,
			Mode:       mode,
			Turn:       turnIndex,
			Status:     status,
		})
	}

	statusCode := http.StatusOK
	if status == "timeout" {
		statusCode = http.StatusGatewayTimeout
	}
	turn := &subagentTurnResponse{
		SubagentID: id,
		Mode:       mode,
		Status:     status,
		Turn:       turnIndex,
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
	c.subagentMu.Lock()
	defer c.subagentMu.Unlock()
	record, ok := c.subagents[id]
	if !ok {
		return nil
	}
	payload := toSubagentStatusResponse(record)
	return &payload
}

func toSubagentStatusResponse(record *subagentRecord) subagentStatusResponse {
	payload := subagentStatusResponse{
		SubagentID: record.ID,
		Mode:       record.Mode,
		Status:     record.Status,
		UserID:     record.UserID,
		AgentID:    record.AgentID,
		CreatedAt:  record.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  record.UpdatedAt.Format(time.RFC3339),
		Error:      record.Error,
		Announce:   record.Announce,
	}
	if record.Result != nil {
		copyResult := *record.Result
		payload.Result = &copyResult
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
				copyResult := *turn.Result
				entry.Result = &copyResult
			}
			payload.Turns = append(payload.Turns, entry)
		}
	}
	return payload
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
