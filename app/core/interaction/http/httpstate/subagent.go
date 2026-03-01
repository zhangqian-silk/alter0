package httpstate

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	SubagentModeRun     = "run"
	SubagentModeSession = "session"
)

var (
	ErrSubagentNotFound        = errors.New("subagent not found")
	ErrSubagentSessionClosed   = errors.New("session is not active")
	ErrSubagentRunAlreadyDone  = errors.New("run mode already finished")
	ErrSubagentModeUnsupported = errors.New("unsupported subagent mode")
)

type SubagentAnnounceTarget struct {
	ChannelID string
	To        string
}

type SubagentTurnRecord struct {
	Turn   int
	Status string
	Result *TaskResult
	Error  string
}

type SubagentRecord struct {
	ID        string
	Mode      string
	Status    string
	UserID    string
	AgentID   string
	Announce  *SubagentAnnounceTarget
	CreatedAt time.Time
	UpdatedAt time.Time
	Result    *TaskResult
	Error     string
	Turns     []SubagentTurnRecord
}

type SubagentTurnStart struct {
	ID       string
	Mode     string
	Turn     int
	UserID   string
	AgentID  string
	Announce *SubagentAnnounceTarget
}

type SubagentStore struct {
	mu      sync.Mutex
	records map[string]*SubagentRecord
}

func NewSubagentStore() *SubagentStore {
	return &SubagentStore{records: map[string]*SubagentRecord{}}
}

func (s *SubagentStore) Create(id string, mode string, userID string, agentID string, announce *SubagentAnnounceTarget, now time.Time) (SubagentRecord, error) {
	if mode != SubagentModeRun && mode != SubagentModeSession {
		return SubagentRecord{}, ErrSubagentModeUnsupported
	}
	if strings.TrimSpace(userID) == "" {
		userID = "local_user"
	}
	rec := &SubagentRecord{
		ID:        id,
		Mode:      mode,
		Status:    "pending",
		UserID:    strings.TrimSpace(userID),
		AgentID:   strings.TrimSpace(agentID),
		Announce:  copyAnnounceTarget(announce),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if mode == SubagentModeSession {
		rec.Status = "active"
	}

	s.mu.Lock()
	s.records[id] = rec
	s.mu.Unlock()
	return copySubagentRecord(rec), nil
}

func (s *SubagentStore) List(statusFilter string, modeFilter string, limit int) []SubagentRecord {
	s.mu.Lock()
	items := make([]SubagentRecord, 0, len(s.records))
	for _, rec := range s.records {
		if statusFilter != "" && statusFilter != "all" && rec.Status != statusFilter {
			continue
		}
		if modeFilter != "" && modeFilter != "all" && rec.Mode != modeFilter {
			continue
		}
		items = append(items, copySubagentRecord(rec))
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (s *SubagentStore) Get(id string) (SubagentRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[id]
	if !ok {
		return SubagentRecord{}, false
	}
	return copySubagentRecord(rec), true
}

func (s *SubagentStore) CloseSession(id string, now time.Time) (SubagentRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[id]
	if !ok {
		return SubagentRecord{}, ErrSubagentNotFound
	}
	if rec.Mode != SubagentModeSession {
		return SubagentRecord{}, ErrSubagentModeUnsupported
	}
	if rec.Status == "active" || rec.Status == "pending" {
		rec.Status = "closed"
		rec.UpdatedAt = now
	}
	return copySubagentRecord(rec), nil
}

func (s *SubagentStore) BeginTurn(id string, now time.Time) (SubagentTurnStart, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[id]
	if !ok {
		return SubagentTurnStart{}, ErrSubagentNotFound
	}
	if rec.Mode == SubagentModeSession && rec.Status != "active" {
		return SubagentTurnStart{}, ErrSubagentSessionClosed
	}
	if rec.Mode == SubagentModeRun && len(rec.Turns) > 0 {
		return SubagentTurnStart{}, ErrSubagentRunAlreadyDone
	}
	turn := len(rec.Turns) + 1
	rec.Status = "running"
	rec.UpdatedAt = now
	return SubagentTurnStart{
		ID:       rec.ID,
		Mode:     rec.Mode,
		Turn:     turn,
		UserID:   rec.UserID,
		AgentID:  rec.AgentID,
		Announce: copyAnnounceTarget(rec.Announce),
	}, nil
}

func (s *SubagentStore) FinishTurn(id string, turn int, status string, result *TaskResult, errMsg string, now time.Time) (SubagentRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[id]
	if !ok {
		return SubagentRecord{}, ErrSubagentNotFound
	}

	rec.UpdatedAt = now
	rec.Error = strings.TrimSpace(errMsg)
	rec.Result = copyTaskResult(result)
	rec.Turns = append(rec.Turns, SubagentTurnRecord{
		Turn:   turn,
		Status: status,
		Result: copyTaskResult(result),
		Error:  strings.TrimSpace(errMsg),
	})

	if rec.Mode == SubagentModeRun {
		rec.Status = status
	} else {
		rec.Status = "active"
	}

	return copySubagentRecord(rec), nil
}

func copySubagentRecord(src *SubagentRecord) SubagentRecord {
	if src == nil {
		return SubagentRecord{}
	}
	cp := *src
	cp.Announce = copyAnnounceTarget(src.Announce)
	cp.Result = copyTaskResult(src.Result)
	cp.Turns = make([]SubagentTurnRecord, 0, len(src.Turns))
	for _, turn := range src.Turns {
		cp.Turns = append(cp.Turns, SubagentTurnRecord{
			Turn:   turn.Turn,
			Status: turn.Status,
			Result: copyTaskResult(turn.Result),
			Error:  turn.Error,
		})
	}
	return cp
}

func copyTaskResult(src *TaskResult) *TaskResult {
	if src == nil {
		return nil
	}
	cp := *src
	return &cp
}

func copyAnnounceTarget(src *SubagentAnnounceTarget) *SubagentAnnounceTarget {
	if src == nil {
		return nil
	}
	cp := *src
	return &cp
}
