package telegram

import (
	"bytes"
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

const defaultAPIRoot = "https://api.telegram.org"

type Config struct {
	BotToken       string
	PollInterval   time.Duration
	TimeoutSeconds int
	DefaultChatID  string
	APIRoot        string
}

type Channel struct {
	cfg Config
	id  string

	counter uint64
	offset  int64

	mu      sync.RWMutex
	handler func(types.Message)
}

func NewChannel(cfg Config) *Channel {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 20
	}
	if strings.TrimSpace(cfg.APIRoot) == "" {
		cfg.APIRoot = defaultAPIRoot
	}
	return &Channel{cfg: cfg, id: "telegram"}
}

func (c *Channel) ID() string {
	return c.id
}

func (c *Channel) Start(ctx context.Context, handler func(types.Message)) error {
	c.mu.Lock()
	c.handler = handler
	c.mu.Unlock()

	if strings.TrimSpace(c.cfg.BotToken) == "" {
		return fmt.Errorf("telegram bot token is required")
	}

	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()

	for {
		if err := c.pollOnce(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("[Telegram] poll error: %v", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (c *Channel) Send(ctx context.Context, msg types.Message) error {
	chatID := c.resolveChatID(msg)
	if chatID == "" {
		return fmt.Errorf("telegram chat id is required")
	}

	imageURL := firstImageURL(msg)
	if imageURL != "" {
		payload := map[string]interface{}{
			"chat_id": chatID,
			"photo":   imageURL,
		}
		if strings.TrimSpace(msg.Content) != "" {
			payload["caption"] = msg.Content
		}
		return c.call(ctx, "sendPhoto", payload, nil)
	}

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    msg.Content,
	}
	if replyTo := replyToID(msg); replyTo != "" {
		if id, err := strconv.ParseInt(replyTo, 10, 64); err == nil {
			payload["reply_to_message_id"] = id
		}
	}
	return c.call(ctx, "sendMessage", payload, nil)
}

func (c *Channel) pollOnce(ctx context.Context) error {
	result := getUpdatesResponse{}
	offset := atomic.LoadInt64(&c.offset)
	payload := map[string]interface{}{
		"timeout": c.cfg.TimeoutSeconds,
	}
	if offset > 0 {
		payload["offset"] = offset
	}
	if err := c.call(ctx, "getUpdates", payload, &result); err != nil {
		return err
	}

	c.mu.RLock()
	handler := c.handler
	c.mu.RUnlock()
	if handler == nil {
		return nil
	}

	for _, upd := range result.Result {
		if upd.UpdateID >= atomic.LoadInt64(&c.offset) {
			atomic.StoreInt64(&c.offset, upd.UpdateID+1)
		}
		if upd.Message.MessageID == 0 {
			continue
		}
		if strings.TrimSpace(upd.Message.Text) == "" && strings.TrimSpace(upd.Message.Caption) == "" {
			continue
		}
		msg := c.toMessage(upd)
		handler(msg)
	}
	return nil
}

func (c *Channel) toMessage(upd update) types.Message {
	text := strings.TrimSpace(upd.Message.Text)
	if text == "" {
		text = strings.TrimSpace(upd.Message.Caption)
	}

	msgID := c.newID("telegram")
	peerID := strconv.FormatInt(upd.Message.Chat.ID, 10)
	userID := strconv.FormatInt(upd.Message.From.ID, 10)
	parts := []types.EnvelopePart{{Type: types.EnvelopePartText, Text: text}}
	if len(upd.Message.Photo) > 0 {
		parts = append(parts, types.EnvelopePart{Type: types.EnvelopePartImage, Name: "telegram-photo"})
	}

	msg := types.Message{
		ID:        msgID,
		Content:   text,
		Role:      types.MessageRoleUser,
		ChannelID: c.id,
		UserID:    userID,
		RequestID: c.newID("req"),
		Envelope: &types.MessageEnvelope{
			Direction: types.EnvelopeDirectionInbound,
			Channel:   c.id,
			PeerID:    peerID,
			PeerType:  "chat",
			MessageID: strconv.FormatInt(upd.Message.MessageID, 10),
			ReplyToID: strconv.FormatInt(upd.Message.ReplyTo.MessageID, 10),
			Parts:     parts,
		},
		Meta: map[string]interface{}{
			"user_id": userID,
			"peer_id": peerID,
		},
	}
	return msg
}

func (c *Channel) call(ctx context.Context, method string, payload interface{}, out interface{}) error {
	url := strings.TrimRight(c.cfg.APIRoot, "/") + "/bot" + c.cfg.BotToken + "/" + method
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram api status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var base apiResponse
	if err := json.Unmarshal(respBody, &base); err != nil {
		return err
	}
	if !base.OK {
		return fmt.Errorf("telegram api error: %s", base.Description)
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return err
		}
	}
	return nil
}

func (c *Channel) resolveChatID(msg types.Message) string {
	if msg.Envelope != nil && strings.TrimSpace(msg.Envelope.PeerID) != "" {
		return strings.TrimSpace(msg.Envelope.PeerID)
	}
	if strings.TrimSpace(msg.UserID) != "" {
		return strings.TrimSpace(msg.UserID)
	}
	return strings.TrimSpace(c.cfg.DefaultChatID)
}

func replyToID(msg types.Message) string {
	if msg.Envelope == nil {
		return ""
	}
	return strings.TrimSpace(msg.Envelope.ReplyToID)
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

func (c *Channel) newID(prefix string) string {
	seq := atomic.AddUint64(&c.counter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), seq)
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

type getUpdatesResponse struct {
	apiResponse
	Result []update `json:"result"`
}

type update struct {
	UpdateID int64           `json:"update_id"`
	Message  telegramMessage `json:"message"`
}

type telegramMessage struct {
	MessageID int64 `json:"message_id"`
	From      struct {
		ID int64 `json:"id"`
	} `json:"from"`
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text    string `json:"text"`
	Caption string `json:"caption"`
	Photo   []struct {
		FileID string `json:"file_id"`
	} `json:"photo"`
	ReplyTo struct {
		MessageID int64 `json:"message_id"`
	} `json:"reply_to_message"`
}
