package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AuthFile struct {
	AuthMode      string      `json:"auth_mode,omitempty"`
	OpenAIAPIKey  interface{} `json:"OPENAI_API_KEY,omitempty"`
	BaseURL       string      `json:"base_url,omitempty"`
	APIBaseURL    string      `json:"api_base_url,omitempty"`
	APIBaseURLAlt string      `json:"apiBaseUrl,omitempty"`
	Tokens        *AuthTokens `json:"tokens,omitempty"`
	LastRefresh   interface{} `json:"last_refresh,omitempty"`
}

type AuthTokens struct {
	IDToken      string `json:"id_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
}

func (t AuthTokens) IsZero() bool {
	return strings.TrimSpace(t.IDToken) == "" &&
		strings.TrimSpace(t.AccessToken) == "" &&
		strings.TrimSpace(t.RefreshToken) == "" &&
		strings.TrimSpace(t.AccountID) == ""
}

type Snapshot struct {
	AuthMode      string    `json:"auth_mode"`
	AccountName   string    `json:"account_name,omitempty"`
	Email         string    `json:"email,omitempty"`
	UserID        string    `json:"user_id,omitempty"`
	AccountID     string    `json:"account_id,omitempty"`
	Plan          string    `json:"plan,omitempty"`
	BaseURL       string    `json:"base_url,omitempty"`
	IdentityKey   string    `json:"identity_key"`
	AuthHash      string    `json:"auth_hash"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	LastRefreshAt time.Time `json:"last_refresh_at,omitempty"`
}

type QuotaWindow struct {
	RemainingPercent int       `json:"remaining_percent"`
	ResetAt          time.Time `json:"reset_at,omitempty"`
}

type QuotaStatus struct {
	Hourly    QuotaWindow `json:"hourly"`
	Weekly    QuotaWindow `json:"weekly"`
	Plan      string      `json:"plan,omitempty"`
	Refreshed bool        `json:"refreshed"`
}

type jwtClaims struct {
	Name     string          `json:"name"`
	Email    string          `json:"email"`
	Sub      string          `json:"sub"`
	Exp      int64           `json:"exp"`
	AuthData json.RawMessage `json:"https://api.openai.com/auth"`
}

type authClaims struct {
	ChatGPTUserID string `json:"chatgpt_user_id"`
	UserID        string `json:"user_id"`
	AccountID     string `json:"account_id"`
	ChatGPTAccID  string `json:"chatgpt_account_id"`
	PlanType      string `json:"chatgpt_plan_type"`
}

func ParseAuthFile(raw []byte) (*AuthFile, error) {
	var auth AuthFile
	if err := json.Unmarshal(raw, &auth); err != nil {
		return nil, err
	}
	return &auth, nil
}

func SnapshotFromRawAuth(raw []byte) (*Snapshot, error) {
	auth, err := ParseAuthFile(raw)
	if err != nil {
		return nil, fmt.Errorf("parse auth.json: %w", err)
	}

	hash := sha256.Sum256(raw)
	snapshot := &Snapshot{
		AuthHash: hex.EncodeToString(hash[:]),
		BaseURL:  firstNonEmpty(auth.BaseURL, auth.APIBaseURL, auth.APIBaseURLAlt),
	}
	snapshot.LastRefreshAt = parseLastRefresh(auth.LastRefresh)

	authMode := normalizeAuthMode(auth)
	snapshot.AuthMode = authMode

	switch authMode {
	case "apikey":
		apiKey := extractAPIKey(auth.OpenAIAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("auth.json missing OPENAI_API_KEY")
		}
		snapshot.IdentityKey = "apikey:" + hashString(apiKey+"|"+strings.ToLower(snapshot.BaseURL))
		return snapshot, nil
	default:
		if auth.Tokens == nil {
			return nil, fmt.Errorf("auth.json missing tokens")
		}
		if strings.TrimSpace(auth.Tokens.IDToken) == "" && strings.TrimSpace(auth.Tokens.AccessToken) == "" {
			return nil, fmt.Errorf("auth.json missing usable token")
		}

		claims, _ := decodeJWTClaims(auth.Tokens.IDToken)
		if claims != nil {
			snapshot.AccountName = strings.TrimSpace(claims.Name)
			snapshot.Email = strings.TrimSpace(claims.Email)
			snapshot.UserID = strings.TrimSpace(claims.Sub)
			if claims.Exp > 0 {
				snapshot.ExpiresAt = time.Unix(claims.Exp, 0).UTC()
			}
		}

		authMeta, _ := decodeAuthClaims(claims)
		if authMeta != nil {
			snapshot.UserID = firstNonEmpty(authMeta.ChatGPTUserID, authMeta.UserID, snapshot.UserID)
			snapshot.AccountID = firstNonEmpty(auth.Tokens.AccountID, authMeta.ChatGPTAccID, authMeta.AccountID)
			snapshot.Plan = strings.TrimSpace(authMeta.PlanType)
		} else {
			snapshot.AccountID = strings.TrimSpace(auth.Tokens.AccountID)
		}

		switch {
		case snapshot.AccountID != "":
			snapshot.IdentityKey = "oauth:account:" + strings.ToLower(snapshot.AccountID)
		case snapshot.UserID != "":
			snapshot.IdentityKey = "oauth:user:" + strings.ToLower(snapshot.UserID)
		case snapshot.Email != "":
			snapshot.IdentityKey = "oauth:email:" + strings.ToLower(snapshot.Email)
		case strings.TrimSpace(auth.Tokens.RefreshToken) != "":
			snapshot.IdentityKey = "oauth:refresh:" + hashString(auth.Tokens.RefreshToken)
		default:
			snapshot.IdentityKey = "oauth:access:" + hashString(auth.Tokens.AccessToken)
		}
		return snapshot, nil
	}
}

func (s Snapshot) SuggestedName() string {
	if value := strings.TrimSpace(s.AccountName); value != "" {
		return value
	}
	if value := strings.TrimSpace(s.Email); value != "" {
		if at := strings.Index(value, "@"); at > 0 {
			return value[:at]
		}
		return value
	}
	if value := strings.TrimSpace(s.AccountID); value != "" {
		return value
	}
	if value := strings.TrimSpace(s.UserID); value != "" {
		return value
	}
	return "codex-account"
}

func (s Snapshot) IsNewerThan(other Snapshot) bool {
	if !s.LastRefreshAt.IsZero() || !other.LastRefreshAt.IsZero() {
		if s.LastRefreshAt.After(other.LastRefreshAt) {
			return true
		}
		if other.LastRefreshAt.After(s.LastRefreshAt) {
			return false
		}
	}
	if !s.ExpiresAt.IsZero() || !other.ExpiresAt.IsZero() {
		if s.ExpiresAt.After(other.ExpiresAt) {
			return true
		}
		if other.ExpiresAt.After(s.ExpiresAt) {
			return false
		}
	}
	return false
}

func TokenExpiry(token string) time.Time {
	claims, err := decodeJWTClaims(token)
	if err != nil || claims == nil || claims.Exp <= 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0).UTC()
}

func normalizeAuthMode(auth *AuthFile) string {
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

func decodeJWTClaims(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid jwt")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

func decodeAuthClaims(claims *jwtClaims) (*authClaims, error) {
	if claims == nil || len(claims.AuthData) == 0 {
		return nil, nil
	}
	var auth authClaims
	if err := json.Unmarshal(claims.AuthData, &auth); err != nil {
		return nil, err
	}
	return &auth, nil
}

func hashString(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
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

func parseLastRefresh(value interface{}) time.Time {
	text, ok := value.(string)
	if !ok {
		return time.Time{}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
