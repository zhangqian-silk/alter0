package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"alter0/app/pkg/types"
)

const (
	defaultAPIRoot   = "https://slack.com/api"
	defaultEventPath = "/events/slack"
)

type Config struct {
	BotToken       string
	AppID          string
	ListenPort     int
	EventPath      string
	DefaultChannel string
	APIRoot        string
}

type Channel struct {
	cfg     Config
	id      string
	server  *http.Server
	handler func(types.Message)

	counter uint64
	mu      sync.RWMutex
}

func NewChannel(cfg Config) *Channel {
	if strings.TrimSpace(cfg.EventPath) == "" {
		cfg.EventPath = defaultEventPath
	}
	if cfg.ListenPort <= 0 {
		cfg.ListenPort = 8091
	}
	if strings.TrimSpace(cfg.APIRoot) == "" {
		cfg.APIRoot = defaultAPIRoot
	}
	cfg.EventPath = ensureLeadingSlash(cfg.EventPath)
	return &Channel{
		cfg: cfg,
		id:  "slack",
	}
}

func (c *Channel) ID() string {
	return c.id
}

func (c *Channel) Start(ctx context.Context, handler func(types.Message)) error {
	c.mu.Lock()
	c.handler = handler
	c.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc(c.cfg.EventPath, c.handleEvent)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	c.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.cfg.ListenPort),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[Slack] Shutdown error: %v", err)
		}
	}()

	log.Printf("[Slack] Listening on :%d%s", c.cfg.ListenPort, c.cfg.EventPath)
	err := c.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (c *Channel) Send(ctx context.Context, msg types.Message) error {
	token := strings.TrimSpace(c.cfg.BotToken)
	if token == "" {
		return fmt.Errorf("slack bot token is required")
	}

	chatID := c.resolvePeer(msg)
	if chatID == "" {
		return fmt.Errorf("slack channel id is required")
	}

	payload := map[string]interface{}{
		"channel": chatID,
		"text":    msg.Content,
	}
	if imageURL := firstImageURL(msg); imageURL != "" {
		payload["blocks"] = []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{"type": "mrkdwn", "text": msg.Content},
			},
			{
				"type":      "image",
				"image_url": imageURL,
				"alt_text":  "image",
			},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.cfg.APIRoot, "/")+"/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack api status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var ack struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &ack); err == nil && !ack.OK {
		return fmt.Errorf("slack api error: %s", ack.Error)
	}
	return nil
}

func (c *Channel) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var envelope eventEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if envelope.Type == "url_verification" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"challenge": envelope.Challenge})
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))

	if envelope.Type != "event_callback" {
		return
	}
	if envelope.Event.Type != "message" || envelope.Event.Subtype != "" {
		return
	}
	if strings.TrimSpace(envelope.Event.Text) == "" {
		return
	}

	msgID := c.newID("slack")
	peerID := strings.TrimSpace(envelope.Event.Channel)
	userID := strings.TrimSpace(envelope.Event.User)

	msg := types.Message{
		ID:        msgID,
		Content:   envelope.Event.Text,
		Role:      types.MessageRoleUser,
		ChannelID: c.id,
		UserID:    userID,
		RequestID: c.newID("req"),
		Envelope: &types.MessageEnvelope{
			Direction: types.EnvelopeDirectionInbound,
			Channel:   c.id,
			AccountID: strings.TrimSpace(c.cfg.AppID),
			PeerID:    peerID,
			PeerType:  "channel",
			MessageID: msgID,
			ReplyToID: strings.TrimSpace(envelope.Event.ThreadTs),
			Parts: []types.EnvelopePart{
				{Type: types.EnvelopePartText, Text: envelope.Event.Text},
			},
		},
		Meta: map[string]interface{}{
			"user_id":  userID,
			"peer_id":  peerID,
			"channel":  c.id,
			"event_ts": envelope.Event.Ts,
		},
	}

	c.mu.RLock()
	handler := c.handler
	c.mu.RUnlock()
	if handler != nil {
		handler(msg)
	}
}

func (c *Channel) newID(prefix string) string {
	seq := atomic.AddUint64(&c.counter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), seq)
}

func (c *Channel) resolvePeer(msg types.Message) string {
	if msg.Envelope != nil && strings.TrimSpace(msg.Envelope.PeerID) != "" {
		return strings.TrimSpace(msg.Envelope.PeerID)
	}
	if strings.TrimSpace(msg.UserID) != "" {
		return strings.TrimSpace(msg.UserID)
	}
	return strings.TrimSpace(c.cfg.DefaultChannel)
}

func firstImageURL(msg types.Message) string {
	if msg.Envelope == nil {
		return ""
	}
	for _, part := range msg.Envelope.Parts {
		if part.Type == types.EnvelopePartImage && strings.TrimSpace(part.URL) != "" {
			return strings.TrimSpace(part.URL)
		}
	}
	return ""
}

func ensureLeadingSlash(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

type eventEnvelope struct {
	Type      string     `json:"type"`
	Challenge string     `json:"challenge"`
	Event     eventInner `json:"event"`
}

type eventInner struct {
	Type     string `json:"type"`
	Subtype  string `json:"subtype"`
	User     string `json:"user"`
	Text     string `json:"text"`
	Ts       string `json:"ts"`
	ThreadTs string `json:"thread_ts"`
	Channel  string `json:"channel"`
}
