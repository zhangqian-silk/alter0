package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	sharedapp "alter0/internal/shared/application"
	shareddomain "alter0/internal/shared/domain"
)

type OverloadPolicy string

const (
	OverloadPolicyRejectNew OverloadPolicy = "reject_new"
)

const (
	ErrorCodeQueueTimeout  = "queue_timeout"
	ErrorCodeQueueCanceled = "queue_canceled"
	ErrorCodeRateLimited   = "rate_limited"
)

var (
	ErrQueueTimeout  = errors.New("queue wait timeout")
	ErrQueueCanceled = errors.New("queue wait canceled")
	ErrRateLimited   = errors.New("system overloaded")
)

type ConcurrencyOptions struct {
	WorkerCount    int
	MaxQueueSize   int
	QueueTimeout   time.Duration
	OverloadPolicy OverloadPolicy
}

type queueTelemetry interface {
	CountQueueEvent(event string)
	ObserveQueueWait(d time.Duration)
	SetQueueDepth(depth int)
	SetWorkerInFlight(inFlight int)
}

type queuedRequest struct {
	ctx        context.Context
	msg        shareddomain.UnifiedMessage
	enqueuedAt time.Time
	done       chan queuedResponse
}

type queuedResponse struct {
	result shareddomain.OrchestrationResult
	err    error
}

type sessionQueue struct {
	pending      []*queuedRequest
	running      bool
	inReadyQueue bool
}

type ConcurrentService struct {
	downstream Orchestrator
	telemetry  sharedapp.Telemetry
	queueStats queueTelemetry
	logger     *slog.Logger
	options    ConcurrencyOptions

	ctx context.Context

	mu           sync.Mutex
	cond         *sync.Cond
	sessions     map[string]*sessionQueue
	readySession []string
	waiting      int
	stopped      bool

	inFlight int64
}

func NewConcurrentService(
	ctx context.Context,
	downstream Orchestrator,
	telemetry sharedapp.Telemetry,
	logger *slog.Logger,
	options ConcurrencyOptions,
) *ConcurrentService {
	normalized := normalizeConcurrencyOptions(options)
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}

	svc := &ConcurrentService{
		downstream: downstream,
		telemetry:  telemetry,
		logger:     logger,
		options:    normalized,
		ctx:        ctx,
		sessions:   map[string]*sessionQueue{},
	}
	if stats, ok := telemetry.(queueTelemetry); ok {
		svc.queueStats = stats
		svc.queueStats.SetQueueDepth(0)
		svc.queueStats.SetWorkerInFlight(0)
	}
	svc.cond = sync.NewCond(&svc.mu)

	for workerID := 0; workerID < normalized.WorkerCount; workerID++ {
		go svc.runWorker(ctx)
	}

	go func() {
		<-ctx.Done()
		svc.mu.Lock()
		svc.stopped = true
		svc.mu.Unlock()
		svc.cond.Broadcast()
	}()

	return svc
}

func normalizeConcurrencyOptions(options ConcurrencyOptions) ConcurrencyOptions {
	if options.WorkerCount <= 0 {
		options.WorkerCount = 4
	}
	if options.MaxQueueSize <= 0 {
		options.MaxQueueSize = options.WorkerCount * 32
	}
	if options.QueueTimeout <= 0 {
		options.QueueTimeout = 5 * time.Second
	}
	if options.OverloadPolicy == "" {
		options.OverloadPolicy = OverloadPolicyRejectNew
	}
	return options
}

func (s *ConcurrentService) Handle(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.OrchestrationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := shareddomain.OrchestrationResult{
		MessageID: msg.MessageID,
		SessionID: msg.SessionID,
	}

	req := &queuedRequest{
		ctx:        ctx,
		msg:        msg,
		enqueuedAt: time.Now(),
		done:       make(chan queuedResponse, 1),
	}

	depth, err := s.enqueue(req)
	if err != nil {
		wait := time.Duration(0)
		result.ErrorCode = ErrorCodeRateLimited
		result = withQueueMetadata(result, wait, false, true)
		s.logQueueOutcome("queue request rejected", msg, wait, false, true, depth)
		return result, fmt.Errorf("%w: queue capacity reached", ErrRateLimited)
	}

	ctxDone := ctx.Done()
	var timer *time.Timer
	var timeoutC <-chan time.Time
	if s.options.QueueTimeout > 0 {
		timer = time.NewTimer(s.options.QueueTimeout)
		timeoutC = timer.C
		defer timer.Stop()
	}

	for {
		select {
		case response := <-req.done:
			return response.result, response.err
		case <-ctxDone:
			removed, queueDepth := s.removePending(msg.SessionID, req)
			if removed {
				wait := time.Since(req.enqueuedAt)
				result.ErrorCode = ErrorCodeQueueCanceled
				result = withQueueMetadata(result, wait, false, false)
				s.countQueueEvent("canceled")
				s.logQueueOutcome("queue wait canceled", msg, wait, false, false, queueDepth)
				return result, fmt.Errorf("%w: %v", ErrQueueCanceled, ctx.Err())
			}
			ctxDone = nil
			timeoutC = nil
		case <-timeoutC:
			removed, queueDepth := s.removePending(msg.SessionID, req)
			if removed {
				wait := time.Since(req.enqueuedAt)
				result.ErrorCode = ErrorCodeQueueTimeout
				result = withQueueMetadata(result, wait, true, false)
				s.countQueueEvent("timeout")
				s.logQueueOutcome("queue wait timeout", msg, wait, true, false, queueDepth)
				return result, fmt.Errorf("%w after %s", ErrQueueTimeout, s.options.QueueTimeout)
			}
			timeoutC = nil
		}
	}
}

func (s *ConcurrentService) runWorker(ctx context.Context) {
	for {
		req, sessionID, ok := s.nextRequest(ctx)
		if !ok {
			return
		}

		wait := time.Since(req.enqueuedAt)
		s.observeQueueWait(wait)
		s.countQueueEvent("dequeued")

		inFlight := int(atomic.AddInt64(&s.inFlight, 1))
		s.setWorkerInFlight(inFlight)

		result, err := s.downstream.Handle(req.ctx, req.msg)
		if result.MessageID == "" {
			result.MessageID = req.msg.MessageID
		}
		if result.SessionID == "" {
			result.SessionID = req.msg.SessionID
		}
		result = withQueueMetadata(result, wait, false, false)
		req.done <- queuedResponse{result: result, err: err}

		inFlight = int(atomic.AddInt64(&s.inFlight, -1))
		s.setWorkerInFlight(inFlight)

		s.finishSession(sessionID)
	}
}

func (s *ConcurrentService) nextRequest(ctx context.Context) (*queuedRequest, string, bool) {
	for {
		s.mu.Lock()
		for len(s.readySession) == 0 && !s.stopped && ctx.Err() == nil {
			s.cond.Wait()
		}

		if s.stopped || ctx.Err() != nil {
			s.mu.Unlock()
			return nil, "", false
		}

		sessionID := s.readySession[0]
		s.readySession = s.readySession[1:]
		state, ok := s.sessions[sessionID]
		if !ok {
			s.mu.Unlock()
			continue
		}
		state.inReadyQueue = false
		if state.running || len(state.pending) == 0 {
			if len(state.pending) == 0 && !state.running {
				delete(s.sessions, sessionID)
			}
			s.mu.Unlock()
			continue
		}

		req := state.pending[0]
		state.pending = state.pending[1:]
		state.running = true
		s.waiting--
		queueDepth := s.waiting
		s.mu.Unlock()

		s.setQueueDepth(queueDepth)
		return req, sessionID, true
	}
}

func (s *ConcurrentService) enqueue(req *queuedRequest) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return s.waiting, ErrRateLimited
	}

	if s.options.OverloadPolicy == OverloadPolicyRejectNew && s.waiting >= s.options.MaxQueueSize {
		s.countQueueEvent("rejected")
		return s.waiting, ErrRateLimited
	}

	state, ok := s.sessions[req.msg.SessionID]
	if !ok {
		state = &sessionQueue{}
		s.sessions[req.msg.SessionID] = state
	}

	state.pending = append(state.pending, req)
	s.waiting++
	if !state.running && !state.inReadyQueue {
		state.inReadyQueue = true
		s.readySession = append(s.readySession, req.msg.SessionID)
		s.cond.Signal()
	}

	queueDepth := s.waiting
	s.countQueueEvent("enqueued")
	s.setQueueDepth(queueDepth)
	return queueDepth, nil
}

func (s *ConcurrentService) removePending(sessionID string, req *queuedRequest) (bool, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.sessions[sessionID]
	if !ok {
		return false, s.waiting
	}

	idx := -1
	for i := range state.pending {
		if state.pending[i] == req {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false, s.waiting
	}

	state.pending = append(state.pending[:idx], state.pending[idx+1:]...)
	s.waiting--
	queueDepth := s.waiting

	if len(state.pending) == 0 {
		if state.inReadyQueue {
			s.removeReadySessionLocked(sessionID)
			state.inReadyQueue = false
		}
		if !state.running {
			delete(s.sessions, sessionID)
		}
	}

	s.setQueueDepth(queueDepth)
	return true, queueDepth
}

func (s *ConcurrentService) removeReadySessionLocked(sessionID string) {
	for idx := range s.readySession {
		if s.readySession[idx] != sessionID {
			continue
		}
		s.readySession = append(s.readySession[:idx], s.readySession[idx+1:]...)
		return
	}
}

func (s *ConcurrentService) finishSession(sessionID string) {
	s.mu.Lock()
	state, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return
	}

	state.running = false
	if len(state.pending) == 0 {
		if !state.inReadyQueue {
			delete(s.sessions, sessionID)
		}
		s.mu.Unlock()
		return
	}

	if !state.inReadyQueue {
		state.inReadyQueue = true
		s.readySession = append(s.readySession, sessionID)
		s.cond.Signal()
	}
	s.mu.Unlock()
}

func (s *ConcurrentService) logQueueOutcome(message string, msg shareddomain.UnifiedMessage, wait time.Duration, timeout bool, degraded bool, queueDepth int) {
	attrs := []any{
		slog.String("trace_id", msg.TraceID),
		slog.String("session_id", msg.SessionID),
		slog.String("message_id", msg.MessageID),
		slog.Int64("queue_wait_ms", wait.Milliseconds()),
		slog.Bool("timeout", timeout),
		slog.Bool("degraded", degraded),
		slog.Int("queue_depth", queueDepth),
	}
	s.logger.Warn(message, attrs...)
}

func withQueueMetadata(
	result shareddomain.OrchestrationResult,
	wait time.Duration,
	timeout bool,
	degraded bool,
) shareddomain.OrchestrationResult {
	metadata := cloneMetadata(result.Metadata)
	metadata["queue_wait_ms"] = strconv.FormatInt(wait.Milliseconds(), 10)
	metadata["timeout"] = strconv.FormatBool(timeout)
	if degraded {
		metadata["degraded"] = "true"
	}
	result.Metadata = metadata
	return result
}

func cloneMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func (s *ConcurrentService) countQueueEvent(event string) {
	if s.queueStats != nil {
		s.queueStats.CountQueueEvent(event)
	}
}

func (s *ConcurrentService) observeQueueWait(wait time.Duration) {
	if s.queueStats != nil {
		s.queueStats.ObserveQueueWait(wait)
	}
}

func (s *ConcurrentService) setQueueDepth(depth int) {
	if s.queueStats != nil {
		s.queueStats.SetQueueDepth(depth)
	}
}

func (s *ConcurrentService) setWorkerInFlight(inFlight int) {
	if s.queueStats != nil {
		s.queueStats.SetWorkerInFlight(inFlight)
	}
}
