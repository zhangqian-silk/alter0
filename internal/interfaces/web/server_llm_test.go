package web

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	llmapp "alter0/internal/llm/application"
	llmdomain "alter0/internal/llm/domain"
	llminfra "alter0/internal/llm/infrastructure"
)

func TestLLMProviderListReturnsDisabledProviders(t *testing.T) {
	service := newTestLLMService(t)
	ctx := context.Background()
	if err := service.AddProvider(ctx, llmdomain.ModelProvider{
		ID:        "enabled-provider",
		Name:      "Enabled Provider",
		BaseURL:   "https://enabled.example/v1",
		APIKey:    "sk-enabled",
		IsEnabled: true,
		Models: []llmdomain.ModelInfo{
			{ID: "enabled-model", Name: "Enabled Model", IsEnabled: true},
		},
		DefaultModel: "enabled-model",
	}); err != nil {
		t.Fatalf("add enabled provider failed: %v", err)
	}
	if err := service.AddProvider(ctx, llmdomain.ModelProvider{
		ID:        "disabled-provider",
		Name:      "Disabled Provider",
		BaseURL:   "https://disabled.example/v1",
		APIKey:    "sk-disabled",
		IsEnabled: false,
		Models: []llmdomain.ModelInfo{
			{ID: "disabled-model", Name: "Disabled Model", IsEnabled: true},
		},
		DefaultModel: "disabled-model",
	}); err != nil {
		t.Fatalf("add disabled provider failed: %v", err)
	}

	server := &Server{
		llm:    service,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/control/llm/providers", nil)
	rec := httptest.NewRecorder()
	server.llmProviderListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Items []llmProviderResponse `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(resp.Items))
	}
}

func TestLLMProviderUpdateKeepsAPIKeyWhenBlank(t *testing.T) {
	service := newTestLLMService(t)
	ctx := context.Background()
	if err := service.AddProvider(ctx, llmdomain.ModelProvider{
		ID:        "qwen",
		Name:      "Qwen",
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-original",
		IsEnabled: true,
		Models: []llmdomain.ModelInfo{
			{ID: "qwen-plus", Name: "Qwen Plus", IsEnabled: true},
		},
		DefaultModel: "qwen-plus",
	}); err != nil {
		t.Fatalf("add provider failed: %v", err)
	}

	server := &Server{
		llm:    service,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/control/llm/providers/qwen",
		strings.NewReader(`{"name":"Qwen Updated","base_url":"https://example.com/v2","api_key":"","default_model":"qwen-plus","models":[{"id":"qwen-plus","name":"Qwen Plus","is_enabled":true}],"is_enabled":true}`),
	)
	rec := httptest.NewRecorder()
	server.llmProviderItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	provider, err := service.GetProvider(ctx, "qwen")
	if err != nil {
		t.Fatalf("get provider failed: %v", err)
	}
	if provider == nil {
		t.Fatalf("expected provider")
	}
	if provider.APIKey != "sk-original" {
		t.Fatalf("expected api key to be preserved, got %s", provider.APIKey)
	}
}

func newTestLLMService(t *testing.T) *llmapp.ModelConfigService {
	t.Helper()
	storage := llminfra.NewModelConfigStorage(filepath.Join(t.TempDir(), "model_config.json"))
	return llmapp.NewModelConfigService(storage)
}
