package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	chatruntimedomain "alter0/internal/chatruntime/domain"
)

func TestGitHubRepositoryProviderListsSearchesAndPaginatesAccessibleRepositories(t *testing.T) {
	var calls [][]string
	provider := newGitHubRepositoryProviderWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return []byte(`[[
			{"id":101,"full_name":"owner/alter0","private":true,"default_branch":"master","updated_at":"2026-07-11T10:00:00Z"},
			{"id":102,"full_name":"owner/notes","private":false,"default_branch":"main","updated_at":"2026-07-10T10:00:00Z"},
			{"id":103,"full_name":"team/alter0-tools","private":false,"default_branch":"main","updated_at":"2026-07-09T10:00:00Z"}
		]]`), nil
	})

	page, err := provider.List(context.Background(), "alter0", "1")
	if err != nil {
		t.Fatalf("list repositories: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "103" || page.Items[0].FullName != "team/alter0-tools" {
		t.Fatalf("expected filtered page after cursor offset, got %+v", page)
	}
	if page.Items[0].Provider != chatruntimedomain.RepositoryProviderGitHub {
		t.Fatalf("expected github provider, got %+v", page.Items[0])
	}
	if len(calls) != 1 {
		t.Fatalf("expected one gh call, got %+v", calls)
	}
	joined := strings.Join(calls[0], " ")
	for _, want := range []string{"gh api", "--paginate", "--slurp", "user/repos"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected repository list command to contain %q, got %q", want, joined)
		}
	}
}

func TestGitHubRepositoryProviderResolvesTrustedMetadataByStableID(t *testing.T) {
	var call []string
	provider := newGitHubRepositoryProviderWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		call = append([]string{name}, args...)
		return []byte(`{"id":123456789,"full_name":"owner/repository","private":true,"default_branch":"main","updated_at":"2026-07-11T10:00:00Z"}`), nil
	})

	repository, err := provider.Resolve(context.Background(), chatruntimedomain.RepositoryRef{
		Provider: chatruntimedomain.RepositoryProviderGitHub,
		ID:       "123456789",
		FullName: "untrusted/name",
	})
	if err != nil {
		t.Fatalf("resolve repository: %v", err)
	}
	if repository.ID != "123456789" || repository.FullName != "owner/repository" || repository.DefaultBranch != "main" {
		t.Fatalf("expected trusted metadata, got %+v", repository)
	}
	if got := strings.Join(call, " "); !strings.Contains(got, "gh api repositories/123456789") {
		t.Fatalf("expected stable id lookup, got %q", got)
	}
}

func TestGitHubRepositoryProviderPreparesCheckoutThroughTokenFreeArguments(t *testing.T) {
	workspaceDir := t.TempDir()
	var calls [][]string
	provider := newGitHubRepositoryProviderWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		if name == "gh" && len(args) >= 4 && args[0] == "repo" && args[1] == "clone" {
			if err := os.MkdirAll(filepath.Join(args[3], ".git"), 0o755); err != nil {
				t.Fatalf("create fake checkout: %v", err)
			}
			return nil, nil
		}
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "--abbrev-ref HEAD") {
			return []byte("main\n"), nil
		}
		if strings.HasSuffix(joined, "rev-parse HEAD") {
			return []byte("abc123\n"), nil
		}
		return nil, nil
	})

	checkout, err := provider.Prepare(context.Background(), chatruntimedomain.Repository{
		Provider:      chatruntimedomain.RepositoryProviderGitHub,
		ID:            "123456789",
		FullName:      "owner/repository",
		DefaultBranch: "main",
	}, workspaceDir)
	if err != nil {
		t.Fatalf("prepare repository: %v", err)
	}
	if checkout.Branch != "main" || checkout.HeadSHA != "abc123" {
		t.Fatalf("expected checkout metadata, got %+v", checkout)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, chatruntimedomain.RepositoryWorkspacePath, ".git")); err != nil {
		t.Fatalf("expected ready checkout at repo/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, ".alter0", "repository-preparing")); !os.IsNotExist(err) {
		t.Fatalf("expected staging directory to be atomically moved, got err=%v", err)
	}
	for _, call := range calls {
		for _, arg := range call {
			normalized := strings.ToLower(arg)
			if strings.Contains(normalized, "token=") || strings.Contains(normalized, "authorization:") || strings.Contains(arg, "@github.com") {
				t.Fatalf("expected token-free command arguments, got %+v", call)
			}
		}
	}
}

func TestGitHubRepositoryProviderRejectsSymlinkedRepositoryTarget(t *testing.T) {
	workspaceDir := t.TempDir()
	externalDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(externalDir, ".git"), 0o755); err != nil {
		t.Fatalf("create external checkout: %v", err)
	}
	if err := os.Symlink(externalDir, filepath.Join(workspaceDir, chatruntimedomain.RepositoryWorkspacePath)); err != nil {
		t.Fatalf("create repository symlink: %v", err)
	}
	provider := newGitHubRepositoryProviderWithRunner(func(_ context.Context, _ string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "remote get-url origin"):
			return []byte("https://github.com/owner/repository.git\n"), nil
		case strings.Contains(joined, "--abbrev-ref HEAD"):
			return []byte("main\n"), nil
		default:
			return []byte("abc123\n"), nil
		}
	})

	_, err := provider.Prepare(context.Background(), chatruntimedomain.Repository{
		Provider: chatruntimedomain.RepositoryProviderGitHub,
		ID:       "123456789",
		FullName: "owner/repository",
	}, workspaceDir)
	if err != ErrRepositoryPreparationFailed {
		t.Fatalf("expected symlinked repository target to be rejected, got %v", err)
	}
}

func TestNormalizeGitHubRemoteFullNameIsCaseInsensitiveForURLPrefix(t *testing.T) {
	if got := normalizeGitHubRemoteFullName("HTTPS://GITHUB.COM/Owner/Repository.git"); got != "owner/repository" {
		t.Fatalf("expected normalized repository name, got %q", got)
	}
}
