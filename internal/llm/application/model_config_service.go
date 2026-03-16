package application

import (
	"context"
	"errors"
	"sync"

	"alter0/internal/llm/domain"
	"alter0/internal/llm/infrastructure"
)

// Ensure ModelConfigService implements LLMService
var _ LLMService = (*ModelConfigService)(nil)

// ModelConfigService provides model configuration management.
type ModelConfigService struct {
	storage  *infrastructure.ModelConfigStorage
	clients  map[string]domain.LLMClient
	mu       sync.RWMutex
}

// NewModelConfigService creates a new model config service.
func NewModelConfigService(storage *infrastructure.ModelConfigStorage) *ModelConfigService {
	return &ModelConfigService{
		storage: storage,
		clients: make(map[string]domain.LLMClient),
	}
}

// GetConfig returns the current configuration.
func (s *ModelConfigService) GetConfig(ctx context.Context) (*domain.ModelConfig, error) {
	return s.storage.Get()
}

// GetProvider returns a provider by ID.
func (s *ModelConfigService) GetProvider(ctx context.Context, providerID string) (*domain.ModelProvider, error) {
	return s.storage.GetProvider(providerID)
}

// GetDefaultProvider returns the default provider.
func (s *ModelConfigService) GetDefaultProvider(ctx context.Context) (*domain.ModelProvider, error) {
	return s.storage.GetDefaultProvider()
}

// GetEnabledProviders returns all enabled providers.
func (s *ModelConfigService) GetEnabledProviders(ctx context.Context) ([]domain.ModelProvider, error) {
	return s.storage.GetEnabledProviders()
}

// AddProvider adds a new provider.
func (s *ModelConfigService) AddProvider(ctx context.Context, provider domain.ModelProvider) error {
	if err := s.storage.AddProvider(provider); err != nil {
		return err
	}

	// Invalidate cached client
	s.mu.Lock()
	delete(s.clients, provider.ID)
	s.mu.Unlock()

	return nil
}

// UpdateProvider updates an existing provider.
func (s *ModelConfigService) UpdateProvider(ctx context.Context, currentProviderID string, provider domain.ModelProvider) error {
	if err := s.storage.UpdateProvider(currentProviderID, provider); err != nil {
		return err
	}

	// Invalidate cached client
	s.mu.Lock()
	delete(s.clients, currentProviderID)
	delete(s.clients, provider.ID)
	s.mu.Unlock()

	return nil
}

// RemoveProvider removes a provider.
func (s *ModelConfigService) RemoveProvider(ctx context.Context, providerID string) error {
	if err := s.storage.RemoveProvider(providerID); err != nil {
		return err
	}

	// Invalidate cached client
	s.mu.Lock()
	delete(s.clients, providerID)
	s.mu.Unlock()

	return nil
}

// SetDefaultProvider sets the default provider.
func (s *ModelConfigService) SetDefaultProvider(ctx context.Context, providerID string) error {
	return s.storage.SetDefaultProvider(providerID)
}

// EnableProvider enables or disables a provider.
func (s *ModelConfigService) EnableProvider(ctx context.Context, providerID string, enabled bool) error {
	return s.storage.EnableProvider(providerID, enabled)
}

// GetClient returns an LLM client for the specified provider.
// If providerID is empty, returns the client for the default provider.
func (s *ModelConfigService) GetClient(ctx context.Context, providerID string) (domain.LLMClient, error) {
	// Get provider
	var provider *domain.ModelProvider
	var err error
	if providerID == "" {
		provider, err = s.storage.GetDefaultProvider()
	} else {
		provider, err = s.storage.GetProvider(providerID)
	}
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, errors.New("no provider configured")
	}

	// Check if provider is enabled
	if !provider.IsEnabled {
		return nil, errors.New("provider is disabled: " + provider.ID)
	}

	// Check cache
	s.mu.RLock()
	client, ok := s.clients[provider.ID]
	s.mu.RUnlock()

	if ok {
		return client, nil
	}

	// Create new client
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check
	if client, ok := s.clients[provider.ID]; ok {
		return client, nil
	}

	client = infrastructure.NewOpenAIClient(infrastructure.OpenAIClientConfig{
		APIKey:  provider.APIKey,
		BaseURL: provider.BaseURL,
		Model:   provider.DefaultModel,
	})

	s.clients[provider.ID] = client
	return client, nil
}

// GetDefaultClient returns an LLM client for the default provider.
func (s *ModelConfigService) GetDefaultClient(ctx context.Context) (domain.LLMClient, error) {
	return s.GetClient(ctx, "")
}

// GetReActAgent creates a ReAct agent with the specified provider and tools.
func (s *ModelConfigService) GetReActAgent(ctx context.Context, providerID string, config domain.ReActAgentConfig) (*domain.ReActAgent, error) {
	client, err := s.GetClient(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// Get provider for model name
	var provider *domain.ModelProvider
	if providerID == "" {
		provider, err = s.storage.GetDefaultProvider()
	} else {
		provider, err = s.storage.GetProvider(providerID)
	}
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, errors.New("no provider configured")
	}

	config.Client = client
	if config.Model == "" {
		config.Model = provider.DefaultModel
	}

	return domain.NewReActAgent(config), nil
}

// CreateDefaultConfig creates a default configuration with common providers.
func CreateDefaultConfig() *domain.ModelConfig {
	return &domain.ModelConfig{
		Providers: []domain.ModelProvider{
			{
				ID:          "openai",
				Name:        "OpenAI",
				BaseURL:     "https://api.openai.com/v1",
				APIKey:      "", // To be filled by user
				DefaultModel: "gpt-4o",
				Models: []domain.ModelInfo{
					{ID: "gpt-4o", Name: "GPT-4o", SupportsTools: true, SupportsVision: true, SupportsStreaming: true, IsEnabled: true},
					{ID: "gpt-4o-mini", Name: "GPT-4o Mini", SupportsTools: true, SupportsVision: true, SupportsStreaming: true, IsEnabled: true},
					{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", SupportsTools: true, SupportsVision: true, SupportsStreaming: true, IsEnabled: true},
					{ID: "o1", Name: "o1", SupportsTools: false, SupportsVision: false, SupportsStreaming: false, IsEnabled: true},
					{ID: "o1-mini", Name: "o1 Mini", SupportsTools: false, SupportsVision: false, SupportsStreaming: false, IsEnabled: true},
				},
				IsEnabled: false,
				IsDefault: true,
			},
		},
		DefaultProviderID: "openai",
	}
}
