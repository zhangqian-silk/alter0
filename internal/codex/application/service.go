package application

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	codexdomain "alter0/internal/codex/domain"
)

const codexOAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"

var (
	codexUsageURL = "https://chatgpt.com/backend-api/wham/usage"
	codexTokenURL = "https://auth.openai.com/oauth/token"
)

type Record struct {
	Name           string               `json:"name"`
	SavedAt        time.Time            `json:"saved_at"`
	LastSwitchedAt time.Time            `json:"last_switched_at,omitempty"`
	Snapshot       codexdomain.Snapshot `json:"snapshot"`
	RawAuth        json.RawMessage      `json:"raw_auth"`
}

type AccountStatus struct {
	Record    Record                   `json:"record"`
	Current   bool                     `json:"current"`
	Quota     *codexdomain.QuotaStatus `json:"quota,omitempty"`
	Error     string                   `json:"error,omitempty"`
	Refreshed bool                     `json:"refreshed"`
}

type CurrentStatus struct {
	Live     *codexdomain.Snapshot `json:"live,omitempty"`
	Managed  *Record               `json:"managed,omitempty"`
	AuthPath string                `json:"auth_path,omitempty"`
}

type LoginSessionStatus string

const (
	LoginSessionStatusPending   LoginSessionStatus = "pending"
	LoginSessionStatusRunning   LoginSessionStatus = "running"
	LoginSessionStatusSucceeded LoginSessionStatus = "succeeded"
	LoginSessionStatusFailed    LoginSessionStatus = "failed"
)

type LoginSession struct {
	ID          string             `json:"id"`
	AccountName string             `json:"account_name,omitempty"`
	Status      LoginSessionStatus `json:"status"`
	Logs        string             `json:"logs,omitempty"`
	Error       string             `json:"error,omitempty"`
	StartedAt   time.Time          `json:"started_at,omitempty"`
	CompletedAt time.Time          `json:"completed_at,omitempty"`
	Account     *Record            `json:"account,omitempty"`
}

type Store interface {
	Save(record Record, overwrite bool) error
	Load(name string) (*Record, error)
	List() ([]Record, error)
	CreateBackup(raw []byte) (string, error)
	PrepareLoginHome(sessionID string) (string, error)
}

type CommandOptions struct {
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type QuotaQueryOptions struct {
	Now    time.Time
	Client *http.Client
}

type ServiceOptions struct {
	Store             Store
	Command           string
	ResolveActiveHome func() (string, error)
	RunCommand        func(ctx context.Context, name string, args []string, options CommandOptions) error
	QueryQuota        func(rawAuth []byte, options QuotaQueryOptions) (*codexdomain.QuotaStatus, []byte, error)
	NewID             func() string
	Now               func() time.Time
	LoginStdout       io.Writer
}

type Service struct {
	store             Store
	command           string
	resolveActiveHome func() (string, error)
	runCommand        func(ctx context.Context, name string, args []string, options CommandOptions) error
	queryQuota        func(rawAuth []byte, options QuotaQueryOptions) (*codexdomain.QuotaStatus, []byte, error)
	newID             func() string
	now               func() time.Time
	loginStdout       io.Writer

	mu           sync.RWMutex
	loginSession map[string]LoginSession
}

func NewService(options ServiceOptions) *Service {
	command := strings.TrimSpace(options.Command)
	if command == "" {
		command = "codex"
	}
	resolveActiveHome := options.ResolveActiveHome
	if resolveActiveHome == nil {
		resolveActiveHome = ResolveActiveHome
	}
	runCommand := options.RunCommand
	if runCommand == nil {
		runCommand = defaultRunCommand
	}
	queryQuota := options.QueryQuota
	if queryQuota == nil {
		queryQuota = QueryQuota
	}
	now := options.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	newID := options.NewID
	if newID == nil {
		newID = func() string { return fmt.Sprintf("login-%d", now().UnixNano()) }
	}
	return &Service{
		store:             options.Store,
		command:           command,
		resolveActiveHome: resolveActiveHome,
		runCommand:        runCommand,
		queryQuota:        queryQuota,
		newID:             newID,
		now:               now,
		loginStdout:       options.LoginStdout,
		loginSession:      map[string]LoginSession{},
	}
}

func (s *Service) AddFromRaw(name string, raw []byte, overwrite bool) (*Record, error) {
	if s.store == nil {
		return nil, errors.New("codex account store is not configured")
	}
	raw = compactJSON(raw)
	snapshot, err := codexdomain.SnapshotFromRawAuth(raw)
	if err != nil {
		return nil, err
	}
	resolvedName, err := s.resolveRecordName(name, *snapshot)
	if err != nil {
		return nil, err
	}

	record := Record{
		Name:     resolvedName,
		SavedAt:  s.now(),
		Snapshot: *snapshot,
		RawAuth:  append([]byte(nil), raw...),
	}
	if err := s.store.Save(record, overwrite); err != nil {
		return nil, err
	}
	return s.store.Load(resolvedName)
}

func (s *Service) Current() (*CurrentStatus, error) {
	activeHome, err := s.resolveActiveHome()
	if err != nil {
		return nil, err
	}
	authPath := AuthFilePath(activeHome)
	raw, err := os.ReadFile(authPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &CurrentStatus{AuthPath: authPath}, nil
		}
		return nil, fmt.Errorf("read active auth.json: %w", err)
	}
	snapshot, err := codexdomain.SnapshotFromRawAuth(raw)
	if err != nil {
		return nil, err
	}
	current := &CurrentStatus{
		Live:     snapshot,
		AuthPath: authPath,
	}
	if s.store == nil {
		return current, nil
	}
	records, err := s.store.List()
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.Snapshot.AuthHash == snapshot.AuthHash {
			cloned := record
			current.Managed = &cloned
			return current, nil
		}
	}
	for _, record := range records {
		if record.Snapshot.IdentityKey != "" && record.Snapshot.IdentityKey == snapshot.IdentityKey {
			cloned := record
			current.Managed = &cloned
			return current, nil
		}
	}
	return current, nil
}

func (s *Service) ListStatuses(ctx context.Context) ([]AccountStatus, *CurrentStatus, error) {
	if s.store == nil {
		return nil, nil, errors.New("codex account store is not configured")
	}
	records, err := s.store.List()
	if err != nil {
		return nil, nil, err
	}
	current, err := s.Current()
	if err != nil {
		return nil, nil, err
	}

	items := make([]AccountStatus, 0, len(records))
	for _, record := range records {
		item := AccountStatus{
			Record:  record,
			Current: current.Managed != nil && strings.EqualFold(current.Managed.Name, record.Name),
		}
		if s.queryQuota != nil {
			quota, updatedRaw, quotaErr := s.queryQuota(record.RawAuth, QuotaQueryOptions{Now: s.now()})
			if quotaErr != nil {
				item.Error = quotaErr.Error()
			} else {
				item.Quota = quota
				item.Refreshed = quota.Refreshed
				if strings.TrimSpace(quota.Plan) != "" {
					record.Snapshot.Plan = strings.TrimSpace(quota.Plan)
					item.Record = record
				}
			}
			if len(updatedRaw) > 0 {
				record.RawAuth = append([]byte(nil), updatedRaw...)
				if snapshot, snapshotErr := codexdomain.SnapshotFromRawAuth(updatedRaw); snapshotErr == nil {
					record.Snapshot = *snapshot
				}
			}
			if len(updatedRaw) > 0 || (item.Quota != nil && strings.TrimSpace(item.Quota.Plan) != "") {
				if saveErr := s.store.Save(record, true); saveErr != nil && item.Error == "" {
					item.Error = saveErr.Error()
				}
				item.Record = record
			}
		}
		items = append(items, item)
	}
	_ = ctx
	return items, current, nil
}

func (s *Service) Switch(name string) (*Record, string, error) {
	if s.store == nil {
		return nil, "", errors.New("codex account store is not configured")
	}
	record, err := s.store.Load(name)
	if err != nil {
		return nil, "", err
	}
	activeHome, err := s.resolveActiveHome()
	if err != nil {
		return nil, "", err
	}

	var backupPath string
	currentRaw, err := os.ReadFile(AuthFilePath(activeHome))
	switch {
	case err == nil && len(currentRaw) > 0:
		if bytes.Equal(compactJSON(currentRaw), compactJSON(record.RawAuth)) {
			record.LastSwitchedAt = s.now()
			if err := s.store.Save(*record, true); err != nil {
				return nil, "", err
			}
			return record, "", nil
		}
		currentSnapshot, snapshotErr := codexdomain.SnapshotFromRawAuth(currentRaw)
		if snapshotErr == nil &&
			currentSnapshot.IdentityKey != "" &&
			currentSnapshot.IdentityKey == record.Snapshot.IdentityKey &&
			currentSnapshot.IsNewerThan(record.Snapshot) {
			record.RawAuth = append([]byte(nil), currentRaw...)
			record.Snapshot = *currentSnapshot
			record.LastSwitchedAt = s.now()
			if err := s.store.Save(*record, true); err != nil {
				return nil, "", err
			}
			return record, "", nil
		}
		backupPath, err = s.store.CreateBackup(currentRaw)
		if err != nil {
			return nil, "", err
		}
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return nil, "", fmt.Errorf("read current auth.json: %w", err)
	}

	if err := WriteFileWithBackup(AuthFilePath(activeHome), compactJSON(record.RawAuth)); err != nil {
		return nil, "", err
	}
	record.LastSwitchedAt = s.now()
	if err := s.store.Save(*record, true); err != nil {
		return nil, "", err
	}
	return record, backupPath, nil
}

func (s *Service) StartLoginSession(ctx context.Context, name string, overwrite bool) (LoginSession, error) {
	if s.store == nil {
		return LoginSession{}, errors.New("codex account store is not configured")
	}
	if err := s.ensureNameAvailable(name, overwrite); err != nil {
		return LoginSession{}, err
	}

	session := LoginSession{
		ID:          strings.TrimSpace(s.newID()),
		AccountName: strings.TrimSpace(name),
		Status:      LoginSessionStatusPending,
		StartedAt:   s.now(),
	}
	if session.ID == "" {
		return LoginSession{}, errors.New("failed to allocate login session id")
	}
	loginHome, err := s.store.PrepareLoginHome(session.ID)
	if err != nil {
		return LoginSession{}, err
	}

	s.mu.Lock()
	s.loginSession[session.ID] = session
	s.mu.Unlock()

	go s.runLoginSession(ctx, session.ID, loginHome, name, overwrite)
	return session, nil
}

func (s *Service) GetLoginSession(id string) (LoginSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.loginSession[strings.TrimSpace(id)]
	return session, ok
}

func (s *Service) runLoginSession(ctx context.Context, sessionID string, loginHome string, name string, overwrite bool) {
	s.updateLoginSession(sessionID, func(session *LoginSession) {
		session.Status = LoginSessionStatusRunning
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	stdoutWriter := io.Writer(&stdout)
	if s.loginStdout != nil {
		stdoutWriter = io.MultiWriter(&stdout, s.loginStdout)
	}
	stderrWriter := io.Writer(&stderr)
	if s.loginStdout != nil {
		stderrWriter = io.MultiWriter(&stderr, s.loginStdout)
	}
	err := s.runCommand(ctx, s.command, []string{"login"}, CommandOptions{
		Env:    append(os.Environ(), "CODEX_HOME="+loginHome),
		Stdout: stdoutWriter,
		Stderr: stderrWriter,
	})

	logs := strings.TrimSpace(stdout.String())
	if stderrText := strings.TrimSpace(stderr.String()); stderrText != "" {
		if logs != "" {
			logs += "\n"
		}
		logs += stderrText
	}
	if err != nil {
		s.updateLoginSession(sessionID, func(session *LoginSession) {
			session.Status = LoginSessionStatusFailed
			session.Error = err.Error()
			session.Logs = logs
			session.CompletedAt = s.now()
		})
		return
	}

	raw, readErr := os.ReadFile(AuthFilePath(loginHome))
	if readErr != nil {
		s.updateLoginSession(sessionID, func(session *LoginSession) {
			session.Status = LoginSessionStatusFailed
			session.Error = fmt.Sprintf("read login auth.json: %v", readErr)
			session.Logs = logs
			session.CompletedAt = s.now()
		})
		return
	}
	record, addErr := s.AddFromRaw(name, raw, overwrite)
	if addErr != nil {
		s.updateLoginSession(sessionID, func(session *LoginSession) {
			session.Status = LoginSessionStatusFailed
			session.Error = addErr.Error()
			session.Logs = logs
			session.CompletedAt = s.now()
		})
		return
	}
	s.updateLoginSession(sessionID, func(session *LoginSession) {
		session.Status = LoginSessionStatusSucceeded
		session.Account = record
		session.Logs = logs
		session.CompletedAt = s.now()
	})
}

func (s *Service) updateLoginSession(id string, update func(session *LoginSession)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.loginSession[id]
	if !ok {
		return
	}
	update(&session)
	s.loginSession[id] = session
}

func (s *Service) resolveRecordName(name string, snapshot codexdomain.Snapshot) (string, error) {
	name = strings.TrimSpace(name)
	if name != "" {
		return name, nil
	}
	return s.nextAvailableName(shellName(snapshot.SuggestedName()))
}

func (s *Service) nextAvailableName(base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "codex-account"
	}
	records, err := s.store.List()
	if err != nil {
		return "", err
	}
	used := make(map[string]struct{}, len(records))
	for _, record := range records {
		used[strings.ToLower(strings.TrimSpace(record.Name))] = struct{}{}
	}
	candidate := base
	for index := 2; ; index++ {
		if _, exists := used[strings.ToLower(candidate)]; !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, index)
	}
}

func (s *Service) ensureNameAvailable(name string, overwrite bool) error {
	name = strings.TrimSpace(name)
	if overwrite || name == "" {
		return nil
	}
	records, err := s.store.List()
	if err != nil {
		return err
	}
	for _, record := range records {
		if strings.EqualFold(record.Name, name) {
			return fmt.Errorf("account %q already exists", name)
		}
	}
	return nil
}

func ResolveDefaultHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(home, ".codex"), nil
}

func ResolveActiveHome() (string, error) {
	if env := strings.TrimSpace(os.Getenv("CODEX_HOME")); env != "" {
		return filepath.Clean(env), nil
	}
	return ResolveDefaultHome()
}

func AuthFilePath(home string) string {
	return filepath.Join(home, "auth.json")
}

func WriteFileWithBackup(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmp, err)
	}
	backup := path + ".bak"
	_ = os.Remove(backup)
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, backup); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("create backup %s: %w", backup, err)
		}
	}
	if err := os.Rename(tmp, path); err != nil {
		if _, restoreErr := os.Stat(backup); restoreErr == nil {
			_ = os.Rename(backup, path)
		}
		_ = os.Remove(tmp)
		return fmt.Errorf("replace file %s: %w", path, err)
	}
	_ = os.Remove(backup)
	return nil
}

func defaultRunCommand(ctx context.Context, name string, args []string, options CommandOptions) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = options.Dir
	cmd.Env = options.Env
	cmd.Stdin = options.Stdin
	cmd.Stdout = options.Stdout
	cmd.Stderr = options.Stderr
	return cmd.Run()
}

type usageResponse struct {
	PlanType  string         `json:"plan_type"`
	RateLimit *rateLimitInfo `json:"rate_limit"`
}

type rateLimitInfo struct {
	PrimaryWindow   *usageWindow `json:"primary_window"`
	SecondaryWindow *usageWindow `json:"secondary_window"`
}

type usageWindow struct {
	UsedPercent       *int   `json:"used_percent"`
	ResetAfterSeconds *int64 `json:"reset_after_seconds"`
	ResetAt           *int64 `json:"reset_at"`
}

type tokenRefreshResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type quotaAPIError struct {
	Detail struct {
		Code string `json:"code"`
	} `json:"detail"`
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
	Code string `json:"code"`
}

func QueryQuota(rawAuth []byte, options QuotaQueryOptions) (*codexdomain.QuotaStatus, []byte, error) {
	auth, err := codexdomain.ParseAuthFile(rawAuth)
	if err != nil {
		return nil, nil, fmt.Errorf("parse auth.json: %w", err)
	}
	if normalizeAuthMode(auth) == "apikey" {
		return nil, nil, fmt.Errorf("api key accounts do not support quota status")
	}
	if auth.Tokens == nil || auth.Tokens.IsZero() {
		return nil, nil, fmt.Errorf("auth.json missing usable oauth token")
	}

	now := options.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	currentAuth := *auth
	refreshed := false
	if tokenExpirySoon(currentAuth.Tokens.AccessToken, now) {
		if err := refreshTokens(&currentAuth, client, now); err != nil {
			return nil, nil, err
		}
		refreshed = true
	}

	status, err := fetchQuotaStatus(&currentAuth, client, now)
	if err != nil && shouldRetryWithRefresh(err) {
		if refreshErr := refreshTokens(&currentAuth, client, now); refreshErr == nil {
			refreshed = true
			status, err = fetchQuotaStatus(&currentAuth, client, now)
		}
	}
	if err != nil {
		return nil, nil, err
	}
	status.Refreshed = refreshed
	if !refreshed {
		return status, nil, nil
	}
	updatedRaw, err := serializeAuthFile(currentAuth)
	if err != nil {
		return nil, nil, err
	}
	return status, updatedRaw, nil
}

func fetchQuotaStatus(auth *codexdomain.AuthFile, client *http.Client, now time.Time) (*codexdomain.QuotaStatus, error) {
	req, err := http.NewRequest(http.MethodGet, codexUsageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build quota request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(auth.Tokens.AccessToken))

	accountID := firstNonEmpty(
		auth.Tokens.AccountID,
		extractAccountIDFromJWT(auth.Tokens.AccessToken),
		extractAccountIDFromJWT(auth.Tokens.IDToken),
	)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request quota api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read quota response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorCode := extractQuotaErrorCode(body)
		message := fmt.Sprintf("quota api returned %d", resp.StatusCode)
		if errorCode != "" {
			message += " [error_code:" + errorCode + "]"
		}
		return nil, errors.New(message)
	}

	var usage usageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, fmt.Errorf("decode quota response: %w", err)
	}
	return &codexdomain.QuotaStatus{
		Hourly: codexdomain.QuotaWindow{
			RemainingPercent: remainingPercent(rateWindow(usage.RateLimit, true)),
			ResetAt:          resetTime(rateWindow(usage.RateLimit, true), now),
		},
		Weekly: codexdomain.QuotaWindow{
			RemainingPercent: remainingPercent(rateWindow(usage.RateLimit, false)),
			ResetAt:          resetTime(rateWindow(usage.RateLimit, false), now),
		},
		Plan: strings.TrimSpace(usage.PlanType),
	}, nil
}

func refreshTokens(auth *codexdomain.AuthFile, client *http.Client, now time.Time) error {
	refreshToken := strings.TrimSpace(auth.Tokens.RefreshToken)
	if refreshToken == "" {
		return fmt.Errorf("token expired and refresh_token is missing")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", codexOAuthClientID)

	req, err := http.NewRequest(http.MethodPost, codexTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("refresh token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read refresh response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("refresh token failed: status=%d", resp.StatusCode)
	}
	var payload tokenRefreshResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode refresh response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return fmt.Errorf("refresh response missing access_token")
	}
	auth.Tokens.AccessToken = strings.TrimSpace(payload.AccessToken)
	if strings.TrimSpace(payload.IDToken) != "" {
		auth.Tokens.IDToken = strings.TrimSpace(payload.IDToken)
	}
	if strings.TrimSpace(payload.RefreshToken) != "" {
		auth.Tokens.RefreshToken = strings.TrimSpace(payload.RefreshToken)
	}
	auth.LastRefresh = now.Format(time.RFC3339Nano)
	if accountID := extractAccountIDFromJWT(auth.Tokens.IDToken); accountID != "" && strings.TrimSpace(auth.Tokens.AccountID) == "" {
		auth.Tokens.AccountID = accountID
	}
	return nil
}

func serializeAuthFile(auth codexdomain.AuthFile) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(auth); err != nil {
		return nil, fmt.Errorf("marshal updated auth.json: %w", err)
	}
	return bytes.TrimSpace(buffer.Bytes()), nil
}

func tokenExpirySoon(accessToken string, now time.Time) bool {
	expiry := codexdomain.TokenExpiry(accessToken)
	if expiry.IsZero() {
		return false
	}
	return expiry.Before(now.Add(5 * time.Minute))
}

func shouldRetryWithRefresh(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "401") || strings.Contains(lower, "token_invalidated") || strings.Contains(lower, "expired")
}

func normalizeAuthMode(auth *codexdomain.AuthFile) string {
	mode := strings.ToLower(strings.TrimSpace(auth.AuthMode))
	if mode != "" {
		return mode
	}
	if extractAPIKey(auth.OpenAIAPIKey) != "" && auth.Tokens == nil {
		return "apikey"
	}
	return "oauth"
}

func extractAPIKey(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func extractAccountIDFromJWT(token string) string {
	claims, err := parseJWTClaims(token)
	if err != nil || claims == nil {
		return ""
	}
	return firstNonEmpty(claims.Auth.AccountID, claims.Auth.ChatGPTAccID)
}

type jwtQuotaClaims struct {
	Auth struct {
		AccountID    string `json:"account_id"`
		ChatGPTAccID string `json:"chatgpt_account_id"`
	} `json:"https://api.openai.com/auth"`
}

func parseJWTClaims(token string) (*jwtQuotaClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid jwt")
	}
	payload, err := decodeJWTPart(parts[1])
	if err != nil {
		return nil, err
	}
	var claims jwtQuotaClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

func decodeJWTPart(value string) ([]byte, error) {
	return decodeBase64URL(value)
}

func decodeBase64URL(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}

func rateWindow(info *rateLimitInfo, primary bool) *usageWindow {
	if info == nil {
		return nil
	}
	if primary {
		return info.PrimaryWindow
	}
	return info.SecondaryWindow
}

func remainingPercent(window *usageWindow) int {
	if window == nil || window.UsedPercent == nil {
		return 0
	}
	remaining := 100 - *window.UsedPercent
	if remaining < 0 {
		return 0
	}
	return remaining
}

func resetTime(window *usageWindow, now time.Time) time.Time {
	if window == nil {
		return time.Time{}
	}
	if window.ResetAt != nil && *window.ResetAt > 0 {
		return time.Unix(*window.ResetAt, 0).UTC()
	}
	if window.ResetAfterSeconds != nil && *window.ResetAfterSeconds > 0 {
		return now.Add(time.Duration(*window.ResetAfterSeconds) * time.Second).UTC()
	}
	return time.Time{}
}

func extractQuotaErrorCode(body []byte) string {
	var payload quotaAPIError
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return firstNonEmpty(payload.Detail.Code, payload.Error.Code, payload.Code)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func shellName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			builder.WriteByte(byte(r - 'A' + 'a'))
			lastDash = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastDash = false
		case builder.Len() > 0 && !lastDash:
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "codex-account"
	}
	return result
}

func compactJSON(raw []byte) []byte {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return []byte{}
	}
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, trimmed); err != nil {
		return append([]byte(nil), trimmed...)
	}
	return buffer.Bytes()
}
