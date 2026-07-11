package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	chatruntimedomain "alter0/internal/chatruntime/domain"
)

const githubRepositoryPageSize = 50

type repositoryCommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

type githubRepositoryProvider struct {
	runner repositoryCommandRunner
}

type githubRepositoryPayload struct {
	ID            json.Number `json:"id"`
	FullName      string      `json:"full_name"`
	Private       bool        `json:"private"`
	DefaultBranch string      `json:"default_branch"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

func newGitHubRepositoryProvider() *githubRepositoryProvider {
	return newGitHubRepositoryProviderWithRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("github command failed")
		}
		return output, nil
	})
}

func newGitHubRepositoryProviderWithRunner(runner repositoryCommandRunner) *githubRepositoryProvider {
	return &githubRepositoryProvider{runner: runner}
}

func (p *githubRepositoryProvider) List(ctx context.Context, query string, cursor string) (RepositoryPage, error) {
	if p == nil || p.runner == nil {
		return RepositoryPage{}, ErrRepositoryUnavailable
	}
	output, err := p.runner(ctx,
		"gh", "api", "--paginate", "--slurp", "--method", "GET", "user/repos",
		"-f", "per_page=100",
		"-f", "sort=updated",
		"-f", "affiliation=owner,collaborator,organization_member",
	)
	if err != nil {
		return RepositoryPage{}, ErrRepositoryUnavailable
	}
	payloads, err := decodeGitHubRepositoryPages(output)
	if err != nil {
		return RepositoryPage{}, ErrRepositoryUnavailable
	}

	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	items := make([]chatruntimedomain.Repository, 0, len(payloads))
	for _, payload := range payloads {
		item, ok := repositoryFromGitHubPayload(payload)
		if !ok || (normalizedQuery != "" && !strings.Contains(strings.ToLower(item.FullName), normalizedQuery)) {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i int, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	offset := 0
	if parsed, parseErr := strconv.Atoi(strings.TrimSpace(cursor)); parseErr == nil && parsed > 0 {
		offset = parsed
	}
	if offset > len(items) {
		offset = len(items)
	}
	end := offset + githubRepositoryPageSize
	if end > len(items) {
		end = len(items)
	}
	nextCursor := ""
	if end < len(items) {
		nextCursor = strconv.Itoa(end)
	}
	return RepositoryPage{
		Items:      append([]chatruntimedomain.Repository{}, items[offset:end]...),
		NextCursor: nextCursor,
	}, nil
}

func (p *githubRepositoryProvider) Resolve(ctx context.Context, ref chatruntimedomain.RepositoryRef) (chatruntimedomain.Repository, error) {
	if p == nil || p.runner == nil {
		return chatruntimedomain.Repository{}, ErrRepositoryUnavailable
	}
	normalized, err := normalizeRepositoryRef(ref)
	if err != nil {
		return chatruntimedomain.Repository{}, err
	}
	if _, err := strconv.ParseUint(normalized.ID, 10, 64); err != nil {
		return chatruntimedomain.Repository{}, ErrRepositoryInvalid
	}
	output, err := p.runner(ctx, "gh", "api", "repositories/"+normalized.ID)
	if err != nil {
		return chatruntimedomain.Repository{}, ErrRepositoryUnavailable
	}
	var payload githubRepositoryPayload
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return chatruntimedomain.Repository{}, ErrRepositoryUnavailable
	}
	repository, ok := repositoryFromGitHubPayload(payload)
	if !ok || repository.ID != normalized.ID {
		return chatruntimedomain.Repository{}, ErrRepositoryInvalid
	}
	return repository, nil
}

func (p *githubRepositoryProvider) Prepare(ctx context.Context, repository chatruntimedomain.Repository, workspaceDir string) (RepositoryCheckout, error) {
	if p == nil || p.runner == nil {
		return RepositoryCheckout{}, ErrRepositoryUnavailable
	}
	fullName, err := normalizeGitHubRepositoryFullName(repository.FullName)
	if err != nil {
		return RepositoryCheckout{}, err
	}
	workspaceDir = strings.TrimSpace(workspaceDir)
	if workspaceDir == "" {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	targetDir := filepath.Join(workspaceDir, chatruntimedomain.RepositoryWorkspacePath)
	if info, statErr := os.Lstat(targetDir); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return RepositoryCheckout{}, ErrRepositoryPreparationFailed
		}
		return p.inspectExistingCheckout(ctx, targetDir, fullName)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}

	stagingDir := filepath.Join(workspaceDir, ".alter0", "repository-preparing")
	if err := os.RemoveAll(stagingDir); err != nil {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	if err := os.MkdirAll(filepath.Dir(stagingDir), 0o755); err != nil {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	cloneArgs := []string{"repo", "clone", fullName, stagingDir}
	if branch := strings.TrimSpace(repository.DefaultBranch); branch != "" {
		cloneArgs = append(cloneArgs, "--", "--branch", branch, "--single-branch")
	}
	if _, err := p.runner(ctx, "gh", cloneArgs...); err != nil {
		_ = os.RemoveAll(stagingDir)
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	checkout, err := p.inspectCheckout(ctx, stagingDir)
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return RepositoryCheckout{}, err
	}
	if err := os.Rename(stagingDir, targetDir); err != nil {
		_ = os.RemoveAll(stagingDir)
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	return checkout, nil
}

func (p *githubRepositoryProvider) inspectExistingCheckout(ctx context.Context, targetDir string, expectedFullName string) (RepositoryCheckout, error) {
	gitInfo, err := os.Lstat(filepath.Join(targetDir, ".git"))
	if err != nil || gitInfo.Mode()&os.ModeSymlink != 0 {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	remote, err := p.runner(ctx, "git", "-C", targetDir, "remote", "get-url", "origin")
	if err != nil || normalizeGitHubRemoteFullName(string(remote)) != strings.ToLower(expectedFullName) {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	return p.inspectCheckout(ctx, targetDir)
}

func (p *githubRepositoryProvider) inspectCheckout(ctx context.Context, checkoutDir string) (RepositoryCheckout, error) {
	branchOutput, err := p.runner(ctx, "git", "-C", checkoutDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	headOutput, err := p.runner(ctx, "git", "-C", checkoutDir, "rev-parse", "HEAD")
	if err != nil {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	checkout := RepositoryCheckout{
		Branch:  strings.TrimSpace(string(branchOutput)),
		HeadSHA: strings.TrimSpace(string(headOutput)),
	}
	if checkout.Branch == "" || checkout.HeadSHA == "" {
		return RepositoryCheckout{}, ErrRepositoryPreparationFailed
	}
	return checkout, nil
}

func decodeGitHubRepositoryPages(data []byte) ([]githubRepositoryPayload, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var pages [][]githubRepositoryPayload
	if err := decoder.Decode(&pages); err == nil {
		items := make([]githubRepositoryPayload, 0)
		for _, page := range pages {
			items = append(items, page...)
		}
		return items, nil
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var items []githubRepositoryPayload
	if err := decoder.Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func repositoryFromGitHubPayload(payload githubRepositoryPayload) (chatruntimedomain.Repository, bool) {
	id := strings.TrimSpace(payload.ID.String())
	fullName := strings.TrimSpace(payload.FullName)
	if id == "" || fullName == "" {
		return chatruntimedomain.Repository{}, false
	}
	return chatruntimedomain.Repository{
		Provider:      chatruntimedomain.RepositoryProviderGitHub,
		ID:            id,
		FullName:      fullName,
		Private:       payload.Private,
		DefaultBranch: strings.TrimSpace(payload.DefaultBranch),
		UpdatedAt:     payload.UpdatedAt.UTC(),
	}, true
}

func normalizeGitHubRepositoryFullName(value string) (string, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(value), "/"), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", ErrRepositoryInvalid
	}
	for _, part := range parts {
		if part == "." || part == ".." || strings.ContainsAny(part, "\\\x00\r\n") {
			return "", ErrRepositoryInvalid
		}
	}
	return strings.TrimSpace(parts[0]) + "/" + strings.TrimSpace(parts[1]), nil
}

func normalizeGitHubRemoteFullName(value string) string {
	remote := strings.TrimSpace(value)
	if len(remote) >= len(".git") && strings.EqualFold(remote[len(remote)-len(".git"):], ".git") {
		remote = remote[:len(remote)-len(".git")]
	}
	for _, prefix := range []string{
		"https://github.com/",
		"http://github.com/",
		"ssh://git@github.com/",
		"git@github.com:",
	} {
		if strings.HasPrefix(strings.ToLower(remote), strings.ToLower(prefix)) {
			return strings.ToLower(strings.Trim(remote[len(prefix):], "/"))
		}
	}
	return ""
}
