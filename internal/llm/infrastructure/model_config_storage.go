package infrastructure

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"alter0/internal/llm/domain"
)

// ModelConfigStorage handles persistence of model configuration.
type ModelConfigStorage struct {
	mu       sync.RWMutex
	filePath string
	config   *domain.ModelConfig
}

// NewModelConfigStorage creates a new model config storage.
func NewModelConfigStorage(filePath string) *ModelConfigStorage {
	if filePath == "" {
		filePath = ".alter0/model_config.json"
	}
	return &ModelConfigStorage{
		filePath: filePath,
	}
}

// Load loads the configuration from disk.
func (s *ModelConfigStorage) Load() (*domain.ModelConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Return empty config
			s.config = &domain.ModelConfig{}
			return cloneModelConfig(s.config), nil
		}
		return nil, err
	}

	var config domain.ModelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	s.config = cloneModelConfig(&config)
	return cloneModelConfig(s.config), nil
}

// Save saves the configuration to disk.
func (s *ModelConfigStorage) Save(config *domain.ModelConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := config.Validate(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return err
	}

	s.config = cloneModelConfig(config)
	return nil
}

// Get returns the current configuration.
func (s *ModelConfigStorage) Get() (*domain.ModelConfig, error) {
	s.mu.RLock()
	if s.config != nil {
		config := cloneModelConfig(s.config)
		s.mu.RUnlock()
		return config, nil
	}
	s.mu.RUnlock()

	return s.Load()
}

// GetProvider returns a provider by ID.
func (s *ModelConfigStorage) GetProvider(providerID string) (*domain.ModelProvider, error) {
	config, err := s.Get()
	if err != nil {
		return nil, err
	}
	return config.GetProvider(providerID), nil
}

// GetDefaultProvider returns the default provider.
func (s *ModelConfigStorage) GetDefaultProvider() (*domain.ModelProvider, error) {
	config, err := s.Get()
	if err != nil {
		return nil, err
	}
	return config.GetDefaultProvider(), nil
}

// AddProvider adds a new provider.
func (s *ModelConfigStorage) AddProvider(provider domain.ModelProvider) error {
	config, err := s.Get()
	if err != nil {
		return err
	}

	if err := config.AddProvider(provider); err != nil {
		return err
	}

	return s.Save(config)
}

// UpdateProvider updates an existing provider.
func (s *ModelConfigStorage) UpdateProvider(currentProviderID string, provider domain.ModelProvider) error {
	config, err := s.Get()
	if err != nil {
		return err
	}

	if err := config.UpdateProvider(currentProviderID, provider); err != nil {
		return err
	}

	return s.Save(config)
}

// RemoveProvider removes a provider.
func (s *ModelConfigStorage) RemoveProvider(providerID string) error {
	config, err := s.Get()
	if err != nil {
		return err
	}

	if err := config.RemoveProvider(providerID); err != nil {
		return err
	}

	return s.Save(config)
}

// SetDefaultProvider sets the default provider.
func (s *ModelConfigStorage) SetDefaultProvider(providerID string) error {
	config, err := s.Get()
	if err != nil {
		return err
	}

	if err := config.SetDefaultProvider(providerID); err != nil {
		return err
	}

	return s.Save(config)
}

// EnableProvider enables or disables a provider.
func (s *ModelConfigStorage) EnableProvider(providerID string, enabled bool) error {
	config, err := s.Get()
	if err != nil {
		return err
	}

	if err := config.EnableProvider(providerID, enabled); err != nil {
		return err
	}

	return s.Save(config)
}

// GetEnabledProviders returns all enabled providers.
func (s *ModelConfigStorage) GetEnabledProviders() ([]domain.ModelProvider, error) {
	config, err := s.Get()
	if err != nil {
		return nil, err
	}
	return config.GetEnabledProviders(), nil
}

func cloneModelConfig(config *domain.ModelConfig) *domain.ModelConfig {
	if config == nil {
		return &domain.ModelConfig{}
	}

	cloned := &domain.ModelConfig{
		DefaultProviderID: config.DefaultProviderID,
		UpdatedAt:         config.UpdatedAt,
	}
	if len(config.Providers) == 0 {
		cloned.Providers = []domain.ModelProvider{}
		return cloned
	}

	cloned.Providers = make([]domain.ModelProvider, 0, len(config.Providers))
	for _, provider := range config.Providers {
		providerCopy := provider
		if len(provider.Models) > 0 {
			providerCopy.Models = append([]domain.ModelInfo(nil), provider.Models...)
		} else {
			providerCopy.Models = []domain.ModelInfo{}
		}
		cloned.Providers = append(cloned.Providers, providerCopy)
	}

	return cloned
}
