package application

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	sharedapp "alter0/internal/shared/application"
	terminaldomain "alter0/internal/terminal/domain"
)

const (
	defaultCodexCommand             = "codex"
	defaultCodexSandbox             = "danger-full-access"
	defaultLinuxSandboxBwrapFeature = "use_linux_sandbox_bwrap"
	maxEntryPageLimit               = 200
)

var (
	ErrSessionOwnerRequired     = errors.New("terminal session owner is required")
	ErrSessionNotFound          = errors.New("terminal session not found")
	ErrSessionNotRunning        = errors.New("terminal session is not running")
	ErrSessionLimitReached      = errors.New("terminal session limit reached")
	ErrSessionBusy              = errors.New("terminal session is processing another turn")
	ErrSessionInputRequired     = errors.New("terminal input is required")
	ErrSessionRecoverIDRequired = errors.New("terminal recovery session id is required")
	ErrTurnNotFound             = errors.New("terminal turn not found")
	ErrStepNotFound             = errors.New("terminal step not found")
)

type Options struct {
	MaxSessions   int
	WorkingDir    string
	Shell         string
	ShellArgs     []string
	ShellArgsLine string
}

type CreateRequest struct {
	OwnerID string
	Title   string
}

type RecoverRequest struct {
	OwnerID           string
	SessionID         string
	TerminalSessionID string
	Title             string
	CreatedAt         time.Time
	LastOutputAt      time.Time
	UpdatedAt         time.Time
}

type EntryPage struct {
	Items      []terminaldomain.Entry `json:"items"`
	Cursor     int                    `json:"cursor"`
	NextCursor int                    `json:"next_cursor"`
	HasMore    bool                   `json:"has_more"`
}

type commandRunner func(ctx context.Context, name string, args ...string) *exec.Cmd

type Service struct {
	rootCtx     context.Context
	idGenerator sharedapp.IDGenerator
	logger      *slog.Logger
	options     Options
	runner      commandRunner

	mu       sync.RWMutex
	sessions map[string]*runtimeSession
}

type codexCommand struct {
	path       string
	globalArgs []string
	label      string
}

type codexExecEvent struct {
	Type     string         `json:"type"`
	ThreadID string         `json:"thread_id,omitempty"`
	Item     *codexExecItem `json:"item,omitempty"`
}

type codexExecItem struct {
	ID               string `json:"id,omitempty"`
	Type             string `json:"type,omitempty"`
	Text             string `json:"text,omitempty"`
	Delta            string `json:"delta,omitempty"`
	Command          string `json:"command,omitempty"`
	AggregatedOutput string `json:"aggregated_output,omitempty"`
	Status           string `json:"status,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`
}

type runtimeSession struct {
	mu sync.RWMutex

	summary      terminaldomain.Session
	entries      []terminaldomain.Entry
	nextID       int
	turns        []*runtimeTurn
	activeTurnID string
	nextTurnID   int
	nextStepID   int
	threadID     string
	turnRunning  bool
	turnCancel   context.CancelFunc
	closedByUser bool
}

type runtimeTurn struct {
	ID          string
	Prompt      string
	Status      string
	StartedAt   time.Time
	FinishedAt  time.Time
	FinalOutput string
	steps       []*runtimeStep
}

type runtimeStep struct {
	ID         string
	ItemID     string
	Type       string
	Title      string
	Status     string
	Preview    string
	StartedAt  time.Time
	FinishedAt time.Time
	Blocks     []StepDetailBlock
	Searchable bool
}

func NewService(ctx context.Context, idGenerator sharedapp.IDGenerator, logger *slog.Logger, options Options) *Service {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	service := &Service{
		rootCtx:     ctx,
		idGenerator: idGenerator,
		logger:      logger,
		options:     normalizeOptions(options),
		runner:      exec.CommandContext,
		sessions:    map[string]*runtimeSession{},
	}
	go func() {
		<-ctx.Done()
		service.shutdown()
	}()
	return service
}

func (s *Service) MaxSessions() int {
	return s.options.MaxSessions
}

func (s *Service) Create(req CreateRequest) (terminaldomain.Session, error) {
	ownerID := strings.TrimSpace(req.OwnerID)
	if ownerID == "" {
		return terminaldomain.Session{}, ErrSessionOwnerRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.countActiveLocked() >= s.options.MaxSessions {
		return terminaldomain.Session{}, ErrSessionLimitReached
	}

	command := resolveCodexCommand(s.options)
	sessionID := "terminal-" + s.newID()
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = sessionID
	}
	now := time.Now().UTC()
	session := &runtimeSession{
		summary: terminaldomain.Session{
			ID:                sessionID,
			TerminalSessionID: sessionID,
			OwnerID:           ownerID,
			Title:             title,
			Shell:             command.label,
			WorkingDir:        s.options.WorkingDir,
			Status:            terminaldomain.SessionStatusRunning,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		entries: []terminaldomain.Entry{},
	}
	s.sessions[sessionID] = session
	return session.snapshot(), nil
}

func (s *Service) Recover(req RecoverRequest) (terminaldomain.Session, error) {
	ownerID := strings.TrimSpace(req.OwnerID)
	if ownerID == "" {
		return terminaldomain.Session{}, ErrSessionOwnerRequired
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return terminaldomain.Session{}, ErrSessionRecoverIDRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.sessions[sessionID]; ok {
		snapshot := existing.snapshot()
		if snapshot.OwnerID != ownerID {
			return terminaldomain.Session{}, ErrSessionNotFound
		}
		return snapshot, nil
	}

	if s.countActiveLocked() >= s.options.MaxSessions {
		return terminaldomain.Session{}, ErrSessionLimitReached
	}

	command := resolveCodexCommand(s.options)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = sessionID
	}
	createdAt := normalizeRecoveredSessionTime(req.CreatedAt, time.Now().UTC())
	updatedAt := normalizeRecoveredSessionTime(req.UpdatedAt, createdAt)
	lastOutputAt := normalizeRecoveredOptionalTime(req.LastOutputAt)
	if !lastOutputAt.IsZero() && updatedAt.Before(lastOutputAt) {
		updatedAt = lastOutputAt
	}
	terminalSessionID := strings.TrimSpace(req.TerminalSessionID)
	if terminalSessionID == "" {
		terminalSessionID = sessionID
	}
	session := &runtimeSession{
		summary: terminaldomain.Session{
			ID:                sessionID,
			TerminalSessionID: terminalSessionID,
			OwnerID:           ownerID,
			Title:             title,
			Shell:             command.label,
			WorkingDir:        s.options.WorkingDir,
			Status:            terminaldomain.SessionStatusRunning,
			CreatedAt:         createdAt,
			LastOutputAt:      lastOutputAt,
			UpdatedAt:         updatedAt,
		},
		entries:  []terminaldomain.Entry{},
		threadID: resolveRecoveredThreadID(sessionID, terminalSessionID),
	}
	s.sessions[sessionID] = session
	return session.snapshot(), nil
}

func (s *Service) List(ownerID string) []terminaldomain.Session {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return []terminaldomain.Session{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]terminaldomain.Session, 0, len(s.sessions))
	for _, item := range s.sessions {
		snapshot := item.snapshot()
		if snapshot.OwnerID != ownerID {
			continue
		}
		items = append(items, snapshot)
	}
	sort.SliceStable(items, func(i, j int) bool {
		leftAt := terminalSessionSortAt(items[i])
		rightAt := terminalSessionSortAt(items[j])
		if leftAt.Equal(rightAt) {
			if items[i].CreatedAt.Equal(items[j].CreatedAt) {
				return items[i].ID > items[j].ID
			}
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return leftAt.After(rightAt)
	})
	return items
}

func terminalSessionSortAt(session terminaldomain.Session) time.Time {
	if !session.LastOutputAt.IsZero() {
		return session.LastOutputAt
	}
	if !session.CreatedAt.IsZero() {
		return session.CreatedAt
	}
	return session.UpdatedAt
}

func (s *Service) Get(ownerID string, sessionID string) (terminaldomain.Session, bool) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return terminaldomain.Session{}, false
	}
	return item.snapshot(), true
}

func (s *Service) ListTurns(ownerID string, sessionID string) ([]TurnSummary, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return nil, err
	}

	item.mu.RLock()
	defer item.mu.RUnlock()

	items := make([]TurnSummary, 0, len(item.turns))
	for _, turn := range item.turns {
		if turn == nil {
			continue
		}
		items = append(items, turn.summary())
	}
	return items, nil
}

func (s *Service) GetStepDetail(ownerID string, sessionID string, turnID string, stepID string) (StepDetail, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return StepDetail{}, err
	}

	item.mu.RLock()
	defer item.mu.RUnlock()

	turn := item.turnByIDLocked(turnID)
	if turn == nil {
		return StepDetail{}, ErrTurnNotFound
	}
	step := turn.stepByID(stepID)
	if step == nil {
		return StepDetail{}, ErrStepNotFound
	}
	return step.detail(turn.ID), nil
}

func (s *Service) ListEntries(ownerID string, sessionID string, cursor int, limit int) (EntryPage, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return EntryPage{}, err
	}

	if cursor < 0 {
		cursor = 0
	}
	if limit <= 0 || limit > maxEntryPageLimit {
		limit = maxEntryPageLimit
	}

	item.mu.RLock()
	defer item.mu.RUnlock()

	if cursor > item.nextID {
		cursor = item.nextID
	}

	items := make([]terminaldomain.Entry, 0, limit)
	nextCursor := cursor
	for _, entry := range item.entries {
		if entry.Cursor < cursor {
			continue
		}
		items = append(items, entry)
		nextCursor = entry.Cursor + 1
		if len(items) >= limit {
			break
		}
	}
	if len(items) == 0 {
		nextCursor = cursor
	}
	return EntryPage{
		Items:      items,
		Cursor:     cursor,
		NextCursor: nextCursor,
		HasMore:    nextCursor < item.nextID,
	}, nil
}

func (s *Service) Input(ownerID string, sessionID string, input string) (terminaldomain.Session, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return terminaldomain.Session{}, err
	}

	prompt := strings.TrimSpace(input)
	if prompt == "" {
		return terminaldomain.Session{}, ErrSessionInputRequired
	}

	turnCtx, turnCancel := context.WithCancel(s.rootCtx)

	item.mu.Lock()
	if item.turnRunning {
		item.mu.Unlock()
		turnCancel()
		return terminaldomain.Session{}, ErrSessionBusy
	}
	if item.closedByUser || item.summary.Status != terminaldomain.SessionStatusRunning {
		item.mu.Unlock()
		turnCancel()
		return terminaldomain.Session{}, ErrSessionNotRunning
	}
	now := time.Now().UTC()
	if item.summary.Title == "" || item.summary.Title == item.summary.ID {
		item.summary.Title = deriveSessionTitle(prompt, item.summary.ID)
	}
	item.summary.Status = terminaldomain.SessionStatusStarting
	item.summary.UpdatedAt = now
	item.summary.ErrorMessage = ""
	item.summary.ExitCode = nil
	item.turnRunning = true
	item.turnCancel = turnCancel
	turn := item.beginTurnLocked(prompt, now)
	item.appendEntryLocked("input", prompt)
	snapshot := item.summary
	item.mu.Unlock()

	go s.runTurn(item, turnCtx, turn.ID, prompt)

	return snapshot, nil
}

func (s *Service) Close(ownerID string, sessionID string) (terminaldomain.Session, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return terminaldomain.Session{}, err
	}

	item.mu.Lock()
	defer item.mu.Unlock()

	now := time.Now().UTC()
	item.closedByUser = true
	if item.turnCancel != nil {
		item.turnCancel()
		item.turnCancel = nil
	}
	item.turnRunning = false
	item.summary.Status = terminaldomain.SessionStatusExited
	item.summary.ErrorMessage = ""
	item.summary.ExitCode = nil
	item.summary.UpdatedAt = now
	item.summary.FinishedAt = now
	item.appendEntryLocked("system", "codex session closed")
	return item.summary, nil
}

func (s *Service) shutdown() {
	s.mu.RLock()
	sessions := make([]*runtimeSession, 0, len(s.sessions))
	for _, item := range s.sessions {
		sessions = append(sessions, item)
	}
	s.mu.RUnlock()

	for _, item := range sessions {
		item.mu.Lock()
		if item.turnCancel != nil {
			item.turnCancel()
			item.turnCancel = nil
		}
		if !item.closedByUser && item.summary.Status == terminaldomain.SessionStatusStarting {
			item.summary.Status = terminaldomain.SessionStatusInterrupted
			item.summary.ErrorMessage = "terminal host unavailable"
			item.summary.UpdatedAt = time.Now().UTC()
			item.appendEntryLocked("system", "terminal interrupted: terminal host unavailable")
		}
		item.turnRunning = false
		item.mu.Unlock()
	}
}

func (s *Service) runTurn(item *runtimeSession, ctx context.Context, turnID string, prompt string) {
	command := resolveCodexCommand(s.options)
	threadID := item.thread()
	args := buildCodexTurnArgs(command, threadID, prompt)
	runner := s.runner
	if runner == nil {
		runner = exec.CommandContext
	}
	cmd := runner(ctx, command.path, args...)
	if s.options.WorkingDir != "" {
		cmd.Dir = s.options.WorkingDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.finishTurn(item, turnID, fmt.Errorf("create codex stdout pipe: %w", err), "")
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.finishTurn(item, turnID, fmt.Errorf("create codex stderr pipe: %w", err), "")
		return
	}

	if err := cmd.Start(); err != nil {
		s.finishTurn(item, turnID, fmt.Errorf("start codex command: %w", err), "")
		return
	}

	stderrCh := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(stderr)
		stderrCh <- strings.TrimSpace(string(data))
	}()

	parseErr := s.consumeCodexOutput(item, turnID, stdout)
	waitErr := cmd.Wait()
	stderrText := <-stderrCh

	if parseErr != nil {
		s.finishTurn(item, turnID, parseErr, stderrText)
		return
	}
	if waitErr != nil {
		s.finishTurn(item, turnID, fmt.Errorf("codex command failed: %w", waitErr), stderrText)
		return
	}
	s.finishTurn(item, turnID, nil, "")
}

func (s *Service) consumeCodexOutput(item *runtimeSession, turnID string, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		event := codexExecEvent{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		s.applyCodexEvent(item, turnID, event)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read codex output: %w", err)
	}
	return nil
}

func (s *Service) applyCodexEvent(item *runtimeSession, turnID string, event codexExecEvent) {
	switch strings.TrimSpace(event.Type) {
	case "thread.started":
		if threadID := strings.TrimSpace(event.ThreadID); threadID != "" {
			item.mu.Lock()
			item.threadID = threadID
			item.summary.TerminalSessionID = threadID
			item.summary.UpdatedAt = time.Now().UTC()
			item.mu.Unlock()
		}
	case "item.delta":
		return
	case "item.started":
		if event.Item == nil {
			return
		}
		item.mu.Lock()
		defer item.mu.Unlock()

		turn := item.turnByIDLocked(turnID)
		if turn == nil {
			return
		}
		switch normalizeCodexItemType(event.Item.Type) {
		case "command_execution":
			command := strings.TrimSpace(event.Item.Command)
			if command == "" {
				return
			}
			step := item.ensureCommandStepLocked(turn, event.Item.ID, command, time.Now().UTC())
			step.Status = "running"
			item.appendEntryLocked("system", "running command: "+command)
		}
	case "item.completed":
		if event.Item == nil {
			return
		}
		now := time.Now().UTC()
		item.mu.Lock()
		defer item.mu.Unlock()

		turn := item.turnByIDLocked(turnID)
		if turn == nil {
			return
		}
		switch normalizeCodexItemType(event.Item.Type) {
		case "agent_message":
			text := normalizeChunk(event.Item.Text)
			if strings.TrimSpace(text) == "" {
				return
			}
			step := item.newStepLocked(turn, "message", deriveStepTitle("message", text), now)
			step.Status = "completed"
			step.FinishedAt = now
			step.Preview = summarizeStepPreview(text)
			step.Blocks = []StepDetailBlock{{
				Type:    "text",
				Title:   "Message",
				Content: text,
				Status:  step.Status,
			}}
			step.Searchable = true
			item.appendEntryLocked("stdout", text)
		case "reasoning", "plan":
			text := normalizeChunk(event.Item.Text)
			if strings.TrimSpace(text) == "" {
				return
			}
			step := item.newStepLocked(turn, normalizeCodexItemType(event.Item.Type), deriveStepTitle(normalizeCodexItemType(event.Item.Type), text), now)
			step.Status = "completed"
			step.FinishedAt = now
			step.Preview = summarizeStepPreview(text)
			step.Blocks = []StepDetailBlock{{
				Type:    "text",
				Title:   step.Title,
				Content: text,
				Status:  step.Status,
			}}
			step.Searchable = true
		case "command_execution":
			command := strings.TrimSpace(event.Item.Command)
			step := item.ensureCommandStepLocked(turn, event.Item.ID, command, now)
			step.Status = normalizeStepStatus(strings.TrimSpace(event.Item.Status), event.Item.ExitCode)
			step.FinishedAt = now
			output := normalizeChunk(event.Item.AggregatedOutput)
			step.Preview = summarizeStepPreview(strings.TrimSpace(command))
			step.Blocks = []StepDetailBlock{{
				Type:     "terminal",
				Title:    "Shell",
				Content:  strings.TrimSpace(strings.Join([]string{command, output}, "\n\n")),
				Language: "shell",
				Status:   step.Status,
				ExitCode: event.Item.ExitCode,
			}}
			if command != "" && step.Status == "failed" {
				item.appendEntryLocked("system", "command failed: "+command)
			}
			if strings.TrimSpace(output) != "" {
				stream := "stdout"
				if event.Item.ExitCode != nil && *event.Item.ExitCode != 0 {
					stream = "stderr"
				}
				item.appendEntryLocked(stream, output)
			}
		}
	}
}

func (s *Service) finishTurn(item *runtimeSession, turnID string, turnErr error, stderrText string) {
	now := time.Now().UTC()
	item.mu.Lock()
	defer item.mu.Unlock()

	turn := item.turnByIDLocked(turnID)

	if item.turnCancel != nil {
		item.turnCancel()
		item.turnCancel = nil
	}
	item.turnRunning = false
	if item.activeTurnID == turnID {
		item.activeTurnID = ""
	}

	if item.closedByUser {
		item.summary.UpdatedAt = now
		item.summary.FinishedAt = now
		item.summary.Status = terminaldomain.SessionStatusExited
		if turn != nil && turn.FinishedAt.IsZero() {
			turn.Status = "completed"
			turn.FinishedAt = now
			turn.promoteFinalOutput()
		}
		return
	}

	item.summary.Status = terminaldomain.SessionStatusRunning
	item.summary.UpdatedAt = now
	item.summary.FinishedAt = time.Time{}
	item.summary.ExitCode = nil

	if turnErr == nil {
		item.summary.ErrorMessage = ""
		if turn != nil {
			turn.Status = "completed"
			turn.FinishedAt = now
			turn.promoteFinalOutput()
		}
		return
	}

	if errors.Is(turnErr, context.Canceled) || errors.Is(s.rootCtx.Err(), context.Canceled) {
		item.summary.Status = terminaldomain.SessionStatusInterrupted
		item.summary.ErrorMessage = "terminal host unavailable"
		item.summary.FinishedAt = now
		item.appendEntryLocked("system", "terminal interrupted: terminal host unavailable")
		if turn != nil {
			turn.Status = "interrupted"
			turn.FinishedAt = now
			item.newSystemStepLocked(turn, "Interrupted", "terminal host unavailable", now, "failed")
			turn.promoteFinalOutput()
		}
		return
	}

	message := compactCodexError(stderrText, turnErr)
	item.summary.ErrorMessage = message
	item.appendEntryLocked("system", "codex request failed: "+message)
	if turn != nil {
		turn.Status = "failed"
		turn.FinishedAt = now
		item.newSystemStepLocked(turn, "Request failed", message, now, "failed")
		turn.promoteFinalOutput()
	}
}

func (s *runtimeSession) beginTurnLocked(prompt string, now time.Time) *runtimeTurn {
	s.nextTurnID++
	turn := &runtimeTurn{
		ID:        fmt.Sprintf("turn-%d", s.nextTurnID),
		Prompt:    prompt,
		Status:    "running",
		StartedAt: now,
		steps:     []*runtimeStep{},
	}
	s.turns = append(s.turns, turn)
	s.activeTurnID = turn.ID
	return turn
}

func (s *runtimeSession) turnByIDLocked(turnID string) *runtimeTurn {
	for _, turn := range s.turns {
		if turn != nil && turn.ID == strings.TrimSpace(turnID) {
			return turn
		}
	}
	return nil
}

func (s *runtimeSession) newStepLocked(turn *runtimeTurn, stepType string, title string, now time.Time) *runtimeStep {
	if turn == nil {
		return nil
	}
	s.nextStepID++
	step := &runtimeStep{
		ID:        fmt.Sprintf("step-%d", s.nextStepID),
		Type:      stepType,
		Title:     strings.TrimSpace(title),
		Status:    "running",
		StartedAt: now,
	}
	turn.steps = append(turn.steps, step)
	return step
}

func (s *runtimeSession) ensureCommandStepLocked(turn *runtimeTurn, itemID string, command string, now time.Time) *runtimeStep {
	if turn == nil {
		return nil
	}
	for _, step := range turn.steps {
		if step != nil && step.Type == "command" && strings.TrimSpace(step.ItemID) == strings.TrimSpace(itemID) && strings.TrimSpace(itemID) != "" {
			return step
		}
	}
	step := s.newStepLocked(turn, "command", deriveStepTitle("command", command), now)
	if step != nil {
		step.ItemID = strings.TrimSpace(itemID)
		step.Preview = summarizeStepPreview(command)
		step.Blocks = []StepDetailBlock{{
			Type:     "terminal",
			Title:    "Shell",
			Content:  strings.TrimSpace(command),
			Language: "shell",
			Status:   "running",
		}}
	}
	return step
}

func (s *runtimeSession) newSystemStepLocked(turn *runtimeTurn, title string, message string, now time.Time, status string) *runtimeStep {
	step := s.newStepLocked(turn, "log", title, now)
	if step == nil {
		return nil
	}
	step.Status = normalizeFallbackStatus(status)
	step.FinishedAt = now
	step.Preview = summarizeStepPreview(message)
	step.Blocks = []StepDetailBlock{{
		Type:    "log",
		Title:   title,
		Content: message,
		Status:  step.Status,
	}}
	step.Searchable = true
	return step
}

func (t *runtimeTurn) stepByID(stepID string) *runtimeStep {
	for _, step := range t.steps {
		if step != nil && step.ID == strings.TrimSpace(stepID) {
			return step
		}
	}
	return nil
}

func (t *runtimeTurn) summary() TurnSummary {
	steps := make([]StepSummary, 0, len(t.steps))
	for _, step := range t.steps {
		if step == nil {
			continue
		}
		steps = append(steps, step.summary())
	}
	return TurnSummary{
		ID:          t.ID,
		Prompt:      t.Prompt,
		Status:      normalizeFallbackStatus(t.Status),
		StartedAt:   t.StartedAt,
		FinishedAt:  t.FinishedAt,
		DurationMS:  durationMilliseconds(t.StartedAt, t.FinishedAt),
		FinalOutput: t.FinalOutput,
		Steps:       steps,
	}
}

func (t *runtimeTurn) promoteFinalOutput() {
	if len(t.steps) == 0 {
		return
	}
	lastMessageIndex := -1
	for index := len(t.steps) - 1; index >= 0; index-- {
		if t.steps[index] != nil && t.steps[index].Type == "message" {
			lastMessageIndex = index
			break
		}
	}
	if lastMessageIndex < 0 {
		return
	}
	messageStep := t.steps[lastMessageIndex]
	if messageStep == nil || len(messageStep.Blocks) == 0 {
		return
	}
	t.FinalOutput = strings.TrimSpace(messageStep.Blocks[0].Content)
	t.steps = append(append([]*runtimeStep{}, t.steps[:lastMessageIndex]...), t.steps[lastMessageIndex+1:]...)
}

func (s *runtimeStep) summary() StepSummary {
	return StepSummary{
		ID:         s.ID,
		Type:       normalizeStepType(s.Type),
		Title:      s.Title,
		Status:     normalizeFallbackStatus(s.Status),
		StartedAt:  s.StartedAt,
		FinishedAt: s.FinishedAt,
		DurationMS: durationMilliseconds(s.StartedAt, s.FinishedAt),
		Preview:    s.Preview,
		HasDetail:  len(s.Blocks) > 0,
	}
}

func (s *runtimeStep) detail(turnID string) StepDetail {
	return StepDetail{
		TurnID:     turnID,
		Step:       s.summary(),
		Blocks:     append([]StepDetailBlock{}, s.Blocks...),
		Searchable: s.Searchable,
	}
}

func durationMilliseconds(start time.Time, finish time.Time) int64 {
	if start.IsZero() {
		return 0
	}
	if finish.IsZero() {
		finish = time.Now().UTC()
	}
	if finish.Before(start) {
		return 0
	}
	return finish.Sub(start).Milliseconds()
}

func deriveStepTitle(stepType string, content string) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	switch normalizeStepType(stepType) {
	case "command":
		if text == "" {
			return "Run command"
		}
		return shortenText(text, 96)
	case "reasoning":
		if text == "" {
			return "Reasoning"
		}
		return shortenText(text, 96)
	case "plan":
		if text == "" {
			return "Plan"
		}
		return shortenText(text, 96)
	case "log":
		if text == "" {
			return "Log"
		}
		return shortenText(text, 96)
	default:
		if text == "" {
			return "Message"
		}
		return shortenText(text, 96)
	}
}

func summarizeStepPreview(content string) string {
	return shortenText(strings.Join(strings.Fields(strings.TrimSpace(content)), " "), 120)
}

func shortenText(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func normalizeFallbackStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "running", "completed", "failed", "interrupted":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "completed"
	}
}

func normalizeStepStatus(value string, exitCode *int) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "completed", "succeeded", "success":
		if exitCode != nil && *exitCode != 0 {
			return "failed"
		}
		return "completed"
	case "failed", "error":
		return "failed"
	case "running", "in_progress", "inprogress":
		return "running"
	default:
		if exitCode != nil && *exitCode != 0 {
			return "failed"
		}
		return "completed"
	}
}

func normalizeStepType(stepType string) string {
	switch strings.ToLower(strings.TrimSpace(stepType)) {
	case "command", "command_execution":
		return "command"
	case "reasoning":
		return "reasoning"
	case "plan":
		return "plan"
	case "log", "system":
		return "log"
	case "diff", "file_change":
		return "diff"
	default:
		return "message"
	}
}

func normalizeOptions(options Options) Options {
	if options.MaxSessions <= 0 {
		options.MaxSessions = 5
	}
	if options.MaxSessions > 5 {
		options.MaxSessions = 5
	}
	options.WorkingDir = strings.TrimSpace(options.WorkingDir)
	if options.WorkingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			options.WorkingDir = cwd
		}
	}
	options.Shell = strings.TrimSpace(options.Shell)
	options.ShellArgsLine = strings.TrimSpace(options.ShellArgsLine)
	args := make([]string, 0, len(options.ShellArgs))
	for _, item := range options.ShellArgs {
		if value := strings.TrimSpace(item); value != "" {
			args = append(args, value)
		}
	}
	if options.ShellArgsLine != "" {
		args = append(args, splitCommandLineArgs(options.ShellArgsLine)...)
	}
	options.ShellArgs = args
	return options
}

func splitCommandLineArgs(line string) []string {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	args := make([]string, 0, 4)
	var current strings.Builder
	var quote rune
	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}
	for _, char := range line {
		switch {
		case quote == 0 && (char == '\'' || char == '"'):
			quote = char
		case quote != 0 && char == quote:
			quote = 0
		case quote == 0 && (char == ' ' || char == '\t' || char == '\n' || char == '\r'):
			flush()
		default:
			current.WriteRune(char)
		}
	}
	flush()
	return args
}

func resolveCodexCommand(options Options) codexCommand {
	path := strings.TrimSpace(options.Shell)
	if path == "" {
		path = defaultCodexCommand
	}
	args := append([]string{}, options.ShellArgs...)
	return codexCommand{
		path:       path,
		globalArgs: args,
		label:      buildCodexLabel(path, args),
	}
}

func buildCodexTurnArgs(command codexCommand, threadID string, prompt string) []string {
	args := append([]string{}, command.globalArgs...)
	args = append(args, "exec", "--enable", defaultLinuxSandboxBwrapFeature)
	if strings.TrimSpace(threadID) != "" {
		args = append(args, "resume", "--json", "--skip-git-repo-check", threadID, prompt)
		return args
	}
	args = append(args,
		"--json",
		"--color", "never",
		"--skip-git-repo-check",
		"--sandbox", defaultCodexSandbox,
		prompt,
	)
	return args
}

func buildCodexLabel(commandPath string, args []string) string {
	parts := []string{commandPath}
	parts = append(parts, args...)
	parts = append(parts, "exec")
	return strings.Join(parts, " ")
}

func deriveSessionTitle(prompt string, fallback string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(prompt)), " ")
	if normalized == "" {
		return fallback
	}
	if len(normalized) > 64 {
		return normalized[:61] + "..."
	}
	return normalized
}

func normalizeCodexItemType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "agentmessage", "agent_message", "agent-message":
		return "agent_message"
	case "commandexecution", "command_execution", "command-execution":
		return "command_execution"
	default:
		return normalized
	}
}

func compactCodexError(stderrText string, turnErr error) string {
	lines := make([]string, 0, 4)
	for _, raw := range strings.Split(normalizeChunk(stderrText), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.Contains(line, " WARN ") {
			continue
		}
		lines = append(lines, line)
		if len(lines) >= 4 {
			break
		}
	}
	if len(lines) > 0 {
		return strings.Join(lines, " | ")
	}
	if turnErr != nil {
		return strings.TrimSpace(turnErr.Error())
	}
	return "unknown error"
}

func normalizeChunk(value string) string {
	text := strings.ReplaceAll(value, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func isTerminalOutputStream(stream string) bool {
	switch strings.ToLower(strings.TrimSpace(stream)) {
	case "stdout", "stderr":
		return true
	default:
		return false
	}
}

func (s *Service) getOwnedSession(ownerID string, sessionID string) (*runtimeSession, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, ErrSessionOwnerRequired
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, ErrSessionNotFound
	}

	s.mu.RLock()
	item, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrSessionNotFound
	}

	item.mu.RLock()
	matched := item.summary.OwnerID == ownerID
	item.mu.RUnlock()
	if !matched {
		return nil, ErrSessionNotFound
	}
	return item, nil
}

func (s *Service) countActiveLocked() int {
	total := 0
	for _, item := range s.sessions {
		snapshot := item.snapshot()
		if snapshot.Status == terminaldomain.SessionStatusStarting || snapshot.Status == terminaldomain.SessionStatusRunning {
			total++
		}
	}
	return total
}

func (s *Service) newID() string {
	if s.idGenerator != nil {
		if value := strings.TrimSpace(s.idGenerator.NewID()); value != "" {
			return value
		}
	}
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}

func normalizeRecoveredSessionTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value.UTC()
}

func normalizeRecoveredOptionalTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC()
}

func resolveRecoveredThreadID(sessionID string, terminalSessionID string) string {
	threadID := strings.TrimSpace(terminalSessionID)
	if threadID == "" || threadID == strings.TrimSpace(sessionID) {
		return ""
	}
	return threadID
}

func (s *runtimeSession) thread() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.threadID
}

func (s *runtimeSession) appendEntry(stream string, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appendEntryLocked(stream, text)
}

func (s *runtimeSession) appendEntryLocked(stream string, text string) {
	content := normalizeChunk(text)
	if content == "" {
		return
	}
	now := time.Now().UTC()
	s.entries = append(s.entries, terminaldomain.Entry{
		Cursor:    s.nextID,
		Stream:    strings.TrimSpace(stream),
		Text:      content,
		CreatedAt: now,
	})
	s.nextID++
	if isTerminalOutputStream(stream) {
		s.summary.LastOutputAt = now
	}
	s.summary.UpdatedAt = now
}

func (s *runtimeSession) snapshot() terminaldomain.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.summary
}
