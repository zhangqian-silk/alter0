package slash

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"alter0/app/pkg/types"
)

type Handler func(context.Context, types.Message, []string) (string, error)

type Authorizer func(userID string, parts []string) error

type HelpProvider func() string

type Executor struct {
	mu             sync.RWMutex
	handlers       map[string]Handler
	helpProvider   HelpProvider
	authorizer     Authorizer
	statusProvider func(context.Context) map[string]interface{}
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

func NewExecutor() *Executor {
	return &Executor{handlers: map[string]Handler{}}
}

func (e *Executor) Register(name string, handler Handler) {
	if e == nil || handler == nil {
		return
	}
	commandName := strings.ToLower(strings.TrimSpace(name))
	if commandName == "" {
		return
	}
	e.mu.Lock()
	e.handlers[commandName] = handler
	e.mu.Unlock()
}

func (e *Executor) SetHelpProvider(provider HelpProvider) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.helpProvider = provider
	e.mu.Unlock()
}

func (e *Executor) SetAuthorizer(authorizer Authorizer) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.authorizer = authorizer
	e.mu.Unlock()
}

func (e *Executor) SetStatusProvider(provider func(context.Context) map[string]interface{}) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.statusProvider = provider
	e.mu.Unlock()
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
	commandName := strings.ToLower(strings.TrimSpace(parts[0]))
	if commandName == "" {
		return "", false, nil
	}

	auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "attempt", "")
	if err := e.authorize(msg.UserID, parts); err != nil {
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
		return "", true, err
	}

	switch commandName {
	case "help":
		out := e.helpText()
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "allow", "")
		return out, true, nil
	case "status":
		out, err := e.runtimeStatusOutput(ctx)
		if err != nil {
			auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
			return "", true, err
		}
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "allow", "")
		return out, true, nil
	default:
		handler := e.handlerFor(commandName)
		if handler == nil {
			err := fmt.Errorf("unknown command: %s", commandName)
			auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
			return "", true, err
		}
		out, err := handler(ctx, msg, parts[1:])
		if err != nil {
			auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "deny", err.Error())
			return out, true, err
		}
		auditCommand(msg.UserID, msg.ChannelID, msg.RequestID, cmd, "allow", "")
		return out, true, nil
	}
}

func (e *Executor) handlerFor(name string) Handler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.handlers[strings.ToLower(strings.TrimSpace(name))]
}

func (e *Executor) authorize(userID string, parts []string) error {
	e.mu.RLock()
	authorizer := e.authorizer
	e.mu.RUnlock()
	if authorizer == nil {
		return nil
	}
	return authorizer(userID, parts)
}

func (e *Executor) helpText() string {
	e.mu.RLock()
	provider := e.helpProvider
	commands := make([]string, 0, len(e.handlers))
	for name := range e.handlers {
		commands = append(commands, name)
	}
	e.mu.RUnlock()

	if provider != nil {
		return strings.TrimSpace(provider())
	}
	sort.Strings(commands)
	var b strings.Builder
	b.WriteString("Commands:\n")
	b.WriteString("  /help\n")
	b.WriteString("  /status\n")
	for _, name := range commands {
		b.WriteString("  /")
		b.WriteString(name)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func (e *Executor) runtimeStatusOutput(ctx context.Context) (string, error) {
	e.mu.RLock()
	provider := e.statusProvider
	e.mu.RUnlock()
	if provider == nil {
		return "", fmt.Errorf("status provider is not configured")
	}
	payload := provider(ctx)
	if payload == nil {
		payload = map[string]interface{}{}
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
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
