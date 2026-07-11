package application

import (
	"bufio"
	"context"
	"crypto/rand"
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
	"strconv"
	"strings"
	"sync"
	"time"

	chatruntimedomain "alter0/internal/chatruntime/domain"
	"alter0/internal/codex/infrastructure/runtimeconfig"
	execdomain "alter0/internal/execution/domain"
	sharedapp "alter0/internal/shared/application"
)

const (
	defaultCodexCommand                  = "codex"
	defaultCodexSandbox                  = "danger-full-access"
	defaultLinuxSandboxBwrapFeature      = "use_linux_sandbox_bwrap"
	workspaceDirectoryName               = "workspaces"
	workspaceChatRuntimeDirName          = "chat"
	workspaceSessionsDirName             = "sessions"
	chatRuntimeTurnAttachmentDirName     = "input-attachments"
	chatRuntimeCodexHomeDirName          = "codex-home"
	defaultChatRuntimeSessionTitle       = "New"
	maxEntryPageLimit                    = 200
	chatRuntimeHostUnavailableMessage    = "chatRuntime host unavailable"
	chatRuntimeCompactionRecoveryMessage = "codex context compaction failed; next input will continue the previous runtime thread in the same workspace"
	chatRuntimeSessionIDPrefix           = "c_"
	chatRuntimeSessionIDLength           = 16
	chatRuntimeSessionIDAlphabet         = "abcdefghijklmnopqrstuvwxyz0123456789"
)

var (
	ErrSessionOwnerRequired     = errors.New("chat session owner is required")
	ErrSessionNotFound          = errors.New("chat session not found")
	ErrSessionNotRunning        = errors.New("chat session is not running")
	ErrSessionBusy              = errors.New("chat session is processing another turn")
	ErrSessionInputRequired     = errors.New("chat input is required")
	ErrSessionRecoverIDRequired = errors.New("chat recovery session id is required")
	ErrTurnNotFound             = errors.New("chat turn not found")
	ErrRuntimeEventNotFound     = errors.New("chatRuntime event not found")
)

const chatRuntimeOwnerID = "chat"

type Options struct {
	WorkingDir         string
	Shell              string
	ShellArgs          []string
	ShellArgsLine      string
	RepositoryCatalog  RepositoryCatalog
	RepositoryPreparer RepositoryWorkspacePreparer
}

type CreateRequest struct {
	OwnerID string
	Title   string
}

type RecoverRequest struct {
	OwnerID          string
	SessionID        string
	RuntimeSessionID string
	Title            string
	CreatedAt        time.Time
	LastOutputAt     time.Time
	UpdatedAt        time.Time
}

type InputRequest struct {
	OwnerID         string
	SessionID       string
	Input           string
	ClientRequestID string
	Attachments     []execdomain.UserAttachment
	SkillContext    *execdomain.SkillContext
	Repository      *chatruntimedomain.RepositoryRef
}

type EntryPage struct {
	Items      []chatruntimedomain.Entry `json:"items"`
	Cursor     int                       `json:"cursor"`
	NextCursor int                       `json:"next_cursor"`
	HasMore    bool                      `json:"has_more"`
}

type commandRunner func(ctx context.Context, name string, args ...string) *exec.Cmd
type SessionEventHook func(event SessionEvent)

const (
	SessionEventSessionCreated    = "session.created"
	SessionEventSessionUpdated    = "session.updated"
	SessionEventSessionDeleted    = "session.deleted"
	SessionEventTurnStarted       = "turn.started"
	SessionEventTurnEventAppended = "turn.event.appended"
	SessionEventTurnEventUpdated  = "turn.event.updated"
	SessionEventTurnCompleted     = "turn.completed"
	SessionEventTurnFailed        = "turn.failed"
	SessionEventTurnInterrupted   = "turn.interrupted"
)

type SessionEvent struct {
	OwnerID      string
	SessionID    string
	EventType    string
	Session      chatruntimedomain.Session
	Turn         *TurnSummary
	RuntimeEvent *RuntimeTraceEvent
}

type Service struct {
	rootCtx            context.Context
	idGenerator        sharedapp.IDGenerator
	logger             *slog.Logger
	options            Options
	runner             commandRunner
	repositoryCatalog  RepositoryCatalog
	repositoryPreparer RepositoryWorkspacePreparer
	hookMu             sync.RWMutex
	eventHook          SessionEventHook

	mu       sync.RWMutex
	sessions map[string]*runtimeSession
}

type codexCommand struct {
	path       string
	globalArgs []string
	label      string
}

type codexExecEvent struct {
	Type              string           `json:"type"`
	ThreadID          string           `json:"thread_id,omitempty"`
	Message           string           `json:"message,omitempty"`
	Title             string           `json:"title,omitempty"`
	Name              string           `json:"name,omitempty"`
	ThreadTitle       string           `json:"thread_title,omitempty"`
	ConversationTitle string           `json:"conversation_title,omitempty"`
	Thread            *codexExecThread `json:"thread,omitempty"`
	Session           *codexExecThread `json:"session,omitempty"`
	Conversation      *codexExecThread `json:"conversation,omitempty"`
	Item              *codexExecItem   `json:"item,omitempty"`
}

type codexExecThread struct {
	Title string `json:"title,omitempty"`
	Name  string `json:"name,omitempty"`
}

type codexExecItem struct {
	ID               string `json:"id,omitempty"`
	Type             string `json:"type,omitempty"`
	Channel          string `json:"channel,omitempty"`
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

	summary       chatruntimedomain.Session
	titleManual   bool
	titleAuto     bool
	titleExternal bool
	titleScore    int
	entries       []chatruntimedomain.Entry
	nextID        int
	turns         []*runtimeTurn
	activeTurnID  string
	nextTurnID    int
	nextEventID   int
	threadID      string
	turnRunning   bool
	turnCancel    context.CancelFunc
	closedByUser  bool
	deleted       bool
}

type runtimeTurn struct {
	ID              string
	ClientRequestID string
	Prompt          string
	Attachments     []TurnAttachment
	SkillContext    *execdomain.SkillContext
	Status          string
	StartedAt       time.Time
	FinishedAt      time.Time
	FinalOutput     string
	events          []*runtimeEventRecord
}

type runtimeEventRecord struct {
	ID         string
	ItemID     string
	Type       string
	Title      string
	Status     string
	Preview    string
	StartedAt  time.Time
	FinishedAt time.Time
	Blocks     []RuntimeDetailBlock
	Searchable bool
}

func nextSessionContentUpdatedAt(previous time.Time, now time.Time) time.Time {
	previous = previous.UTC().Truncate(time.Millisecond)
	candidate := now.UTC().Truncate(time.Millisecond)
	if !previous.IsZero() && !candidate.After(previous) {
		return previous.Add(time.Millisecond)
	}
	return candidate
}

func (s *runtimeSession) advanceContentUpdatedAtLocked(now time.Time) {
	s.summary.UpdatedAt = nextSessionContentUpdatedAt(s.summary.UpdatedAt, now)
}

func NewService(ctx context.Context, idGenerator sharedapp.IDGenerator, logger *slog.Logger, options Options) *Service {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	options = normalizeOptions(options)
	repositoryCatalog := options.RepositoryCatalog
	repositoryPreparer := options.RepositoryPreparer
	if repositoryCatalog == nil || repositoryPreparer == nil {
		githubRepositories := newGitHubRepositoryProvider()
		if repositoryCatalog == nil {
			repositoryCatalog = githubRepositories
		}
		if repositoryPreparer == nil {
			repositoryPreparer = githubRepositories
		}
	}
	service := &Service{
		rootCtx:            ctx,
		idGenerator:        idGenerator,
		logger:             logger,
		options:            options,
		runner:             exec.CommandContext,
		repositoryCatalog:  repositoryCatalog,
		repositoryPreparer: repositoryPreparer,
		sessions:           map[string]*runtimeSession{},
	}
	go func() {
		<-ctx.Done()
		service.shutdown()
	}()
	service.loadPersistedSessions()
	return service
}

func (s *Service) SetSessionEventHook(hook SessionEventHook) {
	if s == nil {
		return
	}
	s.hookMu.Lock()
	defer s.hookMu.Unlock()
	s.eventHook = hook
}

func (s *Service) currentSessionEventHook() SessionEventHook {
	if s == nil {
		return nil
	}
	s.hookMu.RLock()
	defer s.hookMu.RUnlock()
	return s.eventHook
}

func (s *Service) publishSessionEvent(item *runtimeSession, eventType string) {
	hook := s.currentSessionEventHook()
	if hook == nil || item == nil || strings.TrimSpace(eventType) == "" {
		return
	}

	item.mu.RLock()
	session := item.summary
	session.Repository = cloneRepositoryBinding(item.summary.Repository)
	item.mu.RUnlock()

	if strings.TrimSpace(session.OwnerID) == "" || strings.TrimSpace(session.ID) == "" {
		return
	}
	hook(SessionEvent{
		OwnerID:   session.OwnerID,
		SessionID: session.ID,
		EventType: eventType,
		Session:   session,
	})
}

func (s *Service) publishTurnSessionEvent(item *runtimeSession, eventType string, turnID string, runtimeEventID string) {
	hook := s.currentSessionEventHook()
	if hook == nil || item == nil || strings.TrimSpace(eventType) == "" {
		return
	}

	item.mu.RLock()
	session := item.summary
	session.Repository = cloneRepositoryBinding(item.summary.Repository)
	var turnSummary *TurnSummary
	var runtimeEvent *RuntimeTraceEvent
	if turn := item.turnByIDLocked(turnID); turn != nil {
		summary := turn.summary(session.ID)
		if strings.TrimSpace(runtimeEventID) != "" {
			if event, seq := turn.runtimeEventByID(runtimeEventID); event != nil {
				detail := chatRuntimeRuntimeTraceEvent(session.ID, turn.ID, seq, event.summary())
				runtimeEvent = &detail
			}
			summary.RuntimeTraceEvents = nil
		}
		turnSummary = &summary
	}
	item.mu.RUnlock()

	if strings.TrimSpace(session.OwnerID) == "" || strings.TrimSpace(session.ID) == "" {
		return
	}
	if (eventType == SessionEventTurnEventAppended || eventType == SessionEventTurnEventUpdated) && runtimeEvent == nil {
		return
	}
	hook(SessionEvent{
		OwnerID:      session.OwnerID,
		SessionID:    session.ID,
		EventType:    eventType,
		Session:      session,
		Turn:         turnSummary,
		RuntimeEvent: runtimeEvent,
	})
}

func sessionEventTypeForTurn(turn *runtimeTurn) string {
	if turn == nil {
		return ""
	}
	switch normalizeFallbackStatus(turn.Status) {
	case "failed":
		return SessionEventTurnFailed
	case "interrupted", "canceled":
		return SessionEventTurnInterrupted
	default:
		return SessionEventTurnCompleted
	}
}

func (s *Service) Create(req CreateRequest) (chatruntimedomain.Session, error) {
	ownerID := normalizeChatRuntimeOwnerID(req.OwnerID)

	s.mu.Lock()

	command := resolveCodexCommand(s.options)
	sessionID := s.newSessionIDLocked()
	workspaceDir, err := resolveSessionWorkspaceDir(s.options.WorkingDir, sessionID)
	if err != nil {
		s.mu.Unlock()
		return chatruntimedomain.Session{}, err
	}
	title := strings.TrimSpace(req.Title)
	titleManual := false
	titleAuto := false
	if title == "" {
		title = defaultChatRuntimeSessionTitle
		titleAuto = true
	} else {
		titleManual = true
	}
	now := time.Now().UTC()
	session := &runtimeSession{
		summary: chatruntimedomain.Session{
			ID:               sessionID,
			RuntimeSessionID: sessionID,
			OwnerID:          ownerID,
			Title:            title,
			Shell:            command.label,
			WorkingDir:       workspaceDir,
			Status:           chatruntimedomain.SessionStatusReady,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		titleManual: titleManual,
		titleAuto:   titleAuto,
		entries:     []chatruntimedomain.Entry{},
	}
	s.sessions[sessionID] = session
	s.mu.Unlock()

	s.persistSession(session)
	return session.snapshot(), nil
}

func (s *Service) Recover(req RecoverRequest) (chatruntimedomain.Session, error) {
	ownerID := normalizeChatRuntimeOwnerID(req.OwnerID)

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return chatruntimedomain.Session{}, ErrSessionRecoverIDRequired
	}

	s.mu.Lock()

	if existing, ok := s.sessions[sessionID]; ok {
		snapshot := existing.snapshot()
		if normalizeChatRuntimeOwnerID(snapshot.OwnerID) != ownerID {
			s.mu.Unlock()
			return chatruntimedomain.Session{}, ErrSessionNotFound
		}
		s.mu.Unlock()
		return snapshot, nil
	}

	command := resolveCodexCommand(s.options)
	workspaceDir, err := resolveSessionWorkspaceDir(s.options.WorkingDir, sessionID)
	if err != nil {
		s.mu.Unlock()
		return chatruntimedomain.Session{}, err
	}
	title := strings.TrimSpace(req.Title)
	titleManual := false
	titleAuto := false
	titleScore := 0
	if title == "" {
		title = defaultChatRuntimeSessionTitle
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
	chatRuntimeSessionID := strings.TrimSpace(req.RuntimeSessionID)
	if chatRuntimeSessionID == "" {
		chatRuntimeSessionID = sessionID
	}
	session := &runtimeSession{
		summary: chatruntimedomain.Session{
			ID:               sessionID,
			RuntimeSessionID: chatRuntimeSessionID,
			OwnerID:          ownerID,
			Title:            title,
			Shell:            command.label,
			WorkingDir:       workspaceDir,
			Status:           chatruntimedomain.SessionStatusReady,
			CreatedAt:        createdAt,
			LastOutputAt:     lastOutputAt,
			UpdatedAt:        updatedAt,
		},
		titleManual: titleManual,
		titleAuto:   titleAuto,
		titleScore:  titleScore,
		entries:     []chatruntimedomain.Entry{},
		threadID:    resolveRecoveredThreadID(sessionID, chatRuntimeSessionID),
	}
	s.sessions[sessionID] = session
	s.mu.Unlock()

	s.persistSession(session)
	return session.snapshot(), nil
}

func (s *Service) List(ownerID string) []chatruntimedomain.Session {
	ownerID = normalizeChatRuntimeOwnerID(ownerID)
	s.syncMissingPersistedSessions()

	s.mu.RLock()
	sessions := make([]*runtimeSession, 0, len(s.sessions))
	for _, item := range s.sessions {
		sessions = append(sessions, item)
	}
	s.mu.RUnlock()

	items := make([]chatruntimedomain.Session, 0, len(sessions))
	for _, item := range sessions {
		s.reconcileOrphanedRuntimeSession(item)
		snapshot := item.snapshot()
		if normalizeChatRuntimeOwnerID(snapshot.OwnerID) != ownerID {
			continue
		}
		items = append(items, snapshot)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		leftAt := chatRuntimeSessionSortAt(items[i])
		rightAt := chatRuntimeSessionSortAt(items[j])
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

func chatRuntimeSessionSortAt(session chatruntimedomain.Session) time.Time {
	if !session.UpdatedAt.IsZero() {
		return session.UpdatedAt
	}
	if !session.CreatedAt.IsZero() {
		return session.CreatedAt
	}
	return time.Time{}
}

func (s *Service) Get(ownerID string, sessionID string) (chatruntimedomain.Session, bool) {
	item, err := s.getOrRestoreOwnedSession(ownerID, sessionID)
	if err != nil {
		return chatruntimedomain.Session{}, false
	}
	s.reconcileOrphanedRuntimeSession(item)
	return item.snapshot(), true
}

func (s *Service) GetDetail(ownerID string, sessionID string) (SessionDetail, bool) {
	item, err := s.getOrRestoreOwnedSession(ownerID, sessionID)
	if err != nil {
		return SessionDetail{}, false
	}
	s.reconcileOrphanedRuntimeSession(item)

	item.mu.RLock()
	defer item.mu.RUnlock()
	turns := make([]TurnSummary, 0, len(item.turns))
	for _, turn := range item.turns {
		if turn != nil {
			turns = append(turns, turn.summary(item.summary.ID))
		}
	}
	snapshot := item.summary
	snapshot.Status = chatruntimedomain.NormalizeSessionStatus(snapshot.Status)
	return SessionDetail{Session: snapshot, Turns: turns}, true
}

func (s *Service) SetPinned(ownerID string, sessionID string, pinned bool) (chatruntimedomain.Session, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		if !errors.Is(err, ErrSessionNotFound) {
			return chatruntimedomain.Session{}, err
		}
		item, err = s.restorePersistedOwnedSession(ownerID, sessionID)
		if err != nil {
			return chatruntimedomain.Session{}, err
		}
	}

	item.mu.Lock()
	item.summary.Pinned = pinned
	snapshot := item.summary
	snapshot.Repository = cloneRepositoryBinding(item.summary.Repository)
	item.mu.Unlock()

	s.persistSession(item)
	snapshot.Status = chatruntimedomain.NormalizeSessionStatus(snapshot.Status)
	return snapshot, nil
}

func (s *Service) ListTurns(ownerID string, sessionID string) ([]TurnSummary, error) {
	item, err := s.getOrRestoreOwnedSession(ownerID, sessionID)
	if err != nil {
		return nil, err
	}
	s.reconcileOrphanedRuntimeSession(item)

	item.mu.RLock()
	defer item.mu.RUnlock()

	items := make([]TurnSummary, 0, len(item.turns))
	for _, turn := range item.turns {
		if turn == nil {
			continue
		}
		items = append(items, turn.summary(item.summary.ID))
	}
	return items, nil
}

func (s *Service) GetRuntimeTraceEventDetail(ownerID string, sessionID string, turnID string, eventID string) (RuntimeTraceEventDetail, error) {
	item, err := s.getOrRestoreOwnedSession(ownerID, sessionID)
	if err != nil {
		return RuntimeTraceEventDetail{}, err
	}
	s.reconcileOrphanedRuntimeSession(item)

	item.mu.RLock()
	defer item.mu.RUnlock()

	turn := item.turnByIDLocked(turnID)
	if turn == nil {
		return RuntimeTraceEventDetail{}, ErrTurnNotFound
	}
	event, seq := turn.runtimeEventByID(eventID)
	if event == nil {
		return RuntimeTraceEventDetail{}, ErrRuntimeEventNotFound
	}
	return event.runtimeTraceEventDetail(item.summary.ID, turn.ID, seq), nil
}

func (s *Service) ListEntries(ownerID string, sessionID string, cursor int, limit int) (EntryPage, error) {
	item, err := s.getOrRestoreOwnedSession(ownerID, sessionID)
	if err != nil {
		return EntryPage{}, err
	}
	s.reconcileOrphanedRuntimeSession(item)

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

	items := make([]chatruntimedomain.Entry, 0, limit)
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

func (s *Service) Input(ownerID string, sessionID string, input string) (chatruntimedomain.Session, error) {
	return s.InputWithAttachments(InputRequest{
		OwnerID:   ownerID,
		SessionID: sessionID,
		Input:     input,
	})
}

func (s *Service) InputWithAttachments(req InputRequest) (chatruntimedomain.Session, error) {
	item, err := s.getOwnedSession(req.OwnerID, req.SessionID)
	if err != nil {
		if !errors.Is(err, ErrSessionNotFound) {
			return chatruntimedomain.Session{}, err
		}
		item, err = s.restorePersistedOwnedSession(req.OwnerID, req.SessionID)
		if err != nil {
			return chatruntimedomain.Session{}, err
		}
	}
	s.reconcileOrphanedRuntimeSession(item)

	attachments := normalizeTurnAttachments(req.Attachments)
	prompt := strings.TrimSpace(req.Input)
	if prompt == "" && len(attachments) > 0 {
		prompt = defaultAttachmentPrompt(attachments)
	}
	if prompt == "" {
		return chatruntimedomain.Session{}, ErrSessionInputRequired
	}

	var resolvedRepository *chatruntimedomain.Repository
	if req.Repository != nil {
		ref, refErr := normalizeRepositoryRef(*req.Repository)
		if refErr != nil {
			return chatruntimedomain.Session{}, refErr
		}
		item.mu.RLock()
		existingBinding := cloneRepositoryBinding(item.summary.Repository)
		hasTurns := len(item.turns) > 0
		item.mu.RUnlock()
		if existingBinding != nil {
			if !existingBinding.Matches(ref) {
				return chatruntimedomain.Session{}, ErrRepositoryBindingConflict
			}
		} else {
			if hasTurns {
				return chatruntimedomain.Session{}, ErrRepositoryBindingConflict
			}
			if s.repositoryCatalog == nil {
				return chatruntimedomain.Session{}, ErrRepositoryUnavailable
			}
			resolved, resolveErr := s.repositoryCatalog.Resolve(s.rootCtx, ref)
			if resolveErr != nil {
				return chatruntimedomain.Session{}, fmt.Errorf("%w: %v", ErrRepositoryUnavailable, resolveErr)
			}
			resolvedRef, resolveErr := normalizeRepositoryRef(chatruntimedomain.RepositoryRef{
				Provider: resolved.Provider,
				ID:       resolved.ID,
				FullName: resolved.FullName,
			})
			if resolveErr != nil || resolvedRef.ID != ref.ID || strings.TrimSpace(resolved.FullName) == "" {
				return chatruntimedomain.Session{}, ErrRepositoryInvalid
			}
			resolved.Provider = resolvedRef.Provider
			resolved.ID = resolvedRef.ID
			resolved.FullName = strings.TrimSpace(resolved.FullName)
			resolved.DefaultBranch = strings.TrimSpace(resolved.DefaultBranch)
			resolvedRepository = &resolved
		}
	}

	turnCtx, turnCancel := context.WithCancel(s.rootCtx)

	item.mu.Lock()
	if item.turnRunning {
		item.mu.Unlock()
		turnCancel()
		return chatruntimedomain.Session{}, ErrSessionBusy
	}
	if chatruntimedomain.NormalizeSessionStatus(item.summary.Status) == chatruntimedomain.SessionStatusBusy {
		item.mu.Unlock()
		turnCancel()
		return chatruntimedomain.Session{}, ErrSessionNotRunning
	}
	if resolvedRepository != nil {
		if item.summary.Repository != nil || len(item.turns) > 0 {
			item.mu.Unlock()
			turnCancel()
			return chatruntimedomain.Session{}, ErrRepositoryBindingConflict
		}
		binding := chatruntimedomain.NewRepositoryBinding(*resolvedRepository)
		item.summary.Repository = &binding
	}
	now := time.Now().UTC()
	if nextTitle, nextAuto, nextScore, changed := nextAutoSessionTitle(
		item.summary.Title,
		item.titleManual || item.titleExternal,
		item.titleScore,
		prompt,
		item.summary.ID,
		64,
	); changed {
		item.summary.Title = nextTitle
		item.titleAuto = nextAuto
		item.titleScore = nextScore
	}
	item.summary.Status = chatruntimedomain.SessionStatusBusy
	item.summary.FinishedAt = time.Time{}
	item.summary.ErrorMessage = ""
	item.summary.ExitCode = nil
	item.closedByUser = false
	item.turnRunning = true
	item.turnCancel = turnCancel
	turn := item.beginTurnLocked(prompt, strings.TrimSpace(req.ClientRequestID), attachments, req.SkillContext, now)
	item.appendEntryLocked("input", prompt)
	snapshot := item.summary
	snapshot.Repository = cloneRepositoryBinding(item.summary.Repository)
	item.mu.Unlock()
	s.persistSession(item)
	s.publishTurnSessionEvent(item, SessionEventTurnStarted, turn.ID, "")

	go s.runTurn(item, turnCtx, turn.ID, prompt, attachments, cloneChatRuntimeSkillContext(req.SkillContext))

	return snapshot, nil
}

func (s *Service) Delete(ownerID string, sessionID string) (chatruntimedomain.Session, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		if !errors.Is(err, ErrSessionNotFound) {
			return chatruntimedomain.Session{}, err
		}
		item, err = s.restorePersistedOwnedSession(ownerID, sessionID)
		if err != nil {
			return chatruntimedomain.Session{}, err
		}
	}

	snapshot := item.snapshot()
	statePath, err := resolveChatRuntimeSessionStateFilePath(s.options.WorkingDir, snapshot.ID)
	if err != nil {
		return chatruntimedomain.Session{}, err
	}
	workspaceDir := strings.TrimSpace(snapshot.WorkingDir)
	if workspaceDir == "" {
		workspaceDir, err = resolveSessionWorkspacePath(s.options.WorkingDir, snapshot.ID)
		if err != nil {
			return chatruntimedomain.Session{}, err
		}
	}

	s.mu.Lock()
	current, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return chatruntimedomain.Session{}, ErrSessionNotFound
	}
	item = current
	item.mu.Lock()
	wasRunning := item.turnRunning || item.turnCancel != nil
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
	if err := removeChatRuntimeSessionStateFile(statePath); err != nil {
		cleanupErr = errors.Join(cleanupErr, err)
	}
	if err := os.RemoveAll(workspaceDir); err != nil {
		if wasRunning {
			s.cleanupChatRuntimeWorkspaceAfterDelete(snapshot.ID, workspaceDir)
		} else {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove chatRuntime workspace: %w", err))
		}
	}
	if cleanupErr != nil {
		return snapshot, cleanupErr
	}
	return snapshot, nil
}

func (s *Service) cleanupChatRuntimeWorkspaceAfterDelete(sessionID string, workspaceDir string) {
	workspaceDir = strings.TrimSpace(workspaceDir)
	if workspaceDir == "" {
		return
	}
	go func() {
		for attempt := 0; attempt < 5; attempt++ {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			if err := os.RemoveAll(workspaceDir); err != nil {
				if attempt == 4 {
					s.logger.Warn("remove deleted running chatRuntime workspace failed", "session_id", sessionID, "path", workspaceDir, "error", err.Error())
				}
				continue
			}
			return
		}
	}()
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
		if !item.closedByUser && chatruntimedomain.NormalizeSessionStatus(item.summary.Status) == chatruntimedomain.SessionStatusBusy {
			item.markInterruptedLocked(item.turnByIDLocked(item.activeTurnID), time.Now().UTC(), chatRuntimeHostUnavailableMessage)
		}
		item.turnRunning = false
		item.mu.Unlock()
		s.persistSession(item)
	}
}

func (s *Service) reconcileOrphanedRuntimeSession(item *runtimeSession) {
	if item == nil {
		return
	}

	now := time.Now().UTC()
	item.mu.Lock()
	if item.turnRunning || item.turnCancel != nil || item.closedByUser {
		item.mu.Unlock()
		return
	}
	turn := item.orphanedRuntimeTurnLocked()
	if turn == nil && chatruntimedomain.NormalizeSessionStatus(item.summary.Status) != chatruntimedomain.SessionStatusBusy {
		item.mu.Unlock()
		return
	}
	item.turnRunning = false
	item.turnCancel = nil
	item.activeTurnID = ""
	item.markInterruptedLocked(turn, now, chatRuntimeHostUnavailableMessage)
	item.mu.Unlock()

	s.persistSession(item)
}

func (s *Service) runTurn(item *runtimeSession, ctx context.Context, turnID string, prompt string, attachments []TurnAttachment, skillContext *execdomain.SkillContext) {
	command := resolveCodexCommand(s.options)
	threadID := item.thread()
	repository, err := s.prepareRepositoryForTurn(item, ctx)
	if err != nil {
		s.finishTurn(item, turnID, err, "")
		return
	}
	preparedAttachments, err := prepareTurnInputAttachments(item.workspaceDir(), turnID, attachments)
	if err != nil {
		s.finishTurn(item, turnID, fmt.Errorf("prepare input attachments: %w", err), "")
		return
	}
	args := buildCodexTurnArgs(
		command,
		threadID,
		buildCodexTurnPrompt(prompt, preparedAttachments, repository),
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
		env, runtimeErr := prepareChatRuntimeCodexRuntime(workspaceDir, skillContext)
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

func externalThreadTitleFromCodexEvent(event codexExecEvent) string {
	for _, candidate := range []string{
		event.Title,
		event.ThreadTitle,
		event.ConversationTitle,
		event.Name,
		nestedCodexThreadTitle(event.Thread),
		nestedCodexThreadTitle(event.Session),
		nestedCodexThreadTitle(event.Conversation),
	} {
		if title := strings.TrimSpace(candidate); title != "" {
			return title
		}
	}
	return ""
}

func nestedCodexThreadTitle(thread *codexExecThread) string {
	if thread == nil {
		return ""
	}
	if title := strings.TrimSpace(thread.Title); title != "" {
		return title
	}
	return strings.TrimSpace(thread.Name)
}

func applyExternalThreadTitleLocked(item *runtimeSession, title string, _ time.Time) bool {
	title = strings.TrimSpace(title)
	if item == nil || title == "" {
		return false
	}
	if item.summary.Title == title && item.titleExternal && !item.titleAuto && item.titleScore == 0 {
		return false
	}
	item.summary.Title = title
	item.titleAuto = false
	item.titleScore = 0
	item.titleExternal = true
	return true
}

func (s *Service) applyCodexEvent(item *runtimeSession, turnID string, event codexExecEvent) {
	switch strings.TrimSpace(event.Type) {
	case "thread.started":
		now := time.Now().UTC()
		titleChanged := false
		threadChanged := false
		if threadID := strings.TrimSpace(event.ThreadID); threadID != "" {
			item.mu.Lock()
			if item.threadID != threadID || item.summary.RuntimeSessionID != threadID {
				item.threadID = threadID
				item.summary.RuntimeSessionID = threadID
				threadChanged = true
			}
			titleChanged = applyExternalThreadTitleLocked(item, externalThreadTitleFromCodexEvent(event), now)
			item.mu.Unlock()
		} else if title := externalThreadTitleFromCodexEvent(event); title != "" {
			item.mu.Lock()
			titleChanged = applyExternalThreadTitleLocked(item, title, now)
			item.mu.Unlock()
		}
		if threadChanged || titleChanged {
			s.persistSession(item)
		}
		if titleChanged {
			s.publishSessionEvent(item, SessionEventSessionUpdated)
		}
	case "thread.updated", "session.updated", "conversation.updated":
		title := externalThreadTitleFromCodexEvent(event)
		if title == "" {
			return
		}
		item.mu.Lock()
		titleChanged := applyExternalThreadTitleLocked(item, title, time.Now().UTC())
		item.mu.Unlock()
		if titleChanged {
			s.persistSession(item)
			s.publishSessionEvent(item, SessionEventSessionUpdated)
		}
	case "item.delta":
		return
	case "item.started":
		if event.Item == nil {
			return
		}
		sessionEventType := ""
		runtimeEventID := ""
		item.mu.Lock()
		contentChanged := false

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
			step, created := item.ensureCommandEventResultLocked(turn, event.Item.ID, command, time.Now().UTC())
			step.Status = "running"
			runtimeEventID = step.ID
			if created {
				sessionEventType = SessionEventTurnEventAppended
			} else {
				sessionEventType = SessionEventTurnEventUpdated
			}
			item.appendEntryLocked("system", "running command: "+command)
			contentChanged = true
		}
		if contentChanged {
			item.advanceContentUpdatedAtLocked(time.Now().UTC())
		}
		item.mu.Unlock()
		s.persistSession(item)
		s.publishTurnSessionEvent(item, sessionEventType, turnID, runtimeEventID)
	case "item.completed":
		if event.Item == nil {
			return
		}
		now := time.Now().UTC()
		sessionEventType := ""
		runtimeEventID := ""
		item.mu.Lock()
		contentChanged := false

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
			if !isVisibleCodexAgentProcessMessage(event.Item) && !isFinalCodexAgentMessage(event.Item) {
				item.mu.Unlock()
				return
			}
			step := item.newEventLocked(turn, "message", deriveRuntimeEventTitle("message", text), now)
			step.Status = "completed"
			step.FinishedAt = now
			step.Preview = summarizeRuntimeEventSummary(text)
			step.Blocks = []RuntimeDetailBlock{{
				Type:    "text",
				Title:   "Message",
				Content: text,
				Status:  step.Status,
			}}
			step.Searchable = true
			item.appendEntryLocked("stdout", text)
			if isVisibleCodexAgentProcessMessage(event.Item) {
				runtimeEventID = step.ID
				sessionEventType = SessionEventTurnEventAppended
			}
			contentChanged = true
		case "reasoning", "plan":
			text := normalizeChunk(event.Item.Text)
			if strings.TrimSpace(text) == "" {
				item.mu.Unlock()
				return
			}
			step := item.newEventLocked(turn, normalizeCodexItemType(event.Item.Type), deriveRuntimeEventTitle(normalizeCodexItemType(event.Item.Type), text), now)
			step.Status = "completed"
			step.FinishedAt = now
			step.Preview = summarizeRuntimeEventSummary(text)
			step.Blocks = []RuntimeDetailBlock{{
				Type:    "text",
				Title:   step.Title,
				Content: text,
				Status:  step.Status,
			}}
			step.Searchable = true
			runtimeEventID = step.ID
			sessionEventType = SessionEventTurnEventAppended
			contentChanged = true
		case "command_execution":
			command := strings.TrimSpace(event.Item.Command)
			step, created := item.ensureCommandEventResultLocked(turn, event.Item.ID, command, now)
			step.Status = normalizeRuntimeEventStatus(strings.TrimSpace(event.Item.Status), event.Item.ExitCode)
			step.FinishedAt = now
			output := normalizeChunk(event.Item.AggregatedOutput)
			step.Preview = summarizeRuntimeEventSummary(strings.TrimSpace(command))
			step.Blocks = []RuntimeDetailBlock{{
				Type:     "chatRuntime",
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
			runtimeEventID = step.ID
			if created {
				sessionEventType = SessionEventTurnEventAppended
			} else {
				sessionEventType = SessionEventTurnEventUpdated
			}
			contentChanged = true
		}
		if contentChanged {
			item.advanceContentUpdatedAtLocked(now)
		}
		item.mu.Unlock()
		s.persistSession(item)
		s.publishTurnSessionEvent(item, sessionEventType, turnID, runtimeEventID)
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
		sessionEventType := sessionEventTypeForTurn(turn)
		item.mu.Unlock()
		s.persistSession(item)
		s.publishTurnSessionEvent(item, sessionEventType, turnID, "")
		return
	}

	if item.closedByUser {
		item.advanceContentUpdatedAtLocked(now)
		item.summary.FinishedAt = now
		item.summary.Status = chatruntimedomain.SessionStatusExited
		if turn != nil && turn.FinishedAt.IsZero() {
			turn.Status = "completed"
			turn.FinishedAt = now
			turn.promoteFinalOutput()
		}
		item.mu.Unlock()
		s.persistSession(item)
		s.publishTurnSessionEvent(item, SessionEventTurnCompleted, turnID, "")
		return
	}

	item.advanceContentUpdatedAtLocked(now)
	item.summary.ExitCode = nil

	if turnErr == nil {
		item.summary.Status = chatruntimedomain.SessionStatusReady
		item.summary.FinishedAt = time.Time{}
		item.summary.ErrorMessage = ""
		if turn != nil {
			turn.Status = "completed"
			turn.FinishedAt = now
			turn.promoteFinalOutput()
		}
		item.mu.Unlock()
		s.persistSession(item)
		s.publishTurnSessionEvent(item, SessionEventTurnCompleted, turnID, "")
		return
	}

	if errors.Is(turnErr, context.Canceled) || errors.Is(s.rootCtx.Err(), context.Canceled) {
		item.markInterruptedLocked(turn, now, chatRuntimeHostUnavailableMessage)
		item.mu.Unlock()
		s.persistSession(item)
		s.publishTurnSessionEvent(item, SessionEventTurnInterrupted, turnID, "")
		return
	}

	if isCodexCompactionFailure(stderrText, turnErr) {
		item.summary.Status = chatruntimedomain.SessionStatusFailed
		item.summary.FinishedAt = now
		item.summary.ErrorMessage = chatRuntimeCompactionRecoveryMessage
		item.appendEntryLocked("system", "codex previous runtime thread retained after context compaction failure")
		if turn != nil {
			turn.Status = "failed"
			turn.FinishedAt = now
			item.newSystemEventLocked(turn, "Compaction failed", chatRuntimeCompactionRecoveryMessage, now, "failed")
			turn.promoteFinalOutput()
		}
		item.mu.Unlock()
		s.persistSession(item)
		s.publishTurnSessionEvent(item, SessionEventTurnFailed, turnID, "")
		return
	}

	message := compactCodexError(stderrText, turnErr)
	item.summary.Status = chatruntimedomain.SessionStatusFailed
	item.summary.FinishedAt = now
	item.summary.ErrorMessage = message
	item.appendEntryLocked("system", "codex request failed: "+message)
	if turn != nil {
		turn.Status = "failed"
		turn.FinishedAt = now
		item.newSystemEventLocked(turn, "Request failed", message, now, "failed")
		turn.promoteFinalOutput()
	}
	item.mu.Unlock()
	s.persistSession(item)
	s.publishTurnSessionEvent(item, SessionEventTurnFailed, turnID, "")
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
		reason := chatRuntimeHostUnavailableMessage
		if turn.Status != "interrupted" {
			turn.Status = "interrupted"
		}
		if turn.FinishedAt.IsZero() {
			turn.FinishedAt = now
		}
		for _, step := range turn.events {
			if step == nil || !isRuntimeEventLive(step.Status) {
				continue
			}
			step.Status = "interrupted"
			if step.FinishedAt.IsZero() {
				step.FinishedAt = now
			}
		}
		if !hasRuntimeTurnSystemEvent(turn, "Interrupted", reason) {
			item.newSystemEventLocked(turn, "Interrupted", reason, now, "failed")
		}
		turn.promoteFinalOutput()
		return
	}

	message := compactCodexError(stderrText, turnErr)
	turn.Status = "failed"
	if turn.FinishedAt.IsZero() {
		turn.FinishedAt = now
	}
	if !hasRuntimeTurnSystemEvent(turn, "Request failed", message) {
		item.newSystemEventLocked(turn, "Request failed", message, now, "failed")
	}
	turn.promoteFinalOutput()
}

func (s *runtimeSession) beginTurnLocked(prompt string, clientRequestID string, attachments []TurnAttachment, skillContext *execdomain.SkillContext, now time.Time) *runtimeTurn {
	s.nextTurnID++
	turn := &runtimeTurn{
		ID:              fmt.Sprintf("turn-%d", s.nextTurnID),
		ClientRequestID: strings.TrimSpace(clientRequestID),
		Prompt:          prompt,
		Attachments:     cloneTurnAttachments(attachments),
		SkillContext:    cloneChatRuntimeSkillContext(skillContext),
		Status:          "running",
		StartedAt:       now,
		events:          []*runtimeEventRecord{},
	}
	s.turns = append(s.turns, turn)
	s.activeTurnID = turn.ID
	return turn
}

func (s *runtimeSession) turnByIDLocked(turnID string) *runtimeTurn {
	normalized := strings.TrimSpace(turnID)
	numericID := numericRuntimeLookupID(normalized)
	index := 0
	for _, turn := range s.turns {
		if turn == nil {
			continue
		}
		index++
		if turn.ID == normalized {
			return turn
		}
		if numericID > 0 && (numericRuntimeLookupID(turn.ID) == numericID || index == numericID) {
			return turn
		}
	}
	return nil
}

func (s *runtimeSession) orphanedRuntimeTurnLocked() *runtimeTurn {
	if turn := s.turnByIDLocked(s.activeTurnID); isRuntimeTurnLive(turn) {
		return turn
	}
	for index := len(s.turns) - 1; index >= 0; index-- {
		turn := s.turns[index]
		if isRuntimeTurnLive(turn) {
			return turn
		}
	}
	return nil
}

func (s *runtimeSession) newEventLocked(turn *runtimeTurn, stepType string, title string, now time.Time) *runtimeEventRecord {
	if turn == nil {
		return nil
	}
	s.nextEventID++
	step := &runtimeEventRecord{
		ID:        fmt.Sprintf("event-%d", s.nextEventID),
		Type:      stepType,
		Title:     strings.TrimSpace(title),
		Status:    "running",
		StartedAt: now,
	}
	turn.events = append(turn.events, step)
	return step
}

func (s *runtimeSession) ensureCommandEventLocked(turn *runtimeTurn, itemID string, command string, now time.Time) *runtimeEventRecord {
	step, _ := s.ensureCommandEventResultLocked(turn, itemID, command, now)
	return step
}

func (s *runtimeSession) ensureCommandEventResultLocked(turn *runtimeTurn, itemID string, command string, now time.Time) (*runtimeEventRecord, bool) {
	if turn == nil {
		return nil, false
	}
	for _, step := range turn.events {
		if step != nil && step.Type == "command" && strings.TrimSpace(step.ItemID) == strings.TrimSpace(itemID) && strings.TrimSpace(itemID) != "" {
			return step, false
		}
	}
	step := s.newEventLocked(turn, "command", deriveRuntimeEventTitle("command", command), now)
	if step != nil {
		step.ItemID = strings.TrimSpace(itemID)
		step.Preview = summarizeRuntimeEventSummary(command)
		step.Blocks = []RuntimeDetailBlock{{
			Type:     "chatRuntime",
			Title:    "Shell",
			Content:  strings.TrimSpace(command),
			Language: "shell",
			Status:   "running",
		}}
	}
	return step, step != nil
}

func (s *runtimeSession) newSystemEventLocked(turn *runtimeTurn, title string, message string, now time.Time, status string) *runtimeEventRecord {
	step := s.newEventLocked(turn, "log", title, now)
	if step == nil {
		return nil
	}
	step.Status = normalizeFallbackStatus(status)
	step.FinishedAt = now
	step.Preview = summarizeRuntimeEventSummary(message)
	step.Blocks = []RuntimeDetailBlock{{
		Type:    "log",
		Title:   title,
		Content: message,
		Status:  step.Status,
	}}
	step.Searchable = true
	return step
}

func (t *runtimeTurn) runtimeEventByID(eventID string) (*runtimeEventRecord, int) {
	normalized := strings.TrimSpace(eventID)
	numericID := numericRuntimeLookupID(normalized)
	seq := 0
	for _, event := range t.events {
		if event == nil {
			continue
		}
		seq++
		if event.ID == normalized {
			return event, seq
		}
		if numericID > 0 && (numericRuntimeLookupID(event.ID) == numericID || seq == numericID) {
			return event, seq
		}
	}
	return nil, 0
}

func numericRuntimeLookupID(value string) int {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return 0
	}
	if parsed, err := strconv.Atoi(normalized); err == nil && parsed > 0 {
		return parsed
	}
	digits := ""
	for index := len(normalized) - 1; index >= 0; index-- {
		if normalized[index] < '0' || normalized[index] > '9' {
			break
		}
		digits = string(normalized[index]) + digits
	}
	if digits == "" {
		return 0
	}
	parsed, err := strconv.Atoi(digits)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func (t *runtimeTurn) summary(sessionID string) TurnSummary {
	runtimeTraceEvents := make([]RuntimeTraceEvent, 0, len(t.events))
	seq := 0
	for _, event := range t.events {
		if event == nil {
			continue
		}
		seq++
		runtimeTraceEvents = append(runtimeTraceEvents, chatRuntimeRuntimeTraceEvent(sessionID, t.ID, seq, event.summary()))
	}
	return TurnSummary{
		ID:                 t.ID,
		ClientRequestID:    t.ClientRequestID,
		Prompt:             t.Prompt,
		Attachments:        cloneTurnAttachments(t.Attachments),
		Status:             normalizeFallbackStatus(t.Status),
		StartedAt:          t.StartedAt,
		FinishedAt:         t.FinishedAt,
		DurationMS:         durationMilliseconds(t.StartedAt, t.FinishedAt),
		FinalOutput:        t.FinalOutput,
		RuntimeTraceEvents: runtimeTraceEvents,
	}
}

func (t *runtimeTurn) promoteFinalOutput() {
	if len(t.events) == 0 {
		return
	}
	lastMessageIndex := -1
	for index := len(t.events) - 1; index >= 0; index-- {
		if t.events[index] != nil && t.events[index].Type == "message" {
			lastMessageIndex = index
			break
		}
	}
	if lastMessageIndex < 0 {
		return
	}
	messageStep := t.events[lastMessageIndex]
	if messageStep == nil || len(messageStep.Blocks) == 0 {
		return
	}
	t.FinalOutput = strings.TrimSpace(messageStep.Blocks[0].Content)
	t.events = append(append([]*runtimeEventRecord{}, t.events[:lastMessageIndex]...), t.events[lastMessageIndex+1:]...)
}

func (s *runtimeEventRecord) summary() runtimeEventSummary {
	return runtimeEventSummary{
		ID:         s.ID,
		Type:       normalizeRuntimeEventType(s.Type),
		Title:      s.Title,
		Status:     normalizeFallbackStatus(s.Status),
		StartedAt:  s.StartedAt,
		FinishedAt: s.FinishedAt,
		DurationMS: durationMilliseconds(s.StartedAt, s.FinishedAt),
		Preview:    s.Preview,
		HasDetail:  len(s.Blocks) > 0,
		Blocks:     append([]RuntimeDetailBlock{}, s.Blocks...),
	}
}

func (s *runtimeEventRecord) runtimeTraceEventDetail(sessionID string, turnID string, seq int) RuntimeTraceEventDetail {
	summary := s.summary()
	event := chatRuntimeRuntimeTraceEvent(sessionID, turnID, seq, summary)
	event.Blocks = chatRuntimeRuntimeBlocks(summary.Blocks, summary.Preview, event.Kind, false)
	return RuntimeTraceEventDetail{
		TurnID:     turnID,
		Event:      event,
		Blocks:     append([]RuntimeBlock{}, event.Blocks...),
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

func deriveRuntimeEventTitle(stepType string, content string) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	switch normalizeRuntimeEventType(stepType) {
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

func summarizeRuntimeEventSummary(content string) string {
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

func normalizeRuntimeEventStatus(value string, exitCode *int) string {
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

func normalizeRuntimeEventType(stepType string) string {
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

func buildCodexTurnPrompt(prompt string, attachments []preparedTurnAttachment, repository *chatruntimedomain.RepositoryBinding) string {
	files := make([]preparedTurnAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment.IsImage {
			continue
		}
		files = append(files, attachment)
	}
	lines := []string{strings.TrimSpace(prompt)}
	if len(files) > 0 {
		lines = append(lines, "", "Attached files are available in the workspace:")
		for _, attachment := range files {
			lines = append(lines, fmt.Sprintf("- %s (%s): %s", attachment.Name, attachment.ContentType, attachment.PromptPath))
		}
		lines = append(lines, "Read the files directly from disk when needed.")
	}
	if repository != nil && repository.Status == chatruntimedomain.RepositoryPreparationStatusReady {
		branch := strings.TrimSpace(repository.Branch)
		if branch == "" {
			branch = strings.TrimSpace(repository.DefaultBranch)
		}
		lines = append(lines,
			"",
			"Repository context:",
			"- repository: "+strings.TrimSpace(repository.FullName),
			"- path: "+strings.TrimSuffix(strings.TrimSpace(repository.WorkspacePath), "/")+"/",
			"- branch: "+branch,
			"- head: "+strings.TrimSpace(repository.HeadSHA),
			"",
			"This user message is associated with the repository above. Treat it as the default code target when the request relates to repository work.",
		)
	}
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

func cloneChatRuntimeSkillContext(input *execdomain.SkillContext) *execdomain.SkillContext {
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
		return nil, errors.New("chatRuntime workspace is empty")
	}
	dir := filepath.Join(workspaceDir, chatRuntimeTurnAttachmentDirName, sanitizeWorkspaceSegment(turnID))
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

func isFinalCodexAgentMessage(item *codexExecItem) bool {
	if item == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(item.Channel)) {
	case "", "final":
		return true
	default:
		return false
	}
}

func isVisibleCodexAgentProcessMessage(item *codexExecItem) bool {
	if item == nil {
		return false
	}
	return strings.ToLower(strings.TrimSpace(item.Channel)) == "commentary"
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

func isChatRuntimeOutputStream(stream string) bool {
	switch strings.ToLower(strings.TrimSpace(stream)) {
	case "stdout", "stderr":
		return true
	default:
		return false
	}
}

func (s *Service) getOwnedSession(ownerID string, sessionID string) (*runtimeSession, error) {
	ownerID = normalizeChatRuntimeOwnerID(ownerID)
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
	matched := normalizeChatRuntimeOwnerID(item.summary.OwnerID) == ownerID
	item.mu.RUnlock()
	if !matched {
		return nil, ErrSessionNotFound
	}
	return item, nil
}

func (s *Service) getOrRestoreOwnedSession(ownerID string, sessionID string) (*runtimeSession, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, ErrSessionNotFound) {
		return nil, err
	}
	return s.restorePersistedOwnedSession(ownerID, sessionID)
}

func normalizeChatRuntimeOwnerID(ownerID string) string {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return chatRuntimeOwnerID
	}
	return ownerID
}

func (s *Service) countActiveLocked() int {
	total := 0
	for _, item := range s.sessions {
		snapshot := item.snapshot()
		if chatruntimedomain.IsSessionOpenStatus(snapshot.Status) {
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

func (s *Service) newSessionIDLocked() string {
	for attempts := 0; attempts < 16; attempts++ {
		sessionID := generateCompactChatRuntimeSessionID()
		if _, exists := s.sessions[sessionID]; !exists {
			return sessionID
		}
	}
	for {
		sessionID := chatRuntimeSessionIDPrefix + sanitizeCompactSessionIDSeed(s.newID())
		if _, exists := s.sessions[sessionID]; !exists {
			return sessionID
		}
	}
}

func generateCompactChatRuntimeSessionID() string {
	var bytes [chatRuntimeSessionIDLength]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		var builder strings.Builder
		builder.Grow(len(chatRuntimeSessionIDPrefix) + chatRuntimeSessionIDLength)
		builder.WriteString(chatRuntimeSessionIDPrefix)
		for _, value := range bytes {
			builder.WriteByte(chatRuntimeSessionIDAlphabet[int(value)%len(chatRuntimeSessionIDAlphabet)])
		}
		return builder.String()
	}
	return chatRuntimeSessionIDPrefix + sanitizeCompactSessionIDSeed(fmt.Sprintf("%d", time.Now().UTC().UnixNano()))
}

func sanitizeCompactSessionIDSeed(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	builder.Grow(chatRuntimeSessionIDLength)
	for _, char := range normalized {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		}
		if builder.Len() >= chatRuntimeSessionIDLength {
			break
		}
	}
	for builder.Len() < chatRuntimeSessionIDLength {
		builder.WriteByte('0')
	}
	return builder.String()
}

func isCompactChatRuntimeSessionID(value string) bool {
	normalized := strings.TrimSpace(value)
	if len(normalized) != len(chatRuntimeSessionIDPrefix)+chatRuntimeSessionIDLength {
		return false
	}
	if !strings.HasPrefix(normalized, chatRuntimeSessionIDPrefix) {
		return false
	}
	for _, char := range normalized[len(chatRuntimeSessionIDPrefix):] {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= '0' && char <= '9':
		default:
			return false
		}
	}
	return true
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

func resolveRecoveredThreadID(sessionID string, chatRuntimeSessionID string) string {
	threadID := strings.TrimSpace(chatRuntimeSessionID)
	if threadID == "" || threadID == strings.TrimSpace(sessionID) {
		return ""
	}
	return threadID
}

func prepareChatRuntimeCodexRuntime(workspaceDir string, skillContext *execdomain.SkillContext) ([]string, error) {
	materializedSkillContext, skillFiles, err := materializeChatRuntimeSkillContextFiles(skillContext)
	if err != nil {
		return nil, err
	}
	managedFiles := append([]runtimeconfig.ManagedFile{}, skillFiles...)
	managedFiles = append(managedFiles, runtimeconfig.ManagedFile{
		RelativePath: ".alter0/codex-runtime/skills.md",
		Content:      renderChatRuntimeSkillContextMarkdown(materializedSkillContext),
		Mode:         0o644,
	})
	prepared, err := runtimeconfig.Prepare(runtimeconfig.Spec{
		RuntimeHome:      filepath.Join(workspaceDir, chatRuntimeCodexHomeDirName),
		WorkspaceDir:     workspaceDir,
		ManagedFiles:     managedFiles,
		RootInstructions: "- Read `.alter0/codex-runtime/skills.md` before acting. Apply only the skills selected for the current ChatRuntime turn.",
	})
	if err != nil {
		return nil, fmt.Errorf("prepare chatRuntime codex runtime: %w", err)
	}
	return prepared.Env, nil
}

func materializeChatRuntimeSkillContextFiles(skillContext *execdomain.SkillContext) (*execdomain.SkillContext, []runtimeconfig.ManagedFile, error) {
	if skillContext == nil || len(skillContext.Skills) == 0 {
		return skillContext, nil, nil
	}
	refs := make([]runtimeconfig.FileBackedSkillReference, 0, len(skillContext.Skills))
	for i, skill := range skillContext.Skills {
		refs = append(refs, runtimeconfig.FileBackedSkillReference{
			Key:      fmt.Sprintf("%d", i),
			ID:       skill.ID,
			FilePath: skill.FilePath,
		})
	}
	materialized, err := runtimeconfig.MaterializeFileBackedSkillReferences(refs)
	if err != nil {
		return nil, nil, fmt.Errorf("materialize chatRuntime skill files: %w", err)
	}
	updated := cloneChatRuntimeSkillContext(skillContext)
	for i := range updated.Skills {
		if filePath := materialized.FilePaths[fmt.Sprintf("%d", i)]; strings.TrimSpace(filePath) != "" {
			updated.Skills[i].FilePath = filePath
		}
	}
	return updated, materialized.ManagedFiles, nil
}

func renderChatRuntimeSkillContextMarkdown(skillContext *execdomain.SkillContext) string {
	lines := []string{"# Skills", ""}
	if skillContext == nil || len(skillContext.Skills) == 0 {
		lines = append(lines, "No skills selected for this Chat turn.", "")
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
		workspaceDirectoryName,
		workspaceChatRuntimeDirName,
		workspaceSessionsDirName,
		sanitizedSessionID,
	)
	absolute, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("resolve chatRuntime workspace path: %w", err)
	}
	return absolute, nil
}

func resolveSessionWorkspaceDir(baseDir string, sessionID string) (string, error) {
	workspaceDir, err := resolveSessionWorkspacePath(baseDir, sessionID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return "", fmt.Errorf("prepare chatRuntime workspace: %w", err)
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
	s.entries = append(s.entries, chatruntimedomain.Entry{
		Cursor:    s.nextID,
		Stream:    strings.TrimSpace(stream),
		Text:      content,
		CreatedAt: now,
	})
	s.nextID++
	if isChatRuntimeOutputStream(stream) {
		s.summary.LastOutputAt = now
	}
	s.advanceContentUpdatedAtLocked(now)
}

func (s *runtimeSession) markInterruptedLocked(turn *runtimeTurn, now time.Time, message string) {
	reason := strings.TrimSpace(message)
	if reason == "" {
		reason = chatRuntimeHostUnavailableMessage
	}
	summaryText := "chatRuntime interrupted: " + reason
	alreadyRecorded := hasRuntimeTurnSystemEvent(turn, "Interrupted", reason)

	s.summary.Status = chatruntimedomain.SessionStatusInterrupted
	s.summary.ErrorMessage = reason
	s.summary.FinishedAt = now
	s.advanceContentUpdatedAtLocked(now)
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
	for _, step := range turn.events {
		if step == nil || !isRuntimeEventLive(step.Status) {
			continue
		}
		step.Status = "interrupted"
		if step.FinishedAt.IsZero() {
			step.FinishedAt = now
		}
	}
	if !alreadyRecorded {
		s.newSystemEventLocked(turn, "Interrupted", reason, now, "failed")
	}
	turn.promoteFinalOutput()
}

func hasRuntimeTurnSystemEvent(turn *runtimeTurn, title string, message string) bool {
	if turn == nil {
		return false
	}
	targetTitle := strings.TrimSpace(title)
	targetMessage := strings.TrimSpace(message)
	for _, step := range turn.events {
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

func isRuntimeEventLive(status string) bool {
	normalized := strings.TrimSpace(strings.ToLower(status))
	return normalized == "" || normalized == "running" || normalized == "starting"
}

func isRuntimeTurnLive(turn *runtimeTurn) bool {
	if turn == nil {
		return false
	}
	normalized := strings.TrimSpace(strings.ToLower(turn.Status))
	return normalized == "" || normalized == "running" || normalized == "starting" || normalized == "queued" || normalized == "in_progress"
}

func (s *runtimeSession) snapshot() chatruntimedomain.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := s.summary
	snapshot.Repository = cloneRepositoryBinding(s.summary.Repository)
	snapshot.Status = chatruntimedomain.NormalizeSessionStatus(snapshot.Status)
	return snapshot
}
