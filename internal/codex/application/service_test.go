package application_test

import (
	"bufio"
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
	if active.Quota == nil || active.Quota.Plan != "enterprise" {
		t.Fatalf("active quota = %+v, want enterprise plan", active.Quota)
	}
	if !active.Refreshed {
		t.Fatalf("expected active quota to be marked refreshed")
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

func TestServiceListStatusesRefreshesQuotaForUnmanagedCurrentAuth(t *testing.T) {
	activeHome := filepath.Join(t.TempDir(), ".codex")
	if err := os.MkdirAll(activeHome, 0o755); err != nil {
		t.Fatalf("mkdir active home: %v", err)
	}
	raw := buildServiceOAuthAuth(t, serviceOAuthInput{
		Name:      "CLI Account",
		Email:     "cli@example.com",
		UserID:    "user-cli",
		AccountID: "acct-cli",
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
	refreshedRaw := buildServiceOAuthAuth(t, serviceOAuthInput{
		Name:      "CLI Account",
		Email:     "cli@example.com",
		UserID:    "user-cli",
		AccountID: "acct-cli",
		Plan:      "enterprise",
		ExpiresAt: time.Date(2026, 4, 19, 8, 0, 0, 0, time.UTC),
	})
	service := codexapp.NewService(codexapp.ServiceOptions{
		Store:             store,
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		QueryQuota: func(rawAuth []byte, _ codexapp.QuotaQueryOptions) (*codexdomain.QuotaStatus, []byte, error) {
			if !bytes.Equal(rawAuth, raw) {
				t.Fatalf("unexpected raw auth payload")
			}
			return &codexdomain.QuotaStatus{
				Hourly: codexdomain.QuotaWindow{RemainingPercent: 61},
				Weekly: codexdomain.QuotaWindow{RemainingPercent: 84},
				Plan:   "enterprise",
				Refreshed: true,
			}, refreshedRaw, nil
		},
	})

	statuses, active, err := service.ListStatuses(context.Background())
	if err != nil {
		t.Fatalf("ListStatuses returned error: %v", err)
	}
	if len(statuses) != 0 {
		t.Fatalf("len(statuses) = %d, want 0", len(statuses))
	}
	if active == nil || active.Live == nil || active.Live.AccountName != "CLI Account" {
		t.Fatalf("active live account = %+v, want CLI Account", active)
	}
	if active.Quota == nil || active.Quota.Hourly.RemainingPercent != 61 || active.Quota.Weekly.RemainingPercent != 84 {
		t.Fatalf("active quota = %+v, want live quota", active.Quota)
	}
	if active.Live.Plan != "enterprise" {
		t.Fatalf("active live plan = %q, want enterprise", active.Live.Plan)
	}
	gotRaw, err := os.ReadFile(codexapp.AuthFilePath(activeHome))
	if err != nil {
		t.Fatalf("read active auth: %v", err)
	}
	if !bytes.Equal(gotRaw, refreshedRaw) {
		t.Fatalf("active auth was not refreshed")
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

func TestServiceRuntimeStatusReadsModelsAndReasoningFromAppServer(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(activeHome, "config.toml"), []byte("model = \"gpt-5.4\"\n"), 0o600); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}

	store, err := localcodex.NewStore(filepath.Join(t.TempDir(), "accounts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	service := codexapp.NewService(codexapp.ServiceOptions{
		Store:             store,
		Command:           "/usr/local/bin/codex",
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		RunCommand: func(_ context.Context, name string, args []string, options codexapp.CommandOptions) error {
			if name != "/usr/local/bin/codex" {
				t.Fatalf("command name = %q", name)
			}
			if len(args) < 1 || args[0] != "app-server" {
				t.Fatalf("command args = %v, want app-server", args)
			}
			reqs := decodeInteractiveAppServerRequests(t, options.Stdin, options.Stdout)
			method := appServerMethod(reqs)
			switch method {
			case "model/list":
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"data": []map[string]any{
						{
							"id":                     "gpt-5.4",
							"model":                  "gpt-5.4",
							"displayName":            "gpt-5.4",
							"description":            "Strong model for everyday coding.",
							"hidden":                 false,
							"isDefault":              true,
							"defaultReasoningEffort": "medium",
							"supportedReasoningEfforts": []map[string]any{
								{"reasoningEffort": "low", "description": "Fast"},
								{"reasoningEffort": "medium", "description": "Balanced"},
								{"reasoningEffort": "high", "description": "Deep"},
							},
							"inputModalities": []string{"text", "image"},
						},
						{
							"id":                     "gpt-5.4-mini",
							"model":                  "gpt-5.4-mini",
							"displayName":            "GPT-5.4-Mini",
							"description":            "Fast model.",
							"hidden":                 false,
							"isDefault":              false,
							"defaultReasoningEffort": "medium",
							"supportedReasoningEfforts": []map[string]any{
								{"reasoningEffort": "low", "description": "Fast"},
								{"reasoningEffort": "medium", "description": "Balanced"},
							},
							"inputModalities": []string{"text"},
						},
					},
				})
			case "config/read":
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"config": map[string]any{
						"model":                  "gpt-5.4",
						"model_reasoning_effort": "high",
						"profile":                "auto-max",
					},
					"origins": map[string]any{
						"model": map[string]any{
							"name":    map[string]any{"type": "user", "file": filepath.Join(activeHome, "config.toml")},
							"version": "sha256:test-version",
						},
						"model_reasoning_effort": map[string]any{
							"name":    map[string]any{"type": "user", "file": filepath.Join(activeHome, "config.toml")},
							"version": "sha256:test-version",
						},
					},
				})
			default:
				t.Fatalf("unexpected app-server method %q", method)
			}
			return nil
		},
	})
	if _, err := service.AddFromRaw("work", raw, false); err != nil {
		t.Fatalf("AddFromRaw returned error: %v", err)
	}

	status, err := service.RuntimeStatus()
	if err != nil {
		t.Fatalf("RuntimeStatus returned error: %v", err)
	}
	if status.Command != "/usr/local/bin/codex" {
		t.Fatalf("status.Command = %q, want /usr/local/bin/codex", status.Command)
	}
	if !status.HasAuth {
		t.Fatalf("expected HasAuth = true")
	}
	if !status.HasConfig {
		t.Fatalf("expected HasConfig = true")
	}
	if status.AuthPath != filepath.Join(activeHome, "auth.json") {
		t.Fatalf("status.AuthPath = %q", status.AuthPath)
	}
	if status.ConfigPath != filepath.Join(activeHome, "config.toml") {
		t.Fatalf("status.ConfigPath = %q", status.ConfigPath)
	}
	if status.Model != "gpt-5.4" {
		t.Fatalf("status.Model = %q, want gpt-5.4", status.Model)
	}
	if status.ReasoningEffort != "high" {
		t.Fatalf("status.ReasoningEffort = %q, want high", status.ReasoningEffort)
	}
	if status.Profile != "auto-max" {
		t.Fatalf("status.Profile = %q, want auto-max", status.Profile)
	}
	if status.ModelOrigin == nil || status.ModelOrigin.KeyPath != "model" {
		t.Fatalf("status.ModelOrigin = %+v", status.ModelOrigin)
	}
	if len(status.Models) != 2 {
		t.Fatalf("len(status.Models) = %d, want 2", len(status.Models))
	}
	if len(status.Models[0].SupportedReasoningEffort) != 3 {
		t.Fatalf("status.Models[0].SupportedReasoningEffort = %+v", status.Models[0].SupportedReasoningEffort)
	}
	if status.Current == nil || status.Current.Managed == nil || status.Current.Managed.Name != "work" {
		t.Fatalf("status.Current = %+v, want managed account work", status.Current)
	}
}

func TestServiceRuntimeStatusInitializesAppServerBeforeModelList(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(activeHome, "config.toml"), []byte("model = \"gpt-5.4\"\n"), 0o600); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}

	store, err := localcodex.NewStore(filepath.Join(t.TempDir(), "accounts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	service := codexapp.NewService(codexapp.ServiceOptions{
		Store:             store,
		Command:           "/usr/local/bin/codex",
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		RunCommand: func(_ context.Context, name string, args []string, options codexapp.CommandOptions) error {
			if name != "/usr/local/bin/codex" {
				t.Fatalf("command name = %q", name)
			}
			reqs := decodeInteractiveAppServerRequests(t, options.Stdin, options.Stdout)
			method := appServerMethod(reqs)
			writeAppServerResponse(t, options.Stdout, 1, map[string]any{
				"userAgent": "alter0/0.114.0",
			})
			if !appServerRequestsInitialized(reqs) {
				return nil
			}
			switch method {
			case "model/list":
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"data": []map[string]any{
						{
							"id":                     "gpt-5.4",
							"model":                  "gpt-5.4",
							"displayName":            "gpt-5.4",
							"description":            "Strong model for everyday coding.",
							"hidden":                 false,
							"isDefault":              true,
							"defaultReasoningEffort": "medium",
							"supportedReasoningEfforts": []map[string]any{
								{"reasoningEffort": "low", "description": "Fast"},
								{"reasoningEffort": "medium", "description": "Balanced"},
								{"reasoningEffort": "high", "description": "Deep"},
							},
							"inputModalities": []string{"text"},
						},
					},
				})
			case "config/read":
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"config": map[string]any{
						"model":                  "gpt-5.4",
						"model_reasoning_effort": "medium",
					},
					"origins": map[string]any{
						"model": map[string]any{
							"name":    map[string]any{"type": "user", "file": filepath.Join(activeHome, "config.toml")},
							"version": "sha256:test-version",
						},
						"model_reasoning_effort": map[string]any{
							"name":    map[string]any{"type": "user", "file": filepath.Join(activeHome, "config.toml")},
							"version": "sha256:test-version",
						},
					},
				})
			default:
				t.Fatalf("unexpected app-server method %q", method)
			}
			return nil
		},
	})
	if _, err := service.AddFromRaw("work", raw, false); err != nil {
		t.Fatalf("AddFromRaw returned error: %v", err)
	}

	status, err := service.RuntimeStatus()
	if err != nil {
		t.Fatalf("RuntimeStatus returned error: %v", err)
	}
	if status.Model != "gpt-5.4" {
		t.Fatalf("status.Model = %q, want gpt-5.4", status.Model)
	}
	if len(status.Models) != 1 || status.Models[0].ID != "gpt-5.4" {
		t.Fatalf("status.Models = %+v, want gpt-5.4", status.Models)
	}
}

func TestServiceRuntimeStatusWaitsForInitializeResponseBeforeNextRequest(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(activeHome, "config.toml"), []byte("model = \"gpt-5.4\"\n"), 0o600); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}

	store, err := localcodex.NewStore(filepath.Join(t.TempDir(), "accounts"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	service := codexapp.NewService(codexapp.ServiceOptions{
		Store:             store,
		Command:           "/usr/local/bin/codex",
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		RunCommand: func(_ context.Context, name string, args []string, options codexapp.CommandOptions) error {
			if name != "/usr/local/bin/codex" {
				t.Fatalf("command name = %q", name)
			}
			reader := bufio.NewReader(options.Stdin)
			initializeLine, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("read initialize request: %v", err)
			}
			initialize := map[string]any{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(initializeLine)), &initialize); err != nil {
				t.Fatalf("decode initialize request: %v", err)
			}
			if method, _ := initialize["method"].(string); method != "initialize" {
				t.Fatalf("initialize method = %q, want initialize", method)
			}

			nextLineCh := make(chan string, 1)
			nextErrCh := make(chan error, 1)
			go func() {
				line, readErr := reader.ReadString('\n')
				if readErr != nil {
					nextErrCh <- readErr
					return
				}
				nextLineCh <- line
			}()

			select {
			case line := <-nextLineCh:
				t.Fatalf("received next request before initialize response: %s", strings.TrimSpace(line))
			case readErr := <-nextErrCh:
				t.Fatalf("read next request before initialize response: %v", readErr)
			case <-time.After(20 * time.Millisecond):
			}

			writeAppServerResponse(t, options.Stdout, 1, map[string]any{
				"userAgent": "alter0/0.114.0",
			})

			initializedLine := <-nextLineCh
			requestLine, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("read app-server request: %v", err)
			}
			initialized := map[string]any{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(initializedLine)), &initialized); err != nil {
				t.Fatalf("decode initialized notification: %v", err)
			}
			request := map[string]any{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(requestLine)), &request); err != nil {
				t.Fatalf("decode app-server request: %v", err)
			}
			if method, _ := initialized["method"].(string); method != "notifications/initialized" {
				t.Fatalf("initialized method = %q, want notifications/initialized", method)
			}
			switch method, _ := request["method"].(string); method {
			case "model/list":
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"data": []map[string]any{
						{
							"id":                     "gpt-5.4",
							"model":                  "gpt-5.4",
							"displayName":            "gpt-5.4",
							"description":            "Strong model",
							"hidden":                 false,
							"isDefault":              true,
							"defaultReasoningEffort": "medium",
							"supportedReasoningEfforts": []map[string]any{
								{"reasoningEffort": "low", "description": "Fast"},
								{"reasoningEffort": "medium", "description": "Balanced"},
							},
							"inputModalities": []string{"text"},
						},
					},
				})
			case "config/read":
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"config": map[string]any{
						"model":                  "gpt-5.4",
						"model_reasoning_effort": "medium",
					},
					"origins": map[string]any{
						"model": map[string]any{
							"name":    map[string]any{"type": "user", "file": filepath.Join(activeHome, "config.toml")},
							"version": "sha256:test-version",
						},
						"model_reasoning_effort": map[string]any{
							"name":    map[string]any{"type": "user", "file": filepath.Join(activeHome, "config.toml")},
							"version": "sha256:test-version",
						},
					},
				})
			default:
				t.Fatalf("unexpected app-server method %q", method)
			}
			return nil
		},
	})
	if _, err := service.AddFromRaw("work", raw, false); err != nil {
		t.Fatalf("AddFromRaw returned error: %v", err)
	}

	status, err := service.RuntimeStatus()
	if err != nil {
		t.Fatalf("RuntimeStatus returned error: %v", err)
	}
	if status.Model != "gpt-5.4" {
		t.Fatalf("status.Model = %q, want gpt-5.4", status.Model)
	}
}

func TestServiceUpdateRuntimeSettingsUsesAppServerBatchWrite(t *testing.T) {
	activeHome := filepath.Join(t.TempDir(), ".codex")
	if err := os.MkdirAll(activeHome, 0o755); err != nil {
		t.Fatalf("mkdir active home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(activeHome, "config.toml"), []byte("profile = \"auto-max\"\n"), 0o600); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}
	currentModel := "gpt-5.4"
	currentEffort := "high"
	var gotEdits []map[string]any

	service := codexapp.NewService(codexapp.ServiceOptions{
		Command:           "codex",
		ResolveActiveHome: func() (string, error) { return activeHome, nil },
		RunCommand: func(_ context.Context, name string, args []string, options codexapp.CommandOptions) error {
			if name != "codex" {
				t.Fatalf("command name = %q", name)
			}
			reqs := decodeInteractiveAppServerRequests(t, options.Stdin, options.Stdout)
			method := appServerMethod(reqs)
			switch method {
			case "config/read":
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"config": map[string]any{
						"model":                  currentModel,
						"model_reasoning_effort": currentEffort,
						"profile":                "auto-max",
					},
					"origins": map[string]any{
						"model": map[string]any{
							"name":    map[string]any{"type": "user", "file": filepath.Join(activeHome, "config.toml")},
							"version": "sha256:before",
						},
						"model_reasoning_effort": map[string]any{
							"name":    map[string]any{"type": "user", "file": filepath.Join(activeHome, "config.toml")},
							"version": "sha256:before",
						},
					},
				})
			case "config/batchWrite":
				params := reqs[len(reqs)-1]["params"].(map[string]any)
				gotEdits = anySliceToMaps(t, params["edits"])
				currentModel = gotEdits[0]["value"].(string)
				currentEffort = gotEdits[1]["value"].(string)
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"status":   "ok",
					"version":  "sha256:after",
					"filePath": filepath.Join(activeHome, "config.toml"),
				})
			case "model/list":
				writeAppServerResponse(t, options.Stdout, 2, map[string]any{
					"data": []map[string]any{
						{
							"id":                     "gpt-5.4",
							"model":                  "gpt-5.4",
							"displayName":            "gpt-5.4",
							"description":            "Strong model",
							"hidden":                 false,
							"isDefault":              true,
							"defaultReasoningEffort": "medium",
							"supportedReasoningEfforts": []map[string]any{
								{"reasoningEffort": "low", "description": "Fast"},
								{"reasoningEffort": "medium", "description": "Balanced"},
								{"reasoningEffort": "high", "description": "Deep"},
							},
							"inputModalities": []string{"text", "image"},
						},
						{
							"id":                     "gpt-5.4-mini",
							"model":                  "gpt-5.4-mini",
							"displayName":            "GPT-5.4-Mini",
							"description":            "Fast model",
							"hidden":                 false,
							"isDefault":              false,
							"defaultReasoningEffort": "medium",
							"supportedReasoningEfforts": []map[string]any{
								{"reasoningEffort": "low", "description": "Fast"},
								{"reasoningEffort": "medium", "description": "Balanced"},
							},
							"inputModalities": []string{"text"},
						},
					},
				})
			default:
				t.Fatalf("unexpected app-server method %q", method)
			}
			return nil
		},
	})

	status, err := service.UpdateRuntimeSettings("gpt-5.4-mini", "medium")
	if err != nil {
		t.Fatalf("UpdateRuntimeSettings returned error: %v", err)
	}
	if len(gotEdits) != 2 {
		t.Fatalf("gotEdits = %+v, want 2 edits", gotEdits)
	}
	if gotEdits[0]["keyPath"] != "model" || gotEdits[1]["keyPath"] != "model_reasoning_effort" {
		t.Fatalf("gotEdits keyPath = %+v", gotEdits)
	}
	if status.Model != "gpt-5.4-mini" {
		t.Fatalf("status.Model = %q, want gpt-5.4-mini", status.Model)
	}
	if status.ReasoningEffort != "medium" {
		t.Fatalf("status.ReasoningEffort = %q, want medium", status.ReasoningEffort)
	}
	if currentModel != "gpt-5.4-mini" || currentEffort != "medium" {
		t.Fatalf("current settings = (%q, %q), want (gpt-5.4-mini, medium)", currentModel, currentEffort)
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

func decodeAppServerRequests(t *testing.T, reader io.Reader) []map[string]any {
	t.Helper()
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read app-server stdin: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	requests := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("decode app-server request %q: %v", line, err)
		}
		requests = append(requests, payload)
	}
	return requests
}

func decodeInteractiveAppServerRequests(t *testing.T, reader io.Reader, writer io.Writer) []map[string]any {
	t.Helper()
	buffered := bufio.NewReader(reader)
	decodeRequestLine := func(line string) map[string]any {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			t.Fatalf("app-server request line is empty")
		}
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			t.Fatalf("decode app-server request %q: %v", trimmed, err)
		}
		return payload
	}
	readRequestLine := func(label string) map[string]any {
		line, err := buffered.ReadString('\n')
		if err != nil {
			t.Fatalf("read %s request: %v", label, err)
		}
		return decodeRequestLine(line)
	}

	initialize := readRequestLine("initialize")
	writeAppServerResponse(t, writer, 1, map[string]any{
		"userAgent": "alter0/0.114.0",
	})
	initialized := readRequestLine("initialized")
	request := readRequestLine("method")
	return []map[string]any{initialize, initialized, request}
}

func appServerMethod(requests []map[string]any) string {
	if len(requests) == 0 {
		return ""
	}
	method, _ := requests[len(requests)-1]["method"].(string)
	return method
}

func appServerRequestsInitialized(requests []map[string]any) bool {
	if len(requests) < 3 {
		return false
	}
	initializeMethod, _ := requests[0]["method"].(string)
	notificationMethod, _ := requests[1]["method"].(string)
	requestMethod, _ := requests[2]["method"].(string)
	return initializeMethod == "initialize" &&
		notificationMethod == "notifications/initialized" &&
		requestMethod != ""
}

func writeAppServerResponse(t *testing.T, writer io.Writer, id int, payload any) {
	t.Helper()
	message, err := json.Marshal(map[string]any{
		"id":     id,
		"result": payload,
	})
	if err != nil {
		t.Fatalf("marshal app-server response: %v", err)
	}
	if _, err := io.WriteString(writer, string(message)+"\n"); err != nil {
		t.Fatalf("write app-server response: %v", err)
	}
}

func anySliceToMaps(t *testing.T, value any) []map[string]any {
	t.Helper()
	rawItems, ok := value.([]any)
	if !ok {
		t.Fatalf("value = %T, want []any", value)
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, item := range rawItems {
		entry, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("item = %T, want map[string]any", item)
		}
		items = append(items, entry)
	}
	return items
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
