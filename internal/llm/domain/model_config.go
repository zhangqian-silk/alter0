package domain

import (
	"errors"
	"strings"
	"time"
)

// ModelProvider represents a configured LLM provider.
type ModelProvider struct {
	// ID is the unique identifier for the provider.
	ID string `json:"id"`
	// Name is the human-readable name.
	Name string `json:"name"`
	// BaseURL is the API base URL (e.g., "https://api.openai.com/v1").
	BaseURL string `json:"base_url"`
	// APIKey is the API key for authentication.
	APIKey string `json:"api_key"`
	// Models is the list of available models for this provider.
	Models []ModelInfo `json:"models"`
	// DefaultModel is the default model to use.
	DefaultModel string `json:"default_model"`
	// IsEnabled indicates whether the provider is enabled.
	IsEnabled bool `json:"is_enabled"`
	// IsDefault indicates whether this is the default provider.
	IsDefault bool `json:"is_default"`
	// CreatedAt is the creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the last update timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

// ModelInfo represents a model available from a provider.
type ModelInfo struct {
	// ID is the model identifier (e.g., "gpt-4o", "gpt-5.4").
	ID string `json:"id"`
	// Name is the human-readable name.
	Name string `json:"name"`
	// MaxTokens is the maximum context length.
	MaxTokens int `json:"max_tokens,omitempty"`
	// SupportsTools indicates whether the model supports tool calling.
	SupportsTools bool `json:"supports_tools"`
	// SupportsVision indicates whether the model supports vision.
	SupportsVision bool `json:"supports_vision"`
	// SupportsStreaming indicates whether the model supports streaming.
	SupportsStreaming bool `json:"supports_streaming"`
	// IsEnabled indicates whether the model is enabled.
	IsEnabled bool `json:"is_enabled"`
}

// ModelConfig represents the full model configuration.
type ModelConfig struct {
	// Providers is the list of configured providers.
	Providers []ModelProvider `json:"providers"`
	// DefaultProviderID is the ID of the default provider.
	DefaultProviderID string `json:"default_provider_id"`
	// UpdatedAt is the last update timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate validates the model provider configuration.
func (p *ModelProvider) Validate() error {
	if _, err := normalizeProvider(*p); err != nil {
		return err
	}
	return nil
}

func normalizeProvider(provider ModelProvider) (ModelProvider, error) {
	provider.ID = strings.TrimSpace(provider.ID)
	provider.Name = strings.TrimSpace(provider.Name)
	provider.BaseURL = strings.TrimSpace(provider.BaseURL)
	provider.APIKey = strings.TrimSpace(provider.APIKey)
	provider.DefaultModel = strings.TrimSpace(provider.DefaultModel)

	if provider.ID == "" {
		return ModelProvider{}, errors.New("provider id is required")
	}
	if provider.Name == "" {
		return ModelProvider{}, errors.New("provider name is required")
	}
	if provider.BaseURL == "" {
		return ModelProvider{}, errors.New("base_url is required")
	}
	if provider.APIKey == "" {
		return ModelProvider{}, errors.New("api_key is required")
	}

	if len(provider.Models) == 0 {
		return ModelProvider{}, errors.New("at least one model is required")
	}

	seenModelIDs := make(map[string]struct{}, len(provider.Models))
	hasEnabledModel := false
	normalizedModels := make([]ModelInfo, 0, len(provider.Models))
	for _, model := range provider.Models {
		model.ID = strings.TrimSpace(model.ID)
		model.Name = strings.TrimSpace(model.Name)
		if model.ID == "" {
			return ModelProvider{}, errors.New("model id is required")
		}
		if model.Name == "" {
			model.Name = model.ID
		}
		if _, exists := seenModelIDs[model.ID]; exists {
			return ModelProvider{}, errors.New("duplicate model id: " + model.ID)
		}
		seenModelIDs[model.ID] = struct{}{}
		if model.IsEnabled {
			hasEnabledModel = true
		}
		normalizedModels = append(normalizedModels, model)
	}
	if !hasEnabledModel {
		return ModelProvider{}, errors.New("at least one enabled model is required")
	}
	provider.Models = normalizedModels

	if provider.DefaultModel == "" {
		for _, model := range provider.Models {
			if model.IsEnabled {
				provider.DefaultModel = model.ID
				break
			}
		}
	}

	defaultModel := provider.GetModel(provider.DefaultModel)
	if defaultModel == nil {
		return ModelProvider{}, errors.New("default model not found: " + provider.DefaultModel)
	}
	if !defaultModel.IsEnabled {
		return ModelProvider{}, errors.New("default model must be enabled: " + provider.DefaultModel)
	}

	return provider, nil
}

func providerNameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// GetModel returns the model info by ID.
func (p *ModelProvider) GetModel(modelID string) *ModelInfo {
	for i := range p.Models {
		if p.Models[i].ID == modelID {
			return &p.Models[i]
		}
	}
	return nil
}

// GetEnabledModels returns all enabled models.
func (p *ModelProvider) GetEnabledModels() []ModelInfo {
	var result []ModelInfo
	for _, m := range p.Models {
		if m.IsEnabled {
			result = append(result, m)
		}
	}
	return result
}

// Validate validates the model configuration.
func (c *ModelConfig) Validate() error {
	if len(c.Providers) == 0 {
		return nil // Empty config is valid
	}

	// Check for duplicate provider IDs and names.
	ids := make(map[string]bool)
	names := make(map[string]bool)
	for _, p := range c.Providers {
		if ids[p.ID] {
			return errors.New("duplicate provider id: " + p.ID)
		}
		ids[p.ID] = true
		nameKey := providerNameKey(p.Name)
		if names[nameKey] {
			return errors.New("duplicate provider name: " + p.Name)
		}
		names[nameKey] = true

		if err := p.Validate(); err != nil {
			return err
		}
	}

	// Check default provider exists
	if c.DefaultProviderID != "" {
		found := false
		for _, p := range c.Providers {
			if p.ID == c.DefaultProviderID {
				found = true
				break
			}
		}
		if !found {
			return errors.New("default provider not found: " + c.DefaultProviderID)
		}
	}

	return nil
}

// GetProvider returns the provider by ID.
func (c *ModelConfig) GetProvider(providerID string) *ModelProvider {
	for i := range c.Providers {
		if c.Providers[i].ID == providerID {
			return &c.Providers[i]
		}
	}
	return nil
}

// GetDefaultProvider returns the default provider.
func (c *ModelConfig) GetDefaultProvider() *ModelProvider {
	if c.DefaultProviderID == "" {
		// Return first enabled provider
		for i := range c.Providers {
			if c.Providers[i].IsEnabled {
				return &c.Providers[i]
			}
		}
		return nil
	}
	return c.GetProvider(c.DefaultProviderID)
}

// GetEnabledProviders returns all enabled providers.
func (c *ModelConfig) GetEnabledProviders() []ModelProvider {
	var result []ModelProvider
	for _, p := range c.Providers {
		if p.IsEnabled {
			result = append(result, p)
		}
	}
	return result
}

// AddProvider adds a new provider.
func (c *ModelConfig) AddProvider(provider ModelProvider) error {
	normalizedProvider, err := normalizeProvider(provider)
	if err != nil {
		return err
	}

	// Check for duplicate ID and name.
	for _, p := range c.Providers {
		if p.ID == normalizedProvider.ID {
			return errors.New("provider already exists: " + normalizedProvider.ID)
		}
		if providerNameKey(p.Name) == providerNameKey(normalizedProvider.Name) {
			return errors.New("provider name already exists: " + normalizedProvider.Name)
		}
	}

	normalizedProvider.CreatedAt = time.Now()
	normalizedProvider.UpdatedAt = time.Now()

	// Set as default if first provider
	if len(c.Providers) == 0 {
		normalizedProvider.IsDefault = true
		c.DefaultProviderID = normalizedProvider.ID
	} else {
		normalizedProvider.IsDefault = false
	}

	c.Providers = append(c.Providers, normalizedProvider)
	c.UpdatedAt = time.Now()

	return nil
}

// UpdateProvider updates an existing provider and supports provider ID rename.
func (c *ModelConfig) UpdateProvider(currentProviderID string, provider ModelProvider) error {
	normalizedProvider, err := normalizeProvider(provider)
	if err != nil {
		return err
	}
	for i := range c.Providers {
		if c.Providers[i].ID == currentProviderID {
			if normalizedProvider.ID != currentProviderID {
				for j := range c.Providers {
					if i != j && c.Providers[j].ID == normalizedProvider.ID {
						return errors.New("provider already exists: " + normalizedProvider.ID)
					}
				}
			}
			for j := range c.Providers {
				if i != j && providerNameKey(c.Providers[j].Name) == providerNameKey(normalizedProvider.Name) {
					return errors.New("provider name already exists: " + normalizedProvider.Name)
				}
			}
			normalizedProvider.UpdatedAt = time.Now()
			normalizedProvider.CreatedAt = c.Providers[i].CreatedAt
			normalizedProvider.IsDefault = c.DefaultProviderID == currentProviderID
			if normalizedProvider.IsDefault {
				c.DefaultProviderID = normalizedProvider.ID
			}
			c.Providers[i] = normalizedProvider
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("provider not found: " + currentProviderID)
}

// RemoveProvider removes a provider by ID.
func (c *ModelConfig) RemoveProvider(providerID string) error {
	for i := range c.Providers {
		if c.Providers[i].ID == providerID {
			c.Providers = append(c.Providers[:i], c.Providers[i+1:]...)
			c.UpdatedAt = time.Now()

			// Update default if removed
			if c.DefaultProviderID == providerID {
				c.DefaultProviderID = ""
				if len(c.Providers) > 0 {
					// Set first enabled provider as default
					for j := range c.Providers {
						if c.Providers[j].IsEnabled {
							c.Providers[j].IsDefault = true
							c.DefaultProviderID = c.Providers[j].ID
							break
						}
					}
				}
			}
			return nil
		}
	}
	return errors.New("provider not found: " + providerID)
}

// SetDefaultProvider sets the default provider.
func (c *ModelConfig) SetDefaultProvider(providerID string) error {
	found := false
	for i := range c.Providers {
		if c.Providers[i].ID == providerID {
			if !c.Providers[i].IsEnabled {
				return errors.New("default provider must be enabled: " + providerID)
			}
			found = true
			break
		}
	}
	if !found {
		return errors.New("provider not found: " + providerID)
	}

	for i := range c.Providers {
		c.Providers[i].IsDefault = c.Providers[i].ID == providerID
	}

	c.DefaultProviderID = providerID
	c.UpdatedAt = time.Now()
	return nil
}

// EnableProvider enables or disables a provider.
func (c *ModelConfig) EnableProvider(providerID string, enabled bool) error {
	for i := range c.Providers {
		if c.Providers[i].ID == providerID {
			c.Providers[i].IsEnabled = enabled
			c.Providers[i].UpdatedAt = time.Now()
			if !enabled && c.DefaultProviderID == providerID {
				c.DefaultProviderID = ""
				c.Providers[i].IsDefault = false
				for j := range c.Providers {
					if c.Providers[j].ID != providerID && c.Providers[j].IsEnabled {
						c.Providers[j].IsDefault = true
						c.DefaultProviderID = c.Providers[j].ID
						break
					}
				}
			}
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("provider not found: " + providerID)
}
