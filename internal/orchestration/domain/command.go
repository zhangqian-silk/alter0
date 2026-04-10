package domain

import (
	"context"

	shareddomain "alter0/internal/shared/domain"
)

type CommandRequest struct {
	Message shareddomain.UnifiedMessage
	Name    string
	Args    []string
}

type CommandResult struct {
	Output       string
	Metadata     map[string]string
	ProcessSteps []shareddomain.ProcessStep
}

type CommandHandler interface {
	Name() string
	Aliases() []string
	Execute(ctx context.Context, req CommandRequest) (CommandResult, error)
}

type CommandRegistry interface {
	Register(handler CommandHandler) error
	Resolve(name string) (CommandHandler, bool)
	Exists(name string) bool
	List() []string
}
