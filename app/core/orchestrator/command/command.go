package command

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"alter0/app/core/orchestrator/skills"
	"alter0/app/core/orchestrator/task"
	"alter0/app/pkg/types"
)

type Executor struct {
	skillMgr       *skills.Manager
	taskStore      *task.Store
	statusProvider func(context.Context) map[string]interface{}

	mu         sync.RWMutex
	adminUsers map[string]struct{}
}

type commandAuditEntry struct {
	Timestamp string `json:"timestamp"`
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	RequestID string `json:"request_id"`
	Command   string `json:"command"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason,omitempty"`
}

var (
	commandAuditMu       sync.Mutex
	commandAuditBasePath = filepath.Join("output", "audit")
)

func NewExecutor(skillMgr *skills.Manager, taskStore *task.Store, adminUserIDs []string) *Executor {
	e := &Executor{
		skillMgr:  skillMgr,
		taskStore: taskStore,
	}
	e.SetAdminUsers(adminUserIDs)
	return e
}

func (e *Executor) SetAdminUsers(adminUserIDs []string) {
	clean := map[string]struct{}{}
	for _, userID := range adminUserIDs {
		trimmed := strings.TrimSpace(userID)
		if trimmed == "" {
			continue
		}
		clean[trimmed] = struct{}{}
	}
	e.mu.Lock()
	e.adminUsers = clean
	e.mu.Unlock()
}

func (e *Executor) SetStatusProvider(provider func(context.Context) map[string]interface{}) {
	e.statusProvider = provider
}

func (e *Executor) ExecuteSlash(ctx context.Context, msg types.Message) (string, bool, error) {
	cmd := strings.TrimSpace(strings.TrimPrefix(msg.Content, "/"))
	if cmd == "" {
		return "", false, nil
	}
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", false, nil
	}
	skillName := parts[0]
	auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "attempt", "")
	if err := e.authorizeCommand(msg.UserID, parts); err != nil {
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
		return "", true, err
	}
	switch skillName {
	case "help":
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "allow", "")
		return e.helpText(), true, nil
	case "status":
		out, err := e.runtimeStatusOutput(ctx)
		if err != nil {
			auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
			return "", true, err
		}
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "allow", "")
		return out, true, nil
	case "task":
		out, err := e.executeTaskCommand(ctx, msg.UserID, parts[1:])
		if err != nil {
			auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
			return out, true, err
		}
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "allow", "")
		return out, true, nil
	case "config", "executor":
		args := map[string]interface{}{}
		if len(parts) > 1 {
			args["command"] = strings.Join(parts[1:], " ")
		}
		if skillName == "config" {
			e.fillConfigArgs(parts, args)
		}
		result, err := e.skillMgr.Execute(ctx, skillName, args)
		if err != nil {
			auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
			return "", true, err
		}
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "allow", "")
		if skillName == "config" {
			return formatConfigResult(result), true, nil
		}
		return fmt.Sprintf("%v", result), true, nil
	default:
		err := fmt.Errorf("unknown command: %s", skillName)
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
		return "", true, err
	}
}

func auditCommand(userID string, channelID string, requestID string, command string, decision string, reason string) {
	line := formatAuditCommandLine(userID, channelID, requestID, command, decision, reason)
	log.Print(line)
	if err := appendCommandAuditEntry(time.Now(), userID, channelID, requestID, command, decision, reason); err != nil {
		log.Printf("[AUDIT] failed to append command audit entry: %v", err)
	}
}

func formatAuditCommandLine(userID string, channelID string, requestID string, command string, decision string, reason string) string {
	user := normalizeAuditUserID(userID)
	channel := normalizeAuditChannelID(channelID)
	request := normalizeAuditRequestID(requestID)
	line := fmt.Sprintf("[AUDIT] user=%s channel=%s request=%s decision=%s command=%q", user, channel, request, decision, command)
	if strings.TrimSpace(reason) != "" {
		line += fmt.Sprintf(" reason=%q", reason)
	}
	return line
}

func appendCommandAuditEntry(ts time.Time, userID string, channelID string, requestID string, command string, decision string, reason string) error {
	record := commandAuditEntry{
		Timestamp: ts.UTC().Format(time.RFC3339Nano),
		UserID:    normalizeAuditUserID(userID),
		ChannelID: normalizeAuditChannelID(channelID),
		RequestID: normalizeAuditRequestID(requestID),
		Command:   strings.TrimSpace(command),
		Decision:  strings.TrimSpace(decision),
		Reason:    strings.TrimSpace(reason),
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	dayDir := filepath.Join(commandAuditBasePath, ts.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		return err
	}
	logPath := filepath.Join(dayDir, "command_permission.jsonl")

	commandAuditMu.Lock()
	defer commandAuditMu.Unlock()

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(payload, '\n'))
	return err
}

func normalizeAuditUserID(userID string) string {
	user := strings.TrimSpace(userID)
	if user == "" {
		return "anonymous"
	}
	return user
}

func normalizeAuditChannelID(channelID string) string {
	channel := strings.TrimSpace(channelID)
	if channel == "" {
		return "unknown"
	}
	return channel
}

func normalizeAuditRequestID(requestID string) string {
	request := strings.TrimSpace(requestID)
	if request == "" {
		return "n/a"
	}
	return request
}

func (e *Executor) authorizeCommand(userID string, parts []string) error {
	if len(parts) == 0 {
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(parts[0]))
	requiresAdmin := false
	switch name {
	case "executor":
		requiresAdmin = true
	case "config":
		if len(parts) > 1 && strings.EqualFold(parts[1], "set") {
			requiresAdmin = true
		}
	}
	if !requiresAdmin {
		return nil
	}
	if e.isAdminUser(userID) {
		return nil
	}
	return fmt.Errorf("permission denied: admin required for /%s", name)
}

func (e *Executor) isAdminUser(userID string) bool {
	trimmed := strings.TrimSpace(userID)
	if trimmed == "" {
		return false
	}
	e.mu.RLock()
	_, ok := e.adminUsers[trimmed]
	e.mu.RUnlock()
	return ok
}

func (e *Executor) helpText() string {
	configNames, taskAvailable := e.splitCommands()
	sort.Strings(configNames)
	var b strings.Builder
	b.WriteString("Commands:\n")
	b.WriteString("  /help\n")
	b.WriteString("  /status\n")
	b.WriteString("Config:\n")
	for _, name := range configNames {
		switch name {
		case "config":
			b.WriteString("  /config\n")
			b.WriteString("  /config get [key]\n")
			b.WriteString("  /config set <key> <value>\n")
		case "executor":
			b.WriteString("  /executor [name]\n")
		}
	}
	if taskAvailable {
		b.WriteString("Task:\n")
		b.WriteString("  /task list [open|closed|all]\n")
		b.WriteString("  /task current\n")
		b.WriteString("  /task use <task_id>\n")
		b.WriteString("  /task new [title]\n")
		b.WriteString("  /task close [task_id]\n")
		b.WriteString("  /task memory [task_id]\n")
		b.WriteString("  /task memory clear [task_id]\n")
		b.WriteString("  /task stats\n")
	}
	return strings.TrimSpace(b.String())
}

func (e *Executor) splitCommands() ([]string, bool) {
	manifests := e.skillMgr.ListSkills()
	configSet := map[string]struct{}{
		"config":   {},
		"executor": {},
	}
	configNames := make([]string, 0, len(configSet))
	for _, m := range manifests {
		if _, ok := configSet[m.Name]; ok {
			configNames = append(configNames, m.Name)
		}
	}
	return configNames, e.taskStore != nil
}

func (e *Executor) runtimeStatusOutput(ctx context.Context) (string, error) {
	if e.statusProvider == nil {
		return "", fmt.Errorf("status provider is not configured")
	}
	payload := e.statusProvider(ctx)
	if payload == nil {
		payload = map[string]interface{}{}
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (e *Executor) fillConfigArgs(parts []string, args map[string]interface{}) {
	if len(parts) == 1 {
		args["action"] = "get"
		return
	}
	action := strings.ToLower(parts[1])
	if action == "get" || action == "set" {
		args["action"] = action
		if len(parts) >= 3 {
			args["key"] = parts[2]
			if len(parts) > 3 {
				args["value"] = strings.Join(parts[3:], " ")
			} else {
				args["value"] = ""
			}
		}
		return
	}
	args["action"] = "get"
	args["key"] = parts[1]
}

func (e *Executor) executeTaskCommand(ctx context.Context, userID string, args []string) (string, error) {
	if e.taskStore == nil {
		return "", fmt.Errorf("task store is not available")
	}
	if strings.TrimSpace(userID) == "" {
		userID = "anonymous"
	}
	if len(args) == 0 || strings.EqualFold(args[0], "current") {
		current, err := e.taskStore.GetLatestOpenTask(ctx, userID)
		if err != nil {
			if err == sql.ErrNoRows {
				return "Current task: <none>", nil
			}
			return "", err
		}
		forced, err := e.taskStore.PeekForcedTask(ctx, userID)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Current task: %s (%s)\n", current.ID, current.Title))
		if strings.TrimSpace(forced) != "" {
			b.WriteString(fmt.Sprintf("Next forced task: %s", forced))
		}
		return strings.TrimSpace(b.String()), nil
	}

	switch strings.ToLower(args[0]) {
	case "list":
		status := "open"
		if len(args) > 1 {
			status = strings.ToLower(strings.TrimSpace(args[1]))
		}
		items, err := e.taskStore.ListTasks(ctx, userID, status, 30)
		if err != nil {
			return "", err
		}
		if len(items) == 0 {
			return "No tasks found.", nil
		}
		var b strings.Builder
		b.WriteString("Tasks:\n")
		for _, t := range items {
			b.WriteString(fmt.Sprintf("  %s [%s] %s\n", t.ID, t.Status, t.Title))
		}
		return strings.TrimSpace(b.String()), nil
	case "use":
		if len(args) < 2 {
			return "", fmt.Errorf("usage: /task use <task_id>")
		}
		taskID := strings.TrimSpace(args[1])
		t, err := e.taskStore.GetTask(ctx, taskID)
		if err != nil {
			return "", fmt.Errorf("task not found: %s", taskID)
		}
		if t.UserID != userID {
			return "", fmt.Errorf("task does not belong to user")
		}
		if err := e.taskStore.SetForcedTask(ctx, userID, taskID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Next message will use task: %s", taskID), nil
	case "new":
		title := strings.TrimSpace(strings.Join(args[1:], " "))
		t, err := e.taskStore.CreateTask(ctx, userID, title, "")
		if err != nil {
			return "", err
		}
		if err := e.taskStore.SetForcedTask(ctx, userID, t.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Created task: %s (%s)", t.ID, t.Title), nil
	case "close":
		taskID := ""
		if len(args) > 1 {
			taskID = strings.TrimSpace(args[1])
		}
		if taskID == "" {
			current, err := e.taskStore.GetLatestOpenTask(ctx, userID)
			if err != nil {
				if err == sql.ErrNoRows {
					return "No open task to close.", nil
				}
				return "", err
			}
			taskID = current.ID
		}
		t, err := e.taskStore.GetTask(ctx, taskID)
		if err != nil {
			return "", fmt.Errorf("task not found: %s", taskID)
		}
		if t.UserID != userID {
			return "", fmt.Errorf("task does not belong to user")
		}
		if err := e.taskStore.CloseTask(ctx, taskID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Closed task: %s", taskID), nil
	case "stats":
		openTasks, err := e.taskStore.ListTasks(ctx, userID, "open", 1000)
		if err != nil {
			return "", err
		}
		closedTasks, err := e.taskStore.ListTasks(ctx, userID, "closed", 1000)
		if err != nil {
			return "", err
		}
		forcedTaskID, err := e.taskStore.PeekForcedTask(ctx, userID)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(forcedTaskID) == "" {
			forcedTaskID = "<none>"
		}
		return fmt.Sprintf("Task stats:\n  open: %d\n  closed: %d\n  next_forced: %s", len(openTasks), len(closedTasks), forcedTaskID), nil
	case "memory":
		if len(args) > 1 && strings.EqualFold(args[1], "clear") {
			taskID, err := e.resolveTaskIDForMemory(ctx, userID, args[2:])
			if err != nil {
				return "", err
			}
			t, err := e.taskStore.GetTask(ctx, taskID)
			if err != nil {
				return "", fmt.Errorf("task not found: %s", taskID)
			}
			if t.UserID != userID {
				return "", fmt.Errorf("task does not belong to user")
			}
			if err := e.taskStore.DeleteTaskMemory(ctx, taskID); err != nil {
				return "", err
			}
			return fmt.Sprintf("Cleared task memory: %s", taskID), nil
		}
		taskID, err := e.resolveTaskIDForMemory(ctx, userID, args[1:])
		if err != nil {
			return "", err
		}
		t, err := e.taskStore.GetTask(ctx, taskID)
		if err != nil {
			return "", fmt.Errorf("task not found: %s", taskID)
		}
		if t.UserID != userID {
			return "", fmt.Errorf("task does not belong to user")
		}
		memory, err := e.taskStore.GetTaskMemory(ctx, taskID)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Sprintf("Task memory is empty: %s", taskID), nil
			}
			return "", err
		}
		return fmt.Sprintf("Task memory (%s):\n%s", taskID, memory.Summary), nil
	default:
		return "", fmt.Errorf("unknown task subcommand: %s", args[0])
	}
}

func (e *Executor) resolveTaskIDForMemory(ctx context.Context, userID string, args []string) (string, error) {
	if len(args) > 0 {
		taskID := strings.TrimSpace(args[0])
		if taskID != "" {
			return taskID, nil
		}
	}
	current, err := e.taskStore.GetLatestOpenTask(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no open task available")
		}
		return "", err
	}
	return current.ID, nil
}

func formatConfigResult(result interface{}) string {
	cfg, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", result)
	}
	var b strings.Builder
	if msg, ok := cfg["message"].(string); ok && strings.TrimSpace(msg) != "" {
		b.WriteString(msg)
		b.WriteString("\n")
	}
	b.WriteString("Config:\n")
	if agent, ok := cfg["agent"].(map[string]interface{}); ok {
		if name, ok := agent["name"].(string); ok {
			b.WriteString("  agent.name: ")
			b.WriteString(name)
			b.WriteString("\n")
		}
	}
	if executorCfg, ok := cfg["executor"].(map[string]interface{}); ok {
		if name, ok := executorCfg["name"].(string); ok {
			b.WriteString("  executor.name: ")
			b.WriteString(name)
			b.WriteString("\n")
		}
	}
	if taskCfg, ok := cfg["task"].(map[string]interface{}); ok {
		for _, key := range []string{
			"routing_timeout_sec",
			"close_timeout_sec",
			"generation_timeout_sec",
			"routing_confidence_threshold",
			"close_confidence_threshold",
			"cli_user_id",
			"open_task_candidate_limit",
		} {
			if v, ok := taskCfg[key]; ok {
				b.WriteString("  task.")
				b.WriteString(key)
				b.WriteString(": ")
				b.WriteString(fmt.Sprintf("%v", v))
				b.WriteString("\n")
			}
		}
	}
	if securityCfg, ok := cfg["security"].(map[string]interface{}); ok {
		if v, ok := securityCfg["admin_user_ids"]; ok {
			b.WriteString("  security.admin_user_ids: ")
			b.WriteString(fmt.Sprintf("%v", v))
			b.WriteString("\n")
		}
		if toolsCfg, ok := securityCfg["tools"].(map[string]interface{}); ok {
			for _, key := range []string{"global_allow", "global_deny", "require_confirm"} {
				if v, ok := toolsCfg[key]; ok {
					b.WriteString("  security.tools.")
					b.WriteString(key)
					b.WriteString(": ")
					b.WriteString(fmt.Sprintf("%v", v))
					b.WriteString("\n")
				}
			}
		}
		if memoryCfg, ok := securityCfg["memory"].(map[string]interface{}); ok {
			for _, key := range []string{"trusted_channels", "restricted_paths"} {
				if v, ok := memoryCfg[key]; ok {
					b.WriteString("  security.memory.")
					b.WriteString(key)
					b.WriteString(": ")
					b.WriteString(fmt.Sprintf("%v", v))
					b.WriteString("\n")
				}
			}
		}
	}
	return strings.TrimSpace(b.String())
}
