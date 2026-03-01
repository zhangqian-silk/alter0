package gateway

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"alter0/app/core/queue"
	"alter0/app/pkg/types"
)

type QueueOptions struct {
	Enabled        bool
	EnqueueTimeout time.Duration
	AttemptTimeout time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
}

type DefaultGateway struct {
	agent    types.Agent
	channels map[string]types.Channel
	mu       sync.RWMutex

	executionQueue *queue.Queue
	queueOptions   QueueOptions

	processedMessages uint64
	lastMessageUnix   atomic.Int64
	startedUnix       atomic.Int64
}

type HealthStatus struct {
	Started            bool
	StartedAt          time.Time
	RegisteredChannels []string
	ProcessedMessages  uint64
	LastMessageAt      time.Time
	QueueEnabled       bool
	Queue              queue.Stats
}

func NewGateway(agent types.Agent) *DefaultGateway {
	return &DefaultGateway{
		agent:    agent,
		channels: make(map[string]types.Channel),
	}
}

func (g *DefaultGateway) RegisterChannel(c types.Channel) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.channels[c.ID()] = c
	log.Printf("[Gateway] Registered channel: %s", c.ID())
}

func (g *DefaultGateway) SetExecutionQueue(q *queue.Queue, opts QueueOptions) {
	if opts.MaxRetries < 0 {
		opts.MaxRetries = 0
	}
	if opts.EnqueueTimeout < 0 {
		opts.EnqueueTimeout = 0
	}
	if opts.AttemptTimeout < 0 {
		opts.AttemptTimeout = 0
	}
	if opts.RetryDelay < 0 {
		opts.RetryDelay = 0
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.executionQueue = q
	g.queueOptions = opts
}

func (g *DefaultGateway) Start(ctx context.Context) error {
	var wg sync.WaitGroup
	g.startedUnix.Store(time.Now().Unix())

	handler := func(msg types.Message) {
		atomic.AddUint64(&g.processedMessages, 1)
		g.lastMessageUnix.Store(time.Now().Unix())
		log.Printf("[Gateway] Received message from channel=%s user=%s", msg.ChannelID, msg.UserID)

		if g.queueEnabled() {
			g.dispatchWithQueue(ctx, msg)
			return
		}

		if err := g.processAndReply(ctx, msg); err != nil {
			log.Printf("[Gateway] Processing failed: %v", err)
			_ = g.sendErrorReply(ctx, msg, "Error: "+err.Error())
		}
	}

	g.mu.RLock()
	for _, c := range g.channels {
		wg.Add(1)
		go func(ch types.Channel) {
			defer wg.Done()
			if err := ch.Start(ctx, handler); err != nil {
				log.Printf("[Gateway] Channel %s error: %v", ch.ID(), err)
			}
		}(c)
	}
	g.mu.RUnlock()

	log.Println("[Gateway] Started all channels")
	wg.Wait()
	return nil
}

func (g *DefaultGateway) queueEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.queueOptions.Enabled && g.executionQueue != nil
}

func (g *DefaultGateway) dispatchWithQueue(ctx context.Context, msg types.Message) {
	g.mu.RLock()
	q := g.executionQueue
	opts := g.queueOptions
	g.mu.RUnlock()

	attempt := 0
	job := queue.Job{
		MaxRetries:     opts.MaxRetries,
		RetryDelay:     opts.RetryDelay,
		AttemptTimeout: opts.AttemptTimeout,
		Run: func(runCtx context.Context) error {
			attempt++
			err := g.processAndReply(runCtx, msg)
			if err == nil {
				return nil
			}
			if attempt <= opts.MaxRetries {
				log.Printf("[Gateway] Queue job retrying request=%s attempt=%d/%d: %v", msg.RequestID, attempt, opts.MaxRetries+1, err)
				return err
			}
			log.Printf("[Gateway] Queue job failed request=%s after %d attempts: %v", msg.RequestID, attempt, err)
			_ = g.sendErrorReply(runCtx, msg, "Error: "+err.Error())
			return nil
		},
	}

	enqueueCtx := ctx
	cancel := func() {}
	if opts.EnqueueTimeout > 0 {
		enqueueCtx, cancel = context.WithTimeout(ctx, opts.EnqueueTimeout)
	}
	defer cancel()

	if _, err := q.EnqueueContext(enqueueCtx, job); err != nil {
		log.Printf("[Gateway] Queue enqueue failed: %v", err)
		_ = g.sendErrorReply(ctx, msg, fmt.Sprintf("Gateway queue unavailable: %v", err))
	}
}

func (g *DefaultGateway) processAndReply(ctx context.Context, msg types.Message) error {
	response, err := g.agent.Process(ctx, msg)
	if err != nil {
		return fmt.Errorf("agent process: %w", err)
	}
	if strings.TrimSpace(msg.Content) == "" && strings.TrimSpace(response.Content) == "" {
		return nil
	}

	channel, exists := g.channelByID(msg.ChannelID)
	if !exists {
		return fmt.Errorf("channel not found for reply: %s", msg.ChannelID)
	}

	normalizeReply(&response, msg)
	if err := channel.Send(ctx, response); err != nil {
		return fmt.Errorf("send reply: %w", err)
	}
	return nil
}

func (g *DefaultGateway) sendErrorReply(ctx context.Context, msg types.Message, reason string) error {
	channel, exists := g.channelByID(msg.ChannelID)
	if !exists {
		return fmt.Errorf("channel not found for reply: %s", msg.ChannelID)
	}
	response := types.Message{
		ID:        "resp-" + msg.ID,
		Content:   reason,
		Role:      types.MessageRoleAssistant,
		ChannelID: msg.ChannelID,
		UserID:    msg.UserID,
		TaskID:    msg.TaskID,
		RequestID: msg.RequestID,
		Meta:      map[string]interface{}{},
	}
	normalizeReply(&response, msg)
	return channel.Send(ctx, response)
}

func normalizeReply(response *types.Message, request types.Message) {
	if response.ID == "" {
		response.ID = "resp-" + request.ID
	}
	if response.ChannelID == "" {
		response.ChannelID = request.ChannelID
	}
	if response.Role == "" {
		response.Role = types.MessageRoleAssistant
	}
	if response.UserID == "" {
		response.UserID = request.UserID
	}
	if response.RequestID == "" {
		response.RequestID = request.RequestID
	}
	if response.TaskID == "" {
		response.TaskID = request.TaskID
	}
	response.CloneEnvelopeFrom(request)
	envelope := response.EnsureEnvelope()
	envelope.Direction = types.EnvelopeDirectionOutbound
	if envelope.Channel == "" {
		envelope.Channel = response.ChannelID
	}
	if envelope.PeerID == "" {
		envelope.PeerID = response.UserID
	}
	if envelope.MessageID == "" {
		envelope.MessageID = response.ID
	}
	if response.Meta == nil {
		response.Meta = map[string]interface{}{}
	}
	for k, v := range request.Meta {
		if _, exists := response.Meta[k]; !exists {
			response.Meta[k] = v
		}
	}
}

func (g *DefaultGateway) channelByID(channelID string) (types.Channel, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	channel, exists := g.channels[channelID]
	return channel, exists
}

func (g *DefaultGateway) HealthStatus() HealthStatus {
	g.mu.RLock()
	channels := make([]string, 0, len(g.channels))
	for id := range g.channels {
		channels = append(channels, id)
	}
	queueEnabled := g.queueOptions.Enabled && g.executionQueue != nil
	var queueStats queue.Stats
	if queueEnabled {
		queueStats = g.executionQueue.Stats()
	}
	g.mu.RUnlock()
	sort.Strings(channels)

	status := HealthStatus{
		RegisteredChannels: channels,
		ProcessedMessages:  atomic.LoadUint64(&g.processedMessages),
		QueueEnabled:       queueEnabled,
		Queue:              queueStats,
	}

	if started := g.startedUnix.Load(); started > 0 {
		status.Started = true
		status.StartedAt = time.Unix(started, 0).UTC()
	}
	if last := g.lastMessageUnix.Load(); last > 0 {
		status.LastMessageAt = time.Unix(last, 0).UTC()
	}

	return status
}
