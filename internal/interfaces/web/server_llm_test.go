package web

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
		APIType:   llmdomain.ProviderAPITypeOpenAIResponses,
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
		APIType:   llmdomain.ProviderAPITypeOpenAIResponses,
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
		APIType:   llmdomain.ProviderAPITypeOpenAIResponses,
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
		strings.NewReader(`{"id":"qwen","name":"Qwen Updated","api_type":"openai-completions","base_url":"https://example.com/v2","api_key":"","default_model":"qwen-plus","models":[{"id":"qwen-plus","name":"Qwen Plus","is_enabled":true}],"is_enabled":true}`),
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
	if provider.APIType != llmdomain.ProviderAPITypeOpenAICompletions {
		t.Fatalf("expected api type to update, got %s", provider.APIType)
	}
}

func TestLLMProviderUpdateKeepsAPIKeyWhenPlaceholderDash(t *testing.T) {
	service := newTestLLMService(t)
	ctx := context.Background()
	if err := service.AddProvider(ctx, llmdomain.ModelProvider{
		ID:        "openrouter",
		Name:      "OpenRouter",
		APIType:   llmdomain.ProviderAPITypeOpenAICompletions,
		BaseURL:   "https://openrouter.ai/api/v1",
		APIKey:    "sk-or-original",
		IsEnabled: true,
		Models: []llmdomain.ModelInfo{
			{ID: "openai/gpt-5.4", Name: "GPT-5.4", IsEnabled: true},
		},
		DefaultModel: "openai/gpt-5.4",
	}); err != nil {
		t.Fatalf("add provider failed: %v", err)
	}

	server := &Server{
		llm:    service,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/control/llm/providers/openrouter",
		strings.NewReader(`{"id":"openrouter","name":"OpenRouter Updated","provider_type":"openrouter","api_type":"openai-completions","base_url":"https://openrouter.ai/api/v1","api_key":"-","default_model":"openai/gpt-5.4","models":[{"id":"openai/gpt-5.4","name":"GPT-5.4","is_enabled":true}],"is_enabled":true}`),
	)
	rec := httptest.NewRecorder()
	server.llmProviderItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	provider, err := service.GetProvider(ctx, "openrouter")
	if err != nil {
		t.Fatalf("get provider failed: %v", err)
	}
	if provider == nil {
		t.Fatalf("expected provider")
	}
	if provider.APIKey != "sk-or-original" {
		t.Fatalf("expected api key to be preserved, got %s", provider.APIKey)
	}
}

func TestLLMProviderUpdateSupportsRename(t *testing.T) {
	service := newTestLLMService(t)
	ctx := context.Background()
	if err := service.AddProvider(ctx, llmdomain.ModelProvider{
		ID:        "openai",
		Name:      "OpenAI",
		APIType:   llmdomain.ProviderAPITypeOpenAIResponses,
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "sk-original",
		IsEnabled: true,
		Models: []llmdomain.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}); err != nil {
		t.Fatalf("add provider failed: %v", err)
	}

	server := &Server{
		llm:    service,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/control/llm/providers/openai",
		strings.NewReader(`{"id":"openai-cn","name":"OpenAI CN","api_type":"openai-completions","base_url":"https://proxy.example/v1","api_key":"","default_model":"gpt-4o","models":[{"id":"gpt-4o","name":"GPT-4o","is_enabled":true}],"is_enabled":true}`),
	)
	rec := httptest.NewRecorder()
	server.llmProviderItemHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected rename update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	oldProvider, err := service.GetProvider(ctx, "openai")
	if err != nil {
		t.Fatalf("get old provider failed: %v", err)
	}
	if oldProvider != nil {
		t.Fatalf("expected old provider id to be removed")
	}

	provider, err := service.GetProvider(ctx, "openai-cn")
	if err != nil {
		t.Fatalf("get renamed provider failed: %v", err)
	}
	if provider == nil {
		t.Fatalf("expected renamed provider")
	}
	if provider.APIKey != "sk-original" {
		t.Fatalf("expected api key to be preserved, got %s", provider.APIKey)
	}
	if provider.APIType != llmdomain.ProviderAPITypeOpenAICompletions {
		t.Fatalf("expected api type to be preserved on rename, got %s", provider.APIType)
	}
	if !provider.IsDefault {
		t.Fatalf("expected renamed provider to remain default")
	}
}

func TestLLMProviderCreateGeneratesInternalIDWhenMissing(t *testing.T) {
	service := newTestLLMService(t)
	server := &Server{
		llm:         service,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		idGenerator: &sequenceIDGenerator{ids: []string{"provider-seed"}},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/control/llm/providers",
		strings.NewReader(`{"name":"OpenAI","api_type":"openai-completions","base_url":"https://api.openai.com/v1","api_key":"sk-created","default_model":"gpt-4o","models":[{"id":"gpt-4o","name":"GPT-4o","is_enabled":true}],"is_enabled":true}`),
	)
	rec := httptest.NewRecorder()
	server.llmProviderListHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp llmProviderResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if !strings.HasPrefix(resp.ID, "prov_") {
		t.Fatalf("expected generated provider id prefix, got %s", resp.ID)
	}
	if resp.Name != "OpenAI" {
		t.Fatalf("expected provider name OpenAI, got %s", resp.Name)
	}
	if resp.APIType != llmdomain.ProviderAPITypeOpenAICompletions {
		t.Fatalf("expected api type openai-completions, got %s", resp.APIType)
	}
}

func TestLLMProviderCreateRejectsDuplicateName(t *testing.T) {
	service := newTestLLMService(t)
	ctx := context.Background()
	if err := service.AddProvider(ctx, llmdomain.ModelProvider{
		ID:        "existing-provider",
		Name:      "OpenAI",
		APIType:   llmdomain.ProviderAPITypeOpenAIResponses,
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "sk-existing",
		IsEnabled: true,
		Models: []llmdomain.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}); err != nil {
		t.Fatalf("add provider failed: %v", err)
	}

	server := &Server{
		llm:         service,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		idGenerator: &sequenceIDGenerator{ids: []string{"provider-seed"}},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/control/llm/providers",
		strings.NewReader(`{"name":" openai ","api_type":"openai-responses","base_url":"https://proxy.example/v1","api_key":"sk-created","default_model":"gpt-4o","models":[{"id":"gpt-4o","name":"GPT-4o","is_enabled":true}],"is_enabled":true}`),
	)
	rec := httptest.NewRecorder()
	server.llmProviderListHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected create 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "provider name already exists") {
		t.Fatalf("expected duplicate provider name error, got %s", rec.Body.String())
	}
}

func TestLLMProviderCreateSupportsOpenRouterFields(t *testing.T) {
	service := newTestLLMService(t)
	server := &Server{
		llm:         service,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		idGenerator: &sequenceIDGenerator{ids: []string{"provider-openrouter"}},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/control/llm/providers",
		strings.NewReader(`{
			"name":"OpenRouter",
			"provider_type":"openrouter",
			"api_type":"openai-completions",
			"base_url":"https://openrouter.ai/api/v1",
			"api_key":"sk-or-created",
			"openrouter":{
				"site_url":"https://alter0.example",
				"app_name":"Alter0",
				"fallback_models":["anthropic/claude-3.7-sonnet"],
				"provider_order":["openai","anthropic"],
				"allow_fallbacks":true,
				"require_parameters":true
			},
			"default_model":"openai/gpt-5.4",
			"models":[{"id":"openai/gpt-5.4","name":"GPT-5.4","is_enabled":true}],
			"is_enabled":true
		}`),
	)
	rec := httptest.NewRecorder()
	server.llmProviderListHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp llmProviderResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.ProviderType != llmdomain.ProviderTypeOpenRouter {
		t.Fatalf("expected openrouter provider type, got %q", resp.ProviderType)
	}
	if resp.OpenRouter == nil || resp.OpenRouter.SiteURL != "https://alter0.example" {
		t.Fatalf("expected openrouter site url in response, got %+v", resp.OpenRouter)
	}
}

func TestLLMProviderCreateRejectsPlaceholderDashAPIKey(t *testing.T) {
	service := newTestLLMService(t)
	server := &Server{
		llm:         service,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		idGenerator: &sequenceIDGenerator{ids: []string{"provider-placeholder-dash"}},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/control/llm/providers",
		strings.NewReader(`{"name":"OpenAI","api_type":"openai-completions","base_url":"https://api.openai.com/v1","api_key":"-","default_model":"gpt-4o","models":[{"id":"gpt-4o","name":"GPT-4o","is_enabled":true}],"is_enabled":true}`),
	)
	rec := httptest.NewRecorder()
	server.llmProviderListHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected create 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "api_key is required") {
		t.Fatalf("expected api_key required error, got %s", rec.Body.String())
	}
}

func TestLLMProviderListRecoversLegacyProviderWithoutAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	storage := llminfra.NewModelConfigStorage(filepath.Join(tempDir, "model_config.json"))
	data := `{
  "providers": [
    {
      "id": "legacy-openai",
      "name": "Legacy OpenAI",
      "base_url": "https://api.openai.com/v1",
      "api_key": "",
      "default_model": "gpt-4o",
      "models": [
        { "id": "gpt-4o", "name": "GPT-4o", "is_enabled": true }
      ],
      "is_enabled": true,
      "is_default": true
    }
  ],
  "default_provider_id": "legacy-openai"
}`
	if err := os.WriteFile(filepath.Join(tempDir, "model_config.json"), []byte(data), 0644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}

	server := &Server{
		llm:    llmapp.NewModelConfigService(storage),
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
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(resp.Items))
	}
	if resp.Items[0].IsEnabled {
		t.Fatalf("expected legacy provider to be disabled in response")
	}
	if resp.Items[0].APIKey != "" {
		t.Fatalf("expected empty api key in response, got %q", resp.Items[0].APIKey)
	}
}

func newTestLLMService(t *testing.T) *llmapp.ModelConfigService {
	t.Helper()
	storage := llminfra.NewModelConfigStorage(filepath.Join(t.TempDir(), "model_config.json"))
	return llmapp.NewModelConfigService(storage)
}
