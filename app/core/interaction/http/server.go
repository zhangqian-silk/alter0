package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
}

func NewHTTPChannel(port int) *HTTPChannel {
	return &HTTPChannel{
		id:              "http",
		port:            port,
		pending:         map[string]chan types.Message{},
		shutdownTimeout: 5 * time.Second,
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
	if strings.TrimSpace(req.UserID) == "" {
		req.UserID = "local_user"
	}

	requestID := c.newID("req")
	respCh := make(chan types.Message, 1)
	c.pendingMu.Lock()
	c.pending[requestID] = respCh
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, requestID)
		c.pendingMu.Unlock()
	}()

	msg := types.Message{
		ID:        c.newID("http"),
		Content:   req.Content,
		Role:      "user",
		ChannelID: c.id,
		UserID:    req.UserID,
		TaskID:    req.TaskID,
		RequestID: requestID,
		Meta: map[string]interface{}{
			"user_id": req.UserID,
		},
	}

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

func (c *HTTPChannel) newID(prefix string) string {
	seq := atomic.AddUint64(&c.counter, 1)
	return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(seq, 10)
}
