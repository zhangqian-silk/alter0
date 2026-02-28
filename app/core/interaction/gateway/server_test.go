package gateway

import (
	"context"
	"sync"
	"testing"
	"time"

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
