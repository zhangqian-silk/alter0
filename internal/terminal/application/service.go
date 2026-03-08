package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	sharedapp "alter0/internal/shared/application"
	terminaldomain "alter0/internal/terminal/domain"
)

var (
	ErrSessionOwnerRequired = errors.New("terminal session owner is required")
	ErrSessionNotFound      = errors.New("terminal session not found")
	ErrSessionNotRunning    = errors.New("terminal session is not running")
	ErrSessionLimitReached  = errors.New("terminal session limit reached")
)

type Options struct {
	MaxSessions int
	WorkingDir  string
	Shell       string
	ShellArgs   []string
}

type CreateRequest struct {
	OwnerID string
	Title   string
}

type EntryPage struct {
	Items      []terminaldomain.Entry `json:"items"`
	Cursor     int                    `json:"cursor"`
	NextCursor int                    `json:"next_cursor"`
	HasMore    bool                   `json:"has_more"`
}

type Service struct {
	rootCtx     context.Context
	idGenerator sharedapp.IDGenerator
	logger      *slog.Logger
	options     Options

	mu       sync.RWMutex
	sessions map[string]*runtimeSession
}

type shellCommand struct {
	path  string
	args  []string
	label string
}

func (s *Service) MaxSessions() int {
	return s.options.MaxSessions
}

type runtimeSession struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu           sync.RWMutex
	summary      terminaldomain.Session
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	entries      []terminaldomain.Entry
	nextID       int
	closedByUser bool
}

func NewService(ctx context.Context, idGenerator sharedapp.IDGenerator, logger *slog.Logger, options Options) *Service {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	resolved := normalizeOptions(options)
	service := &Service{
		rootCtx:     ctx,
		idGenerator: idGenerator,
		logger:      logger,
		options:     resolved,
		sessions:    map[string]*runtimeSession{},
	}
	go func() {
		<-ctx.Done()
		service.shutdown()
	}()
	return service
}

func (s *Service) Create(req CreateRequest) (terminaldomain.Session, error) {
	ownerID := strings.TrimSpace(req.OwnerID)
	if ownerID == "" {
		return terminaldomain.Session{}, ErrSessionOwnerRequired
	}

	s.mu.Lock()
	if s.countActiveLocked() >= s.options.MaxSessions {
		s.mu.Unlock()
		return terminaldomain.Session{}, ErrSessionLimitReached
	}
	s.mu.Unlock()

	shell := resolveShellCommand(s.options)
	sessionID := "terminal-" + s.newID()
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = sessionID
	}
	sessionCtx, cancel := context.WithCancel(s.rootCtx)
	cmd := exec.CommandContext(sessionCtx, shell.path, shell.args...)
	cmd.Dir = s.options.WorkingDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return terminaldomain.Session{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return terminaldomain.Session{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return terminaldomain.Session{}, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return terminaldomain.Session{}, err
	}

	now := time.Now().UTC()
	session := &runtimeSession{
		ctx:    sessionCtx,
		cancel: cancel,
		cmd:    cmd,
		stdin:  stdin,
		summary: terminaldomain.Session{
			ID:         sessionID,
			OwnerID:    ownerID,
			Title:      title,
			Shell:      shell.label,
			WorkingDir: s.options.WorkingDir,
			Status:     terminaldomain.SessionStatusRunning,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		entries: []terminaldomain.Entry{},
	}
	session.appendEntry("system", fmt.Sprintf("shell started: %s", session.summary.Shell))

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	go s.readOutput(session, "stdout", stdout)
	go s.readOutput(session, "stderr", stderr)
	go s.waitForExit(session)

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
		return session.LastOutputAt.UTC()
	}
	if !session.CreatedAt.IsZero() {
		return session.CreatedAt.UTC()
	}
	if !session.UpdatedAt.IsZero() {
		return session.UpdatedAt.UTC()
	}
	return time.Time{}
}

func isTerminalOutputStream(stream string) bool {
	switch strings.ToLower(strings.TrimSpace(stream)) {
	case "stdout", "stderr":
		return true
	default:
		return false
	}
}

func (s *Service) Get(ownerID string, sessionID string) (terminaldomain.Session, bool) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return terminaldomain.Session{}, false
	}
	return item.snapshot(), true
}

func (s *Service) ListEntries(ownerID string, sessionID string, cursor int, limit int) (EntryPage, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return EntryPage{}, err
	}
	if cursor < 0 {
		cursor = 0
	}
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}

	item.mu.RLock()
	defer item.mu.RUnlock()

	total := len(item.entries)
	if cursor > total {
		cursor = total
	}
	nextCursor := cursor + limit
	if nextCursor > total {
		nextCursor = total
	}
	items := make([]terminaldomain.Entry, 0, nextCursor-cursor)
	items = append(items, item.entries[cursor:nextCursor]...)
	return EntryPage{
		Items:      items,
		Cursor:     cursor,
		NextCursor: nextCursor,
		HasMore:    nextCursor < total,
	}, nil
}

func (s *Service) Input(ownerID string, sessionID string, input string) (terminaldomain.Session, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return terminaldomain.Session{}, err
	}

	text := strings.TrimRight(input, "\r\n")
	if strings.TrimSpace(text) == "" {
		return terminaldomain.Session{}, errors.New("terminal input is required")
	}

	item.mu.Lock()
	defer item.mu.Unlock()

	if item.summary.Status != terminaldomain.SessionStatusRunning {
		return terminaldomain.Session{}, ErrSessionNotRunning
	}
	if item.stdin == nil {
		return terminaldomain.Session{}, ErrSessionNotRunning
	}
	if _, err := io.WriteString(item.stdin, text+"\n"); err != nil {
		item.summary.Status = terminaldomain.SessionStatusInterrupted
		item.summary.UpdatedAt = time.Now().UTC()
		item.summary.ErrorMessage = strings.TrimSpace(err.Error())
		item.appendEntryLocked("system", "terminal input rejected: "+strings.TrimSpace(err.Error()))
		return terminaldomain.Session{}, ErrSessionNotRunning
	}
	item.appendEntryLocked("input", text)
	return item.summary, nil
}

func (s *Service) Close(ownerID string, sessionID string) (terminaldomain.Session, error) {
	item, err := s.getOwnedSession(ownerID, sessionID)
	if err != nil {
		return terminaldomain.Session{}, err
	}

	item.mu.Lock()
	defer item.mu.Unlock()

	switch item.summary.Status {
	case terminaldomain.SessionStatusRunning, terminaldomain.SessionStatusStarting:
	default:
		return item.summary, nil
	}

	now := time.Now().UTC()
	item.closedByUser = true
	item.summary.Status = terminaldomain.SessionStatusExited
	item.summary.UpdatedAt = now
	item.summary.FinishedAt = now
	item.summary.ErrorMessage = ""
	item.appendEntryLocked("system", "terminal closed by user")
	if item.stdin != nil {
		_ = item.stdin.Close()
		item.stdin = nil
	}
	item.cancel()
	if item.cmd != nil && item.cmd.Process != nil {
		_ = item.cmd.Process.Kill()
	}
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
		item.cancel()
		if item.cmd != nil && item.cmd.Process != nil {
			_ = item.cmd.Process.Kill()
		}
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

func (s *Service) readOutput(item *runtimeSession, stream string, reader io.ReadCloser) {
	defer reader.Close()
	buffer := make([]byte, 4096)
	for {
		read, err := reader.Read(buffer)
		if read > 0 {
			text := normalizeChunk(string(buffer[:read]))
			if strings.TrimSpace(text) != "" || strings.Contains(text, "\n") {
				item.appendEntry(stream, text)
			}
		}
		if err != nil {
			if err != io.EOF && s.logger != nil {
				s.logger.Warn("terminal output reader stopped",
					slog.String("session_id", item.snapshot().ID),
					slog.String("stream", stream),
					slog.String("error", err.Error()),
				)
			}
			return
		}
	}
}

func (s *Service) waitForExit(item *runtimeSession) {
	err := item.cmd.Wait()
	finishedAt := time.Now().UTC()
	exitCode, hasExitCode := resolveExitCode(item.cmd.ProcessState, err)

	item.mu.Lock()
	defer item.mu.Unlock()

	if item.summary.Status == terminaldomain.SessionStatusInterrupted {
		item.summary.UpdatedAt = finishedAt
		item.summary.FinishedAt = finishedAt
		return
	}

	item.summary.UpdatedAt = finishedAt
	item.summary.FinishedAt = finishedAt
	if hasExitCode {
		item.summary.ExitCode = &exitCode
	}

	if item.closedByUser {
		item.summary.Status = terminaldomain.SessionStatusExited
		item.summary.ErrorMessage = ""
		return
	}

	switch {
	case errors.Is(item.ctx.Err(), context.Canceled):
		item.summary.Status = terminaldomain.SessionStatusInterrupted
		if item.summary.ErrorMessage == "" {
			item.summary.ErrorMessage = "terminal host unavailable"
		}
		item.appendEntryLocked("system", "terminal interrupted: terminal host unavailable")
	case err != nil && !hasExitCode:
		item.summary.Status = terminaldomain.SessionStatusFailed
		item.summary.ErrorMessage = strings.TrimSpace(err.Error())
		item.appendEntryLocked("system", "terminal failed: "+item.summary.ErrorMessage)
	default:
		item.summary.Status = terminaldomain.SessionStatusExited
		if hasExitCode {
			item.appendEntryLocked("system", fmt.Sprintf("terminal exited with code %d", exitCode))
		} else {
			item.appendEntryLocked("system", "terminal exited")
		}
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
	args := make([]string, 0, len(options.ShellArgs))
	for _, item := range options.ShellArgs {
		if value := strings.TrimSpace(item); value != "" {
			args = append(args, value)
		}
	}
	options.ShellArgs = args
	return options
}

func resolveShellCommand(options Options) shellCommand {
	return resolveShellCommandForOS(runtime.GOOS, options, exec.LookPath)
}

func resolveShellCommandForOS(goos string, options Options, lookPath func(string) (string, error)) shellCommand {
	if options.Shell != "" {
		path := options.Shell
		args := append([]string{}, options.ShellArgs...)
		if goos == "windows" {
			return normalizeWindowsShellCommand(path, args)
		}
		return shellCommand{
			path:  path,
			args:  args,
			label: buildShellLabel(path, args),
		}
	}
	if goos == "windows" {
		return resolveDefaultWindowsShell(lookPath)
	}
	if value := strings.TrimSpace(os.Getenv("SHELL")); value != "" {
		return shellCommand{
			path:  value,
			args:  []string{},
			label: value,
		}
	}
	return shellCommand{
		path:  "/bin/sh",
		args:  []string{},
		label: "/bin/sh",
	}
}

func resolveDefaultWindowsShell(lookPath func(string) (string, error)) shellCommand {
	defaultArgs := []string{
		"-NoLogo",
		"-NoProfile",
		"-NoExit",
		"-Command",
		"[Console]::InputEncoding=[System.Text.UTF8Encoding]::new($false);[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false);$OutputEncoding=[Console]::OutputEncoding;chcp.com 65001 > $null",
	}

	if lookPath != nil {
		if resolved, err := lookPath("powershell.exe"); err == nil && strings.TrimSpace(resolved) != "" {
			return shellCommand{
				path:  resolved,
				args:  append([]string{}, defaultArgs...),
				label: resolved,
			}
		}
	}

	if value := strings.TrimSpace(os.Getenv("COMSPEC")); value != "" {
		return shellCommand{
			path:  value,
			args:  []string{},
			label: value,
		}
	}

	return shellCommand{
		path:  "powershell.exe",
		args:  append([]string{}, defaultArgs...),
		label: filepath.Clean("powershell.exe"),
	}
}

func normalizeWindowsShellCommand(path string, args []string) shellCommand {
	switch {
	case isWindowsPowerShellCommand(path):
		if len(args) == 0 {
			return shellCommand{
				path:  path,
				args:  []string{"-NoLogo", "-NoProfile", "-NoExit", "-Command", "[Console]::InputEncoding=[System.Text.UTF8Encoding]::new($false);[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false);$OutputEncoding=[Console]::OutputEncoding;chcp.com 65001 > $null"},
				label: path,
			}
		}
	case isWindowsCmdCommand(path):
		if len(args) == 0 {
			return shellCommand{
				path:  path,
				args:  []string{"/D", "/K", "chcp.com 65001>nul"},
				label: path,
			}
		}
	}
	return shellCommand{
		path:  path,
		args:  args,
		label: buildShellLabel(path, args),
	}
}

func isWindowsPowerShellCommand(path string) bool {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(path)))
	return base == "powershell.exe" || base == "powershell" || base == "pwsh.exe" || base == "pwsh"
}

func isWindowsCmdCommand(path string) bool {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(path)))
	return base == "cmd.exe" || base == "cmd"
}

func buildShellLabel(shellPath string, args []string) string {
	parts := []string{shellPath}
	parts = append(parts, args...)
	return strings.Join(parts, " ")
}

func normalizeChunk(value string) string {
	text := strings.ReplaceAll(value, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func resolveExitCode(state *os.ProcessState, waitErr error) (int, bool) {
	if state != nil {
		return state.ExitCode(), true
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) && exitErr.ProcessState != nil {
		return exitErr.ProcessState.ExitCode(), true
	}
	return 0, false
}

func (s *runtimeSession) appendEntry(stream string, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appendEntryLocked(stream, text)
}

func (s *runtimeSession) appendEntryLocked(stream string, text string) {
	now := time.Now().UTC()
	s.entries = append(s.entries, terminaldomain.Entry{
		Cursor:    s.nextID,
		Stream:    strings.TrimSpace(stream),
		Text:      text,
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
