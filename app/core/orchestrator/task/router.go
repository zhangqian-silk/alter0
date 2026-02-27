package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"alter0/app/core/orchestrator/execlog"
)

type Router struct {
	timeoutSec  int
	confidence  float64
	maxCandSize int
}

func NewRouter(timeoutSec int, confidence float64, maxCandSize int) *Router {
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	if confidence <= 0 || confidence > 1 {
		confidence = 0.55
	}
	if maxCandSize <= 0 {
		maxCandSize = 8
	}
	return &Router{
		timeoutSec:  timeoutSec,
		confidence:  confidence,
		maxCandSize: maxCandSize,
	}
}

func (r *Router) Decide(ctx context.Context, executor string, userID string, channelID string, userInput string, candidates []Task) RouteDecision {
	prior := "prefer_new"
	if strings.EqualFold(channelID, "cli") {
		prior = "prefer_existing"
	}

	if len(candidates) > r.maxCandSize {
		candidates = candidates[:r.maxCandSize]
	}

	prompt := buildRoutePrompt(userID, channelID, prior, userInput, candidates)
	out, err := execlog.RunExecutorWithTimeout(ctx, executor, prompt, time.Duration(r.timeoutSec)*time.Second, execlog.StageRouter)
	if err != nil {
		return RouteDecision{
			Decision: "new",
			Reason:   fmt.Sprintf("router_failed: %v", err),
		}
	}

	decision, err := parseRouteDecision(out)
	if err != nil {
		return RouteDecision{
			Decision: "new",
			Reason:   fmt.Sprintf("router_parse_failed: %v", err),
		}
	}

	decision.Decision = strings.ToLower(strings.TrimSpace(decision.Decision))
	if decision.Decision != "existing" && decision.Decision != "new" {
		return RouteDecision{
			Decision: "new",
			Reason:   "router_invalid_decision",
		}
	}
	if decision.Confidence < r.confidence {
		return RouteDecision{
			Decision: "new",
			Reason:   "router_low_confidence",
		}
	}
	if decision.Decision == "existing" {
		if !hasTask(candidates, decision.TaskID) {
			return RouteDecision{
				Decision: "new",
				Reason:   "router_invalid_task_id",
			}
		}
	}
	if decision.Decision == "new" {
		decision.TaskID = ""
	}
	return decision
}

func buildRoutePrompt(userID string, channelID string, prior string, userInput string, candidates []Task) string {
	var b strings.Builder
	b.WriteString("You are a strict task router.\n")
	b.WriteString("Choose whether the incoming message belongs to an existing OPEN task or should create a NEW task.\n")
	b.WriteString("Return JSON only.\n\n")
	b.WriteString("JSON schema:\n")
	b.WriteString("{\"decision\":\"existing|new\",\"task_id\":\"optional\",\"title\":\"optional\",\"confidence\":0.0,\"reason\":\"short\"}\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- If uncertain, choose new.\n")
	b.WriteString("- Confidence is between 0 and 1.\n")
	b.WriteString("- If decision=existing, task_id must be one of candidate ids.\n")
	b.WriteString("- For CLI channel, prefer existing. For other channels, prefer new.\n\n")
	b.WriteString("Context:\n")
	b.WriteString("user_id: " + userID + "\n")
	b.WriteString("channel_id: " + channelID + "\n")
	b.WriteString("channel_prior: " + prior + "\n")
	b.WriteString("user_input: " + userInput + "\n\n")
	b.WriteString("open_tasks:\n")
	if len(candidates) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, t := range candidates {
			b.WriteString(fmt.Sprintf("- id=%s title=%q updated_at=%d\n", t.ID, t.Title, t.UpdatedAt))
		}
	}
	return b.String()
}

func parseRouteDecision(text string) (RouteDecision, error) {
	payload, err := extractJSONObject(text)
	if err != nil {
		return RouteDecision{}, err
	}
	var d RouteDecision
	if err := json.Unmarshal([]byte(payload), &d); err != nil {
		return RouteDecision{}, err
	}
	return d, nil
}

func hasTask(candidates []Task, taskID string) bool {
	for _, t := range candidates {
		if t.ID == taskID {
			return true
		}
	}
	return false
}

func extractJSONObject(text string) (string, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("json object not found")
	}
	return text[start : end+1], nil
}
