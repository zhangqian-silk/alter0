package domain

import "testing"

func TestModelProviderValidateRequiresEnabledDefaultModel(t *testing.T) {
	provider := ModelProvider{
		ID:        "qwen",
		Name:      "Qwen",
		APIType:   ProviderAPITypeOpenAIResponses,
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
		APIType:   ProviderAPITypeOpenAIResponses,
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
		APIType:   ProviderAPITypeOpenAIResponses,
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
		APIType:   ProviderAPITypeOpenAIResponses,
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
		APIType:   ProviderAPITypeOpenAIResponses,
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
		APIType:   ProviderAPITypeOpenAIResponses,
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
		APIType:   ProviderAPITypeOpenAIResponses,
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
			APIType:   ProviderAPITypeOpenAIResponses,
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
			APIType:   ProviderAPITypeOpenAIResponses,
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
		APIType:   ProviderAPITypeOpenAIResponses,
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

func TestModelProviderValidateDefaultsAPIType(t *testing.T) {
	provider := ModelProvider{
		ID:        "openai",
		Name:      "OpenAI",
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "sk-test",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}

	if err := provider.Validate(); err != nil {
		t.Fatalf("expected empty api_type to default, got %v", err)
	}
	normalized, err := normalizeProvider(provider)
	if err != nil {
		t.Fatalf("normalize provider failed: %v", err)
	}
	if normalized.APIType != DefaultProviderAPIType {
		t.Fatalf("expected default api_type %q, got %q", DefaultProviderAPIType, normalized.APIType)
	}
}

func TestModelProviderValidateRejectsUnsupportedAPIType(t *testing.T) {
	provider := ModelProvider{
		ID:        "openai",
		Name:      "OpenAI",
		APIType:   "custom",
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "sk-test",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}

	if err := provider.Validate(); err == nil {
		t.Fatalf("expected unsupported api_type validation error")
	}
}

func TestModelConfigAddProviderDoesNotAssignDisabledDefault(t *testing.T) {
	config := &ModelConfig{}
	if err := config.AddProvider(ModelProvider{
		ID:        "provider-disabled",
		Name:      "Disabled Provider",
		APIType:   ProviderAPITypeOpenAIResponses,
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-disabled",
		IsEnabled: false,
		Models: []ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}); err != nil {
		t.Fatalf("add provider failed: %v", err)
	}

	if config.DefaultProviderID != "" {
		t.Fatalf("expected no default provider, got %s", config.DefaultProviderID)
	}
	if provider := config.GetDefaultProvider(); provider != nil {
		t.Fatalf("expected no default provider, got %+v", provider)
	}
	if config.Providers[0].IsDefault {
		t.Fatalf("expected disabled provider to not be default")
	}
}

func TestModelConfigEnableProviderPromotesFirstEnabledDefault(t *testing.T) {
	config := &ModelConfig{}
	if err := config.AddProvider(ModelProvider{
		ID:        "provider-disabled",
		Name:      "Disabled Provider",
		APIType:   ProviderAPITypeOpenAIResponses,
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-disabled",
		IsEnabled: false,
		Models: []ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
		},
		DefaultModel: "gpt-4o",
	}); err != nil {
		t.Fatalf("add disabled provider failed: %v", err)
	}
	if err := config.AddProvider(ModelProvider{
		ID:        "provider-enabled",
		Name:      "Enabled Provider",
		APIType:   ProviderAPITypeOpenAIResponses,
		BaseURL:   "https://example.com/v2",
		APIKey:    "sk-enabled",
		IsEnabled: false,
		Models: []ModelInfo{
			{ID: "gpt-4.1", Name: "GPT-4.1", IsEnabled: true},
		},
		DefaultModel: "gpt-4.1",
	}); err != nil {
		t.Fatalf("add enabled candidate failed: %v", err)
	}

	if err := config.EnableProvider("provider-enabled", true); err != nil {
		t.Fatalf("enable provider failed: %v", err)
	}

	if config.DefaultProviderID != "provider-enabled" {
		t.Fatalf("expected provider-enabled as default, got %s", config.DefaultProviderID)
	}
	provider := config.GetDefaultProvider()
	if provider == nil || provider.ID != "provider-enabled" {
		t.Fatalf("expected provider-enabled default, got %+v", provider)
	}
}

func TestModelConfigValidateReconcilesDisabledLegacyDefault(t *testing.T) {
	config := &ModelConfig{
		Providers: []ModelProvider{
			{
				ID:        "provider-disabled",
				Name:      "Disabled Provider",
				APIType:   ProviderAPITypeOpenAIResponses,
				BaseURL:   "https://example.com/v1",
				APIKey:    "sk-disabled",
				IsEnabled: false,
				IsDefault: true,
				Models: []ModelInfo{
					{ID: "gpt-4o", Name: "GPT-4o", IsEnabled: true},
				},
				DefaultModel: "gpt-4o",
			},
			{
				ID:        "provider-enabled",
				Name:      "Enabled Provider",
				APIType:   ProviderAPITypeOpenAIResponses,
				BaseURL:   "https://example.com/v2",
				APIKey:    "sk-enabled",
				IsEnabled: true,
				IsDefault: false,
				Models: []ModelInfo{
					{ID: "gpt-4.1", Name: "GPT-4.1", IsEnabled: true},
				},
				DefaultModel: "gpt-4.1",
			},
		},
		DefaultProviderID: "provider-disabled",
	}

	if err := config.Validate(); err != nil {
		t.Fatalf("validate config failed: %v", err)
	}
	if config.DefaultProviderID != "provider-enabled" {
		t.Fatalf("expected provider-enabled as reconciled default, got %s", config.DefaultProviderID)
	}
	if config.Providers[0].IsDefault {
		t.Fatalf("expected disabled provider to lose default flag")
	}
	if !config.Providers[1].IsDefault {
		t.Fatalf("expected enabled provider to become default")
	}
}

func TestModelProviderValidateDefaultsOpenRouterFields(t *testing.T) {
	provider := ModelProvider{
		ID:           "openrouter",
		Name:         "OpenRouter",
		ProviderType: ProviderTypeOpenRouter,
		APIKey:       "sk-or-test",
		IsEnabled:    true,
		Models: []ModelInfo{
			{ID: "openai/gpt-5.4", Name: "GPT-5.4", IsEnabled: true},
		},
		DefaultModel: "openai/gpt-5.4",
	}

	normalized, err := normalizeProvider(provider)
	if err != nil {
		t.Fatalf("normalize provider failed: %v", err)
	}
	if normalized.APIType != ProviderAPITypeOpenAICompletions {
		t.Fatalf("expected openrouter api type %q, got %q", ProviderAPITypeOpenAICompletions, normalized.APIType)
	}
	if normalized.BaseURL != DefaultOpenRouterBaseURL {
		t.Fatalf("expected openrouter base url %q, got %q", DefaultOpenRouterBaseURL, normalized.BaseURL)
	}
}

func TestModelProviderValidateInfersOpenRouterFromBaseURL(t *testing.T) {
	provider := ModelProvider{
		ID:        "router",
		Name:      "Router",
		BaseURL:   " https://openrouter.ai/api/v1 ",
		APIKey:    "sk-or-test",
		IsEnabled: true,
		Models: []ModelInfo{
			{ID: "openai/gpt-5.4", Name: "GPT-5.4", IsEnabled: true},
		},
		DefaultModel: "openai/gpt-5.4",
	}

	normalized, err := normalizeProvider(provider)
	if err != nil {
		t.Fatalf("normalize provider failed: %v", err)
	}
	if normalized.ProviderType != ProviderTypeOpenRouter {
		t.Fatalf("expected provider type %q, got %q", ProviderTypeOpenRouter, normalized.ProviderType)
	}
	if normalized.APIType != ProviderAPITypeOpenAICompletions {
		t.Fatalf("expected openrouter api type %q, got %q", ProviderAPITypeOpenAICompletions, normalized.APIType)
	}
}

func TestModelProviderValidateNormalizesOpenRouterConfig(t *testing.T) {
	allowFallbacks := true
	requireParameters := true
	provider := ModelProvider{
		ID:           "openrouter",
		Name:         "OpenRouter",
		ProviderType: ProviderTypeOpenRouter,
		BaseURL:      DefaultOpenRouterBaseURL,
		APIKey:       "sk-or-test",
		IsEnabled:    true,
		OpenRouter: &OpenRouterConfig{
			SiteURL:           " https://alter0.example ",
			AppName:           " Alter0 ",
			FallbackModels:    []string{" openai/gpt-5.4 ", "", "openai/gpt-5.4", "anthropic/claude-3.7-sonnet"},
			ProviderOrder:     []string{" openai ", "openai", "anthropic"},
			AllowFallbacks:    &allowFallbacks,
			RequireParameters: &requireParameters,
		},
		Models: []ModelInfo{
			{ID: "openai/gpt-5.4", Name: "GPT-5.4", IsEnabled: true},
		},
		DefaultModel: "openai/gpt-5.4",
	}

	normalized, err := normalizeProvider(provider)
	if err != nil {
		t.Fatalf("normalize provider failed: %v", err)
	}
	if normalized.OpenRouter == nil {
		t.Fatalf("expected normalized openrouter config")
	}
	if normalized.OpenRouter.SiteURL != "https://alter0.example" {
		t.Fatalf("expected trimmed site url, got %q", normalized.OpenRouter.SiteURL)
	}
	if len(normalized.OpenRouter.FallbackModels) != 2 {
		t.Fatalf("expected 2 fallback models, got %d", len(normalized.OpenRouter.FallbackModels))
	}
	if len(normalized.OpenRouter.ProviderOrder) != 2 {
		t.Fatalf("expected 2 provider order entries, got %d", len(normalized.OpenRouter.ProviderOrder))
	}
	if normalized.OpenRouter.AllowFallbacks == nil || !*normalized.OpenRouter.AllowFallbacks {
		t.Fatalf("expected allow_fallbacks to remain true")
	}
	if normalized.OpenRouter.RequireParameters == nil || !*normalized.OpenRouter.RequireParameters {
		t.Fatalf("expected require_parameters to remain true")
	}
}
