package infrastructure

import (
	"os"
	"path/filepath"
	"testing"

	"alter0/internal/llm/domain"
)

func TestModelConfigStorageGetLoadsExistingFileAndReturnsClone(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "model_config.json")
	data := `{
  "providers": [
    {
      "id": "qwen",
      "name": "Qwen",
      "base_url": "https://example.com/v1",
      "api_key": "sk-test",
      "default_model": "qwen-plus",
      "models": [
        { "id": "qwen-plus", "name": "Qwen Plus", "is_enabled": true }
      ],
      "is_enabled": true,
      "is_default": true
    }
  ],
  "default_provider_id": "qwen"
}`
	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}

	storage := NewModelConfigStorage(filePath)
	config, err := storage.Get()
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	if len(config.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(config.Providers))
	}
	if config.Providers[0].APIType != domain.DefaultProviderAPIType {
		t.Fatalf("expected default api_type %q, got %q", domain.DefaultProviderAPIType, config.Providers[0].APIType)
	}

	config.Providers[0].Name = "Changed"
	config.Providers[0].Models[0].Name = "Changed Model"

	reloaded, err := storage.Get()
	if err != nil {
		t.Fatalf("get cloned config failed: %v", err)
	}
	if reloaded.Providers[0].Name != "Qwen" {
		t.Fatalf("expected stored provider name Qwen, got %s", reloaded.Providers[0].Name)
	}
	if reloaded.Providers[0].Models[0].Name != "Qwen Plus" {
		t.Fatalf("expected stored model name Qwen Plus, got %s", reloaded.Providers[0].Models[0].Name)
	}
}

func TestModelConfigStorageGetReturnsClonedOpenRouterConfig(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "model_config.json")
	data := `{
  "providers": [
    {
      "id": "openrouter",
      "name": "OpenRouter",
      "provider_type": "openrouter",
      "api_type": "openai-completions",
      "base_url": "https://openrouter.ai/api/v1",
      "api_key": "sk-or-test",
      "openrouter": {
        "site_url": "https://alter0.example",
        "app_name": "Alter0",
        "fallback_models": ["anthropic/claude-3.7-sonnet"],
        "provider_order": ["openai", "anthropic"],
        "allow_fallbacks": true,
        "require_parameters": true
      },
      "default_model": "openai/gpt-5.4",
      "models": [
        { "id": "openai/gpt-5.4", "name": "GPT-5.4", "is_enabled": true }
      ],
      "is_enabled": true,
      "is_default": true
    }
  ],
  "default_provider_id": "openrouter"
}`
	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}

	storage := NewModelConfigStorage(filePath)
	config, err := storage.Get()
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	config.Providers[0].OpenRouter.SiteURL = "https://mutated.example"
	config.Providers[0].OpenRouter.FallbackModels[0] = "google/gemini-2.5-pro"

	reloaded, err := storage.Get()
	if err != nil {
		t.Fatalf("reload config failed: %v", err)
	}
	if reloaded.Providers[0].OpenRouter == nil {
		t.Fatalf("expected openrouter config")
	}
	if reloaded.Providers[0].OpenRouter.SiteURL != "https://alter0.example" {
		t.Fatalf("expected original site url, got %q", reloaded.Providers[0].OpenRouter.SiteURL)
	}
	if reloaded.Providers[0].OpenRouter.FallbackModels[0] != "anthropic/claude-3.7-sonnet" {
		t.Fatalf("expected original fallback model, got %q", reloaded.Providers[0].OpenRouter.FallbackModels[0])
	}
}

func TestModelConfigStorageGetReconcilesDisabledLegacyDefaultProvider(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "model_config.json")
	data := `{
  "providers": [
    {
      "id": "disabled",
      "name": "Disabled",
      "base_url": "https://disabled.example/v1",
      "api_key": "sk-disabled",
      "default_model": "gpt-4o",
      "models": [
        { "id": "gpt-4o", "name": "GPT-4o", "is_enabled": true }
      ],
      "is_enabled": false,
      "is_default": true
    },
    {
      "id": "enabled",
      "name": "Enabled",
      "base_url": "https://enabled.example/v1",
      "api_key": "sk-enabled",
      "default_model": "gpt-4.1",
      "models": [
        { "id": "gpt-4.1", "name": "GPT-4.1", "is_enabled": true }
      ],
      "is_enabled": true,
      "is_default": false
    }
  ],
  "default_provider_id": "disabled"
}`
	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}

	storage := NewModelConfigStorage(filePath)
	config, err := storage.Get()
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	if config.DefaultProviderID != "enabled" {
		t.Fatalf("expected enabled provider to become default, got %s", config.DefaultProviderID)
	}
	if !config.Providers[1].IsDefault {
		t.Fatalf("expected enabled provider default flag")
	}
	if config.Providers[0].IsDefault {
		t.Fatalf("expected disabled provider default flag to be cleared")
	}
}

func TestModelConfigStorageGetRecoversLegacyProviderWithoutAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "model_config.json")
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
    },
    {
      "id": "enabled",
      "name": "Enabled",
      "base_url": "https://enabled.example/v1",
      "api_key": "sk-enabled",
      "default_model": "gpt-4.1",
      "models": [
        { "id": "gpt-4.1", "name": "GPT-4.1", "is_enabled": true }
      ],
      "is_enabled": true,
      "is_default": false
    }
  ],
  "default_provider_id": "legacy-openai"
}`
	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}

	storage := NewModelConfigStorage(filePath)
	config, err := storage.Get()
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	legacy := config.GetProvider("legacy-openai")
	if legacy == nil {
		t.Fatalf("expected legacy provider")
	}
	if legacy.IsEnabled {
		t.Fatalf("expected legacy provider without api key to be disabled")
	}
	if legacy.IsDefault {
		t.Fatalf("expected legacy provider without api key to lose default flag")
	}
	if config.DefaultProviderID != "enabled" {
		t.Fatalf("expected enabled provider to become default, got %s", config.DefaultProviderID)
	}
}
