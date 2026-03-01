package gateway

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	servicequeue "alter0/app/pkg/queue"
	"alter0/app/pkg/types"
)

const defaultAgentID = "default"

type QueueOptions struct {
	Enabled        bool
	EnqueueTimeout time.Duration
	AttemptTimeout time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
}

type DefaultGateway struct {
	defaultAgentID string
	agents         map[string]types.Agent
	channels       map[string]types.Channel
	mu             sync.RWMutex
	tracer         TraceRecorder

	executionQueue *servicequeue.Queue
	queueOptions   QueueOptions

	processedMessages uint64
	agentFallbacks    uint64
	lastMessageUnix   atomic.Int64
	startedUnix       atomic.Int64
}

type HealthStatus struct {
	Started            bool
	StartedAt          time.Time
	RegisteredChannels []string
	RegisteredAgents   []string
	DefaultAgentID     string
	ProcessedMessages  uint64
	AgentFallbacks     uint64
	LastMessageAt      time.Time
	QueueEnabled       bool
	Queue              servicequeue.Stats
}

func NewGateway(agent types.Agent) *DefaultGateway {
	agents := map[string]types.Agent{}
	if agent != nil {
		agents[defaultAgentID] = agent
	}
	return &DefaultGateway{
		defaultAgentID: defaultAgentID,
		agents:         agents,
		channels:       make(map[string]types.Channel),
	}
}

func (g *DefaultGateway) RegisterAgent(agentID string, agent types.Agent) error {
	id := strings.TrimSpace(agentID)
	if id == "" {
		return fmt.Errorf("agent id is required")
	}
	if agent == nil {
		return fmt.Errorf("agent is required")
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.agents[id] = agent
	if g.defaultAgentID == "" {
		g.defaultAgentID = id
	}
	return nil
}

func (g *DefaultGateway) SetDefaultAgent(agentID string) error {
	id := strings.TrimSpace(agentID)
	if id == "" {
		return fmt.Errorf("default agent id is required")
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.agents[id]; !ok {
		return fmt.Errorf("default agent not found: %s", id)
	}
	g.defaultAgentID = id
	return nil
}

func (g *DefaultGateway) RegisterChannel(c types.Channel) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.channels[c.ID()] = c
	log.Printf("[Gateway] Registered channel: %s", c.ID())
}

func (g *DefaultGateway) SetTraceRecorder(tracer TraceRecorder) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tracer = tracer
}

func (g *DefaultGateway) SetExecutionQueue(q *servicequeue.Queue, opts QueueOptions) {
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
		g.trace(msg, "", "inbound_received", "ok", "")

		routedMsg := g.routeMessageToAgent(msg)
		agentID := metaString(routedMsg.Meta, "agent_id")
		if routeErr := metaString(routedMsg.Meta, "route_error"); routeErr != "" {
			g.trace(routedMsg, agentID, "route_selected", "error", routeErr)
		} else {
			g.trace(routedMsg, agentID, "route_selected", "ok", "")
		}
		if g.queueEnabled() {
			g.dispatchWithQueue(ctx, routedMsg, agentID)
			return
		}

		if err := g.processAndReply(ctx, routedMsg, agentID); err != nil {
			log.Printf("[Gateway] Processing failed: %v", err)
			_ = g.sendErrorReply(ctx, routedMsg, "Error: "+err.Error())
		}
	}

	g.mu.RLock()
	for _, c := range g.channels {
		wg.Add(1)
		go func(ch types.Channel) {
			defer wg.Done()
			if err := ch.Start(ctx, handler); err != nil {
				log.Printf("[Gateway] Channel %s error: %v", ch.ID(), err)
				if ctx.Err() == nil {
					g.trace(types.Message{ChannelID: ch.ID()}, "", "channel_disconnected", "error", err.Error())
				}
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

func (g *DefaultGateway) dispatchWithQueue(ctx context.Context, msg types.Message, agentID string) {
	g.mu.RLock()
	q := g.executionQueue
	opts := g.queueOptions
	g.mu.RUnlock()

	attempt := 0
	job := servicequeue.Job{
		MaxRetries:     opts.MaxRetries,
		RetryDelay:     opts.RetryDelay,
		AttemptTimeout: opts.AttemptTimeout,
		Run: func(runCtx context.Context) error {
			attempt++
			err := g.processAndReply(runCtx, msg, agentID)
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
		g.trace(msg, agentID, "queue_enqueue", "error", err.Error())
		_ = g.sendErrorReply(ctx, msg, fmt.Sprintf("Gateway queue unavailable: %v", err))
		return
	}
	g.trace(msg, agentID, "queue_enqueue", "ok", "")
}

func (g *DefaultGateway) processAndReply(ctx context.Context, msg types.Message, agentID string) error {
	agent, _, err := g.resolveAgent(msg)
	if err != nil {
		g.trace(msg, agentID, "agent_process", "error", err.Error())
		return err
	}

	response, err := agent.Process(ctx, msg)
	if err != nil {
		g.trace(msg, agentID, "agent_process", "error", err.Error())
		return fmt.Errorf("agent process: %w", err)
	}
	g.trace(msg, agentID, "agent_process", "ok", "")
	if strings.TrimSpace(msg.Content) == "" && strings.TrimSpace(response.Content) == "" {
		return nil
	}

	channel, exists := g.channelByID(msg.ChannelID)
	if !exists {
		g.trace(msg, agentID, "deliver_reply", "error", "channel not found for reply")
		return fmt.Errorf("channel not found for reply: %s", msg.ChannelID)
	}

	normalizeReply(&response, msg)
	if err := channel.Send(ctx, response); err != nil {
		g.trace(response, agentID, "deliver_reply", "error", err.Error())
		return fmt.Errorf("send reply: %w", err)
	}
	g.trace(response, agentID, "deliver_reply", "ok", "")
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
	if err := channel.Send(ctx, response); err != nil {
		g.trace(response, metaString(msg.Meta, "agent_id"), "deliver_error_reply", "error", err.Error())
		return err
	}
	g.trace(response, metaString(msg.Meta, "agent_id"), "deliver_error_reply", "ok", "")
	return nil
}

func (g *DefaultGateway) DeliverDirect(ctx context.Context, channelID string, to string, content string, meta map[string]interface{}) error {
	channelID = strings.TrimSpace(channelID)
	to = strings.TrimSpace(to)
	content = strings.TrimSpace(content)
	if channelID == "" {
		return fmt.Errorf("channel id is required")
	}
	if to == "" {
		return fmt.Errorf("delivery target is required")
	}
	if content == "" {
		return fmt.Errorf("delivery content is required")
	}

	channel, exists := g.channelByID(channelID)
	if !exists {
		return fmt.Errorf("channel not found: %s", channelID)
	}

	msg := types.Message{
		ID:        "sched-direct-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Content:   content,
		Role:      types.MessageRoleAssistant,
		ChannelID: channelID,
		UserID:    to,
		Meta:      map[string]interface{}{},
		Envelope: &types.MessageEnvelope{
			Direction: types.EnvelopeDirectionOutbound,
			Channel:   channelID,
			PeerID:    to,
			PeerType:  "user",
			MessageID: "sched-direct-" + strconv.FormatInt(time.Now().UnixNano(), 10),
			Parts: []types.EnvelopePart{{
				Type: types.EnvelopePartText,
				Text: content,
			}},
		},
	}
	for k, v := range meta {
		msg.Meta[k] = v
	}
	return channel.Send(ctx, msg)
}

func (g *DefaultGateway) DeliverAgentTurn(ctx context.Context, channelID string, to string, content string, agentID string, meta map[string]interface{}) error {
	channelID = strings.TrimSpace(channelID)
	to = strings.TrimSpace(to)
	content = strings.TrimSpace(content)
	agentID = strings.TrimSpace(agentID)
	if channelID == "" {
		return fmt.Errorf("channel id is required")
	}
	if to == "" {
		return fmt.Errorf("delivery target is required")
	}
	if content == "" {
		return fmt.Errorf("agent turn content is required")
	}

	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	requestID := "sched-turn-req-" + now
	msg := types.Message{
		ID:        "sched-turn-msg-" + now,
		Content:   content,
		Role:      types.MessageRoleUser,
		ChannelID: channelID,
		UserID:    to,
		RequestID: requestID,
		Envelope: &types.MessageEnvelope{
			Direction: types.EnvelopeDirectionInbound,
			Channel:   channelID,
			PeerID:    to,
			PeerType:  "user",
			MessageID: "sched-turn-msg-" + now,
			Parts: []types.EnvelopePart{{
				Type: types.EnvelopePartText,
				Text: content,
			}},
		},
		Meta: map[string]interface{}{},
	}
	if agentID != "" {
		msg.Meta["agent_id"] = agentID
	}
	for k, v := range meta {
		msg.Meta[k] = v
	}
	return g.processAndReply(ctx, msg, strings.TrimSpace(agentID))
}

func (g *DefaultGateway) routeMessageToAgent(msg types.Message) types.Message {
	agent, resolvedAgentID, err := g.resolveAgent(msg)
	if err != nil {
		if msg.Meta == nil {
			msg.Meta = map[string]interface{}{}
		}
		msg.Meta["route_error"] = err.Error()
		return msg
	}
	_ = agent

	requestedAgentID, requested := requestedAgentID(msg)
	if msg.Meta == nil {
		msg.Meta = map[string]interface{}{}
	}
	msg.Meta["agent_id"] = resolvedAgentID
	if requested {
		msg.Meta["requested_agent_id"] = requestedAgentID
		if requestedAgentID != resolvedAgentID {
			msg.Meta["agent_fallback"] = true
		}
	}
	return msg
}

func (g *DefaultGateway) resolveAgent(msg types.Message) (types.Agent, string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.agents) == 0 {
		return nil, "", fmt.Errorf("gateway has no registered agents")
	}

	defaultAgentID := g.defaultAgentID
	if strings.TrimSpace(defaultAgentID) == "" {
		for id := range g.agents {
			defaultAgentID = id
			break
		}
	}
	defaultAgent, ok := g.agents[defaultAgentID]
	if !ok {
		return nil, "", fmt.Errorf("default agent not found: %s", defaultAgentID)
	}

	requestedID, requested := requestedAgentID(msg)
	if !requested {
		return defaultAgent, defaultAgentID, nil
	}

	agent, ok := g.agents[requestedID]
	if ok {
		return agent, requestedID, nil
	}

	atomic.AddUint64(&g.agentFallbacks, 1)
	log.Printf("[Gateway] Unknown agent requested=%s, fallback=%s", requestedID, defaultAgentID)
	return defaultAgent, defaultAgentID, nil
}

func requestedAgentID(msg types.Message) (string, bool) {
	if msg.Meta == nil {
		return "", false
	}
	for _, key := range []string{"agent_id", "agentId"} {
		raw, ok := msg.Meta[key]
		if !ok {
			continue
		}
		if value, ok := raw.(string); ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed, true
			}
		}
	}
	return "", false
}

func (g *DefaultGateway) trace(msg types.Message, agentID, event, status, detail string) {
	g.mu.RLock()
	tracer := g.tracer
	g.mu.RUnlock()
	if tracer == nil {
		return
	}

	traceEvent := TraceEvent{
		RequestID: strings.TrimSpace(msg.RequestID),
		MessageID: strings.TrimSpace(msg.ID),
		ChannelID: strings.TrimSpace(msg.ChannelID),
		UserID:    strings.TrimSpace(msg.UserID),
		AgentID:   strings.TrimSpace(agentID),
		Event:     strings.TrimSpace(event),
		Status:    strings.TrimSpace(status),
		Detail:    strings.TrimSpace(detail),
	}
	if traceEvent.Event == "" {
		traceEvent.Event = "unknown"
	}
	if traceEvent.Status == "" {
		traceEvent.Status = "ok"
	}
	if err := tracer.Record(traceEvent); err != nil {
		log.Printf("[Gateway] Trace write failed: %v", err)
	}
}

func metaString(meta map[string]interface{}, key string) string {
	if len(meta) == 0 {
		return ""
	}
	raw, ok := meta[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return strings.TrimSpace(value)
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
	agents := make([]string, 0, len(g.agents))
	for id := range g.agents {
		agents = append(agents, id)
	}
	defaultAgentID := g.defaultAgentID
	queueEnabled := g.queueOptions.Enabled && g.executionQueue != nil
	var queueStats servicequeue.Stats
	if queueEnabled {
		queueStats = g.executionQueue.Stats()
	}
	g.mu.RUnlock()
	sort.Strings(channels)
	sort.Strings(agents)

	status := HealthStatus{
		RegisteredChannels: channels,
		RegisteredAgents:   agents,
		DefaultAgentID:     defaultAgentID,
		ProcessedMessages:  atomic.LoadUint64(&g.processedMessages),
		AgentFallbacks:     atomic.LoadUint64(&g.agentFallbacks),
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
