package web

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	codexapp "alter0/internal/codex/application"
	codexdomain "alter0/internal/codex/domain"
)

type stubCodexAccountService struct {
	listStatuses      func(ctx context.Context) ([]codexapp.AccountStatus, *codexapp.CurrentStatus, error)
	addFromRaw        func(name string, raw []byte, overwrite bool) (*codexapp.Record, error)
	switchAccount     func(name string) (*codexapp.Record, string, error)
	startLoginSession func(ctx context.Context, name string, overwrite bool) (codexapp.LoginSession, error)
	getLoginSession   func(id string) (codexapp.LoginSession, bool)
}

func (s *stubCodexAccountService) ListStatuses(ctx context.Context) ([]codexapp.AccountStatus, *codexapp.CurrentStatus, error) {
	if s.listStatuses != nil {
		return s.listStatuses(ctx)
	}
	return nil, nil, nil
}

func (s *stubCodexAccountService) AddFromRaw(name string, raw []byte, overwrite bool) (*codexapp.Record, error) {
	if s.addFromRaw != nil {
		return s.addFromRaw(name, raw, overwrite)
	}
	return nil, nil
}

func (s *stubCodexAccountService) Switch(name string) (*codexapp.Record, string, error) {
	if s.switchAccount != nil {
		return s.switchAccount(name)
	}
	return nil, "", nil
}

func (s *stubCodexAccountService) StartLoginSession(ctx context.Context, name string, overwrite bool) (codexapp.LoginSession, error) {
	if s.startLoginSession != nil {
		return s.startLoginSession(ctx, name, overwrite)
	}
	return codexapp.LoginSession{}, nil
}

func (s *stubCodexAccountService) GetLoginSession(id string) (codexapp.LoginSession, bool) {
	if s.getLoginSession != nil {
		return s.getLoginSession(id)
	}
	return codexapp.LoginSession{}, false
}

func TestCodexAccountCollectionHandlerListsAccounts(t *testing.T) {
	server := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		codexAccounts: &stubCodexAccountService{
			listStatuses: func(context.Context) ([]codexapp.AccountStatus, *codexapp.CurrentStatus, error) {
				record := codexapp.Record{
					Name: "work",
					Snapshot: codexdomain.Snapshot{
						AccountName: "Work Account",
						AccountID:   "acct-work",
						Plan:        "plus",
					},
				}
				return []codexapp.AccountStatus{
						{
							Record:  record,
							Current: true,
							Quota: &codexdomain.QuotaStatus{
								Hourly: codexdomain.QuotaWindow{RemainingPercent: 80},
								Weekly: codexdomain.QuotaWindow{RemainingPercent: 92},
								Plan:   "plus",
							},
						},
					}, &codexapp.CurrentStatus{
						Managed:  &record,
						AuthPath: "/var/lib/alter0/.codex/auth.json",
					}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/codex/accounts", nil)
	rec := httptest.NewRecorder()
	server.codexAccountCollectionHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Items  []codexapp.AccountStatus `json:"items"`
		Active *codexapp.CurrentStatus  `json:"active"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].Record.Name != "work" {
		t.Fatalf("unexpected items: %+v", payload.Items)
	}
	if payload.Active == nil || payload.Active.Managed == nil || payload.Active.Managed.Name != "work" {
		t.Fatalf("unexpected active: %+v", payload.Active)
	}
}

func TestCodexAccountCollectionHandlerCreatesAccountFromAuthFile(t *testing.T) {
	var gotName string
	var gotRaw string
	var gotOverwrite bool
	server := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		codexAccounts: &stubCodexAccountService{
			addFromRaw: func(name string, raw []byte, overwrite bool) (*codexapp.Record, error) {
				gotName = name
				gotRaw = string(raw)
				gotOverwrite = overwrite
				return &codexapp.Record{Name: "work"}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/codex/accounts", strings.NewReader(`{
		"name":"work",
		"overwrite":true,
		"auth_file_content":"{\"auth_mode\":\"apikey\",\"OPENAI_API_KEY\":\"sk-test\"}"
	}`))
	rec := httptest.NewRecorder()
	server.codexAccountCollectionHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotName != "work" {
		t.Fatalf("gotName = %q, want work", gotName)
	}
	if gotRaw != `{"auth_mode":"apikey","OPENAI_API_KEY":"sk-test"}` {
		t.Fatalf("gotRaw = %q", gotRaw)
	}
	if !gotOverwrite {
		t.Fatalf("expected overwrite to be true")
	}
}

func TestCodexAccountLoginSessionHandlersStartAndReadSession(t *testing.T) {
	server := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		codexAccounts: &stubCodexAccountService{
			startLoginSession: func(context.Context, string, bool) (codexapp.LoginSession, error) {
				return codexapp.LoginSession{
					ID:          "login-1",
					AccountName: "fresh",
					Status:      codexapp.LoginSessionStatusRunning,
					StartedAt:   time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC),
				}, nil
			},
			getLoginSession: func(id string) (codexapp.LoginSession, bool) {
				if id != "login-1" {
					return codexapp.LoginSession{}, false
				}
				return codexapp.LoginSession{
					ID:          "login-1",
					AccountName: "fresh",
					Status:      codexapp.LoginSessionStatusSucceeded,
					Logs:        "open browser",
				}, true
			},
		},
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/control/codex/accounts/login-sessions", strings.NewReader(`{"name":"fresh","overwrite":false}`))
	startRec := httptest.NewRecorder()
	server.codexAccountLoginSessionCollectionHandler(startRec, startReq)

	if startRec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d: %s", startRec.Code, startRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/control/codex/accounts/login-sessions/login-1", nil)
	getRec := httptest.NewRecorder()
	server.codexAccountItemHandler(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	var session codexapp.LoginSession
	if err := json.NewDecoder(getRec.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if session.Status != codexapp.LoginSessionStatusSucceeded {
		t.Fatalf("session.Status = %q, want %q", session.Status, codexapp.LoginSessionStatusSucceeded)
	}
}

func TestCodexAccountItemHandlerSwitchesAccount(t *testing.T) {
	var switchedName string
	server := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		codexAccounts: &stubCodexAccountService{
			switchAccount: func(name string) (*codexapp.Record, string, error) {
				switchedName = name
				return &codexapp.Record{Name: name}, "/tmp/auth-backup.json", nil
			},
			listStatuses: func(context.Context) ([]codexapp.AccountStatus, *codexapp.CurrentStatus, error) {
				record := codexapp.Record{Name: "work"}
				return nil, &codexapp.CurrentStatus{Managed: &record}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/control/codex/accounts/work/switch", nil)
	rec := httptest.NewRecorder()
	server.codexAccountItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if switchedName != "work" {
		t.Fatalf("switchedName = %q, want work", switchedName)
	}
	var payload struct {
		Account    codexapp.Record         `json:"account"`
		BackupPath string                  `json:"backup_path"`
		Active     *codexapp.CurrentStatus `json:"active"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode switch response: %v", err)
	}
	if payload.Account.Name != "work" {
		t.Fatalf("payload.Account.Name = %q, want work", payload.Account.Name)
	}
	if payload.BackupPath != "/tmp/auth-backup.json" {
		t.Fatalf("payload.BackupPath = %q, want /tmp/auth-backup.json", payload.BackupPath)
	}
}
