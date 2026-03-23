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
