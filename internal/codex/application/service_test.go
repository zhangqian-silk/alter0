package application_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	codexapp "alter0/internal/codex/application"
	codexdomain "alter0/internal/codex/domain"
	localcodex "alter0/internal/codex/infrastructure/localfile"
)

func TestServiceAddFromRawAndSwitch(t *testing.T) {
	activeHome := filepath.Join(t.TempDir(), ".codex")
	if err := os.MkdirAll(activeHome, 0o755); err != nil {
		t.Fatalf("mkdir active home: %v", err)
	}
	currentRaw := buildServiceOAuthAuth(t, serviceOAuthInput{
		Name:      "Current Account",
		Email:     "current@example.com",
		UserID:    "user-current",
		AccountID: "acct-current",
		Plan:      "plus",
		ExpiresAt: time.Date(2026, 4, 18, 8, 0, 0, 0, time.UTC),
	})
	if err := os.WriteFile(codexapp.AuthFilePath(activeHome), currentRaw, 0o600); err != nil {
		t.Fatalf("write current auth: %v", err)
	}

	store, err := localcodex.NewStore(filepath.Join(t.TempDir(), "accounts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	service := codexapp.NewService(codexapp.ServiceOptions{
		Store:             store,
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		Now: func() time.Time {
			return time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
		},
	})

	targetRaw := buildServiceOAuthAuth(t, serviceOAuthInput{
		Name:      "Work Account",
		Email:     "work@example.com",
		UserID:    "user-work",
		AccountID: "acct-work",
		Plan:      "pro",
		ExpiresAt: time.Date(2026, 4, 19, 8, 0, 0, 0, time.UTC),
	})
	record, err := service.AddFromRaw("work", targetRaw, false)
	if err != nil {
		t.Fatalf("AddFromRaw returned error: %v", err)
	}
	if record.Name != "work" {
		t.Fatalf("record.Name = %q, want work", record.Name)
	}

	switched, backupPath, err := service.Switch("work")
	if err != nil {
		t.Fatalf("Switch returned error: %v", err)
	}
	if switched.Name != "work" {
		t.Fatalf("switched.Name = %q, want work", switched.Name)
	}
	if strings.TrimSpace(backupPath) == "" {
		t.Fatalf("expected backupPath to be populated")
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backupPath stat failed: %v", err)
	}

	gotRaw, err := os.ReadFile(codexapp.AuthFilePath(activeHome))
	if err != nil {
		t.Fatalf("read switched auth: %v", err)
	}
	if !bytes.Equal(gotRaw, targetRaw) {
		t.Fatalf("active auth mismatch after switch")
	}
}

func TestServiceListStatusesMarksCurrentAndRefreshesQuota(t *testing.T) {
	activeHome := filepath.Join(t.TempDir(), ".codex")
	if err := os.MkdirAll(activeHome, 0o755); err != nil {
		t.Fatalf("mkdir active home: %v", err)
	}
	raw := buildServiceOAuthAuth(t, serviceOAuthInput{
		Name:      "Work Account",
		Email:     "work@example.com",
		UserID:    "user-work",
		AccountID: "acct-work",
		Plan:      "plus",
		ExpiresAt: time.Date(2026, 4, 18, 8, 0, 0, 0, time.UTC),
	})
	if err := os.WriteFile(codexapp.AuthFilePath(activeHome), raw, 0o600); err != nil {
		t.Fatalf("write active auth: %v", err)
	}

	store, err := localcodex.NewStore(filepath.Join(t.TempDir(), "accounts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	service := codexapp.NewService(codexapp.ServiceOptions{
		Store:             store,
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		QueryQuota: func(_ []byte, _ codexapp.QuotaQueryOptions) (*codexdomain.QuotaStatus, []byte, error) {
			updated := append([]byte(nil), raw...)
			return &codexdomain.QuotaStatus{
				Hourly: codexdomain.QuotaWindow{
					RemainingPercent: 80,
					ResetAt:          time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
				},
				Weekly: codexdomain.QuotaWindow{
					RemainingPercent: 92,
					ResetAt:          time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
				},
				Plan:      "enterprise",
				Refreshed: true,
			}, updated, nil
		},
	})
	if _, err := service.AddFromRaw("work", raw, false); err != nil {
		t.Fatalf("AddFromRaw returned error: %v", err)
	}

	statuses, active, err := service.ListStatuses(context.Background())
	if err != nil {
		t.Fatalf("ListStatuses returned error: %v", err)
	}
	if active == nil || active.Managed == nil || active.Managed.Name != "work" {
		t.Fatalf("active managed account = %+v, want work", active)
	}
	if len(statuses) != 1 {
		t.Fatalf("len(statuses) = %d, want 1", len(statuses))
	}
	if !statuses[0].Current {
		t.Fatalf("expected status to be marked current")
	}
	if statuses[0].Quota == nil || statuses[0].Quota.Plan != "enterprise" {
		t.Fatalf("quota = %+v, want enterprise plan", statuses[0].Quota)
	}
	if !statuses[0].Refreshed {
		t.Fatalf("expected refreshed quota")
	}
}

func TestServiceStartLoginSessionPersistsAccountOnSuccess(t *testing.T) {
	activeHome := filepath.Join(t.TempDir(), ".codex")
	store, err := localcodex.NewStore(filepath.Join(t.TempDir(), "accounts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	var stdout bytes.Buffer
	service := codexapp.NewService(codexapp.ServiceOptions{
		Store:             store,
		Command:           "codex",
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		NewID:             func() string { return "login-1" },
		RunCommand: func(_ context.Context, name string, args []string, options codexapp.CommandOptions) error {
			if name != "codex" {
				t.Fatalf("command name = %q, want codex", name)
			}
			if len(args) != 1 || args[0] != "login" {
				t.Fatalf("command args = %v, want [login]", args)
			}
			if _, err := io.WriteString(options.Stdout, "open browser\n"); err != nil {
				return err
			}
			loginHome := envValue(options.Env, "CODEX_HOME")
			if strings.TrimSpace(loginHome) == "" {
				t.Fatalf("expected CODEX_HOME env")
			}
			raw := buildServiceOAuthAuth(t, serviceOAuthInput{
				Name:      "Fresh Account",
				Email:     "fresh@example.com",
				UserID:    "user-fresh",
				AccountID: "acct-fresh",
				Plan:      "plus",
				ExpiresAt: time.Date(2026, 4, 19, 8, 0, 0, 0, time.UTC),
			})
			return os.WriteFile(codexapp.AuthFilePath(loginHome), raw, 0o600)
		},
		LoginStdout: &stdout,
	})

	session, err := service.StartLoginSession(context.Background(), "fresh", false)
	if err != nil {
		t.Fatalf("StartLoginSession returned error: %v", err)
	}
	if session.ID != "login-1" {
		t.Fatalf("session.ID = %q, want login-1", session.ID)
	}

	waitForLoginStatus(t, service, "login-1", codexapp.LoginSessionStatusSucceeded)

	record, err := store.Load("fresh")
	if err != nil {
		t.Fatalf("load saved account: %v", err)
	}
	if record.Name != "fresh" {
		t.Fatalf("record.Name = %q, want fresh", record.Name)
	}
	finalSession, ok := service.GetLoginSession("login-1")
	if !ok {
		t.Fatalf("expected login session to exist")
	}
	if !strings.Contains(finalSession.Logs, "open browser") {
		t.Fatalf("expected login logs to capture stdout, got %q", finalSession.Logs)
	}
	if finalSession.Account == nil || finalSession.Account.Name != "fresh" {
		t.Fatalf("expected login session account to be saved, got %+v", finalSession.Account)
	}
}

func TestServiceStartLoginSessionMarksFailure(t *testing.T) {
	activeHome := filepath.Join(t.TempDir(), ".codex")
	store, err := localcodex.NewStore(filepath.Join(t.TempDir(), "accounts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	service := codexapp.NewService(codexapp.ServiceOptions{
		Store:             store,
		Command:           "codex",
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		NewID:             func() string { return "login-fail" },
		RunCommand: func(_ context.Context, _ string, _ []string, options codexapp.CommandOptions) error {
			_, _ = io.WriteString(options.Stderr, "login failed\n")
			return errors.New("exit status 1")
		},
	})

	if _, err := service.StartLoginSession(context.Background(), "broken", false); err != nil {
		t.Fatalf("StartLoginSession returned error: %v", err)
	}

	session := waitForLoginStatus(t, service, "login-fail", codexapp.LoginSessionStatusFailed)
	if !strings.Contains(session.Error, "exit status 1") {
		t.Fatalf("session.Error = %q, want exit status 1", session.Error)
	}
	if !strings.Contains(session.Logs, "login failed") {
		t.Fatalf("session.Logs = %q, want login failed", session.Logs)
	}
}

func waitForLoginStatus(t *testing.T, service *codexapp.Service, sessionID string, want codexapp.LoginSessionStatus) codexapp.LoginSession {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		session, ok := service.GetLoginSession(sessionID)
		if ok && session.Status == want {
			return session
		}
		time.Sleep(10 * time.Millisecond)
	}
	session, _ := service.GetLoginSession(sessionID)
	t.Fatalf("timed out waiting for login session %s to reach %s; last=%+v", sessionID, want, session)
	return codexapp.LoginSession{}
}

func envValue(values []string, key string) string {
	prefix := key + "="
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimPrefix(value, prefix)
		}
	}
	return ""
}

type serviceOAuthInput struct {
	Name      string
	Email     string
	UserID    string
	AccountID string
	Plan      string
	ExpiresAt time.Time
}

func buildServiceOAuthAuth(t *testing.T, input serviceOAuthInput) []byte {
	t.Helper()

	authClaims := map[string]any{
		"chatgpt_user_id":    input.UserID,
		"chatgpt_account_id": input.AccountID,
		"chatgpt_plan_type":  input.Plan,
	}
	jwtClaims := map[string]any{
		"name":                        input.Name,
		"email":                       input.Email,
		"sub":                         input.UserID,
		"exp":                         input.ExpiresAt.Unix(),
		"https://api.openai.com/auth": authClaims,
	}

	rawClaims, err := json.Marshal(jwtClaims)
	if err != nil {
		t.Fatalf("marshal jwt claims: %v", err)
	}
	token := "header." + base64.RawURLEncoding.EncodeToString(rawClaims) + ".sig"

	payload := map[string]any{
		"auth_mode": "oauth",
		"tokens": map[string]any{
			"id_token":      token,
			"access_token":  token,
			"refresh_token": "refresh-token",
			"account_id":    input.AccountID,
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal auth payload: %v", err)
	}
	return raw
}
