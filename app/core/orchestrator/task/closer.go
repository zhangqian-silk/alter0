package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"alter0/app/core/orchestrator/execlog"
)

type Closer struct {
	timeoutSec int
	confidence float64
}

func NewCloser(timeoutSec int, confidence float64) *Closer {
	if timeoutSec <= 0 {
		timeoutSec = 10
	}
	if confidence <= 0 || confidence > 1 {
		confidence = 0.70
	}
	return &Closer{
		timeoutSec: timeoutSec,
		confidence: confidence,
	}
}

func (c *Closer) Decide(ctx context.Context, executor string, task Task, history []TaskMessage, lastUser string, lastAssistant string) CloseDecision {
	prompt := buildClosePrompt(task, history, lastUser, lastAssistant)
	out, err := execlog.RunExecutorWithTimeout(ctx, executor, prompt, time.Duration(c.timeoutSec)*time.Second, execlog.StageCloser)
	if err != nil {
		return CloseDecision{Close: false, Reason: fmt.Sprintf("closer_failed: %v", err)}
	}
	d, err := parseCloseDecision(out)
	if err != nil {
		return CloseDecision{Close: false, Reason: fmt.Sprintf("closer_parse_failed: %v", err)}
	}
	if d.Confidence < c.confidence {
		return CloseDecision{Close: false, Reason: "closer_low_confidence"}
	}
	return d
}

func buildClosePrompt(task Task, history []TaskMessage, lastUser string, lastAssistant string) string {
	var b strings.Builder
	b.WriteString("You are a strict task closer.\n")
	b.WriteString("Decide whether this task is completed and should be closed now.\n")
	b.WriteString("Return JSON only with schema: {\"close\":true|false,\"confidence\":0.0,\"reason\":\"short\"}\n")
	b.WriteString("If uncertain, return close=false.\n\n")
	b.WriteString(fmt.Sprintf("task_id: %s\n", task.ID))
	b.WriteString(fmt.Sprintf("task_title: %s\n", task.Title))
	b.WriteString("recent_history:\n")
	start := 0
	if len(history) > 8 {
		start = len(history) - 8
	}
	for i := start; i < len(history); i++ {
		m := history[i]
		b.WriteString(fmt.Sprintf("- %s: %s\n", m.Role, m.Content))
	}
	b.WriteString("\n")
	b.WriteString("latest_user_message:\n")
	b.WriteString(lastUser + "\n")
	b.WriteString("latest_assistant_message:\n")
	b.WriteString(lastAssistant + "\n")
	return b.String()
}

func parseCloseDecision(text string) (CloseDecision, error) {
	payload, err := extractJSONObject(text)
	if err != nil {
		return CloseDecision{}, err
	}
	var d CloseDecision
	if err := json.Unmarshal([]byte(payload), &d); err != nil {
		return CloseDecision{}, err
	}
	return d, nil
}
