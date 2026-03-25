package domain

import (
	"context"
	"strings"
	"testing"
)

type reactScriptClient struct {
	call int
}

func (c *reactScriptClient) Chat(_ context.Context, req ChatRequest) (*ChatResponse, error) {
	c.call++
	switch c.call {
	case 1:
		return &ChatResponse{
			Message: Message{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{ID: "call-1", Name: "lookup", Arguments: `{}`},
				},
			},
		}, nil
	case 2:
		if len(req.Messages) < 4 {
			return nil, &reactTestError{text: "missing follow-up messages"}
		}
		last := req.Messages[len(req.Messages)-1]
		if last.Role != "user" || !strings.Contains(last.Content, "最新补充") {
			return nil, &reactTestError{text: "latest user message was not injected"}
		}
		previous := req.Messages[len(req.Messages)-2]
		if previous.Role != "tool" || !strings.Contains(previous.Content, "工具观察") {
			return nil, &reactTestError{text: "tool observation missing before latest user message"}
		}
		return &ChatResponse{
			Message: Message{
				Role:    "assistant",
				Content: "已按最新补充继续执行",
			},
		}, nil
	default:
		return &ChatResponse{
			Message: Message{
				Role:    "assistant",
				Content: "unexpected",
			},
		}, nil
	}
}

func (c *reactScriptClient) ChatStream(_ context.Context, _ ChatRequest, _ func(StreamEvent) error) (*ChatResponse, error) {
	return nil, nil
}

func (c *reactScriptClient) Close() error {
	return nil
}

type reactToolExecutor struct {
	call int
}

func (e *reactToolExecutor) Execute(_ context.Context, toolCall ToolCall) (*ToolResult, error) {
	e.call++
	return &ToolResult{
		ToolCallID: toolCall.ID,
		Name:       toolCall.Name,
		Result:     "工具观察",
	}, nil
}

type reactLatestMessageSource struct {
	message string
	used    bool
}

func (s *reactLatestMessageSource) ConsumeLatest(_ context.Context) (string, bool) {
	if s.used || strings.TrimSpace(s.message) == "" {
		return "", false
	}
	s.used = true
	return s.message, true
}

type reactTestError struct {
	text string
}

func (e *reactTestError) Error() string {
	return e.text
}

func TestReActAgentInjectsLatestUserMessageBetweenIterations(t *testing.T) {
	agent := NewReActAgent(ReActAgentConfig{
		Client:            &reactScriptClient{},
		Model:             "test-model",
		Tools:             []Tool{{Name: "lookup", Description: "lookup data"}},
		ToolExecutor:      &reactToolExecutor{},
		MaxIterations:     3,
		UserMessagePuller: (&reactLatestMessageSource{message: "最新补充"}).ConsumeLatest,
	})

	state, err := agent.RunWithState(context.Background(), "初始请求", nil)
	if err != nil {
		t.Fatalf("RunWithState() error = %v", err)
	}
	if strings.TrimSpace(state.Answer) != "已按最新补充继续执行" {
		t.Fatalf("unexpected answer %q", state.Answer)
	}
}
