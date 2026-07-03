package application

import (
	"strconv"
	"strings"
	"time"
)

type TurnSummary struct {
	ID                 string              `json:"id"`
	Prompt             string              `json:"prompt"`
	Attachments        []TurnAttachment    `json:"attachments,omitempty"`
	Status             string              `json:"status"`
	StartedAt          time.Time           `json:"started_at,omitempty"`
	FinishedAt         time.Time           `json:"finished_at,omitempty"`
	DurationMS         int64               `json:"duration_ms,omitempty"`
	FinalOutput        string              `json:"final_output,omitempty"`
	RuntimeTraceEvents []RuntimeTraceEvent `json:"runtime_trace_events,omitempty"`
}

type TurnAttachment struct {
	Name          string `json:"name"`
	ContentType   string `json:"content_type"`
	DataURL       string `json:"data_url,omitempty"`
	AssetURL      string `json:"asset_url,omitempty"`
	PreviewURL    string `json:"preview_url,omitempty"`
	WorkspacePath string `json:"-"`
}

type runtimeEventSummary struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"`
	Title      string               `json:"title"`
	Status     string               `json:"status"`
	StartedAt  time.Time            `json:"started_at,omitempty"`
	FinishedAt time.Time            `json:"finished_at,omitempty"`
	DurationMS int64                `json:"duration_ms,omitempty"`
	Preview    string               `json:"preview,omitempty"`
	HasDetail  bool                 `json:"has_detail"`
	Blocks     []RuntimeDetailBlock `json:"blocks,omitempty"`
}

type RuntimeTraceEventDetail struct {
	TurnID     string            `json:"turn_id"`
	Event      RuntimeTraceEvent `json:"event"`
	Blocks     []RuntimeBlock    `json:"blocks,omitempty"`
	Searchable bool              `json:"searchable,omitempty"`
}

type RuntimeDetailBlock struct {
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Content   string `json:"content,omitempty"`
	Language  string `json:"language,omitempty"`
	File      string `json:"file,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	Status    string `json:"status,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

type RuntimeTraceEvent struct {
	ID          string               `json:"id"`
	SessionID   string               `json:"session_id,omitempty"`
	TurnID      string               `json:"turn_id"`
	Seq         int                  `json:"seq"`
	Source      string               `json:"source"`
	Provider    RuntimeProviderRef   `json:"provider"`
	Role        string               `json:"role"`
	Kind        string               `json:"kind"`
	Lifecycle   string               `json:"lifecycle"`
	Status      string               `json:"status"`
	Title       string               `json:"title,omitempty"`
	Summary     string               `json:"summary,omitempty"`
	Blocks      []RuntimeBlock       `json:"blocks"`
	Action      *RuntimeAction       `json:"action,omitempty"`
	Visibility  string               `json:"visibility"`
	StartedAt   time.Time            `json:"started_at,omitempty"`
	CompletedAt time.Time            `json:"completed_at,omitempty"`
	DurationMS  int64                `json:"duration_ms,omitempty"`
	Raw         RuntimeTraceEventRaw `json:"raw,omitempty"`
}

type RuntimeProviderRef struct {
	Engine    string `json:"engine"`
	Adapter   string `json:"adapter"`
	EventType string `json:"event_type,omitempty"`
	ItemID    string `json:"item_id,omitempty"`
}

type RuntimeAction struct {
	Name   string `json:"name,omitempty"`
	Family string `json:"family,omitempty"`
}

type RuntimeTraceEventRaw struct {
	Ref       string `json:"ref,omitempty"`
	Type      string `json:"type,omitempty"`
	HasDetail bool   `json:"has_detail,omitempty"`
}

type RuntimeBlock struct {
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Text      string `json:"text,omitempty"`
	JSON      any    `json:"json,omitempty"`
	Command   string `json:"command,omitempty"`
	Output    string `json:"output,omitempty"`
	Content   string `json:"content,omitempty"`
	Language  string `json:"language,omitempty"`
	File      string `json:"file,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	Message   string `json:"message,omitempty"`
}

func chatRuntimeRuntimeTraceEvent(sessionID string, turnID string, seq int, summary runtimeEventSummary) RuntimeTraceEvent {
	eventType := strings.ToLower(strings.TrimSpace(summary.Type))
	kind := chatRuntimeRuntimeEventKind(eventType)
	status := chatRuntimeRuntimeStatus(summary.Status)
	title := strings.TrimSpace(summary.Title)
	if title == "" {
		title = strings.TrimSpace(summary.Preview)
	}
	if title == "" {
		title = chatRuntimeDefaultEventTitle(kind)
	}
	event := RuntimeTraceEvent{
		ID:        strings.TrimSpace(summary.ID),
		SessionID: sessionID,
		TurnID:    turnID,
		Seq:       seq,
		Source:    "adapter",
		Provider: RuntimeProviderRef{
			Engine:    "codex",
			Adapter:   "codex_cli_json",
			EventType: summary.Type,
			ItemID:    summary.ID,
		},
		Role:        chatRuntimeRuntimeRole(kind),
		Kind:        kind,
		Lifecycle:   chatRuntimeRuntimeLifecycle(summary.Status),
		Status:      status,
		Title:       title,
		Summary:     firstNonEmpty(summary.Preview, title),
		Blocks:      chatRuntimeRuntimeBlocks(summary.Blocks, summary.Preview, kind, summary.HasDetail),
		Action:      chatRuntimeRuntimeAction(kind),
		Visibility:  chatRuntimeRuntimeVisibility(kind),
		StartedAt:   summary.StartedAt,
		CompletedAt: summary.FinishedAt,
		DurationMS:  summary.DurationMS,
		Raw: RuntimeTraceEventRaw{
			Ref:       summary.ID,
			Type:      summary.Type,
			HasDetail: summary.HasDetail,
		},
	}
	if event.ID == "" {
		event.ID = turnID + ":event:" + strconv.Itoa(seq)
	}
	return event
}

func chatRuntimeRuntimeEventKind(eventType string) string {
	switch eventType {
	case "reasoning":
		return "reasoning"
	case "plan":
		return "plan"
	case "command", "command_execution":
		return "shell_command"
	case "log", "system":
		return "system_event"
	case "diff":
		return "file_edit"
	case "message":
		return "assistant_commentary"
	default:
		return "unknown_provider_event"
	}
}

func chatRuntimeRuntimeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued":
		return "queued"
	case "running", "busy", "starting":
		return "running"
	case "failed", "error":
		return "failed"
	case "interrupted":
		return "interrupted"
	case "cancelled", "canceled":
		return "canceled"
	case "completed", "success", "done", "":
		return "completed"
	default:
		return "unknown"
	}
}

func chatRuntimeRuntimeLifecycle(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "queued", "busy", "starting":
		return "started"
	case "failed", "error":
		return "failed"
	case "interrupted":
		return "interrupted"
	case "cancelled", "canceled":
		return "canceled"
	case "completed", "success", "done", "":
		return "completed"
	default:
		return "updated"
	}
}

func chatRuntimeRuntimeRole(kind string) string {
	if kind == "tool_result" {
		return "tool"
	}
	return "assistant"
}

func chatRuntimeRuntimeVisibility(kind string) string {
	switch kind {
	case "assistant_commentary", "plan", "reasoning", "shell_command":
		return "collapsed"
	default:
		return "developer"
	}
}

func chatRuntimeRuntimeAction(kind string) *RuntimeAction {
	if kind != "shell_command" {
		return nil
	}
	return &RuntimeAction{Family: "shell", Name: "shell"}
}

func chatRuntimeRuntimeBlocks(blocks []RuntimeDetailBlock, preview string, kind string, hasDetail bool) []RuntimeBlock {
	result := make([]RuntimeBlock, 0, len(blocks))
	for _, block := range blocks {
		blockType := strings.ToLower(strings.TrimSpace(block.Type))
		content := block.Content
		switch blockType {
		case "chatRuntime":
			command, output := splitChatRuntimeContent(content)
			result = append(result, RuntimeBlock{
				Type:     "chatRuntime",
				Title:    strings.TrimSpace(block.Title),
				Command:  strings.TrimSpace(command),
				Output:   strings.TrimSpace(output),
				Language: "shell",
				ExitCode: block.ExitCode,
			})
		case "diff":
			result = append(result, RuntimeBlock{
				Type:      "diff",
				Title:     strings.TrimSpace(block.Title),
				Content:   content,
				Language:  strings.TrimSpace(block.Language),
				File:      strings.TrimSpace(block.File),
				StartLine: block.StartLine,
			})
		case "code":
			result = append(result, RuntimeBlock{
				Type:      "code",
				Title:     strings.TrimSpace(block.Title),
				Content:   content,
				Language:  strings.TrimSpace(block.Language),
				File:      strings.TrimSpace(block.File),
				StartLine: block.StartLine,
			})
		default:
			if strings.TrimSpace(content) != "" {
				result = append(result, RuntimeBlock{
					Type:  "markdown",
					Title: strings.TrimSpace(block.Title),
					Text:  content,
				})
			}
		}
	}
	if len(result) > 0 {
		return result
	}
	if hasDetail {
		return []RuntimeBlock{}
	}
	fallback := strings.TrimSpace(preview)
	if fallback == "" {
		return []RuntimeBlock{}
	}
	if kind == "shell_command" {
		return []RuntimeBlock{{Type: "chatRuntime", Command: fallback, Language: "shell"}}
	}
	return []RuntimeBlock{{Type: "markdown", Text: fallback}}
}

func splitChatRuntimeContent(content string) (string, string) {
	parts := strings.SplitN(content, "\n\n", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func chatRuntimeDefaultEventTitle(kind string) string {
	switch kind {
	case "assistant_commentary":
		return "Progress"
	case "reasoning":
		return "Reasoning"
	case "plan":
		return "Plan"
	case "shell_command":
		return "Shell"
	case "system_event":
		return "System"
	default:
		return "Event"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if normalized := strings.TrimSpace(value); normalized != "" {
			return normalized
		}
	}
	return ""
}
