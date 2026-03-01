package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	KindAt   = "at"
	KindCron = "cron"

	StatusActive    = "active"
	StatusPaused    = "paused"
	StatusCanceled  = "canceled"
	StatusCompleted = "completed"
	StatusFailed    = "failed"

	ModeDirect    = "direct"
	ModeAgentTurn = "agent_turn"
)

type DeliveryPayload struct {
	Mode      string `json:"mode"`
	ChannelID string `json:"channel_id"`
	To        string `json:"to"`
	Content   string `json:"content"`
	AgentID   string `json:"agent_id,omitempty"`
}

type Job struct {
	ID             string
	Kind           string
	Spec           string
	Status         string
	IdempotencyKey string
	MaxRetries     int
	Attempt        int
	RetryDelaySec  int
	Payload        DeliveryPayload
	LastRunAt      time.Time
	NextRunAt      time.Time
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RunRecord struct {
	ID         int64     `json:"id"`
	ScheduleID string    `json:"schedule_id"`
	Attempt    int       `json:"attempt"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
	RunAt      time.Time `json:"run_at"`
	FinishedAt time.Time `json:"finished_at"`
}

type CreateRequest struct {
	Kind           string          `json:"kind"`
	At             string          `json:"at,omitempty"`
	Cron           string          `json:"cron,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	MaxRetries     int             `json:"max_retries,omitempty"`
	RetryDelaySec  int             `json:"retry_delay_sec,omitempty"`
	Payload        DeliveryPayload `json:"payload"`
}

type Store struct {
	conn *sql.DB
}

func NewStore(conn *sql.DB) (*Store, error) {
	if conn == nil {
		return nil, errors.New("schedule store: db connection is required")
	}
	store := &Store{conn: conn}
	if err := store.initSchema(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schedules (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			spec TEXT NOT NULL,
			status TEXT NOT NULL,
			idempotency_key TEXT,
			max_retries INTEGER NOT NULL DEFAULT 0,
			attempt INTEGER NOT NULL DEFAULT 0,
			retry_delay_sec INTEGER NOT NULL DEFAULT 30,
			payload_json TEXT NOT NULL,
			last_run_at INTEGER,
			next_run_at INTEGER,
			last_error TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_schedules_idempotency_key ON schedules(idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key != ''`,
		`CREATE INDEX IF NOT EXISTS idx_schedules_status_next_run ON schedules(status, next_run_at)`,
		`CREATE TABLE IF NOT EXISTS schedule_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			schedule_id TEXT NOT NULL,
			attempt INTEGER NOT NULL,
			status TEXT NOT NULL,
			error TEXT,
			run_at INTEGER NOT NULL,
			finished_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_schedule_runs_schedule_run_at ON schedule_runs(schedule_id, run_at DESC)`,
	}
	for _, stmt := range stmts {
		if _, err := s.conn.Exec(stmt); err != nil {
			return fmt.Errorf("schedule store: init schema: %w", err)
		}
	}
	return nil
}

func (s *Store) Create(ctx context.Context, job Job) (Job, error) {
	now := time.Now().UTC()
	if strings.TrimSpace(job.ID) == "" {
		job.ID = uuid.NewString()
	}
	job.CreatedAt = now
	job.UpdatedAt = now
	payloadJSON, err := json.Marshal(job.Payload)
	if err != nil {
		return Job{}, fmt.Errorf("schedule store: marshal payload: %w", err)
	}
	_, err = s.conn.ExecContext(ctx, `
		INSERT INTO schedules(
			id, kind, spec, status, idempotency_key, max_retries, attempt, retry_delay_sec,
			payload_json, last_run_at, next_run_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		job.ID,
		job.Kind,
		job.Spec,
		job.Status,
		nullIfEmpty(job.IdempotencyKey),
		job.MaxRetries,
		job.Attempt,
		job.RetryDelaySec,
		string(payloadJSON),
		unixOrNil(job.LastRunAt),
		unixOrNil(job.NextRunAt),
		nullIfEmpty(job.LastError),
		job.CreatedAt.Unix(),
		job.UpdatedAt.Unix(),
	)
	if err != nil {
		return Job{}, err
	}
	return s.Get(ctx, job.ID)
}

func (s *Store) GetByIdempotencyKey(ctx context.Context, key string) (Job, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Job{}, false, nil
	}
	job, err := s.getByWhere(ctx, "idempotency_key = ?", key)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, false, nil
	}
	if err != nil {
		return Job{}, false, err
	}
	return job, true, nil
}

func (s *Store) Get(ctx context.Context, id string) (Job, error) {
	return s.getByWhere(ctx, "id = ?", id)
}

func (s *Store) getByWhere(ctx context.Context, where string, arg interface{}) (Job, error) {
	row := s.conn.QueryRowContext(ctx, `
		SELECT id, kind, spec, status, idempotency_key, max_retries, attempt, retry_delay_sec,
			payload_json, last_run_at, next_run_at, last_error, created_at, updated_at
		FROM schedules
		WHERE `+where+`
	`, arg)
	return scanJob(row)
}

func (s *Store) List(ctx context.Context, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.conn.QueryContext(ctx, `
		SELECT id, kind, spec, status, idempotency_key, max_retries, attempt, retry_delay_sec,
			payload_json, last_run_at, next_run_at, last_error, created_at, updated_at
		FROM schedules
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (s *Store) ListDue(ctx context.Context, now time.Time, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.conn.QueryContext(ctx, `
		SELECT id, kind, spec, status, idempotency_key, max_retries, attempt, retry_delay_sec,
			payload_json, last_run_at, next_run_at, last_error, created_at, updated_at
		FROM schedules
		WHERE status = ? AND next_run_at IS NOT NULL AND next_run_at <= ?
		ORDER BY next_run_at ASC
		LIMIT ?
	`, StatusActive, now.UTC().Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs, err := scanJobs(rows)
	if err != nil {
		return nil, err
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].NextRunAt.Before(jobs[j].NextRunAt) })
	return jobs, nil
}

func (s *Store) UpdateRunResult(ctx context.Context, id string, status string, attempt int, nextRunAt *time.Time, lastRunAt time.Time, lastErr string) error {
	_, err := s.conn.ExecContext(ctx, `
		UPDATE schedules
		SET status = ?, attempt = ?, next_run_at = ?, last_run_at = ?, last_error = ?, updated_at = ?
		WHERE id = ?
	`, status, attempt, timePtrToUnixOrNil(nextRunAt), lastRunAt.UTC().Unix(), nullIfEmpty(lastErr), time.Now().UTC().Unix(), id)
	return err
}

func (s *Store) UpdateStatus(ctx context.Context, id string, status string) error {
	nextRun := interface{}(nil)
	if status == StatusActive {
		job, err := s.Get(ctx, id)
		if err != nil {
			return err
		}
		computed, err := nextExecutionAt(job, time.Now().UTC())
		if err != nil {
			return err
		}
		nextRun = computed.Unix()
	}
	_, err := s.conn.ExecContext(ctx, `
		UPDATE schedules
		SET status = ?, next_run_at = ?, updated_at = ?
		WHERE id = ?
	`, status, nextRun, time.Now().UTC().Unix(), id)
	return err
}

func (s *Store) AppendRun(ctx context.Context, record RunRecord) error {
	_, err := s.conn.ExecContext(ctx, `
		INSERT INTO schedule_runs(schedule_id, attempt, status, error, run_at, finished_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, record.ScheduleID, record.Attempt, record.Status, nullIfEmpty(record.Error), record.RunAt.UTC().Unix(), record.FinishedAt.UTC().Unix())
	return err
}

func (s *Store) ListRuns(ctx context.Context, scheduleID string, limit int) ([]RunRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.conn.QueryContext(ctx, `
		SELECT id, schedule_id, attempt, status, error, run_at, finished_at
		FROM schedule_runs
		WHERE schedule_id = ?
		ORDER BY run_at DESC
		LIMIT ?
	`, scheduleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RunRecord, 0)
	for rows.Next() {
		var item RunRecord
		var errText sql.NullString
		var runAt int64
		var finishedAt int64
		if err := rows.Scan(&item.ID, &item.ScheduleID, &item.Attempt, &item.Status, &errText, &runAt, &finishedAt); err != nil {
			return nil, err
		}
		item.Error = errText.String
		item.RunAt = time.Unix(runAt, 0).UTC()
		item.FinishedAt = time.Unix(finishedAt, 0).UTC()
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanJob(row *sql.Row) (Job, error) {
	var job Job
	var payloadJSON string
	var idem sql.NullString
	var lastErr sql.NullString
	var lastRun sql.NullInt64
	var nextRun sql.NullInt64
	var createdAt int64
	var updatedAt int64
	if err := row.Scan(
		&job.ID,
		&job.Kind,
		&job.Spec,
		&job.Status,
		&idem,
		&job.MaxRetries,
		&job.Attempt,
		&job.RetryDelaySec,
		&payloadJSON,
		&lastRun,
		&nextRun,
		&lastErr,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Job{}, err
	}
	if err := json.Unmarshal([]byte(payloadJSON), &job.Payload); err != nil {
		return Job{}, err
	}
	job.IdempotencyKey = idem.String
	job.LastError = lastErr.String
	if lastRun.Valid {
		job.LastRunAt = time.Unix(lastRun.Int64, 0).UTC()
	}
	if nextRun.Valid {
		job.NextRunAt = time.Unix(nextRun.Int64, 0).UTC()
	}
	job.CreatedAt = time.Unix(createdAt, 0).UTC()
	job.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return job, nil
}

func scanJobs(rows *sql.Rows) ([]Job, error) {
	items := make([]Job, 0)
	for rows.Next() {
		var (
			job         Job
			payloadJSON string
			idem        sql.NullString
			lastErr     sql.NullString
			lastRun     sql.NullInt64
			nextRun     sql.NullInt64
			createdAt   int64
			updatedAt   int64
		)
		if err := rows.Scan(
			&job.ID,
			&job.Kind,
			&job.Spec,
			&job.Status,
			&idem,
			&job.MaxRetries,
			&job.Attempt,
			&job.RetryDelaySec,
			&payloadJSON,
			&lastRun,
			&nextRun,
			&lastErr,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payloadJSON), &job.Payload); err != nil {
			return nil, err
		}
		job.IdempotencyKey = idem.String
		job.LastError = lastErr.String
		if lastRun.Valid {
			job.LastRunAt = time.Unix(lastRun.Int64, 0).UTC()
		}
		if nextRun.Valid {
			job.NextRunAt = time.Unix(nextRun.Int64, 0).UTC()
		}
		job.CreatedAt = time.Unix(createdAt, 0).UTC()
		job.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		items = append(items, job)
	}
	return items, rows.Err()
}

func nullIfEmpty(v string) interface{} {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func unixOrNil(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Unix()
}

func timePtrToUnixOrNil(t *time.Time) interface{} {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Unix()
}

type Dispatcher interface {
	DeliverDirect(ctx context.Context, channelID string, to string, content string, meta map[string]interface{}) error
	DeliverAgentTurn(ctx context.Context, channelID string, to string, content string, agentID string, meta map[string]interface{}) error
}

type Service struct {
	store      *Store
	dispatcher Dispatcher
	batchSize  int

	runMu   sync.Mutex
	running map[string]struct{}
}

func NewService(store *Store, dispatcher Dispatcher) *Service {
	return &Service{
		store:      store,
		dispatcher: dispatcher,
		batchSize:  20,
		running:    map[string]struct{}{},
	}
}

func (s *Service) SetBatchSize(batchSize int) {
	if s == nil || batchSize <= 0 {
		return
	}
	s.batchSize = batchSize
}

func (s *Service) DispatchDue(ctx context.Context) {
	if s.store == nil || s.dispatcher == nil {
		return
	}
	jobs, err := s.store.ListDue(ctx, time.Now().UTC(), s.batchSize)
	if err != nil {
		return
	}
	for _, job := range jobs {
		job := job
		if !s.markRunning(job.ID) {
			continue
		}
		go func() {
			defer s.unmarkRunning(job.ID)
			s.executeJob(ctx, job)
		}()
	}
}

func (s *Service) executeJob(ctx context.Context, job Job) {
	fresh, err := s.store.Get(ctx, job.ID)
	if err != nil {
		return
	}
	if fresh.Status != StatusActive || fresh.NextRunAt.IsZero() || fresh.NextRunAt.After(time.Now().UTC()) {
		return
	}

	runAt := time.Now().UTC()
	attempt := fresh.Attempt + 1
	deliverErr := s.deliver(ctx, fresh)
	finishedAt := time.Now().UTC()

	runStatus := "success"
	errText := ""
	if deliverErr != nil {
		runStatus = "failed"
		errText = deliverErr.Error()
	}
	_ = s.store.AppendRun(ctx, RunRecord{
		ScheduleID: fresh.ID,
		Attempt:    attempt,
		Status:     runStatus,
		Error:      errText,
		RunAt:      runAt,
		FinishedAt: finishedAt,
	})

	nextStatus, nextAttempt, nextRunAt, nextErr := s.resolveNextState(fresh, attempt, deliverErr, finishedAt)
	_ = s.store.UpdateRunResult(ctx, fresh.ID, nextStatus, nextAttempt, nextRunAt, finishedAt, nextErr)
}

func (s *Service) resolveNextState(job Job, attempt int, runErr error, finishedAt time.Time) (string, int, *time.Time, string) {
	if runErr == nil {
		nextAt, _ := nextExecutionAt(job, finishedAt)
		if job.Kind == KindAt {
			return StatusCompleted, 0, nil, ""
		}
		return StatusActive, 0, &nextAt, ""
	}

	if attempt <= job.MaxRetries {
		delaySec := job.RetryDelaySec
		if delaySec <= 0 {
			delaySec = 30
		}
		next := finishedAt.Add(time.Duration(delaySec) * time.Second)
		return StatusActive, attempt, &next, runErr.Error()
	}

	if job.Kind == KindCron {
		nextAt, err := nextExecutionAt(job, finishedAt)
		if err != nil {
			return StatusFailed, attempt, nil, runErr.Error()
		}
		return StatusActive, 0, &nextAt, runErr.Error()
	}
	return StatusFailed, attempt, nil, runErr.Error()
}

func (s *Service) deliver(ctx context.Context, job Job) error {
	meta := map[string]interface{}{
		"schedule_id":   job.ID,
		"delivery_mode": job.Payload.Mode,
	}
	if job.Payload.Mode == ModeAgentTurn {
		return s.dispatcher.DeliverAgentTurn(ctx, job.Payload.ChannelID, job.Payload.To, job.Payload.Content, job.Payload.AgentID, meta)
	}
	return s.dispatcher.DeliverDirect(ctx, job.Payload.ChannelID, job.Payload.To, job.Payload.Content, meta)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Job, error) {
	job, err := s.buildJob(req)
	if err != nil {
		return Job{}, err
	}
	if job.IdempotencyKey != "" {
		existing, found, err := s.store.GetByIdempotencyKey(ctx, job.IdempotencyKey)
		if err != nil {
			return Job{}, err
		}
		if found {
			return existing, nil
		}
	}
	return s.store.Create(ctx, job)
}

func (s *Service) buildJob(req CreateRequest) (Job, error) {
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind != KindAt && kind != KindCron {
		return Job{}, fmt.Errorf("invalid kind: %s", req.Kind)
	}
	payload, err := validatePayload(req.Payload)
	if err != nil {
		return Job{}, err
	}
	maxRetries := req.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	retryDelaySec := req.RetryDelaySec
	if retryDelaySec <= 0 {
		retryDelaySec = 30
	}

	job := Job{
		ID:             uuid.NewString(),
		Kind:           kind,
		Status:         StatusActive,
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		MaxRetries:     maxRetries,
		RetryDelaySec:  retryDelaySec,
		Payload:        payload,
	}

	now := time.Now().UTC()
	if kind == KindAt {
		atText := strings.TrimSpace(req.At)
		if atText == "" {
			return Job{}, errors.New("at is required for kind=at")
		}
		atValue, err := time.Parse(time.RFC3339, atText)
		if err != nil {
			return Job{}, fmt.Errorf("invalid at timestamp: %w", err)
		}
		atValue = atValue.UTC()
		job.Spec = atValue.Format(time.RFC3339)
		if atValue.Before(now) {
			job.NextRunAt = now
		} else {
			job.NextRunAt = atValue
		}
		return job, nil
	}

	cronSpec := strings.TrimSpace(req.Cron)
	if cronSpec == "" {
		return Job{}, errors.New("cron is required for kind=cron")
	}
	job.Spec = cronSpec
	nextAt, err := nextExecutionAt(job, now)
	if err != nil {
		return Job{}, err
	}
	job.NextRunAt = nextAt
	return job, nil
}

func (s *Service) List(ctx context.Context, limit int) ([]Job, error) {
	return s.store.List(ctx, limit)
}

func (s *Service) Get(ctx context.Context, id string) (Job, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) Pause(ctx context.Context, id string) error {
	return s.store.UpdateStatus(ctx, id, StatusPaused)
}

func (s *Service) Resume(ctx context.Context, id string) error {
	return s.store.UpdateStatus(ctx, id, StatusActive)
}

func (s *Service) Cancel(ctx context.Context, id string) error {
	return s.store.UpdateStatus(ctx, id, StatusCanceled)
}

func (s *Service) Runs(ctx context.Context, id string, limit int) ([]RunRecord, error) {
	return s.store.ListRuns(ctx, id, limit)
}

func (s *Service) markRunning(id string) bool {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if _, exists := s.running[id]; exists {
		return false
	}
	s.running[id] = struct{}{}
	return true
}

func (s *Service) unmarkRunning(id string) {
	s.runMu.Lock()
	delete(s.running, id)
	s.runMu.Unlock()
}

func validatePayload(payload DeliveryPayload) (DeliveryPayload, error) {
	payload.Mode = strings.ToLower(strings.TrimSpace(payload.Mode))
	payload.ChannelID = strings.TrimSpace(payload.ChannelID)
	payload.To = strings.TrimSpace(payload.To)
	payload.Content = strings.TrimSpace(payload.Content)
	payload.AgentID = strings.TrimSpace(payload.AgentID)
	if payload.Mode == "" {
		payload.Mode = ModeDirect
	}
	if payload.Mode != ModeDirect && payload.Mode != ModeAgentTurn {
		return DeliveryPayload{}, fmt.Errorf("invalid payload.mode: %s", payload.Mode)
	}
	if payload.ChannelID == "" {
		return DeliveryPayload{}, errors.New("payload.channel_id is required")
	}
	if payload.To == "" {
		return DeliveryPayload{}, errors.New("payload.to is required")
	}
	if payload.Content == "" {
		return DeliveryPayload{}, errors.New("payload.content is required")
	}
	if payload.Mode == ModeAgentTurn && payload.AgentID == "" {
		return DeliveryPayload{}, errors.New("payload.agent_id is required for mode=agent_turn")
	}
	return payload, nil
}

func nextExecutionAt(job Job, after time.Time) (time.Time, error) {
	after = after.UTC()
	if job.Kind == KindAt {
		atValue, err := time.Parse(time.RFC3339, job.Spec)
		if err != nil {
			return time.Time{}, err
		}
		return atValue.UTC(), nil
	}
	fields := strings.Fields(strings.TrimSpace(job.Spec))
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("invalid cron spec: %s", job.Spec)
	}
	for i, f := range fields {
		minValue, maxValue := cronFieldBounds(i)
		if _, err := parseCronField(f, minValue, maxValue); err != nil {
			return time.Time{}, fmt.Errorf("invalid cron field %d: %w", i+1, err)
		}
	}

	candidate := after.Truncate(time.Minute).Add(time.Minute)
	limit := candidate.Add(366 * 24 * time.Hour)
	for !candidate.After(limit) {
		ok, err := matchCronFields(fields, candidate)
		if err != nil {
			return time.Time{}, err
		}
		if ok {
			return candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}
	return time.Time{}, errors.New("cron spec has no upcoming execution within one year")
}

func matchCronFields(fields []string, t time.Time) (bool, error) {
	values := []int{t.Minute(), t.Hour(), t.Day(), int(t.Month()), int(t.Weekday())}
	for i, field := range fields {
		minValue, maxValue := cronFieldBounds(i)
		matcher, err := parseCronField(field, minValue, maxValue)
		if err != nil {
			return false, err
		}
		if !matcher(values[i]) {
			return false, nil
		}
	}
	return true, nil
}

func cronFieldBounds(index int) (int, int) {
	switch index {
	case 0:
		return 0, 59
	case 1:
		return 0, 23
	case 2:
		return 1, 31
	case 3:
		return 1, 12
	default:
		return 0, 6
	}
}

func parseCronField(raw string, boundsMin int, boundsMax int) (func(int) bool, error) {
	field := strings.TrimSpace(raw)
	if field == "*" {
		return func(v int) bool { return v >= boundsMin && v <= boundsMax }, nil
	}
	if strings.HasPrefix(field, "*/") {
		stepText := strings.TrimPrefix(field, "*/")
		step, err := strconv.Atoi(stepText)
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step: %s", field)
		}
		return func(v int) bool {
			if v < boundsMin || v > boundsMax {
				return false
			}
			return (v-boundsMin)%step == 0
		}, nil
	}
	value, err := strconv.Atoi(field)
	if err != nil {
		return nil, fmt.Errorf("unsupported field: %s", field)
	}
	if value < boundsMin || value > boundsMax {
		return nil, fmt.Errorf("value out of range: %s", field)
	}
	return func(v int) bool { return v == value }, nil
}
