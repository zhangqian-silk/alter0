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

	"alter0/app/pkg/types"
)

type HTTPChannel struct {
	id      string
	port    int
	server  *http.Server
	handler func(types.Message)

	pendingMu sync.Mutex
	pending   map[string]chan types.Message
	counter   uint64
}

func NewHTTPChannel(port int) *HTTPChannel {
	return &HTTPChannel{
		id:      "http",
		port:    port,
		pending: map[string]chan types.Message{},
	}
}

func (c *HTTPChannel) ID() string {
	return c.id
}

func (c *HTTPChannel) Start(ctx context.Context, handler func(types.Message)) error {
	c.handler = handler

	mux := http.NewServeMux()
	mux.HandleFunc("/api/message", c.handleMessage)
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
}

type outgoingResponse struct {
	TaskID   string `json:"task_id"`
	Response string `json:"response"`
	Closed   bool   `json:"closed"`
	Decision string `json:"decision"`
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

	select {
	case response := <-respCh:
		closed, _ := response.Meta["closed"].(bool)
		decision, _ := response.Meta["decision"].(string)
		payload := outgoingResponse{
			TaskID:   response.TaskID,
			Response: response.Content,
			Closed:   closed,
			Decision: decision,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload)
	case <-time.After(60 * time.Second):
		http.Error(w, "request timeout", http.StatusGatewayTimeout)
	}
}

func (c *HTTPChannel) newID(prefix string) string {
	seq := atomic.AddUint64(&c.counter, 1)
	return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(seq, 10)
}
