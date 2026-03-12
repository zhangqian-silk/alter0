package domain

import (
	"errors"
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
	if p.ID == "" {
		return errors.New("provider id is required")
	}
	if p.Name == "" {
		return errors.New("provider name is required")
	}
	if p.BaseURL == "" {
		return errors.New("base_url is required")
	}
	if p.APIKey == "" {
		return errors.New("api_key is required")
	}
	return nil
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

	// Check for duplicate provider IDs
	ids := make(map[string]bool)
	for _, p := range c.Providers {
		if ids[p.ID] {
			return errors.New("duplicate provider id: " + p.ID)
		}
		ids[p.ID] = true

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
	if err := provider.Validate(); err != nil {
		return err
	}

	// Check for duplicate ID
	for _, p := range c.Providers {
		if p.ID == provider.ID {
			return errors.New("provider already exists: " + provider.ID)
		}
	}

	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()

	// Set as default if first provider
	if len(c.Providers) == 0 {
		provider.IsDefault = true
		c.DefaultProviderID = provider.ID
	}

	c.Providers = append(c.Providers, provider)
	c.UpdatedAt = time.Now()

	return nil
}

// UpdateProvider updates an existing provider.
func (c *ModelConfig) UpdateProvider(provider ModelProvider) error {
	for i := range c.Providers {
		if c.Providers[i].ID == provider.ID {
			provider.UpdatedAt = time.Now()
			provider.CreatedAt = c.Providers[i].CreatedAt
			c.Providers[i] = provider
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("provider not found: " + provider.ID)
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
					for _, p := range c.Providers {
						if p.IsEnabled {
							c.DefaultProviderID = p.ID
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
	// Check provider exists
	found := false
	for i := range c.Providers {
		if c.Providers[i].ID == providerID {
			found = true
			c.Providers[i].IsDefault = true
		} else {
			c.Providers[i].IsDefault = false
		}
	}
	if !found {
		return errors.New("provider not found: " + providerID)
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
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("provider not found: " + providerID)
}