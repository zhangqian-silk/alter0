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

	if err := config.UpdateProvider("qwen", ModelProvider{
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

func TestModelConfigUpdateProviderSupportsRenameForDefaultProvider(t *testing.T) {
	config := &ModelConfig{}
	if err := config.AddProvider(ModelProvider{
		ID:        "openai",
		Name:      "OpenAI",
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "sk-test",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}); err != nil {
		t.Fatalf("add provider failed: %v", err)
	}

	if err := config.UpdateProvider("openai", ModelProvider{
		ID:        "openai-cn",
		Name:      "OpenAI CN",
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-updated",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}); err != nil {
		t.Fatalf("rename provider failed: %v", err)
	}

	if config.GetProvider("openai") != nil {
		t.Fatalf("expected old provider id to be removed")
	}
	provider := config.GetProvider("openai-cn")
	if provider == nil {
		t.Fatalf("expected renamed provider")
	}
	if !provider.IsDefault {
		t.Fatalf("expected renamed provider to remain default")
	}
	if config.DefaultProviderID != "openai-cn" {
		t.Fatalf("expected default provider id to update, got %s", config.DefaultProviderID)
	}
}

func TestModelConfigAddProviderRejectsDuplicateName(t *testing.T) {
	config := &ModelConfig{}
	if err := config.AddProvider(ModelProvider{
		ID:        "provider-a",
		Name:      "OpenAI",
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-a",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}); err != nil {
		t.Fatalf("add provider failed: %v", err)
	}

	err := config.AddProvider(ModelProvider{
		ID:        "provider-b",
		Name:      " openai ",
		BaseURL:   "https://example.com/v2",
		APIKey:    "sk-b",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "gpt-4.1", Name: "GPT-4.1", IsEnabled: true},
		},
		DefaultModel: "gpt-4.1",
	})
	if err == nil {
		t.Fatalf("expected duplicate provider name error")
	}
}

func TestModelConfigUpdateProviderRejectsDuplicateName(t *testing.T) {
	config := &ModelConfig{}
	for _, provider := range []ModelProvider{
		{
			ID:        "provider-a",
			Name:      "OpenAI",
			BaseURL:   "https://example.com/v1",
			APIKey:    "sk-a",
			IsEnabled: true,
			Models: []ModelInfo{
				{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
			},
			DefaultModel: "gpt-4o",
		},
		{
			ID:        "provider-b",
			Name:      "Anthropic",
			BaseURL:   "https://example.com/v2",
			APIKey:    "sk-b",
			IsEnabled: true,
			Models: []ModelInfo{
				{ID: "claude-3-7", Name: "Claude 3.7", IsEnabled: true},
			},
			DefaultModel: "claude-3-7",
		},
	} {
		if err := config.AddProvider(provider); err != nil {
			t.Fatalf("add provider failed: %v", err)
		}
	}

	err := config.UpdateProvider("provider-b", ModelProvider{
		ID:        "provider-b",
		Name:      " openai ",
		BaseURL:   "https://example.com/v2",
		APIKey:    "sk-b",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "claude-3-7", Name: "Claude 3.7", IsEnabled: true},
		},
		DefaultModel: "claude-3-7",
	})
	if err == nil {
		t.Fatalf("expected duplicate provider name error")
	}
}
