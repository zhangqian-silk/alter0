package command

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"

	"alter0/app/core/orchestrator/command/slash"
	orctask "alter0/app/core/orchestrator/task"
	"alter0/app/pkg/types"
)

type SkillManager interface {
	ListSkills() []types.SkillManifest
	Execute(context.Context, string, map[string]interface{}) (interface{}, error)
}

type Executor struct {
	skillMgr  SkillManager
	taskStore *orctask.Store
	slash     *slash.Executor

	mu         sync.RWMutex
	adminUsers map[string]struct{}
}

func NewExecutor(skillMgr SkillManager, taskStore *orctask.Store, adminUserIDs []string) *Executor {
	e := &Executor{
		skillMgr:  skillMgr,
		taskStore: taskStore,
		slash:     slash.NewExecutor(),
	}
	e.SetAdminUsers(adminUserIDs)
	e.registerHandlers()
	e.slash.SetAuthorizer(e.authorizeCommand)
	e.slash.SetHelpProvider(e.helpText)
	return e
}

func (e *Executor) registerHandlers() {
	e.slash.Register("task", func(ctx context.Context, msg types.Message, args []string) (string, error) {
		return e.executeTaskCommand(ctx, msg.UserID, args)
	})
	e.slash.Register("config", func(ctx context.Context, msg types.Message, args []string) (string, error) {
		return e.executeConfigCommand(ctx, args)
	})
	e.slash.Register("executor", func(ctx context.Context, msg types.Message, args []string) (string, error) {
		return e.executeExecutorCommand(ctx, args)
	})
}

func (e *Executor) ExecuteSlash(ctx context.Context, msg types.Message) (string, bool, error) {
	if e == nil || e.slash == nil {
		return "", false, nil
	}
	return e.slash.ExecuteSlash(ctx, msg)
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
	if e == nil || e.slash == nil {
		return
	}
	e.slash.SetStatusProvider(provider)
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
	if e.skillMgr == nil {
		return []string{"config", "executor"}, e.taskStore != nil
	}
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

func (e *Executor) executeConfigCommand(ctx context.Context, args []string) (string, error) {
	if e.skillMgr == nil {
		return "", fmt.Errorf("skill manager is not configured")
	}
	params := map[string]interface{}{}
	if len(args) > 0 {
		params["command"] = strings.Join(args, " ")
	}
	e.fillConfigArgs(args, params)
	result, err := e.skillMgr.Execute(ctx, "config", params)
	if err != nil {
		return "", err
	}
	return formatConfigResult(result), nil
}

func (e *Executor) executeExecutorCommand(ctx context.Context, args []string) (string, error) {
	if e.skillMgr == nil {
		return "", fmt.Errorf("skill manager is not configured")
	}
	params := map[string]interface{}{}
	if len(args) > 0 {
		params["command"] = strings.Join(args, " ")
	}
	result, err := e.skillMgr.Execute(ctx, "executor", params)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", result), nil
}

func (e *Executor) fillConfigArgs(args []string, params map[string]interface{}) {
	if len(args) == 0 {
		params["action"] = "get"
		return
	}
	action := strings.ToLower(args[0])
	if action == "get" || action == "set" {
		params["action"] = action
		if len(args) >= 2 {
			params["key"] = args[1]
			if len(args) > 2 {
				params["value"] = strings.Join(args[2:], " ")
			} else {
				params["value"] = ""
			}
		}
		return
	}
	params["action"] = "get"
	params["key"] = args[0]
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
