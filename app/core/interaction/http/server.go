package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"alter0/app/pkg/types"
)

const (
	defaultResponseTimeout = 60 * time.Second
	defaultChunkSizeRunes  = 1200
	maxChunkSizeRunes      = 4000
)

type HTTPChannel struct {
	id              string
	port            int
	server          *http.Server
	handler         func(types.Message)
	statusProvider  func(context.Context) map[string]interface{}
	shutdownTimeout time.Duration

	pendingMu   sync.Mutex
	pending     map[string]chan types.Message
	counter     uint64
	startedUnix atomic.Int64

	asyncMu sync.Mutex
	async   map[string]*asyncRequest
}

func NewHTTPChannel(port int) *HTTPChannel {
	return &HTTPChannel{
		id:              "http",
		port:            port,
		pending:         map[string]chan types.Message{},
		shutdownTimeout: 5 * time.Second,
		async:           map[string]*asyncRequest{},
	}
}

func (c *HTTPChannel) ID() string {
	return c.id
}

func (c *HTTPChannel) SetStatusProvider(provider func(context.Context) map[string]interface{}) {
	c.statusProvider = provider
}

func (c *HTTPChannel) SetShutdownTimeout(timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	c.shutdownTimeout = timeout
}

func (c *HTTPChannel) Start(ctx context.Context, handler func(types.Message)) error {
	c.handler = handler
	c.startedUnix.Store(time.Now().Unix())

	mux := http.NewServeMux()
	mux.HandleFunc("/api/message", c.handleMessage)
	mux.HandleFunc("/api/status", c.handleStatus)
	mux.HandleFunc("/api/tasks", c.handleAsyncTasks)
	mux.HandleFunc("/api/tasks/", c.handleAsyncTasks)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/", fs)

	c.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), c.shutdownTimeout)
		defer cancel()
		if err := c.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[HTTP] Shutdown error: %v", err)
		}
	}()

	log.Printf("[HTTP] Listening on port %d...", c.port)
	if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (c *HTTPChannel) Send(ctx context.Context, msg types.Message) error {
	if strings.TrimSpace(msg.RequestID) == "" {
		log.Printf("[HTTP] Outgoing message without request id: %s", msg.Content)
		return nil
	}

	c.pendingMu.Lock()
	ch, ok := c.pending[msg.RequestID]
	c.pendingMu.Unlock()
	if !ok {
		log.Printf("[HTTP] Pending request not found: %s", msg.RequestID)
		return nil
	}

	select {
	case ch <- msg:
	default:
	}
	return nil
}

type incomingRequest struct {
	Content string `json:"content"`
	UserID  string `json:"user_id"`
	TaskID  string `json:"task_id,omitempty"`
	Stream  bool   `json:"stream,omitempty"`
}

type outgoingResponse struct {
	TaskID   string `json:"task_id"`
	Response string `json:"response"`
	Closed   bool   `json:"closed"`
	Decision string `json:"decision"`
}

type streamResponseEvent struct {
	Type     string `json:"type"`
	TaskID   string `json:"task_id,omitempty"`
	Decision string `json:"decision,omitempty"`
	Closed   bool   `json:"closed,omitempty"`
	Index    int    `json:"index,omitempty"`
	Total    int    `json:"total,omitempty"`
	Chunk    string `json:"chunk,omitempty"`
}

type statusResponse struct {
	ChannelID       string                 `json:"channel_id"`
	PendingRequests int                    `json:"pending_requests"`
	StartedAt       string                 `json:"started_at,omitempty"`
	UptimeSec       int64                  `json:"uptime_sec"`
	Runtime         map[string]interface{} `json:"runtime,omitempty"`
}

type asyncRequest struct {
	RequestID string
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    string
	Content   string
	UserID    string
	TaskID    string
	Result    *outgoingResponse
}

type asyncSubmitResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	StatusURL string `json:"status_url"`
	CancelURL string `json:"cancel_url"`
	Accepted  string `json:"accepted_at"`
}

type asyncTaskResponse struct {
	RequestID string            `json:"request_id"`
	Status    string            `json:"status"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
	Content   string            `json:"content,omitempty"`
	UserID    string            `json:"user_id,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	Result    *outgoingResponse `json:"result,omitempty"`
}

type asyncTaskListResponse struct {
	Tasks []asyncTaskResponse `json:"tasks"`
}

func (c *HTTPChannel) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := statusResponse{ChannelID: c.id}
	c.pendingMu.Lock()
	resp.PendingRequests = len(c.pending)
	c.pendingMu.Unlock()

	if started := c.startedUnix.Load(); started > 0 {
		startAt := time.Unix(started, 0).UTC()
		resp.StartedAt = startAt.Format(time.RFC3339)
		resp.UptimeSec = int64(time.Since(startAt).Seconds())
		if resp.UptimeSec < 0 {
			resp.UptimeSec = 0
		}
	}
	if c.statusProvider != nil {
		resp.Runtime = c.statusProvider(r.Context())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *HTTPChannel) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req incomingRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}
	msg, respCh := c.prepareMessage(req)
	requestID := msg.RequestID
	defer c.removePendingRequest(requestID)

	if c.handler == nil {
		http.Error(w, "handler not ready", http.StatusServiceUnavailable)
		return
	}
	c.handler(msg)

	streamRequested := req.Stream || parseBoolQuery(r.URL.Query().Get("stream"))
	chunkSize := parseChunkSize(r.URL.Query().Get("chunk_size"))

	select {
	case response := <-respCh:
		closed, _ := response.Meta["closed"].(bool)
		decision, _ := response.Meta["decision"].(string)
		if streamRequested {
			c.writeStreamResponse(w, response, decision, closed, chunkSize)
			return
		}

		payload := outgoingResponse{
			TaskID:   response.TaskID,
			Response: response.Content,
			Closed:   closed,
			Decision: decision,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload)
	case <-time.After(defaultResponseTimeout):
		http.Error(w, "request timeout", http.StatusGatewayTimeout)
	}
}

func (c *HTTPChannel) handleAsyncTasks(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/tasks" {
		switch r.Method {
		case http.MethodPost:
			c.handleAsyncSubmit(w, r)
		case http.MethodGet:
			c.handleAsyncList(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id, action, ok := parseTaskPath(r.URL.Path)
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
		c.handleAsyncStatus(w, r, id)
	case "cancel":
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.handleAsyncCancel(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (c *HTTPChannel) handleAsyncSubmit(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req incomingRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}
	if c.handler == nil {
		http.Error(w, "handler not ready", http.StatusServiceUnavailable)
		return
	}

	msg, respCh := c.prepareMessage(req)
	requestID := msg.RequestID
	now := time.Now().UTC()

	c.asyncMu.Lock()
	c.async[requestID] = &asyncRequest{
		RequestID: requestID,
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "pending",
		Content:   req.Content,
		UserID:    msg.UserID,
		TaskID:    strings.TrimSpace(req.TaskID),
	}
	c.asyncMu.Unlock()

	c.handler(msg)
	go c.awaitAsyncResult(requestID, respCh)

	resp := asyncSubmitResponse{
		RequestID: requestID,
		Status:    "pending",
		StatusURL: "/api/tasks/" + requestID,
		CancelURL: "/api/tasks/" + requestID + "/cancel",
		Accepted:  now.Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *HTTPChannel) handleAsyncList(w http.ResponseWriter, r *http.Request) {
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if statusFilter == "" {
		statusFilter = "all"
	}
	limit := parseListLimit(r.URL.Query().Get("limit"))

	c.asyncMu.Lock()
	items := make([]*asyncRequest, 0, len(c.async))
	for _, entry := range c.async {
		if statusFilter != "all" && entry.Status != statusFilter {
			continue
		}
		copyEntry := *entry
		items = append(items, &copyEntry)
	}
	c.asyncMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	if len(items) > limit {
		items = items[:limit]
	}

	resp := asyncTaskListResponse{Tasks: make([]asyncTaskResponse, 0, len(items))}
	for _, entry := range items {
		resp.Tasks = append(resp.Tasks, asyncTaskResponse{
			RequestID: entry.RequestID,
			Status:    entry.Status,
			CreatedAt: entry.CreatedAt.Format(time.RFC3339),
			UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
			Content:   entry.Content,
			UserID:    entry.UserID,
			TaskID:    entry.TaskID,
			Result:    entry.Result,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *HTTPChannel) handleAsyncStatus(w http.ResponseWriter, r *http.Request, requestID string) {
	c.asyncMu.Lock()
	entry, ok := c.async[requestID]
	c.asyncMu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	payload := asyncTaskResponse{
		RequestID: entry.RequestID,
		Status:    entry.Status,
		CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
		Content:   entry.Content,
		UserID:    entry.UserID,
		TaskID:    entry.TaskID,
		Result:    entry.Result,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *HTTPChannel) handleAsyncCancel(w http.ResponseWriter, r *http.Request, requestID string) {
	c.asyncMu.Lock()
	entry, ok := c.async[requestID]
	if !ok {
		c.asyncMu.Unlock()
		http.NotFound(w, r)
		return
	}
	if entry.Status == "pending" {
		entry.Status = "canceled"
		entry.UpdatedAt = time.Now().UTC()
	}
	payload := asyncTaskResponse{
		RequestID: entry.RequestID,
		Status:    entry.Status,
		CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
		Content:   entry.Content,
		UserID:    entry.UserID,
		TaskID:    entry.TaskID,
		Result:    entry.Result,
	}
	c.asyncMu.Unlock()

	if payload.Status == "canceled" {
		c.removePendingRequest(requestID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *HTTPChannel) awaitAsyncResult(requestID string, respCh <-chan types.Message) {
	defer c.removePendingRequest(requestID)

	select {
	case response := <-respCh:
		closed, _ := response.Meta["closed"].(bool)
		decision, _ := response.Meta["decision"].(string)
		result := &outgoingResponse{
			TaskID:   response.TaskID,
			Response: response.Content,
			Closed:   closed,
			Decision: decision,
		}

		c.asyncMu.Lock()
		if entry, ok := c.async[requestID]; ok && entry.Status == "pending" {
			entry.Status = "completed"
			entry.Result = result
			entry.UpdatedAt = time.Now().UTC()
		}
		c.asyncMu.Unlock()
	case <-time.After(defaultResponseTimeout):
		c.asyncMu.Lock()
		if entry, ok := c.async[requestID]; ok && entry.Status == "pending" {
			entry.Status = "timeout"
			entry.UpdatedAt = time.Now().UTC()
		}
		c.asyncMu.Unlock()
	}
}

func (c *HTTPChannel) prepareMessage(req incomingRequest) (types.Message, chan types.Message) {
	if strings.TrimSpace(req.UserID) == "" {
		req.UserID = "local_user"
	}

	requestID := c.newID("req")
	respCh := make(chan types.Message, 1)
	c.pendingMu.Lock()
	c.pending[requestID] = respCh
	c.pendingMu.Unlock()

	msgID := c.newID("http")
	msg := types.Message{
		ID:        msgID,
		Content:   req.Content,
		Role:      types.MessageRoleUser,
		ChannelID: c.id,
		UserID:    req.UserID,
		TaskID:    req.TaskID,
		RequestID: requestID,
		Envelope: &types.MessageEnvelope{
			Direction: types.EnvelopeDirectionInbound,
			Channel:   c.id,
			PeerID:    req.UserID,
			PeerType:  "user",
			MessageID: msgID,
			Parts: []types.EnvelopePart{
				{Type: types.EnvelopePartText, Text: req.Content},
			},
		},
		Meta: map[string]interface{}{
			"user_id": req.UserID,
		},
	}
	return msg, respCh
}

func (c *HTTPChannel) removePendingRequest(requestID string) {
	c.pendingMu.Lock()
	delete(c.pending, requestID)
	c.pendingMu.Unlock()
}

func parseTaskPath(path string) (id string, action string, ok bool) {
	if !strings.HasPrefix(path, "/api/tasks/") {
		return "", "", false
	}
	tail := strings.Trim(strings.TrimPrefix(path, "/api/tasks/"), "/")
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

func (c *HTTPChannel) writeStreamResponse(w http.ResponseWriter, response types.Message, decision string, closed bool, chunkSize int) {
	chunks := splitByRunes(response.Content, chunkSize)
	if len(chunks) == 0 {
		chunks = []string{""}
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	flusher, _ := w.(http.Flusher)
	for i, chunk := range chunks {
		_ = encoder.Encode(streamResponseEvent{
			Type:  "chunk",
			Index: i + 1,
			Total: len(chunks),
			Chunk: chunk,
		})
		if flusher != nil {
			flusher.Flush()
		}
	}

	_ = encoder.Encode(streamResponseEvent{
		Type:     "done",
		TaskID:   response.TaskID,
		Decision: decision,
		Closed:   closed,
		Total:    len(chunks),
	})
	if flusher != nil {
		flusher.Flush()
	}
}

func splitByRunes(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = defaultChunkSizeRunes
	}
	if utf8.RuneCountInString(text) <= chunkSize {
		return []string{text}
	}

	runes := []rune(text)
	chunks := make([]string, 0, (len(runes)+chunkSize-1)/chunkSize)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func parseBoolQuery(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseChunkSize(raw string) int {
	size, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || size <= 0 {
		return defaultChunkSizeRunes
	}
	if size > maxChunkSizeRunes {
		return maxChunkSizeRunes
	}
	return size
}

func parseListLimit(raw string) int {
	const (
		defaultLimit = 20
		maxLimit     = 100
	)
	size, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || size <= 0 {
		return defaultLimit
	}
	if size > maxLimit {
		return maxLimit
	}
	return size
}

func (c *HTTPChannel) newID(prefix string) string {
	seq := atomic.AddUint64(&c.counter, 1)
	return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(seq, 10)
}
