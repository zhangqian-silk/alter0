package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	chatruntimedomain "alter0/internal/chatruntime/domain"
)

const chatRuntimeStateDirectoryName = "state"

type persistedSessionRecord struct {
	Summary      chatruntimedomain.Session `json:"summary"`
	TitleManual  *bool                     `json:"title_manual,omitempty"`
	TitleAuto    *bool                     `json:"title_auto,omitempty"`
	TitleScore   int                       `json:"title_score,omitempty"`
	Entries      []chatruntimedomain.Entry `json:"entries,omitempty"`
	Turns        []persistedTurnRecord     `json:"turns,omitempty"`
	NextID       int                       `json:"next_id,omitempty"`
	NextTurnID   int                       `json:"next_turn_id,omitempty"`
	NextEventID  int                       `json:"next_event_id,omitempty"`
	ThreadID     string                    `json:"thread_id,omitempty"`
	ClosedByUser bool                      `json:"closed_by_user,omitempty"`
}

type persistedTurnRecord struct {
	ID            string                        `json:"id"`
	Prompt        string                        `json:"prompt,omitempty"`
	Attachments   []TurnAttachment              `json:"attachments,omitempty"`
	Status        string                        `json:"status,omitempty"`
	StartedAt     time.Time                     `json:"started_at,omitempty"`
	FinishedAt    time.Time                     `json:"finished_at,omitempty"`
	FinalOutput   string                        `json:"final_output,omitempty"`
	RuntimeEvents []persistedRuntimeEventRecord `json:"runtime_events,omitempty"`
}

type persistedRuntimeEventRecord struct {
	ID         string               `json:"id"`
	ItemID     string               `json:"item_id,omitempty"`
	Type       string               `json:"type,omitempty"`
	Title      string               `json:"title,omitempty"`
	Status     string               `json:"status,omitempty"`
	Preview    string               `json:"preview,omitempty"`
	StartedAt  time.Time            `json:"started_at,omitempty"`
	FinishedAt time.Time            `json:"finished_at,omitempty"`
	Blocks     []RuntimeDetailBlock `json:"blocks,omitempty"`
	Searchable bool                 `json:"searchable,omitempty"`
}

func decodePersistedSessionRecord(data []byte) (persistedSessionRecord, error) {
	record := persistedSessionRecord{}
	if err := json.Unmarshal(data, &record); err != nil {
		return persistedSessionRecord{}, err
	}
	return record, nil
}

func (s *Service) loadPersistedSessions() {
	dir, err := resolveChatRuntimeSessionStateDir(s.options.WorkingDir)
	if err != nil {
		s.logger.Warn("load chatRuntime session store failed", "error", err.Error())
		return
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		s.logger.Warn("read chatRuntime session store failed", "error", err.Error())
		return
	}

	now := time.Now().UTC()
	s.mu.Lock()
	for _, entry := range items {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		recordPath := filepath.Join(dir, entry.Name())
		data, readErr := os.ReadFile(recordPath)
		if readErr != nil {
			s.logger.Warn("read chatRuntime session record failed", "path", recordPath, "error", readErr.Error())
			continue
		}
		record, err := decodePersistedSessionRecord(data)
		if err != nil {
			s.logger.Warn("decode chatRuntime session record failed", "path", recordPath, "error", err.Error())
			continue
		}
		session := restorePersistedSession(record, now, s.options.WorkingDir)
		if session == nil {
			continue
		}
		s.sessions[session.summary.ID] = session
	}
	s.mu.Unlock()
}

func (s *Service) syncMissingPersistedSessions() {
	dir, err := resolveChatRuntimeSessionStateDir(s.options.WorkingDir)
	if err != nil {
		s.logger.Warn("sync chatRuntime session store failed", "error", err.Error())
		return
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		s.logger.Warn("read chatRuntime session store failed", "error", err.Error())
		return
	}

	now := time.Now().UTC()
	for _, entry := range items {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		recordPath := filepath.Join(dir, entry.Name())
		data, readErr := os.ReadFile(recordPath)
		if readErr != nil {
			s.logger.Warn("read chatRuntime session record failed", "path", recordPath, "error", readErr.Error())
			continue
		}
		record, err := decodePersistedSessionRecord(data)
		if err != nil {
			s.logger.Warn("decode chatRuntime session record failed", "path", recordPath, "error", err.Error())
			continue
		}
		session := restorePersistedSession(record, now, s.options.WorkingDir)
		if session == nil {
			continue
		}
		s.mu.Lock()
		_, exists := s.sessions[session.summary.ID]
		if !exists {
			s.sessions[session.summary.ID] = session
		}
		s.mu.Unlock()
	}
}

func (s *Service) persistSession(item *runtimeSession) {
	if item == nil {
		return
	}
	record, deleted := snapshotPersistedSession(item)
	if deleted {
		return
	}
	if strings.TrimSpace(record.Summary.ID) == "" {
		return
	}
	dir, err := resolveChatRuntimeSessionStateDir(s.options.WorkingDir)
	if err != nil {
		s.logger.Warn("prepare chatRuntime session store failed", "session_id", record.Summary.ID, "error", err.Error())
		return
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		s.logger.Warn("encode chatRuntime session record failed", "session_id", record.Summary.ID, "error", err.Error())
		return
	}
	path := filepath.Join(dir, sanitizeWorkspaceSegment(record.Summary.ID)+".json")
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		s.logger.Warn("write chatRuntime session record failed", "path", tempPath, "error", err.Error())
		return
	}
	if isRuntimeSessionDeleted(item) {
		_ = os.Remove(tempPath)
		return
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		s.logger.Warn("commit chatRuntime session record failed", "path", path, "error", err.Error())
		return
	}
	s.hookMu.RLock()
	hook := s.updateHook
	s.hookMu.RUnlock()
	if hook != nil {
		hook(record.Summary.OwnerID, record.Summary.ID, record.Summary)
	}
}

func (s *Service) restorePersistedOwnedSession(ownerID string, sessionID string) (*runtimeSession, error) {
	ownerID = normalizeChatRuntimeOwnerID(ownerID)
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, ErrSessionNotFound
	}

	s.mu.RLock()
	existing, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if ok {
		existing.mu.RLock()
		matched := normalizeChatRuntimeOwnerID(existing.summary.OwnerID) == ownerID
		existing.mu.RUnlock()
		if !matched {
			return nil, ErrSessionNotFound
		}
		return existing, nil
	}

	dir, err := resolveChatRuntimeSessionStateDir(s.options.WorkingDir)
	if err != nil {
		s.logger.Warn("resolve chatRuntime session state dir failed", "session_id", sessionID, "error", err.Error())
		return nil, ErrSessionNotFound
	}
	path := filepath.Join(dir, sanitizeWorkspaceSegment(sessionID)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			s.logger.Warn("read chatRuntime session record failed", "path", path, "error", err.Error())
		}
		return nil, ErrSessionNotFound
	}
	record, err := decodePersistedSessionRecord(data)
	if err != nil {
		s.logger.Warn("decode chatRuntime session record failed", "path", path, "error", err.Error())
		return nil, ErrSessionNotFound
	}
	session := restorePersistedSession(record, time.Now().UTC(), s.options.WorkingDir)
	if session == nil || strings.TrimSpace(session.summary.OwnerID) != ownerID {
		return nil, ErrSessionNotFound
	}

	s.mu.Lock()
	if existing, ok := s.sessions[sessionID]; ok {
		existing.mu.RLock()
		matched := normalizeChatRuntimeOwnerID(existing.summary.OwnerID) == ownerID
		existing.mu.RUnlock()
		if !matched {
			s.mu.Unlock()
			return nil, ErrSessionNotFound
		}
		s.mu.Unlock()
		return existing, nil
	}
	s.sessions[sessionID] = session
	s.mu.Unlock()
	return session, nil
}

func snapshotPersistedSession(item *runtimeSession) (persistedSessionRecord, bool) {
	item.mu.RLock()
	defer item.mu.RUnlock()
	if item.deleted {
		return persistedSessionRecord{}, true
	}

	record := persistedSessionRecord{
		Summary:      item.summary,
		TitleManual:  boolPointer(item.titleManual),
		TitleAuto:    boolPointer(item.titleAuto),
		TitleScore:   item.titleScore,
		Entries:      append([]chatruntimedomain.Entry{}, item.entries...),
		NextID:       item.nextID,
		NextTurnID:   item.nextTurnID,
		NextEventID:  item.nextEventID,
		ThreadID:     strings.TrimSpace(item.threadID),
		ClosedByUser: item.closedByUser,
		Turns:        make([]persistedTurnRecord, 0, len(item.turns)),
	}
	for _, turn := range item.turns {
		if turn == nil {
			continue
		}
		persistedTurn := persistedTurnRecord{
			ID:            turn.ID,
			Prompt:        turn.Prompt,
			Attachments:   cloneTurnAttachments(turn.Attachments),
			Status:        turn.Status,
			StartedAt:     turn.StartedAt,
			FinishedAt:    turn.FinishedAt,
			FinalOutput:   turn.FinalOutput,
			RuntimeEvents: make([]persistedRuntimeEventRecord, 0, len(turn.events)),
		}
		for _, step := range turn.events {
			if step == nil {
				continue
			}
			persistedTurn.RuntimeEvents = append(persistedTurn.RuntimeEvents, persistedRuntimeEventRecord{
				ID:         step.ID,
				ItemID:     step.ItemID,
				Type:       step.Type,
				Title:      step.Title,
				Status:     step.Status,
				Preview:    step.Preview,
				StartedAt:  step.StartedAt,
				FinishedAt: step.FinishedAt,
				Blocks:     append([]RuntimeDetailBlock{}, step.Blocks...),
				Searchable: step.Searchable,
			})
		}
		record.Turns = append(record.Turns, persistedTurn)
	}
	return record, false
}

func isRuntimeSessionDeleted(item *runtimeSession) bool {
	if item == nil {
		return false
	}
	item.mu.RLock()
	defer item.mu.RUnlock()
	return item.deleted
}

func restorePersistedSession(record persistedSessionRecord, now time.Time, baseDir string) *runtimeSession {
	sessionID := strings.TrimSpace(record.Summary.ID)
	if sessionID == "" {
		return nil
	}
	summary := record.Summary
	summary.OwnerID = normalizeChatRuntimeOwnerID(summary.OwnerID)
	summary.Status = chatruntimedomain.NormalizeSessionStatus(summary.Status)
	if workspaceDir, err := resolveSessionWorkspacePath(baseDir, sessionID); err == nil {
		summary.WorkingDir = workspaceDir
	}
	if strings.TrimSpace(summary.RuntimeSessionID) == "" {
		summary.RuntimeSessionID = sessionID
	}
	threadID := strings.TrimSpace(record.ThreadID)
	if threadID == "" {
		threadID = resolveRecoveredThreadID(sessionID, summary.RuntimeSessionID)
	}
	session := &runtimeSession{
		summary:      summary,
		titleScore:   record.TitleScore,
		entries:      append([]chatruntimedomain.Entry{}, record.Entries...),
		nextID:       record.NextID,
		nextTurnID:   record.NextTurnID,
		nextEventID:  record.NextEventID,
		threadID:     threadID,
		closedByUser: record.ClosedByUser,
		turns:        make([]*runtimeTurn, 0, len(record.Turns)),
	}
	if record.TitleManual != nil {
		session.titleManual = *record.TitleManual
	}
	if record.TitleAuto != nil {
		session.titleAuto = *record.TitleAuto
		if record.TitleManual == nil {
			session.titleManual = inferManualSessionTitleState(summary.Title, sessionID, session.titleAuto, session.titleScore)
		}
	} else {
		session.titleAuto, session.titleScore = inferAutoSessionTitleState(summary.Title, sessionID)
	}
	for _, turnRecord := range record.Turns {
		turn := &runtimeTurn{
			ID:          turnRecord.ID,
			Prompt:      turnRecord.Prompt,
			Attachments: cloneTurnAttachments(turnRecord.Attachments),
			Status:      turnRecord.Status,
			StartedAt:   turnRecord.StartedAt,
			FinishedAt:  turnRecord.FinishedAt,
			FinalOutput: turnRecord.FinalOutput,
			events:      make([]*runtimeEventRecord, 0, len(turnRecord.RuntimeEvents)),
		}
		for _, stepRecord := range turnRecord.RuntimeEvents {
			turn.events = append(turn.events, &runtimeEventRecord{
				ID:         stepRecord.ID,
				ItemID:     stepRecord.ItemID,
				Type:       stepRecord.Type,
				Title:      stepRecord.Title,
				Status:     stepRecord.Status,
				Preview:    stepRecord.Preview,
				StartedAt:  stepRecord.StartedAt,
				FinishedAt: stepRecord.FinishedAt,
				Blocks:     append([]RuntimeDetailBlock{}, stepRecord.Blocks...),
				Searchable: stepRecord.Searchable,
			})
		}
		session.turns = append(session.turns, turn)
	}
	if title, titleAuto, titleScore, changed := repairSupplementalConstraintAutoSessionTitle(
		session.summary.Title,
		session.titleManual,
		session.turns,
		sessionID,
		64,
	); changed {
		session.summary.Title = title
		session.titleAuto = titleAuto
		session.titleScore = titleScore
		session.titleManual = false
	}
	normalizeRestoredSessionState(session, now)
	return session
}

func boolPointer(value bool) *bool {
	return &value
}

func normalizeRestoredSessionState(session *runtimeSession, now time.Time) {
	if session == nil {
		return
	}
	if session.nextID <= 0 {
		nextID := 0
		for _, entry := range session.entries {
			if entry.Cursor >= nextID {
				nextID = entry.Cursor + 1
			}
		}
		session.nextID = nextID
	}
	liveStatus := chatruntimedomain.NormalizeSessionStatus(session.summary.Status) == chatruntimedomain.SessionStatusBusy
	for _, turn := range session.turns {
		if turn == nil {
			continue
		}
		if turn.ID != "" && parseRuntimeOrdinal(turn.ID, "turn-") > session.nextTurnID {
			session.nextTurnID = parseRuntimeOrdinal(turn.ID, "turn-")
		}
		if turn.Status == "running" {
			turn.Status = "interrupted"
			if turn.FinishedAt.IsZero() {
				turn.FinishedAt = now
			}
			liveStatus = true
		}
		for _, step := range turn.events {
			if step == nil {
				continue
			}
			if eventOrdinal := maxRuntimeOrdinal(step.ID, "event-", "step-"); eventOrdinal > session.nextEventID {
				session.nextEventID = eventOrdinal
			}
			if step.Status == "running" {
				step.Status = "interrupted"
				if step.FinishedAt.IsZero() {
					step.FinishedAt = now
				}
			}
		}
	}
	if liveStatus {
		if session.closedByUser {
			session.summary.Status = chatruntimedomain.SessionStatusExited
		} else {
			session.summary.Status = chatruntimedomain.SessionStatusInterrupted
			if strings.TrimSpace(session.summary.ErrorMessage) == "" {
				session.summary.ErrorMessage = "codex runtime exited"
			}
		}
		if session.summary.FinishedAt.IsZero() {
			session.summary.FinishedAt = now
		}
		if session.summary.UpdatedAt.Before(now) {
			session.summary.UpdatedAt = now
		}
	}
	session.summary.Status = chatruntimedomain.NormalizeSessionStatus(session.summary.Status)
	session.turnRunning = false
	session.turnCancel = nil
	session.activeTurnID = ""
}

func parseRuntimeOrdinal(value string, prefix string) int {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), prefix)
	if trimmed == "" || trimmed == value {
		return 0
	}
	var ordinal int
	_, err := fmt.Sscanf(trimmed, "%d", &ordinal)
	if err != nil || ordinal < 0 {
		return 0
	}
	return ordinal
}

func maxRuntimeOrdinal(value string, prefixes ...string) int {
	maxOrdinal := 0
	for _, prefix := range prefixes {
		if ordinal := parseRuntimeOrdinal(value, prefix); ordinal > maxOrdinal {
			maxOrdinal = ordinal
		}
	}
	return maxOrdinal
}

func resolveChatRuntimeSessionStateDir(baseDir string) (string, error) {
	dir, err := resolveChatRuntimeSessionStatePath(baseDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("prepare chatRuntime session store: %w", err)
	}
	return dir, nil
}

func resolveChatRuntimeSessionStatePath(baseDir string) (string, error) {
	root := strings.TrimSpace(baseDir)
	if root == "" {
		root = "."
	}
	dir := filepath.Join(
		root,
		chatRuntimeStateDirectoryName,
		workspaceChatRuntimeDirName,
		workspaceSessionsDirName,
	)
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve chatRuntime session store path: %w", err)
	}
	return absolute, nil
}

func resolveChatRuntimeSessionStateFilePath(baseDir string, sessionID string) (string, error) {
	dir, err := resolveChatRuntimeSessionStatePath(baseDir)
	if err != nil {
		return "", err
	}
	sanitizedSessionID := sanitizeWorkspaceSegment(sessionID)
	if sanitizedSessionID == "" {
		return "", ErrSessionRecoverIDRequired
	}
	return filepath.Join(dir, sanitizedSessionID+".json"), nil
}

func removeChatRuntimeSessionStateFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove chatRuntime session record: %w", err)
	}
	return nil
}
