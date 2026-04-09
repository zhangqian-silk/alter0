package application

import (
	"testing"

	"alter0/internal/llm/domain"
)

func TestCreateDefaultConfigKeepsProvidersDisabledUntilConfigured(t *testing.T) {
	config := CreateDefaultConfig()

	if config.DefaultProviderID != "" {
		t.Fatalf("DefaultProviderID = %q, want empty", config.DefaultProviderID)
	}
	if len(config.Providers) != 2 {
		t.Fatalf("Providers length = %d, want 2", len(config.Providers))
	}

	openai := findProvider(t, config.Providers, "openai")
	if openai.ProviderType != domain.ProviderTypeOpenAICompatible {
		t.Fatalf("openai ProviderType = %q, want %q", openai.ProviderType, domain.ProviderTypeOpenAICompatible)
	}
	if openai.APIType != domain.DefaultProviderAPIType {
		t.Fatalf("openai APIType = %q, want %q", openai.APIType, domain.DefaultProviderAPIType)
	}
	if openai.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("openai BaseURL = %q, want https://api.openai.com/v1", openai.BaseURL)
	}
	if openai.IsEnabled || openai.IsDefault {
		t.Fatalf("openai enabled/default = (%v, %v), want false/false", openai.IsEnabled, openai.IsDefault)
	}
	if openai.APIKey != "" {
		t.Fatalf("openai APIKey = %q, want empty", openai.APIKey)
	}
	if len(openai.Models) == 0 {
		t.Fatalf("openai Models empty, want defaults")
	}

	openrouter := findProvider(t, config.Providers, "openrouter")
	if openrouter.ProviderType != domain.ProviderTypeOpenRouter {
		t.Fatalf("openrouter ProviderType = %q, want %q", openrouter.ProviderType, domain.ProviderTypeOpenRouter)
	}
	if openrouter.APIType != domain.ProviderAPITypeOpenAICompletions {
		t.Fatalf("openrouter APIType = %q, want %q", openrouter.APIType, domain.ProviderAPITypeOpenAICompletions)
	}
	if openrouter.BaseURL != domain.DefaultOpenRouterBaseURL {
		t.Fatalf("openrouter BaseURL = %q, want %q", openrouter.BaseURL, domain.DefaultOpenRouterBaseURL)
	}
	if openrouter.IsEnabled || openrouter.IsDefault {
		t.Fatalf("openrouter enabled/default = (%v, %v), want false/false", openrouter.IsEnabled, openrouter.IsDefault)
	}
}

func findProvider(t *testing.T, providers []domain.ModelProvider, id string) domain.ModelProvider {
	t.Helper()
	for _, provider := range providers {
		if provider.ID == id {
			return provider
		}
	}
	t.Fatalf("provider %q not found: %+v", id, providers)
	return domain.ModelProvider{}
}
