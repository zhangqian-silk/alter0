package application

import (
	"context"
	"errors"
	"strings"
	"time"

	chatruntimedomain "alter0/internal/chatruntime/domain"
)

var (
	ErrRepositoryInvalid           = errors.New("repository reference is invalid")
	ErrRepositoryUnavailable       = errors.New("repository service is unavailable")
	ErrRepositoryBindingConflict   = errors.New("chat session is already associated with another repository")
	ErrRepositoryPreparationFailed = errors.New("repository preparation failed")
	ErrRepositoryRetryUnavailable  = errors.New("repository preparation is not retryable")
)

type RepositoryPage struct {
	Items      []chatruntimedomain.Repository `json:"repositories"`
	NextCursor string                         `json:"next_cursor,omitempty"`
}

type RepositoryCheckout struct {
	Branch  string
	HeadSHA string
}

type RepositoryCatalog interface {
	List(ctx context.Context, query string, cursor string) (RepositoryPage, error)
	Resolve(ctx context.Context, ref chatruntimedomain.RepositoryRef) (chatruntimedomain.Repository, error)
}

type RepositoryWorkspacePreparer interface {
	Prepare(ctx context.Context, repository chatruntimedomain.Repository, workspaceDir string) (RepositoryCheckout, error)
}

func (s *Service) ListRepositories(ctx context.Context, query string, cursor string) (RepositoryPage, error) {
	if s == nil || s.repositoryCatalog == nil {
		return RepositoryPage{}, ErrRepositoryUnavailable
	}
	page, err := s.repositoryCatalog.List(ctx, strings.TrimSpace(query), strings.TrimSpace(cursor))
	if err != nil {
		return RepositoryPage{}, ErrRepositoryUnavailable
	}
	return page, nil
}

func (s *Service) RetryRepository(ownerID string, sessionID string) (chatruntimedomain.Session, error) {
	item, err := s.getOrRestoreOwnedSession(ownerID, sessionID)
	if err != nil {
		return chatruntimedomain.Session{}, err
	}
	s.reconcileOrphanedRuntimeSession(item)

	turnCtx, turnCancel := context.WithCancel(s.rootCtx)
	item.mu.Lock()
	if item.turnRunning || chatruntimedomain.NormalizeSessionStatus(item.summary.Status) == chatruntimedomain.SessionStatusBusy {
		item.mu.Unlock()
		turnCancel()
		return chatruntimedomain.Session{}, ErrSessionBusy
	}
	if item.summary.Repository == nil || item.summary.Repository.Status != chatruntimedomain.RepositoryPreparationStatusFailed {
		item.mu.Unlock()
		turnCancel()
		return chatruntimedomain.Session{}, ErrRepositoryRetryUnavailable
	}
	var turn *runtimeTurn
	for index := len(item.turns) - 1; index >= 0; index-- {
		candidate := item.turns[index]
		if candidate == nil {
			continue
		}
		if status := strings.ToLower(strings.TrimSpace(candidate.Status)); status == "failed" || status == "interrupted" {
			turn = candidate
			break
		}
	}
	if turn == nil {
		item.mu.Unlock()
		turnCancel()
		return chatruntimedomain.Session{}, ErrRepositoryRetryUnavailable
	}

	now := time.Now().UTC()
	item.summary.Repository.Status = chatruntimedomain.RepositoryPreparationStatusPreparing
	item.summary.Repository.ErrorCode = ""
	item.summary.Repository.ErrorMessage = ""
	item.summary.Status = chatruntimedomain.SessionStatusBusy
	item.summary.ErrorMessage = ""
	item.summary.FinishedAt = time.Time{}
	item.summary.ExitCode = nil
	item.summary.UpdatedAt = now
	item.turnRunning = true
	item.turnCancel = turnCancel
	item.activeTurnID = turn.ID
	turn.Status = "running"
	turn.FinishedAt = time.Time{}
	turn.FinalOutput = ""
	prompt := turn.Prompt
	attachments := cloneTurnAttachments(turn.Attachments)
	skillContext := cloneChatRuntimeSkillContext(turn.SkillContext)
	turnID := turn.ID
	snapshot := item.summary
	snapshot.Repository = cloneRepositoryBinding(item.summary.Repository)
	item.mu.Unlock()

	s.persistSession(item)
	s.publishTurnSessionEvent(item, SessionEventTurnStarted, turnID, "")
	go s.runTurn(item, turnCtx, turnID, prompt, attachments, skillContext)
	return snapshot, nil
}

func normalizeRepositoryRef(ref chatruntimedomain.RepositoryRef) (chatruntimedomain.RepositoryRef, error) {
	normalized := chatruntimedomain.RepositoryRef{
		Provider: chatruntimedomain.RepositoryProvider(strings.ToLower(strings.TrimSpace(string(ref.Provider)))),
		ID:       strings.TrimSpace(ref.ID),
		FullName: strings.TrimSpace(ref.FullName),
	}
	if normalized.Provider != chatruntimedomain.RepositoryProviderGitHub || normalized.ID == "" {
		return chatruntimedomain.RepositoryRef{}, ErrRepositoryInvalid
	}
	return normalized, nil
}

func repositoryFromBinding(binding *chatruntimedomain.RepositoryBinding) chatruntimedomain.Repository {
	if binding == nil {
		return chatruntimedomain.Repository{}
	}
	return chatruntimedomain.Repository{
		Provider:      binding.Provider,
		ID:            binding.ID,
		FullName:      binding.FullName,
		Private:       binding.Private,
		DefaultBranch: binding.DefaultBranch,
	}
}

func cloneRepositoryBinding(binding *chatruntimedomain.RepositoryBinding) *chatruntimedomain.RepositoryBinding {
	if binding == nil {
		return nil
	}
	cloned := *binding
	return &cloned
}

func (s *Service) prepareRepositoryForTurn(item *runtimeSession, ctx context.Context) (*chatruntimedomain.RepositoryBinding, error) {
	if item == nil {
		return nil, nil
	}
	item.mu.RLock()
	binding := cloneRepositoryBinding(item.summary.Repository)
	workspaceDir := strings.TrimSpace(item.summary.WorkingDir)
	item.mu.RUnlock()
	if binding == nil {
		return nil, nil
	}
	if binding.Status == chatruntimedomain.RepositoryPreparationStatusReady {
		return binding, nil
	}
	if s.repositoryPreparer == nil {
		s.markRepositoryPreparationFailed(item)
		return nil, ErrRepositoryUnavailable
	}

	checkout, err := s.repositoryPreparer.Prepare(ctx, repositoryFromBinding(binding), workspaceDir)
	if err != nil {
		s.logger.Warn("prepare chat repository failed", "repository", binding.FullName, "error", err.Error())
		s.markRepositoryPreparationFailed(item)
		return nil, ErrRepositoryPreparationFailed
	}

	now := time.Now().UTC()
	item.mu.Lock()
	if item.summary.Repository == nil || !item.summary.Repository.Matches(chatruntimedomain.RepositoryRef{
		Provider: binding.Provider,
		ID:       binding.ID,
	}) {
		item.mu.Unlock()
		return nil, ErrRepositoryBindingConflict
	}
	item.summary.Repository.Status = chatruntimedomain.RepositoryPreparationStatusReady
	item.summary.Repository.Branch = strings.TrimSpace(checkout.Branch)
	if item.summary.Repository.Branch == "" {
		item.summary.Repository.Branch = strings.TrimSpace(item.summary.Repository.DefaultBranch)
	}
	item.summary.Repository.HeadSHA = strings.TrimSpace(checkout.HeadSHA)
	item.summary.Repository.WorkspacePath = chatruntimedomain.RepositoryWorkspacePath
	item.summary.Repository.ErrorCode = ""
	item.summary.Repository.ErrorMessage = ""
	item.summary.UpdatedAt = now
	ready := cloneRepositoryBinding(item.summary.Repository)
	item.mu.Unlock()

	s.persistSession(item)
	s.publishSessionEvent(item, SessionEventSessionUpdated)
	return ready, nil
}

func (s *Service) markRepositoryPreparationFailed(item *runtimeSession) {
	if item == nil {
		return
	}
	now := time.Now().UTC()
	item.mu.Lock()
	if item.summary.Repository != nil {
		item.summary.Repository.Status = chatruntimedomain.RepositoryPreparationStatusFailed
		item.summary.Repository.ErrorCode = "repository_prepare_failed"
		item.summary.Repository.ErrorMessage = "Failed to prepare repository."
		item.summary.Repository.Branch = ""
		item.summary.Repository.HeadSHA = ""
		item.summary.Repository.WorkspacePath = chatruntimedomain.RepositoryWorkspacePath
		item.summary.UpdatedAt = now
	}
	item.mu.Unlock()
	s.persistSession(item)
	s.publishSessionEvent(item, SessionEventSessionUpdated)
}

func normalizeRestoredRepositoryBinding(binding *chatruntimedomain.RepositoryBinding) *chatruntimedomain.RepositoryBinding {
	cloned := cloneRepositoryBinding(binding)
	if cloned == nil {
		return nil
	}
	cloned.Provider = chatruntimedomain.RepositoryProvider(strings.ToLower(strings.TrimSpace(string(cloned.Provider))))
	cloned.ID = strings.TrimSpace(cloned.ID)
	cloned.FullName = strings.TrimSpace(cloned.FullName)
	cloned.DefaultBranch = strings.TrimSpace(cloned.DefaultBranch)
	cloned.Branch = strings.TrimSpace(cloned.Branch)
	cloned.HeadSHA = strings.TrimSpace(cloned.HeadSHA)
	cloned.WorkspacePath = chatruntimedomain.RepositoryWorkspacePath
	cloned.Status = chatruntimedomain.NormalizeRepositoryPreparationStatus(cloned.Status)
	if cloned.Status == chatruntimedomain.RepositoryPreparationStatusPreparing {
		cloned.Status = chatruntimedomain.RepositoryPreparationStatusFailed
		cloned.ErrorCode = "repository_prepare_interrupted"
		cloned.ErrorMessage = "Repository preparation was interrupted."
	}
	return cloned
}
