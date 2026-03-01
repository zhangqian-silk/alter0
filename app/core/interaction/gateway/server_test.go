package gateway

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"alter0/app/core/queue"
	"alter0/app/pkg/types"
)

type testAgent struct{}

func (a *testAgent) Process(_ context.Context, msg types.Message) (types.Message, error) {
	return types.Message{Content: "ok", TaskID: msg.TaskID}, nil
}

func (a *testAgent) Name() string {
	return "test"
}

type testChannel struct {
	id       string
	startFn  func(context.Context, func(types.Message)) error
	sendMu   sync.Mutex
	sentMsgs []types.Message
}

func (c *testChannel) Start(ctx context.Context, handler func(types.Message)) error {
	if c.startFn != nil {
		return c.startFn(ctx, handler)
	}
	<-ctx.Done()
	return nil
}

func (c *testChannel) Send(_ context.Context, msg types.Message) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	c.sentMsgs = append(c.sentMsgs, msg)
	return nil
}

func (c *testChannel) ID() string {
	return c.id
}

func TestHealthStatusIncludesRegisteredChannels(t *testing.T) {
	gw := NewGateway(&testAgent{})
	gw.RegisterChannel(&testChannel{id: "http"})
	gw.RegisterChannel(&testChannel{id: "cli"})

	status := gw.HealthStatus()
	if status.Started {
		t.Fatal("expected gateway to be stopped")
	}
	if len(status.RegisteredChannels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(status.RegisteredChannels))
	}
	if status.RegisteredChannels[0] != "cli" || status.RegisteredChannels[1] != "http" {
		t.Fatalf("channels should be sorted, got %v", status.RegisteredChannels)
	}
}

func TestHealthStatusTracksProcessedMessages(t *testing.T) {
	gw := NewGateway(&testAgent{})
	ch := &testChannel{id: "cli"}
	ch.startFn = func(ctx context.Context, handler func(types.Message)) error {
		handler(types.Message{
			ID:        "m1",
			Content:   "hello",
			ChannelID: "cli",
			UserID:    "u-1",
			TaskID:    "t-1",
		})
		<-ctx.Done()
		return nil
	}
	gw.RegisterChannel(ch)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- gw.Start(ctx) }()

	deadline := time.Now().Add(200 * time.Millisecond)
	for {
		status := gw.HealthStatus()
		if status.ProcessedMessages >= 1 {
			if !status.Started {
				t.Fatal("expected started=true")
			}
			if status.StartedAt.IsZero() {
				t.Fatal("expected non-zero started timestamp")
			}
			if status.LastMessageAt.IsZero() {
				t.Fatal("expected non-zero last message timestamp")
			}
			cancel()
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("gateway did not process message in time: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := <-done; err != nil {
		t.Fatalf("gateway start returned error: %v", err)
	}
}

func TestNormalizeReplyClonesEnvelopeAndMarksOutbound(t *testing.T) {
	request := types.Message{
		ID:        "m-1",
		ChannelID: "http",
		UserID:    "u-1",
		RequestID: "r-1",
		TaskID:    "t-1",
		Envelope: &types.MessageEnvelope{
			Direction: types.EnvelopeDirectionInbound,
			Channel:   "http",
			PeerID:    "u-1",
			MessageID: "m-1",
			Parts: []types.EnvelopePart{
				{Type: types.EnvelopePartText, Text: "hello"},
			},
		},
		Meta: map[string]interface{}{"k": "v"},
	}
	response := types.Message{Content: "ok"}

	normalizeReply(&response, request)

	if response.Role != types.MessageRoleAssistant {
		t.Fatalf("unexpected role: %s", response.Role)
	}
	if response.Envelope == nil {
		t.Fatal("expected envelope clone")
	}
	if response.Envelope == request.Envelope {
		t.Fatal("response envelope should be a clone")
	}
	if response.Envelope.Direction != types.EnvelopeDirectionOutbound {
		t.Fatalf("unexpected direction: %s", response.Envelope.Direction)
	}
	if response.Envelope.Channel != "http" || response.Envelope.PeerID != "u-1" {
		t.Fatalf("unexpected envelope routing fields: %+v", response.Envelope)
	}
	if len(response.Envelope.Parts) != 1 || response.Envelope.Parts[0].Text != "hello" {
		t.Fatalf("unexpected envelope parts: %+v", response.Envelope.Parts)
	}
}

type flakyAgent struct {
	calls atomic.Int32
}

func (a *flakyAgent) Process(_ context.Context, msg types.Message) (types.Message, error) {
	if a.calls.Add(1) == 1 {
		return types.Message{}, errors.New("temporary error")
	}
	return types.Message{Content: "ok", TaskID: msg.TaskID}, nil
}

func (a *flakyAgent) Name() string {
	return "flaky"
}

func TestGatewayDispatchWithQueueRetries(t *testing.T) {
	agent := &flakyAgent{}
	gw := NewGateway(agent)
	ch := &testChannel{id: "cli"}
	ch.startFn = func(ctx context.Context, handler func(types.Message)) error {
		handler(types.Message{
			ID:        "m1",
			Content:   "hello",
			ChannelID: "cli",
			UserID:    "u-1",
			TaskID:    "t-1",
		})
		<-ctx.Done()
		return nil
	}
	gw.RegisterChannel(ch)

	q := queue.New(8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := q.Start(ctx, 1); err != nil {
		t.Fatalf("queue start failed: %v", err)
	}
	defer q.Stop(200 * time.Millisecond)

	gw.SetExecutionQueue(q, QueueOptions{Enabled: true, MaxRetries: 1})

	done := make(chan error, 1)
	go func() { done <- gw.Start(ctx) }()

	deadline := time.Now().Add(300 * time.Millisecond)
	for {
		ch.sendMu.Lock()
		sent := len(ch.sentMsgs)
		ch.sendMu.Unlock()
		if sent >= 1 {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("expected queued reply")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := agent.calls.Load(); got != 2 {
		cancel()
		t.Fatalf("expected 2 attempts via queue retry, got %d", got)
	}

	stats := q.Stats()
	if stats.Retried < 1 {
		cancel()
		t.Fatalf("expected retried stats >= 1, got %+v", stats)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("gateway start returned error: %v", err)
	}
}
