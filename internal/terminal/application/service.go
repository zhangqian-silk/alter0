package application

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"alter0/internal/codex/infrastructure/runtimeconfig"
	execdomain "alter0/internal/execution/domain"
	sharedapp "alter0/internal/shared/application"
	terminaldomain "alter0/internal/terminal/domain"
)

const (
	defaultCodexCommand             = "codex"
	defaultCodexSandbox             = "danger-full-access"
	defaultLinuxSandboxBwrapFeature = "use_linux_sandbox_bwrap"
	defaultWorkspaceRootDirName     = ".alter0"
	workspaceDirectoryName          = "workspaces"
	workspaceTerminalDirName        = "terminal"
	workspaceSessionsDirName        = "sessions"
	terminalTurnAttachmentDirName   = "input-attachments"
	terminalCodexHomeDirName        = "codex-home"
	maxEntryPageLimit               = 200
	terminalHostUnavailableMessage  = "terminal host unavailable"
	terminalCompactionResetMessage  = "codex context compaction failed; next input will start a fresh runtime thread in the same workspace"
)

var (
	ErrSessionOwnerRequired     = errors.New("terminal session owner is required")
	ErrSessionNotFound          = errors.New("terminal session not found")
	ErrSessionNotRunning        = errors.New("terminal session is not running")
	ErrSessionBusy              = errors.New("terminal session is processing another turn")
	ErrSessionInputRequired     = errors.New("terminal input is required")
	ErrSessionRecoverIDRequired = errors.New("terminal recovery session id is required")
	ErrTurnNotFound             = errors.New("terminal turn not found")
	ErrStepNotFound             = errors.New("terminal step not found")
)

const sharedTerminalOwnerID = "shared"

type Options struct {
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

type InputRequest struct {
	OwnerID      string
	SessionID    string
	Input        string
	Attachments  []execdomain.UserAttachment
	SkillContext *execdomain.SkillContext
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
	Message  string         `json:"message,omitempty"`
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

type preparedTurnAttachment struct {
	Name        string
	ContentType string
	Path        string
	PromptPath  string
	IsImage     bool
}

type runtimeSession struct {
	mu sync.RWMutex

	summary      terminaldomain.Session
	titleManual  bool
	titleAuto    bool
	titleScore   int
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
	deleted      bool
}

type runtimeTurn struct {
	ID          string
	Prompt      string
	Attachments []TurnAttachment
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
	service.loadPersistedSessions()
	return service
}

func (s *Service) Create(req CreateRequest) (terminaldomain.Session, error) {
	ownerID := normalizeTerminalOwnerID(req.OwnerID)

	s.mu.Lock()
	defer s.mu.Unlock()

	command := resolveCodexCommand(s.options)
	sessionID := "terminal-" + s.newID()
	workspaceDir, err := resolveSessionWorkspaceDir(s.options.WorkingDir, sessionID)
	if err != nil {
		return terminaldomain.Session{}, err
	}
	title := strings.TrimSpace(req.Title)
	titleManual := false
	titleAuto := false
	if title == "" {
		title = sessionID
		titleAuto = true
	} else {
		titleManual = true
	}
	now := time.Now().UTC()
	session := &runtimeSession{
		summary: terminaldomain.Session{
			ID:                sessionID,
			TerminalSessionID: sessionID,
			OwnerID:           ownerID,
			Title:             title,
			Shell:             command.label,
			WorkingDir:        workspaceDir,
			Status:            terminaldomain.SessionStatusReady,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		titleManual: titleManual,
		titleAuto:   titleAuto,
		entries:     []terminaldomain.Entry{},
	}
	s.sessions[sessionID] = session
	s.persistSession(session)
	return session.snapshot(), nil
}

func (s *Service) Recover(req RecoverRequest) (terminaldomain.Session, error) {
	ownerID := normalizeTerminalOwnerID(req.OwnerID)

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return terminaldomain.Session{}, ErrSessionRecoverIDRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.sessions[sessionID]; ok {
		snapshot := existing.snapshot()
		if snapshot.OwnerID != ownerID {
			existing.mu.Lock()
			existing.summary.OwnerID = ownerID
			if workspaceDir, workspaceErr := resolveSessionWorkspacePath(s.options.WorkingDir, sessionID); workspaceErr == nil {
				existing.summary.WorkingDir = workspaceDir
			}
			existing.summary.UpdatedAt = time.Now().UTC()
			snapshot = existing.summary
			existing.mu.Unlock()
			s.persistSession(existing)
		}
		return snapshot, nil
	}

	command := resolveCodexCommand(s.options)
	workspaceDir, err := resolveSessionWorkspaceDir(s.options.WorkingDir, sessionID)
	if err != nil {
		return terminaldomain.Session{}, err
	}
	title := strings.TrimSpace(req.Title)
	titleManual := false
	titleAuto := false
	titleScore := 0
	if title == "" {
		title = sessionID
		titleAuto = true
	} else {
		titleAuto, titleScore = inferAutoSessionTitleState(title, sessionID)
		titleManual = inferManualSessionTitleState(title, sessionID, titleAuto, titleScore)
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
			WorkingDir:        workspaceDir,
			Status:            terminaldomain.SessionStatusReady,
			CreatedAt:         createdAt,
			LastOutputAt:      lastOutputAt,
			UpdatedAt:         updatedAt,
		},
		titleManual: titleManual,
		titleAuto:   titleAuto,
		titleScore:  titleScore,
		entries:     []terminaldomain.Entry{},
		threadID:    resolveRecoveredThreadID(sessionID, terminalSessionID),
	}
	s.sessions[sessionID] = session
	s.persistSession(session)
	return session.snapshot(), nil
}

func (s *Service) List(ownerID string) []terminaldomain.Session {
	ownerID = normalizeTerminalOwnerID(ownerID)

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
	return s.InputWithAttachments(InputRequest{
		OwnerID:   ownerID,
		SessionID: sessionID,
		Input:     input,
	})
}

func (s *Service) InputWithAttachments(req InputRequest) (terminaldomain.Session, error) {
	item, err := s.getOwnedSession(req.OwnerID, req.SessionID)
	if err != nil {
		if !errors.Is(err, ErrSessionNotFound) {
			return terminaldomain.Session{}, err
		}
		item, err = s.restorePersistedOwnedSession(req.OwnerID, req.SessionID)
		if err != nil {
			return terminaldomain.Session{}, err
		}
	}

	attachments := normalizeTurnAttachments(req.Attachments)
	prompt := strings.TrimSpace(req.Input)
	if prompt == "" && len(attachments) > 0 {
		prompt = defaultAttachmentPrompt(attachments)
	}
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
	if terminaldomain.NormalizeSessionStatus(item.summary.Status) == terminaldomain.SessionStatusBusy {
		item.mu.Unlock()
		turnCancel()
		return terminaldomain.Session{}, ErrSessionNotRunning
	}
	now := time.Now().UTC()
	if nextTitle, nextAuto, nextScore, changed := nextAutoSessionTitle(
		item.summary.Title,
		item.titleManual,
		item.titleScore,
		prompt,
		item.summary.ID,
		64,
	); changed {
		item.summary.Title = nextTitle
		item.titleAuto = nextAuto
		item.titleScore = nextScore
	}
	item.summary.Status = terminaldomain.SessionStatusBusy
	item.summary.UpdatedAt = now
	item.summary.FinishedAt = time.Time{}
	item.summary.ErrorMessage = ""
	item.summary.ExitCode = nil
	item.closedByUser = false
	item.turnRunning = true
	item.turnCancel = turnCancel
	turn := item.beginTurnLocked(prompt, attachments, now)
	item.appendEntryLocked("input", prompt)
	snapshot := item.summary
	item.mu.Unlock()
	s.persistSession(item)

	go s.runTurn(item, turnCtx, turn.ID, prompt, attachments, cloneTerminalSkillContext(req.SkillContext))

	return snapshot, nil
}

func (s *Service) Delete(ownerID string, sessionID string) (terminaldomain.Session, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		if !errors.Is(err, ErrSessionNotFound) {
			return terminaldomain.Session{}, err
		}
		item, err = s.restorePersistedOwnedSession(ownerID, sessionID)
		if err != nil {
			return terminaldomain.Session{}, err
		}
	}

	snapshot := item.snapshot()
	statePath, err := resolveTerminalSessionStateFilePath(s.options.WorkingDir, snapshot.ID)
	if err != nil {
		return terminaldomain.Session{}, err
	}
	workspaceDir := strings.TrimSpace(snapshot.WorkingDir)
	if workspaceDir == "" {
		workspaceDir, err = resolveSessionWorkspacePath(s.options.WorkingDir, snapshot.ID)
		if err != nil {
			return terminaldomain.Session{}, err
		}
	}

	s.mu.Lock()
	current, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return terminaldomain.Session{}, ErrSessionNotFound
	}
	item = current
	item.mu.Lock()
	item.deleted = true
	item.closedByUser = true
	if item.turnCancel != nil {
		item.turnCancel()
		item.turnCancel = nil
	}
	item.turnRunning = false
	item.mu.Unlock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()

	var cleanupErr error
	if err := removeTerminalSessionStateFile(statePath); err != nil {
		cleanupErr = errors.Join(cleanupErr, err)
	}
	if err := os.RemoveAll(workspaceDir); err != nil {
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove terminal workspace: %w", err))
	}
	if cleanupErr != nil {
		return snapshot, cleanupErr
	}
	return snapshot, nil
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
		if !item.closedByUser && terminaldomain.NormalizeSessionStatus(item.summary.Status) == terminaldomain.SessionStatusBusy {
			item.markInterruptedLocked(item.turnByIDLocked(item.activeTurnID), time.Now().UTC(), terminalHostUnavailableMessage)
		}
		item.turnRunning = false
		item.mu.Unlock()
		s.persistSession(item)
	}
}

func (s *Service) runTurn(item *runtimeSession, ctx context.Context, turnID string, prompt string, attachments []TurnAttachment, skillContext *execdomain.SkillContext) {
	command := resolveCodexCommand(s.options)
	threadID := item.thread()
	preparedAttachments, err := prepareTurnInputAttachments(item.workspaceDir(), turnID, attachments)
	if err != nil {
		s.finishTurn(item, turnID, fmt.Errorf("prepare input attachments: %w", err), "")
		return
	}
	args := buildCodexTurnArgs(
		command,
		threadID,
		buildCodexTurnPrompt(prompt, preparedAttachments),
		imagePathsFromPreparedTurnAttachments(preparedAttachments),
	)
	runner := s.runner
	if runner == nil {
		runner = exec.CommandContext
	}
	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	cmd := runner(runCtx, command.path, args...)
	if workspaceDir := item.workspaceDir(); workspaceDir != "" {
		cmd.Dir = workspaceDir
		env, runtimeErr := prepareTerminalCodexRuntime(workspaceDir, skillContext)
		if runtimeErr != nil {
			s.finishTurn(item, turnID, runtimeErr, "")
			return
		}
		if len(env) > 0 {
			baseEnv := cmd.Env
			if len(baseEnv) == 0 {
				baseEnv = os.Environ()
			}
			cmd.Env = append(baseEnv, env...)
		}
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

	parseErr := s.consumeCodexOutput(item, turnID, stdout, runCancel)
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

func (s *Service) consumeCodexOutput(item *runtimeSession, turnID string, reader io.Reader, cancel func()) error {
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
		if fatalMessage := fatalCodexEventMessage(event.Message); fatalMessage != "" {
			if cancel != nil {
				cancel()
			}
			return fmt.Errorf("codex authentication failed: %s", fatalMessage)
		}
		s.applyCodexEvent(item, turnID, event)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read codex output: %w", err)
	}
	return nil
}

func fatalCodexEventMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	switch {
	case strings.Contains(normalized, "401 unauthorized"):
		return trimmed
	case strings.Contains(normalized, "403 forbidden"):
		return trimmed
	case strings.Contains(normalized, "missing bearer"):
		return trimmed
	case strings.Contains(normalized, "missing basic authentication"):
		return trimmed
	case strings.Contains(normalized, "missing bearer or basic authentication"):
		return trimmed
	case strings.Contains(normalized, "invalid api key"):
		return trimmed
	case strings.Contains(normalized, "incorrect api key"):
		return trimmed
	default:
		return ""
	}
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
			s.persistSession(item)
		}
	case "item.delta":
		return
	case "item.started":
		if event.Item == nil {
			return
		}
		item.mu.Lock()

		turn := item.turnByIDLocked(turnID)
		if turn == nil {
			item.mu.Unlock()
			return
		}
		switch normalizeCodexItemType(event.Item.Type) {
		case "command_execution":
			command := strings.TrimSpace(event.Item.Command)
			if command == "" {
				item.mu.Unlock()
				return
			}
			step := item.ensureCommandStepLocked(turn, event.Item.ID, command, time.Now().UTC())
			step.Status = "running"
			item.appendEntryLocked("system", "running command: "+command)
		}
		item.mu.Unlock()
		s.persistSession(item)
	case "item.completed":
		if event.Item == nil {
			return
		}
		now := time.Now().UTC()
		item.mu.Lock()

		turn := item.turnByIDLocked(turnID)
		if turn == nil {
			item.mu.Unlock()
			return
		}
		switch normalizeCodexItemType(event.Item.Type) {
		case "agent_message":
			text := normalizeChunk(event.Item.Text)
			if strings.TrimSpace(text) == "" {
				item.mu.Unlock()
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
				item.mu.Unlock()
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
		item.mu.Unlock()
		s.persistSession(item)
	}
}

func (s *Service) finishTurn(item *runtimeSession, turnID string, turnErr error, stderrText string) {
	now := time.Now().UTC()
	item.mu.Lock()

	turn := item.turnByIDLocked(turnID)
	activeTurn := item.activeTurnID == turnID

	if activeTurn {
		if item.turnCancel != nil {
			item.turnCancel()
			item.turnCancel = nil
		}
		item.turnRunning = false
		item.activeTurnID = ""
	} else if item.turnRunning && strings.TrimSpace(item.activeTurnID) != "" {
		s.finishSupersededTurnLocked(item, turn, turnErr, stderrText, now)
		item.mu.Unlock()
		s.persistSession(item)
		return
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
		item.mu.Unlock()
		s.persistSession(item)
		return
	}

	item.summary.Status = terminaldomain.SessionStatusReady
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
		item.mu.Unlock()
		s.persistSession(item)
		return
	}

	if errors.Is(turnErr, context.Canceled) || errors.Is(s.rootCtx.Err(), context.Canceled) {
		item.markInterruptedLocked(turn, now, terminalHostUnavailableMessage)
		item.mu.Unlock()
		s.persistSession(item)
		return
	}

	if isCodexCompactionFailure(stderrText, turnErr) {
		item.threadID = ""
		item.summary.TerminalSessionID = item.summary.ID
		item.summary.ErrorMessage = terminalCompactionResetMessage
		item.appendEntryLocked("system", "codex runtime thread reset after context compaction failure")
		if turn != nil {
			turn.Status = "failed"
			turn.FinishedAt = now
			item.newSystemStepLocked(turn, "Thread reset", terminalCompactionResetMessage, now, "failed")
			turn.promoteFinalOutput()
		}
		item.mu.Unlock()
		s.persistSession(item)
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
	item.mu.Unlock()
	s.persistSession(item)
}

func (s *Service) finishSupersededTurnLocked(item *runtimeSession, turn *runtimeTurn, turnErr error, stderrText string, now time.Time) {
	if turn == nil {
		return
	}

	if turnErr == nil {
		turn.Status = "completed"
		if turn.FinishedAt.IsZero() {
			turn.FinishedAt = now
		}
		turn.promoteFinalOutput()
		return
	}

	if errors.Is(turnErr, context.Canceled) || errors.Is(s.rootCtx.Err(), context.Canceled) {
		reason := terminalHostUnavailableMessage
		if turn.Status != "interrupted" {
			turn.Status = "interrupted"
		}
		if turn.FinishedAt.IsZero() {
			turn.FinishedAt = now
		}
		for _, step := range turn.steps {
			if step == nil || !isRuntimeStepLive(step.Status) {
				continue
			}
			step.Status = "interrupted"
			if step.FinishedAt.IsZero() {
				step.FinishedAt = now
			}
		}
		if !hasRuntimeTurnSystemStep(turn, "Interrupted", reason) {
			item.newSystemStepLocked(turn, "Interrupted", reason, now, "failed")
		}
		turn.promoteFinalOutput()
		return
	}

	message := compactCodexError(stderrText, turnErr)
	turn.Status = "failed"
	if turn.FinishedAt.IsZero() {
		turn.FinishedAt = now
	}
	if !hasRuntimeTurnSystemStep(turn, "Request failed", message) {
		item.newSystemStepLocked(turn, "Request failed", message, now, "failed")
	}
	turn.promoteFinalOutput()
}

func (s *runtimeSession) beginTurnLocked(prompt string, attachments []TurnAttachment, now time.Time) *runtimeTurn {
	s.nextTurnID++
	turn := &runtimeTurn{
		ID:          fmt.Sprintf("turn-%d", s.nextTurnID),
		Prompt:      prompt,
		Attachments: cloneTurnAttachments(attachments),
		Status:      "running",
		StartedAt:   now,
		steps:       []*runtimeStep{},
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
		Attachments: cloneTurnAttachments(t.Attachments),
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

func buildCodexTurnArgs(command codexCommand, threadID string, prompt string, imagePaths []string) []string {
	args := append([]string{}, command.globalArgs...)
	args = append(args, "exec", "--enable", defaultLinuxSandboxBwrapFeature)
	if strings.TrimSpace(threadID) != "" {
		args = append(args, "resume", "--json", "--skip-git-repo-check")
		for _, imagePath := range imagePaths {
			args = append(args, "-i", imagePath)
		}
		args = append(args, threadID, prompt)
		return args
	}
	args = append(args,
		"--json",
		"--color", "never",
		"--skip-git-repo-check",
		"--sandbox", defaultCodexSandbox,
	)
	for _, imagePath := range imagePaths {
		args = append(args, "-i", imagePath)
	}
	args = append(args, prompt)
	return args
}

func buildCodexTurnPrompt(prompt string, attachments []preparedTurnAttachment) string {
	files := make([]preparedTurnAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment.IsImage {
			continue
		}
		files = append(files, attachment)
	}
	if len(files) == 0 {
		return prompt
	}
	lines := []string{strings.TrimSpace(prompt), "", "Attached files are available in the workspace:"}
	for _, attachment := range files {
		lines = append(lines, fmt.Sprintf("- %s (%s): %s", attachment.Name, attachment.ContentType, attachment.PromptPath))
	}
	lines = append(lines, "Read the files directly from disk when needed.")
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func imagePathsFromPreparedTurnAttachments(items []preparedTurnAttachment) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.IsImage && strings.TrimSpace(item.Path) != "" {
			out = append(out, item.Path)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneTerminalSkillContext(input *execdomain.SkillContext) *execdomain.SkillContext {
	if input == nil {
		return nil
	}
	out := *input
	if len(input.Skills) > 0 {
		out.Skills = make([]execdomain.SkillSpec, 0, len(input.Skills))
		for _, skill := range input.Skills {
			cloned := skill
			if len(skill.ParameterTemplate) > 0 {
				cloned.ParameterTemplate = make(map[string]string, len(skill.ParameterTemplate))
				for key, value := range skill.ParameterTemplate {
					cloned.ParameterTemplate[key] = value
				}
			}
			cloned.Constraints = append([]string{}, skill.Constraints...)
			cloned.Abilities = append([]string{}, skill.Abilities...)
			out.Skills = append(out.Skills, cloned)
		}
	}
	if len(input.ResolvedParameters) > 0 {
		out.ResolvedParameters = make(map[string]string, len(input.ResolvedParameters))
		for key, value := range input.ResolvedParameters {
			out.ResolvedParameters[key] = value
		}
	}
	out.Conflicts = append([]execdomain.SkillConflict{}, input.Conflicts...)
	return &out
}

func cloneTurnAttachments(items []TurnAttachment) []TurnAttachment {
	if len(items) == 0 {
		return nil
	}
	out := make([]TurnAttachment, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ContentType) == "" {
			continue
		}
		if strings.TrimSpace(item.DataURL) == "" && strings.TrimSpace(item.AssetURL) == "" && strings.TrimSpace(item.WorkspacePath) == "" {
			continue
		}
		out = append(out, TurnAttachment{
			Name:          strings.TrimSpace(item.Name),
			ContentType:   strings.TrimSpace(item.ContentType),
			DataURL:       strings.TrimSpace(item.DataURL),
			AssetURL:      strings.TrimSpace(item.AssetURL),
			PreviewURL:    strings.TrimSpace(item.PreviewURL),
			WorkspacePath: strings.TrimSpace(item.WorkspacePath),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeTurnAttachments(items []execdomain.UserAttachment) []TurnAttachment {
	if len(items) == 0 {
		return nil
	}
	normalized := execdomain.NormalizeUserAttachments(items)
	if len(normalized) == 0 {
		return nil
	}
	out := make([]TurnAttachment, 0, len(normalized))
	for _, item := range normalized {
		out = append(out, TurnAttachment{
			Name:          item.Name,
			ContentType:   item.ContentType,
			DataURL:       item.DataURL,
			AssetURL:      item.AssetURL,
			PreviewURL:    item.PreviewURL,
			WorkspacePath: item.WorkspacePath,
		})
	}
	return out
}

func defaultAttachmentPrompt(attachments []TurnAttachment) string {
	count := len(attachments)
	if count <= 0 {
		return ""
	}
	imageCount := 0
	for _, attachment := range attachments {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(attachment.ContentType)), "image/") {
			imageCount += 1
		}
	}
	if imageCount == count {
		if count <= 1 {
			return "Inspect the attached image."
		}
		return fmt.Sprintf("Inspect the attached %d images.", count)
	}
	if count <= 1 {
		return "Review the attached file."
	}
	if imageCount == 0 {
		return fmt.Sprintf("Review the attached %d files.", count)
	}
	return fmt.Sprintf("Review the attached %d files, including %d images.", count, imageCount)
}

func prepareTurnInputAttachments(workspaceDir string, turnID string, attachments []TurnAttachment) ([]preparedTurnAttachment, error) {
	if len(attachments) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(workspaceDir) == "" {
		return nil, errors.New("terminal workspace is empty")
	}
	dir := filepath.Join(workspaceDir, terminalTurnAttachmentDirName, sanitizeWorkspaceSegment(turnID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("prepare turn attachment dir: %w", err)
	}
	items := make([]preparedTurnAttachment, 0, len(attachments))
	for index, attachment := range attachments {
		filename := resolveTurnAttachmentFilename(index, attachment)
		path := filepath.Join(dir, filename)
		data, err := readTurnAttachmentBytes(attachment)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, fmt.Errorf("write turn attachment: %w", err)
		}
		promptPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			promptPath = path
		}
		items = append(items, preparedTurnAttachment{
			Name:        strings.TrimSpace(attachment.Name),
			ContentType: strings.TrimSpace(attachment.ContentType),
			Path:        path,
			PromptPath:  filepath.ToSlash(promptPath),
			IsImage:     strings.HasPrefix(strings.ToLower(strings.TrimSpace(attachment.ContentType)), "image/"),
		})
	}
	return items, nil
}

func readTurnAttachmentBytes(attachment TurnAttachment) ([]byte, error) {
	if dataURL := strings.TrimSpace(attachment.DataURL); dataURL != "" {
		return decodeAttachmentDataURL(dataURL)
	}
	if workspacePath := strings.TrimSpace(attachment.WorkspacePath); workspacePath != "" {
		data, err := os.ReadFile(workspacePath)
		if err != nil {
			return nil, fmt.Errorf("read workspace attachment: %w", err)
		}
		return data, nil
	}
	return nil, errors.New("attachment payload is empty")
}

func resolveTurnAttachmentFilename(index int, attachment TurnAttachment) string {
	name := sanitizeWorkspaceSegment(strings.TrimSpace(attachment.Name))
	if name == "" {
		name = fmt.Sprintf("attachment-%d%s", index+1, attachmentExtension(attachment.ContentType))
	}
	if filepath.Ext(name) == "" {
		name += attachmentExtension(attachment.ContentType)
	}
	return name
}

func attachmentExtension(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	default:
		extensions, err := mime.ExtensionsByType(strings.TrimSpace(contentType))
		if err == nil && len(extensions) > 0 {
			return extensions[0]
		}
		return ".bin"
	}
}

func decodeAttachmentDataURL(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("attachment data url is empty")
	}
	if !strings.HasPrefix(value, "data:") {
		return nil, errors.New("attachment data url is invalid")
	}
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return nil, errors.New("attachment data url is invalid")
	}
	if !strings.HasSuffix(strings.ToLower(parts[0]), ";base64") {
		return nil, errors.New("attachment data url is invalid")
	}
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode attachment data url: %w", err)
	}
	return decoded, nil
}

func buildCodexLabel(commandPath string, args []string) string {
	parts := []string{commandPath}
	parts = append(parts, args...)
	parts = append(parts, "exec")
	return strings.Join(parts, " ")
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

func isCodexCompactionFailure(stderrText string, turnErr error) bool {
	parts := []string{normalizeChunk(stderrText)}
	if turnErr != nil {
		parts = append(parts, turnErr.Error())
	}
	normalized := strings.ToLower(strings.Join(parts, "\n"))
	switch {
	case strings.Contains(normalized, "remote compaction failed"):
		return true
	case strings.Contains(normalized, "failed to run pre-sampling compact"):
		return true
	case strings.Contains(normalized, "responses/compact"):
		return true
	default:
		return false
	}
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
	ownerID = normalizeTerminalOwnerID(ownerID)
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

func normalizeTerminalOwnerID(_ string) string {
	return sharedTerminalOwnerID
}

func (s *Service) countActiveLocked() int {
	total := 0
	for _, item := range s.sessions {
		snapshot := item.snapshot()
		if terminaldomain.IsSessionOpenStatus(snapshot.Status) {
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

func prepareTerminalCodexRuntime(workspaceDir string, skillContext *execdomain.SkillContext) ([]string, error) {
	prepared, err := runtimeconfig.Prepare(runtimeconfig.Spec{
		RuntimeHome:  filepath.Join(workspaceDir, terminalCodexHomeDirName),
		WorkspaceDir: workspaceDir,
		ManagedFiles: []runtimeconfig.ManagedFile{{
			RelativePath: ".alter0/codex-runtime/skills.md",
			Content:      renderTerminalSkillContextMarkdown(skillContext),
			Mode:         0o644,
		}},
		RootInstructions: "- Read `.alter0/codex-runtime/skills.md` before acting. Apply only the skills selected for the current Terminal turn.",
	})
	if err != nil {
		return nil, fmt.Errorf("prepare terminal codex runtime: %w", err)
	}
	return prepared.Env, nil
}

func renderTerminalSkillContextMarkdown(skillContext *execdomain.SkillContext) string {
	lines := []string{"# Skills", ""}
	if skillContext == nil || len(skillContext.Skills) == 0 {
		lines = append(lines, "No skills selected for this Terminal turn.", "")
		return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
	}
	protocol := strings.TrimSpace(skillContext.Protocol)
	if protocol == "" {
		protocol = execdomain.SkillContextProtocolVersion
	}
	lines = append(lines, "- protocol: "+protocol, "")
	for _, skill := range skillContext.Skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			name = strings.TrimSpace(skill.ID)
		}
		if name == "" {
			continue
		}
		lines = append(lines, "## "+name, "")
		if strings.TrimSpace(skill.ID) != "" {
			lines = append(lines, "- id: "+strings.TrimSpace(skill.ID))
		}
		if strings.TrimSpace(skill.Description) != "" {
			lines = append(lines, "- description: "+strings.TrimSpace(skill.Description))
		}
		if strings.TrimSpace(skill.FilePath) != "" {
			lines = append(lines, "- file_path: "+strings.TrimSpace(skill.FilePath))
		}
		if strings.TrimSpace(skill.Guide) != "" {
			lines = append(lines, "", "### Guide", "", strings.TrimSpace(skill.Guide))
		}
		if len(skill.Constraints) > 0 {
			lines = append(lines, "", "### Constraints")
			for _, constraint := range skill.Constraints {
				if strings.TrimSpace(constraint) != "" {
					lines = append(lines, "- "+strings.TrimSpace(constraint))
				}
			}
		}
		if len(skill.Abilities) > 0 {
			lines = append(lines, "", "### Abilities")
			for _, ability := range skill.Abilities {
				if strings.TrimSpace(ability) != "" {
					lines = append(lines, "- "+strings.TrimSpace(ability))
				}
			}
		}
		lines = append(lines, "")
	}
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

func resolveSessionWorkspacePath(baseDir string, sessionID string) (string, error) {
	root := strings.TrimSpace(baseDir)
	if root == "" {
		root = "."
	}
	sanitizedSessionID := sanitizeWorkspaceSegment(sessionID)
	if sanitizedSessionID == "" {
		return "", ErrSessionRecoverIDRequired
	}
	workspaceDir := filepath.Join(
		root,
		defaultWorkspaceRootDirName,
		workspaceDirectoryName,
		workspaceTerminalDirName,
		workspaceSessionsDirName,
		sanitizedSessionID,
	)
	absolute, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("resolve terminal workspace path: %w", err)
	}
	return absolute, nil
}

func resolveSessionWorkspaceDir(baseDir string, sessionID string) (string, error) {
	workspaceDir, err := resolveSessionWorkspacePath(baseDir, sessionID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return "", fmt.Errorf("prepare terminal workspace: %w", err)
	}
	return workspaceDir, nil
}

func sanitizeWorkspaceSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	hyphenPending := false
	for _, ch := range trimmed {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.' {
			builder.WriteRune(ch)
			hyphenPending = false
			continue
		}
		if hyphenPending {
			continue
		}
		builder.WriteByte('-')
		hyphenPending = true
	}
	sanitized := strings.Trim(builder.String(), "-._")
	if sanitized == "" {
		return ""
	}
	if len(sanitized) > 96 {
		sanitized = sanitized[:96]
	}
	return strings.ToLower(sanitized)
}

func (s *runtimeSession) thread() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.threadID
}

func (s *runtimeSession) workspaceDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.summary.WorkingDir)
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

func (s *runtimeSession) markInterruptedLocked(turn *runtimeTurn, now time.Time, message string) {
	reason := strings.TrimSpace(message)
	if reason == "" {
		reason = terminalHostUnavailableMessage
	}
	summaryText := "terminal interrupted: " + reason
	alreadyRecorded := hasRuntimeTurnSystemStep(turn, "Interrupted", reason)

	s.summary.Status = terminaldomain.SessionStatusInterrupted
	s.summary.ErrorMessage = reason
	s.summary.FinishedAt = now
	if s.summary.UpdatedAt.Before(now) {
		s.summary.UpdatedAt = now
	}
	if !alreadyRecorded {
		s.appendEntryLocked("system", summaryText)
	}
	if turn == nil {
		return
	}
	if turn.Status != "interrupted" {
		turn.Status = "interrupted"
	}
	if turn.FinishedAt.IsZero() {
		turn.FinishedAt = now
	}
	for _, step := range turn.steps {
		if step == nil || !isRuntimeStepLive(step.Status) {
			continue
		}
		step.Status = "interrupted"
		if step.FinishedAt.IsZero() {
			step.FinishedAt = now
		}
	}
	if !alreadyRecorded {
		s.newSystemStepLocked(turn, "Interrupted", reason, now, "failed")
	}
	turn.promoteFinalOutput()
}

func hasRuntimeTurnSystemStep(turn *runtimeTurn, title string, message string) bool {
	if turn == nil {
		return false
	}
	targetTitle := strings.TrimSpace(title)
	targetMessage := strings.TrimSpace(message)
	for _, step := range turn.steps {
		if step == nil || step.Type != "log" {
			continue
		}
		if strings.TrimSpace(step.Title) != targetTitle {
			continue
		}
		for _, block := range step.Blocks {
			if strings.TrimSpace(block.Content) == targetMessage {
				return true
			}
		}
	}
	return false
}

func isRuntimeStepLive(status string) bool {
	normalized := strings.TrimSpace(strings.ToLower(status))
	return normalized == "" || normalized == "running" || normalized == "starting"
}

func (s *runtimeSession) snapshot() terminaldomain.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := s.summary
	snapshot.Status = terminaldomain.NormalizeSessionStatus(snapshot.Status)
	return snapshot
}
