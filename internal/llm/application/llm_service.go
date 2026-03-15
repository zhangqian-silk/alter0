package application

import (
	"context"

	"alter0/internal/llm/domain"
)

// LLMService provides LLM configuration and client management.
type LLMService interface {
	// Configuration management
	GetConfig(ctx context.Context) (*domain.ModelConfig, error)
	GetProvider(ctx context.Context, providerID string) (*domain.ModelProvider, error)
	GetDefaultProvider(ctx context.Context) (*domain.ModelProvider, error)
	GetEnabledProviders(ctx context.Context) ([]domain.ModelProvider, error)
	AddProvider(ctx context.Context, provider domain.ModelProvider) error
	UpdateProvider(ctx context.Context, provider domain.ModelProvider) error
	RemoveProvider(ctx context.Context, providerID string) error
	SetDefaultProvider(ctx context.Context, providerID string) error
	EnableProvider(ctx context.Context, providerID string, enabled bool) error

	// Client management
	GetClient(ctx context.Context, providerID string) (domain.LLMClient, error)
	GetDefaultClient(ctx context.Context) (domain.LLMClient, error)

	// Agent creation
	GetReActAgent(ctx context.Context, providerID string, config domain.ReActAgentConfig) (*domain.ReActAgent, error)
}