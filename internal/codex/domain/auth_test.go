package domain

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSnapshotFromRawAuthParsesOAuthIdentity(t *testing.T) {
	raw := buildOAuthAuth(t, oauthAuthInput{
		Name:      "Work Account",
		Email:     "work@example.com",
		UserID:    "user-work",
		AccountID: "acct-work",
		Plan:      "plus",
		ExpiresAt: time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC),
	})

	snapshot, err := SnapshotFromRawAuth(raw)
	if err != nil {
		t.Fatalf("SnapshotFromRawAuth returned error: %v", err)
	}

	if snapshot.AuthMode != "oauth" {
		t.Fatalf("AuthMode = %q, want oauth", snapshot.AuthMode)
	}
	if snapshot.AccountName != "Work Account" {
		t.Fatalf("AccountName = %q, want Work Account", snapshot.AccountName)
	}
	if snapshot.Email != "work@example.com" {
		t.Fatalf("Email = %q, want work@example.com", snapshot.Email)
	}
	if snapshot.UserID != "user-work" {
		t.Fatalf("UserID = %q, want user-work", snapshot.UserID)
	}
	if snapshot.AccountID != "acct-work" {
		t.Fatalf("AccountID = %q, want acct-work", snapshot.AccountID)
	}
	if snapshot.Plan != "plus" {
		t.Fatalf("Plan = %q, want plus", snapshot.Plan)
	}
	if snapshot.IdentityKey != "oauth:account:acct-work" {
		t.Fatalf("IdentityKey = %q, want oauth:account:acct-work", snapshot.IdentityKey)
	}
	if snapshot.ExpiresAt.IsZero() {
		t.Fatalf("ExpiresAt should be populated")
	}
	if snapshot.AuthHash == "" {
		t.Fatalf("AuthHash should not be empty")
	}
}

func TestSnapshotFromRawAuthParsesAPIKeyIdentity(t *testing.T) {
	raw := []byte(`{"auth_mode":"apikey","OPENAI_API_KEY":"sk-test","base_url":"https://api.example.com"}`)

	snapshot, err := SnapshotFromRawAuth(raw)
	if err != nil {
		t.Fatalf("SnapshotFromRawAuth returned error: %v", err)
	}

	if snapshot.AuthMode != "apikey" {
		t.Fatalf("AuthMode = %q, want apikey", snapshot.AuthMode)
	}
	if !strings.HasPrefix(snapshot.IdentityKey, "apikey:") {
		t.Fatalf("IdentityKey = %q, want apikey:*", snapshot.IdentityKey)
	}
	if snapshot.BaseURL != "https://api.example.com" {
		t.Fatalf("BaseURL = %q, want https://api.example.com", snapshot.BaseURL)
	}
}

func TestSnapshotSuggestedNamePrefersAccountName(t *testing.T) {
	snapshot := Snapshot{
		AccountName: "  Work Account  ",
		Email:       "work@example.com",
		AccountID:   "acct-work",
		UserID:      "user-work",
	}

	if got := snapshot.SuggestedName(); got != "Work Account" {
		t.Fatalf("SuggestedName = %q, want Work Account", got)
	}
}

type oauthAuthInput struct {
	Name      string
	Email     string
	UserID    string
	AccountID string
	Plan      string
	ExpiresAt time.Time
}

func buildOAuthAuth(t *testing.T, input oauthAuthInput) []byte {
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
	idToken := "header." + base64.RawURLEncoding.EncodeToString(rawClaims) + ".sig"

	payload := map[string]any{
		"auth_mode": "oauth",
		"tokens": map[string]any{
			"id_token":      idToken,
			"access_token":  idToken,
			"refresh_token": "refresh-token",
			"account_id":    input.AccountID,
		},
		"last_refresh": time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal auth payload: %v", err)
	}
	return raw
}
