package domain

import "testing"

func TestModelProviderValidateRequiresEnabledDefaultModel(t *testing.T) {
	provider := ModelProvider{
		ID:        "qwen",
		Name:      "Qwen",
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-test",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "qwen-plus", Name: "Qwen Plus", IsEnabled: true},
			{ID: "qwen-max", Name: "Qwen Max", IsEnabled: false},
		},
		DefaultModel: "qwen-max",
	}

	if err := provider.Validate(); err == nil {
		t.Fatalf("expected disabled default model validation error")
	}
}

func TestModelConfigUpdateProviderPreservesDefaultFlag(t *testing.T) {
	config := &ModelConfig{}
	if err := config.AddProvider(ModelProvider{
		ID:        "qwen",
		Name:      "Qwen",
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-test",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "qwen-plus", Name: "Qwen Plus", IsEnabled: true},
		},
		DefaultModel: "qwen-plus",
	}); err != nil {
		t.Fatalf("add provider failed: %v", err)
	}

	if err := config.UpdateProvider(ModelProvider{
		ID:        "qwen",
		Name:      "Qwen Updated",
		BaseURL:   "https://example.com/v2",
		APIKey:    "sk-updated",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "qwen-max", Name: "Qwen Max", IsEnabled: true},
		},
		DefaultModel: "qwen-max",
	}); err != nil {
		t.Fatalf("update provider failed: %v", err)
	}

	provider := config.GetProvider("qwen")
	if provider == nil {
		t.Fatalf("expected provider")
	}
	if !provider.IsDefault {
		t.Fatalf("expected provider to remain default")
	}
	if config.DefaultProviderID != "qwen" {
		t.Fatalf("expected default provider id to stay qwen, got %s", config.DefaultProviderID)
	}
	if provider.DefaultModel != "qwen-max" {
		t.Fatalf("expected default model qwen-max, got %s", provider.DefaultModel)
	}
}
