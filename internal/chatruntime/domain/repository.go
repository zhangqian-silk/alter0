package domain

import (
	"strings"
	"time"
)

type RepositoryProvider string

const RepositoryProviderGitHub RepositoryProvider = "github"

const RepositoryWorkspacePath = "repo"

type RepositoryPreparationStatus string

const (
	RepositoryPreparationStatusPreparing RepositoryPreparationStatus = "preparing"
	RepositoryPreparationStatusReady     RepositoryPreparationStatus = "ready"
	RepositoryPreparationStatusFailed    RepositoryPreparationStatus = "failed"
)

func NormalizeRepositoryPreparationStatus(status RepositoryPreparationStatus) RepositoryPreparationStatus {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case string(RepositoryPreparationStatusPreparing):
		return RepositoryPreparationStatusPreparing
	case string(RepositoryPreparationStatusReady):
		return RepositoryPreparationStatusReady
	case string(RepositoryPreparationStatusFailed):
		return RepositoryPreparationStatusFailed
	default:
		return RepositoryPreparationStatusFailed
	}
}

type RepositoryRef struct {
	Provider RepositoryProvider `json:"provider"`
	ID       string             `json:"id"`
	FullName string             `json:"full_name,omitempty"`
}

type Repository struct {
	Provider      RepositoryProvider `json:"provider"`
	ID            string             `json:"id"`
	FullName      string             `json:"full_name"`
	Private       bool               `json:"private"`
	DefaultBranch string             `json:"default_branch"`
	UpdatedAt     time.Time          `json:"updated_at,omitempty"`
}

type RepositoryBinding struct {
	Provider      RepositoryProvider          `json:"provider"`
	ID            string                      `json:"id"`
	FullName      string                      `json:"full_name"`
	Private       bool                        `json:"private"`
	DefaultBranch string                      `json:"default_branch"`
	Branch        string                      `json:"branch,omitempty"`
	HeadSHA       string                      `json:"head_sha,omitempty"`
	Status        RepositoryPreparationStatus `json:"status"`
	WorkspacePath string                      `json:"workspace_path"`
	ErrorCode     string                      `json:"error_code,omitempty"`
	ErrorMessage  string                      `json:"error_message,omitempty"`
}

func NewRepositoryBinding(repository Repository) RepositoryBinding {
	return RepositoryBinding{
		Provider:      normalizeRepositoryProvider(repository.Provider),
		ID:            strings.TrimSpace(repository.ID),
		FullName:      strings.TrimSpace(repository.FullName),
		Private:       repository.Private,
		DefaultBranch: strings.TrimSpace(repository.DefaultBranch),
		Status:        RepositoryPreparationStatusPreparing,
		WorkspacePath: RepositoryWorkspacePath,
	}
}

func (b RepositoryBinding) Matches(ref RepositoryRef) bool {
	return normalizeRepositoryProvider(b.Provider) == normalizeRepositoryProvider(ref.Provider) &&
		strings.TrimSpace(b.ID) != "" &&
		strings.TrimSpace(b.ID) == strings.TrimSpace(ref.ID)
}

func normalizeRepositoryProvider(provider RepositoryProvider) RepositoryProvider {
	if strings.EqualFold(strings.TrimSpace(string(provider)), string(RepositoryProviderGitHub)) {
		return RepositoryProviderGitHub
	}
	return RepositoryProvider(strings.ToLower(strings.TrimSpace(string(provider))))
}
