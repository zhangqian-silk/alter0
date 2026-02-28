package agent

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	config "alter0/app/configs"
	"alter0/app/core/orchestrator/command"
	"alter0/app/core/orchestrator/execlog"
	"alter0/app/core/orchestrator/skills"
	"alter0/app/core/orchestrator/task"
	"alter0/app/pkg/types"
)

type DefaultAgent struct {
	name         string
	executorName string
	taskConfig   config.TaskConfig

	skillMgr  *skills.Manager
	taskStore *task.Store
	command   *command.Executor
	router    *task.Router
	closer    *task.Closer

	mu sync.RWMutex
}

func NewAgent(name string, skillMgr *skills.Manager, taskStore *task.Store, executorName string, taskConfig config.TaskConfig, securityConfig config.SecurityConfig) *DefaultAgent {
	a := &DefaultAgent{
		name:         name,
		executorName: executorName,
		taskConfig:   taskConfig,
		skillMgr:     skillMgr,
		taskStore:    taskStore,
	}
	a.command = command.NewExecutor(skillMgr, taskStore, securityConfig.AdminUserIDs)
	a.router = task.NewRouter(taskConfig.RoutingTimeoutSec, taskConfig.RoutingConfidenceThreshold, taskConfig.OpenTaskCandidateLimit)
	a.closer = task.NewCloser(taskConfig.CloseTimeoutSec, taskConfig.CloseConfidenceThreshold)
	return a
}

func (a *DefaultAgent) Process(ctx context.Context, msg types.Message) (types.Message, error) {
	sessionChannel := strings.TrimSpace(msg.ChannelID)
	if sessionChannel == "" {
		sessionChannel = "unknown"
	}
	userID := strings.TrimSpace(msg.UserID)
	if userID == "" {
		userID = "anonymous"
	}
	trimmed := strings.TrimSpace(msg.Content)
	if trimmed == "" {
		return types.Message{
			ID:        msg.ID,
			ChannelID: sessionChannel,
			UserID:    userID,
			RequestID: msg.RequestID,
			TaskID:    msg.TaskID,
			Meta:      msg.Meta,
		}, nil
	}

	msg.ChannelID = sessionChannel
	msg.UserID = userID
	msg.Content = trimmed
	if strings.TrimSpace(msg.RequestID) == "" {
		msg.RequestID = msg.ID
	}

	if strings.HasPrefix(trimmed, "/") {
		out, handled, err := a.command.ExecuteSlash(ctx, msg)
		if handled {
			if err != nil {
				return a.newReply(msg, fmt.Sprintf("Command failed: %v", err), map[string]interface{}{"command_error": true}), nil
			}
			return a.newReply(msg, out, nil), nil
		}
	}

	return a.processTaskMessage(ctx, msg), nil
}

func (a *DefaultAgent) processTaskMessage(ctx context.Context, msg types.Message) types.Message {
	sessionID := strings.TrimSpace(msg.RequestID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(msg.ID)
	}
	if sessionID == "" {
		sessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}

	baseExecCtx := execlog.WithMeta(ctx, execlog.Meta{
		SessionID: sessionID,
		TaskHint:  deriveTaskTitle(msg.Content),
		UserID:    msg.UserID,
		ChannelID: msg.ChannelID,
	})

	selectedTask, decision := a.selectTask(baseExecCtx, msg)
	if selectedTask.ID == "" {
		return a.newReply(msg, "Task routing failed: unable to select task", map[string]interface{}{
			"decision": "new",
			"closed":   false,
		})
	}

	if err := a.taskStore.AppendMessage(ctx, selectedTask.ID, msg.UserID, msg.ChannelID, "user", msg.Content, msg.Meta); err != nil {
		log.Printf("[Agent] Failed to persist user message: %v", err)
	}

	history, err := a.taskStore.GetTaskHistory(ctx, selectedTask.ID, 40)
	if err != nil {
		log.Printf("[Agent] Failed to load task history: %v", err)
	}

	memorySummary := ""
	if memory, memErr := a.taskStore.GetTaskMemory(ctx, selectedTask.ID); memErr == nil {
		memorySummary = memory.Summary
	} else if memErr != sql.ErrNoRows {
		log.Printf("[Agent] Failed to load task memory: %v", memErr)
	}

	prompt := a.buildGenerationPrompt(selectedTask, history, memorySummary, msg.Content)
	taskCfg := a.getTaskConfig()
	output, runErr := execlog.RunExecutorWithTimeout(
		execlog.WithMeta(baseExecCtx, execlog.Meta{
			SessionID: sessionID,
			Stage:     execlog.StageGen,
			TaskID:    selectedTask.ID,
			TaskHint:  selectedTask.Title,
			UserID:    msg.UserID,
			ChannelID: msg.ChannelID,
		}),
		a.getExecutorName(),
		prompt,
		time.Duration(taskCfg.GenerationTimeoutSec)*time.Second,
		execlog.StageGen,
	)
	if runErr != nil {
		output = fmt.Sprintf("CLI Agent failed: %v", runErr)
	}

	replyMeta := map[string]interface{}{
		"decision": decision.Decision,
		"closed":   false,
	}
	if decision.Reason != "" {
		replyMeta["route_reason"] = decision.Reason
	}

	if err := a.taskStore.AppendMessage(ctx, selectedTask.ID, msg.UserID, msg.ChannelID, "assistant", output, replyMeta); err != nil {
		log.Printf("[Agent] Failed to persist assistant message: %v", err)
	}

	latestHistory, err := a.taskStore.GetTaskHistory(ctx, selectedTask.ID, 10)
	if err != nil {
		log.Printf("[Agent] Failed to load latest history: %v", err)
		latestHistory = history
	}
	memorySnapshot := buildTaskMemorySnapshot(latestHistory, 1200)
	if memorySnapshot != "" {
		if memErr := a.taskStore.UpsertTaskMemory(ctx, selectedTask.ID, memorySnapshot); memErr != nil {
			log.Printf("[Agent] Failed to persist task memory: %v", memErr)
		}
	}

	closeDecision := a.closer.Decide(
		execlog.WithMeta(baseExecCtx, execlog.Meta{
			SessionID: sessionID,
			Stage:     execlog.StageCloser,
			TaskID:    selectedTask.ID,
			TaskHint:  selectedTask.Title,
			UserID:    msg.UserID,
			ChannelID: msg.ChannelID,
		}),
		a.getExecutorName(),
		selectedTask,
		latestHistory,
		msg.Content,
		output,
	)
	closed := false
	if closeDecision.Close {
		if err := a.taskStore.CloseTask(ctx, selectedTask.ID); err != nil {
			log.Printf("[Agent] Failed to close task %s: %v", selectedTask.ID, err)
		} else {
			closed = true
		}
	}
	replyMeta["closed"] = closed
	if closeDecision.Reason != "" {
		replyMeta["close_reason"] = closeDecision.Reason
	}

	reply := a.newReply(msg, output, replyMeta)
	reply.TaskID = selectedTask.ID
	return reply
}

func (a *DefaultAgent) selectTask(ctx context.Context, msg types.Message) (task.Task, task.RouteDecision) {
	userID := msg.UserID
	channelID := msg.ChannelID
	userInput := msg.Content

	// explicit task id has highest priority
	if explicitID := strings.TrimSpace(msg.TaskID); explicitID != "" {
		if t, err := a.taskStore.GetTask(ctx, explicitID); err == nil && t.UserID == userID && t.Status == "open" {
			return t, task.RouteDecision{
				Decision:   "existing",
				TaskID:     t.ID,
				Confidence: 1,
				Reason:     "explicit_task_id",
			}
		}
	}

	// one-shot forced task from command override
	if forcedID, err := a.taskStore.ConsumeForcedTask(ctx, userID); err == nil && strings.TrimSpace(forcedID) != "" {
		if t, getErr := a.taskStore.GetTask(ctx, forcedID); getErr == nil && t.UserID == userID && t.Status == "open" {
			return t, task.RouteDecision{
				Decision:   "existing",
				TaskID:     t.ID,
				Confidence: 1,
				Reason:     "forced_task_override",
			}
		}
	}

	taskCfg := a.getTaskConfig()
	candidates, err := a.taskStore.ListOpenTasks(ctx, userID, taskCfg.OpenTaskCandidateLimit)
	if err != nil {
		log.Printf("[Agent] Failed to list open tasks: %v", err)
		candidates = nil
	}

	routeCtx := execlog.WithMeta(ctx, execlog.Meta{
		SessionID: strings.TrimSpace(msg.RequestID),
		Stage:     execlog.StageRouter,
		TaskID:    strings.TrimSpace(msg.TaskID),
		TaskHint:  deriveTaskTitle(userInput),
		UserID:    userID,
		ChannelID: channelID,
	})
	decision := a.router.Decide(routeCtx, a.getExecutorName(), userID, channelID, userInput, candidates)
	if decision.Decision == "existing" {
		for _, c := range candidates {
			if c.ID == decision.TaskID {
				return c, decision
			}
		}
	}

	title := strings.TrimSpace(decision.Title)
	if title == "" {
		title = deriveTaskTitle(userInput)
	}
	created, err := a.taskStore.CreateTask(ctx, userID, title, channelID)
	if err != nil {
		log.Printf("[Agent] Failed to create task: %v", err)
		return task.Task{}, task.RouteDecision{Decision: "new", Reason: "task_create_failed"}
	}
	decision.Decision = "new"
	decision.TaskID = created.ID
	if decision.Reason == "" {
		decision.Reason = "new_task_created"
	}
	return created, decision
}

func deriveTaskTitle(text string) string {
	s := strings.TrimSpace(text)
	if s == "" {
		return "Untitled Task"
	}
	max := 30
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

func buildTaskMemorySnapshot(history []task.TaskMessage, maxRunes int) string {
	if len(history) == 0 {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = 1200
	}
	start := 0
	if len(history) > 8 {
		start = len(history) - 8
	}
	var b strings.Builder
	for i := start; i < len(history); i++ {
		m := history[i]
		line := strings.TrimSpace(m.Content)
		if line == "" {
			continue
		}
		b.WriteString(m.Role)
		b.WriteString(": ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return truncateRunes(strings.TrimSpace(b.String()), maxRunes)
}

func truncateRunes(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	if maxRunes <= 1 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-1]) + "..."
}

func (a *DefaultAgent) buildGenerationPrompt(t task.Task, history []task.TaskMessage, memorySummary string, userInput string) string {
	var b strings.Builder
	b.WriteString("You are ")
	b.WriteString(a.Name())
	b.WriteString(", an execution-focused assistant.\n")
	b.WriteString("Respond to the latest user input using the task history.\n")
	b.WriteString("If information is insufficient, ask a concise follow-up question.\n\n")
	b.WriteString("Task:\n")
	b.WriteString(fmt.Sprintf("- id: %s\n", t.ID))
	b.WriteString(fmt.Sprintf("- title: %s\n", t.Title))
	b.WriteString(fmt.Sprintf("- status: %s\n\n", t.Status))
	if strings.TrimSpace(memorySummary) != "" {
		b.WriteString("Task memory snapshot:\n")
		b.WriteString(memorySummary)
		b.WriteString("\n\n")
	}
	b.WriteString("Recent history:\n")
	start := 0
	if len(history) > 20 {
		start = len(history) - 20
	}
	for i := start; i < len(history); i++ {
		m := history[i]
		b.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}
	b.WriteString("\nLatest user input:\n")
	b.WriteString(userInput)
	b.WriteString("\n\nReturn plain text only.")
	return b.String()
}

func (a *DefaultAgent) newReply(msg types.Message, content string, meta map[string]interface{}) types.Message {
	if meta == nil {
		meta = map[string]interface{}{}
	}
	for k, v := range msg.Meta {
		if _, exists := meta[k]; !exists {
			meta[k] = v
		}
	}
	return types.Message{
		ID:        fmt.Sprintf("asst-%d", time.Now().UnixNano()),
		Content:   content,
		Role:      "assistant",
		ChannelID: msg.ChannelID,
		UserID:    msg.UserID,
		TaskID:    msg.TaskID,
		RequestID: msg.RequestID,
		Meta:      meta,
	}
}

func (a *DefaultAgent) Name() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.name
}

func (a *DefaultAgent) SetName(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.name = name
}

func (a *DefaultAgent) SetExecutor(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.executorName = name
}

func (a *DefaultAgent) SetTaskConfig(taskCfg config.TaskConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.taskConfig = taskCfg
	a.router = task.NewRouter(taskCfg.RoutingTimeoutSec, taskCfg.RoutingConfidenceThreshold, taskCfg.OpenTaskCandidateLimit)
	a.closer = task.NewCloser(taskCfg.CloseTimeoutSec, taskCfg.CloseConfidenceThreshold)
}

func (a *DefaultAgent) SetSecurityConfig(securityCfg config.SecurityConfig) {
	a.command.SetAdminUsers(securityCfg.AdminUserIDs)
}

func (a *DefaultAgent) SetStatusProvider(provider func(context.Context) map[string]interface{}) {
	a.command.SetStatusProvider(provider)
}

func (a *DefaultAgent) getExecutorName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.executorName
}

func (a *DefaultAgent) getTaskConfig() config.TaskConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.taskConfig
}
