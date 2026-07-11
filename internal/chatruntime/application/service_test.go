package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	chatruntimedomain "alter0/internal/chatruntime/domain"
	execdomain "alter0/internal/execution/domain"
)

func TestNextSessionContentUpdatedAtIsStrictlyMonotonicAtMillisecondPrecision(t *testing.T) {
	previous := time.Date(2026, time.July, 10, 9, 30, 0, 123_000_000, time.UTC)

	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "same millisecond",
			now:  previous.Add(400 * time.Microsecond),
			want: previous.Add(time.Millisecond),
		},
		{
			name: "clock moved backwards",
			now:  previous.Add(-time.Minute),
			want: previous.Add(time.Millisecond),
		},
		{
			name: "later millisecond",
			now:  previous.Add(7*time.Millisecond + 800*time.Microsecond),
			want: previous.Add(7 * time.Millisecond),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextSessionContentUpdatedAt(previous, tt.now)
			if !got.Equal(tt.want) {
				t.Fatalf("expected %s, got %s", tt.want.Format(time.RFC3339Nano), got.Format(time.RFC3339Nano))
			}
		})
	}
}

func TestApplyExternalThreadTitleDoesNotInvalidateConversationContent(t *testing.T) {
	contentUpdatedAt := time.Date(2026, time.July, 10, 9, 30, 0, 123_000_000, time.UTC)
	session := &runtimeSession{
		summary: chatruntimedomain.Session{
			Title:     "Old title",
			UpdatedAt: contentUpdatedAt,
		},
	}

	if !applyExternalThreadTitleLocked(session, "New title", contentUpdatedAt.Add(time.Minute)) {
		t.Fatal("expected title to change")
	}
	if !session.summary.UpdatedAt.Equal(contentUpdatedAt) {
		t.Fatalf("title-only change invalidated conversation content: got %s", session.summary.UpdatedAt)
	}
}

func TestServiceGetDetailReturnsSessionAndTurnsFromOneSnapshot(t *testing.T) {
	service := newTestService("success")
	session, err := service.Create(CreateRequest{OwnerID: "owner-detail-snapshot"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.InputWithAttachments(InputRequest{
		OwnerID:         "owner-detail-snapshot",
		SessionID:       session.ID,
		Input:           "hello",
		ClientRequestID: "request-123",
	}); err != nil {
		t.Fatalf("input: %v", err)
	}
	_, _ = waitForSessionEntries(t, service, "owner-detail-snapshot", session.ID, 2)

	detail, ok := service.GetDetail("owner-detail-snapshot", session.ID)
	if !ok {
		t.Fatal("expected detail snapshot")
	}
	if detail.Session.ID != session.ID || len(detail.Turns) != 1 {
		t.Fatalf("unexpected detail snapshot: %+v", detail)
	}
	if detail.Turns[0].ClientRequestID != "request-123" {
		t.Fatalf("expected client request id in snapshot, got %+v", detail.Turns[0])
	}
}

func TestResolveCodexCommandUsesDefaultCommand(t *testing.T) {
	command := resolveCodexCommand(Options{})

	if command.path != defaultCodexCommand {
		t.Fatalf("expected default codex command, got %q", command.path)
	}
	if command.label != "codex exec" {
		t.Fatalf("expected codex exec label, got %q", command.label)
	}
}

func TestBuildCodexTurnArgsUsesResumeWhenThreadExists(t *testing.T) {
	command := resolveCodexCommand(Options{
		Shell:     "codex.exe",
		ShellArgs: []string{"--profile", "test"},
	})

	args := buildCodexTurnArgs(command, "thread-123", "reply exactly", nil)

	got := strings.Join(args, " ")
	for _, part := range []string{"--profile", "test", "exec", "resume", "--json", "--skip-git-repo-check", "thread-123", "reply exactly"} {
		if !strings.Contains(got, part) {
			t.Fatalf("expected args to contain %q, got %v", part, args)
		}
	}
}

func TestBuildCodexTurnArgsIncludesImageFlags(t *testing.T) {
	command := resolveCodexCommand(Options{})

	args := buildCodexTurnArgs(command, "", "inspect screenshot", []string{"/tmp/a.png", "/tmp/b.webp"})
	got := strings.Join(args, " ")

	for _, part := range []string{"-i /tmp/a.png", "-i /tmp/b.webp", "inspect screenshot"} {
		if !strings.Contains(got, part) {
			t.Fatalf("expected image args to contain %q, got %v", part, args)
		}
	}
}

func TestBuildCodexTurnPromptIncludesWorkspaceFiles(t *testing.T) {
	prompt := buildCodexTurnPrompt("review the attached docs", []preparedTurnAttachment{
		{
			Name:        "requirements.md",
			ContentType: "text/markdown",
			PromptPath:  "input-attachments/turn-1/requirements.md",
		},
		{
			Name:        "diagram.png",
			ContentType: "image/png",
			PromptPath:  "input-attachments/turn-1/diagram.png",
			IsImage:     true,
		},
	}, nil)

	if !strings.Contains(prompt, "Attached files are available in the workspace:") {
		t.Fatalf("expected file attachment note, got %q", prompt)
	}
	if !strings.Contains(prompt, "requirements.md (text/markdown): input-attachments/turn-1/requirements.md") {
		t.Fatalf("expected markdown file path in prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "diagram.png") {
		t.Fatalf("expected image attachments to stay on CLI flags instead of prompt text, got %q", prompt)
	}
}

func TestBuildCodexTurnPromptIncludesTrustedRepositoryContextSeparately(t *testing.T) {
	binding := chatruntimedomain.NewRepositoryBinding(chatruntimedomain.Repository{
		Provider:      chatruntimedomain.RepositoryProviderGitHub,
		ID:            "123456789",
		FullName:      "owner/repository",
		DefaultBranch: "main",
	})
	binding.Status = chatruntimedomain.RepositoryPreparationStatusReady
	binding.Branch = "main"
	binding.HeadSHA = "abc123"

	prompt := buildCodexTurnPrompt("Update the retry behavior", nil, &binding)

	for _, want := range []string{
		"Update the retry behavior",
		"Repository context:",
		"- repository: owner/repository",
		"- path: repo/",
		"- branch: main",
		"- head: abc123",
		"This user message is associated with the repository above.",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected repository prompt to contain %q, got:\n%s", want, prompt)
		}
	}
}

func TestRenderChatRuntimeSkillContextMarkdownIncludesSelectedSkills(t *testing.T) {
	content := renderChatRuntimeSkillContextMarkdown(&execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills: []execdomain.SkillSpec{{
			ID:          "summary",
			Name:        "Summary",
			Description: "Summarize chat work.",
			Guide:       "Use concise structured summaries.",
			FilePath:    ".alter0/skills/summary/SKILL.md",
			Constraints: []string{"Keep output brief."},
		}},
	})

	for _, want := range []string{
		"# Skills",
		"- protocol: alter0.skill-context/v1",
		"## Summary",
		"- id: summary",
		"- file_path: .alter0/skills/summary/SKILL.md",
		"Use concise structured summaries.",
		"- Keep output brief.",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected skill markdown to contain %q, got:\n%s", want, content)
		}
	}
}

func TestRenderChatRuntimeSkillContextMarkdownMarksEmptySelection(t *testing.T) {
	content := renderChatRuntimeSkillContextMarkdown(nil)

	if !strings.Contains(content, "No skills selected for this Chat turn.") {
		t.Fatalf("expected empty skill selection marker, got:\n%s", content)
	}
}

func TestPrepareChatRuntimeCodexRuntimeMaterializesSelectedSkillFiles(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, "docs", "skills", "frontend-design", "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "docs", "skills", "frontend-design", "SKILL.md"), []byte("# Frontend Design\n"), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "docs", "skills", "frontend-design", "scripts", "helper.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write skill helper: %v", err)
	}
	activeHome := t.TempDir()
	t.Setenv("CODEX_HOME", activeHome)
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	_, err = prepareChatRuntimeCodexRuntime(workspaceDir, &execdomain.SkillContext{
		Protocol: execdomain.SkillContextProtocolVersion,
		Skills: []execdomain.SkillSpec{{
			ID:       "frontend-design",
			Name:     "Frontend Design",
			FilePath: "docs/skills/frontend-design/SKILL.md",
		}},
	})
	if err != nil {
		t.Fatalf("prepareChatRuntimeCodexRuntime() error = %v", err)
	}

	materializedPath := filepath.Join(workspaceDir, ".alter0", "codex-runtime", "skills", "frontend-design", "SKILL.md")
	materialized, err := os.ReadFile(materializedPath)
	if err != nil {
		t.Fatalf("read materialized skill file: %v", err)
	}
	if string(materialized) != "# Frontend Design\n" {
		t.Fatalf("unexpected materialized skill file: %q", string(materialized))
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, ".alter0", "codex-runtime", "skills", "frontend-design", "scripts", "helper.sh")); err != nil {
		t.Fatalf("expected skill helper directory to be materialized: %v", err)
	}
	skillsMarkdown, err := os.ReadFile(filepath.Join(workspaceDir, ".alter0", "codex-runtime", "skills.md"))
	if err != nil {
		t.Fatalf("read runtime skills markdown: %v", err)
	}
	if !strings.Contains(string(skillsMarkdown), "- file_path: .alter0/codex-runtime/skills/frontend-design/SKILL.md") {
		t.Fatalf("expected runtime skill file_path to point inside workspace, got:\n%s", string(skillsMarkdown))
	}
	if strings.Contains(string(skillsMarkdown), "- file_path: docs/skills/frontend-design/SKILL.md") {
		t.Fatalf("expected source-relative skill file_path to be rewritten, got:\n%s", string(skillsMarkdown))
	}
}

func TestPrepareTurnInputAttachmentsUsesWorkspaceFilesWithoutDataURLs(t *testing.T) {
	workspaceDir := t.TempDir()
	sourcePath := filepath.Join(workspaceDir, "source-requirements.md")
	if err := os.WriteFile(sourcePath, []byte("# Requirements\n"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	attachments, err := prepareTurnInputAttachments(workspaceDir, "turn-1", []TurnAttachment{
		{
			Name:          "requirements.md",
			ContentType:   "text/markdown",
			WorkspacePath: sourcePath,
		},
	})
	if err != nil {
		t.Fatalf("prepareTurnInputAttachments() error = %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 prepared attachment, got %+v", attachments)
	}
	if attachments[0].PromptPath != filepath.ToSlash(filepath.Join(chatRuntimeTurnAttachmentDirName, "turn-1", "requirements.md")) {
		t.Fatalf("unexpected prompt path %+v", attachments[0])
	}
	data, err := os.ReadFile(attachments[0].Path)
	if err != nil {
		t.Fatalf("read prepared attachment: %v", err)
	}
	if string(data) != "# Requirements\n" {
		t.Fatalf("unexpected prepared attachment content %q", string(data))
	}
}

func TestNormalizeOptionsParsesShellArgsLine(t *testing.T) {
	options := normalizeOptions(Options{
		Shell:         "bash",
		ShellArgsLine: `"./fixtures/codex mock.sh" --profile test`,
	})

	expected := []string{"./fixtures/codex mock.sh", "--profile", "test"}
	if strings.Join(options.ShellArgs, "|") != strings.Join(expected, "|") {
		t.Fatalf("expected parsed shell args %v, got %v", expected, options.ShellArgs)
	}
}

func TestCreateAssignsSessionWorkspaceDir(t *testing.T) {
	baseDir := t.TempDir()
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: baseDir})

	session, err := service.Create(CreateRequest{OwnerID: "owner-workspace"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	expected := filepath.Join(baseDir, "workspaces", "chat", "sessions", session.ID)
	if filepath.Clean(session.WorkingDir) != filepath.Clean(expected) {
		t.Fatalf("expected workspace %q, got %q", expected, session.WorkingDir)
	}
	info, statErr := os.Stat(session.WorkingDir)
	if statErr != nil {
		t.Fatalf("stat workspace dir: %v", statErr)
	}
	if !info.IsDir() {
		t.Fatalf("expected workspace directory, got file")
	}
}

func TestCreateUsesNewAsDefaultSessionTitle(t *testing.T) {
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: t.TempDir()})

	session, err := service.Create(CreateRequest{OwnerID: "owner-title"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if session.Title != "New" {
		t.Fatalf("expected default chatRuntime session title %q, got %q", "New", session.Title)
	}
	if !regexp.MustCompile(`^c_[a-z0-9]{16}$`).MatchString(session.ID) {
		t.Fatalf("expected generated compact chatRuntime session id, got %q", session.ID)
	}
	if strings.HasPrefix(session.ID, "chat-") || strings.Contains(session.ID, "T") {
		t.Fatalf("expected session id without legacy chat timestamp shape, got %q", session.ID)
	}
	if session.RuntimeSessionID != session.ID {
		t.Fatalf("expected runtime session id to use canonical session id, got %q", session.RuntimeSessionID)
	}
}

func TestRecoverAssignsDeterministicWorkspaceDir(t *testing.T) {
	baseDir := t.TempDir()
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: baseDir})

	session, err := service.Recover(RecoverRequest{
		OwnerID:   "owner-recover",
		SessionID: "chat-recover",
	})
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}

	expected := filepath.Join(baseDir, "workspaces", "chat", "sessions", "chat-recover")
	if filepath.Clean(session.WorkingDir) != filepath.Clean(expected) {
		t.Fatalf("expected recovered workspace %q, got %q", expected, session.WorkingDir)
	}
}

func TestCreateAssignsDistinctWorkspacePerSession(t *testing.T) {
	baseDir := t.TempDir()
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: baseDir})

	first, err := service.Create(CreateRequest{OwnerID: "owner-a"})
	if err != nil {
		t.Fatalf("create first session: %v", err)
	}
	second, err := service.Create(CreateRequest{OwnerID: "owner-b"})
	if err != nil {
		t.Fatalf("create second session: %v", err)
	}

	if filepath.Clean(first.WorkingDir) == filepath.Clean(second.WorkingDir) {
		t.Fatalf("expected distinct workspaces, got %q", first.WorkingDir)
	}
}

func TestChatRuntimeInputUsesSessionScopedCodexHome(t *testing.T) {
	baseDir := t.TempDir()
	activeHome := filepath.Join(t.TempDir(), "active-codex-home")
	if err := os.MkdirAll(activeHome, 0o755); err != nil {
		t.Fatalf("mkdir active home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(activeHome, "auth.json"), []byte(`{"auth_mode":"apikey","OPENAI_API_KEY":"sk-test"}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	t.Setenv("CODEX_HOME", activeHome)

	service := newTestServiceWithBaseDir("success", baseDir)
	session, err := service.Create(CreateRequest{OwnerID: "owner-runtime-home"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Setenv("CHAT_RUNTIME_HELPER_EXPECT_CODEX_HOME_SUFFIX", filepath.Join(session.WorkingDir, chatRuntimeCodexHomeDirName))

	if _, err := service.Input("owner-runtime-home", session.ID, "first prompt"); err != nil {
		t.Fatalf("input: %v", err)
	}
	waitForSessionEntries(t, service, "owner-runtime-home", session.ID, 2)

	if _, err := os.Stat(filepath.Join(session.WorkingDir, chatRuntimeCodexHomeDirName, "auth.json")); err != nil {
		t.Fatalf("expected session codex auth copy: %v", err)
	}
}

func TestCreateStartsSessionReady(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-ready",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.Status != chatruntimedomain.SessionStatusReady {
		t.Fatalf("expected ready status after create, got %q", session.Status)
	}
}

func TestPersistedSessionSnapshotClonesRepositoryBinding(t *testing.T) {
	binding := chatruntimedomain.NewRepositoryBinding(chatruntimedomain.Repository{
		Provider:      chatruntimedomain.RepositoryProviderGitHub,
		ID:            "123456789",
		FullName:      "owner/repository",
		DefaultBranch: "main",
	})
	item := &runtimeSession{summary: chatruntimedomain.Session{
		ID:         "c_snapshot0000000",
		OwnerID:    "owner-snapshot",
		Status:     chatruntimedomain.SessionStatusBusy,
		Repository: &binding,
	}}

	record, deleted := snapshotPersistedSession(item)
	if deleted || record.Summary.Repository == nil {
		t.Fatalf("expected repository in persisted snapshot, got %+v", record.Summary.Repository)
	}
	item.summary.Repository.Status = chatruntimedomain.RepositoryPreparationStatusReady
	item.summary.Repository.HeadSHA = "changed-after-snapshot"

	if record.Summary.Repository.Status != chatruntimedomain.RepositoryPreparationStatusPreparing || record.Summary.Repository.HeadSHA != "" {
		t.Fatalf("expected immutable persisted snapshot, got %+v", record.Summary.Repository)
	}
}

func TestServiceInputStartsAndResumesCodexSession(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-a",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-a", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}

	firstSnapshot, firstEntries := waitForSessionEntries(t, service, "owner-a", session.ID, 2)
	if firstSnapshot.RuntimeSessionID != "thread-first-prompt" {
		t.Fatalf("expected runtime thread id, got %q", firstSnapshot.RuntimeSessionID)
	}
	if firstSnapshot.Status != chatruntimedomain.SessionStatusReady {
		t.Fatalf("expected ready after first turn, got %q", firstSnapshot.Status)
	}
	if got := firstEntries[1].Text; got != "mock:first prompt" {
		t.Fatalf("expected first reply, got %q", got)
	}

	if _, err := service.Input("owner-a", session.ID, "second prompt"); err != nil {
		t.Fatalf("second input: %v", err)
	}

	secondSnapshot, secondEntries := waitForSessionEntries(t, service, "owner-a", session.ID, 4)
	if secondSnapshot.RuntimeSessionID != "thread-first-prompt" {
		t.Fatalf("expected resumed thread id, got %q", secondSnapshot.RuntimeSessionID)
	}
	if got := secondEntries[3].Text; got != "mock:second prompt" {
		t.Fatalf("expected second reply, got %q", got)
	}
}

func TestServiceMissingCodexRolloutExportsHistoryAndDirectsUserToNewSession(t *testing.T) {
	baseDir := t.TempDir()
	service := newTestServiceWithBaseDir("missing-rollout", baseDir)

	session, err := service.Create(CreateRequest{OwnerID: "owner-missing-rollout"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-missing-rollout", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	waitForSessionEntries(t, service, "owner-missing-rollout", session.ID, 2)

	if _, err := service.Input("owner-missing-rollout", session.ID, "continue the work"); err != nil {
		t.Fatalf("follow-up input: %v", err)
	}
	snapshot, entries := waitForSessionStatus(t, service, "owner-missing-rollout", session.ID, chatruntimedomain.SessionStatusFailed)

	historyPath, err := resolveChatRuntimeHistoryExportPath(baseDir, session.ID)
	if err != nil {
		t.Fatalf("resolve history path: %v", err)
	}
	if !strings.Contains(snapshot.ErrorMessage, "请新建会话") || !strings.Contains(snapshot.ErrorMessage, historyPath) {
		t.Fatalf("expected new-session recovery guidance, got %q", snapshot.ErrorMessage)
	}
	if !strings.Contains(snapshot.ErrorMessage, "请读取 "+historyPath+" 中的历史会话消息，并在此基础上继续") {
		t.Fatalf("expected directly copyable history prompt, got %q", snapshot.ErrorMessage)
	}
	if got := entries[len(entries)-1].Text; !strings.Contains(got, historyPath) {
		t.Fatalf("expected timeline error to include history path, got %q", got)
	}
	detail, ok := service.GetDetail("owner-missing-rollout", session.ID)
	if !ok || len(detail.Turns) != 2 {
		t.Fatalf("expected failed turn detail, got %+v", detail)
	}
	if !strings.Contains(detail.Turns[1].FinalOutput, historyPath) {
		t.Fatalf("expected copyable guidance as failed assistant output, got %q", detail.Turns[1].FinalOutput)
	}

	history, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("read exported history: %v", err)
	}
	for _, want := range []string{"# Alter0 会话历史", "first prompt", "mock:first prompt", "continue the work"} {
		if !strings.Contains(string(history), want) {
			t.Fatalf("expected exported history to contain %q, got:\n%s", want, history)
		}
	}

	if _, err := service.Input("owner-missing-rollout", session.ID, "retry once more"); err != nil {
		t.Fatalf("retry missing rollout input: %v", err)
	}
	_, retryEntries := waitForSessionStatus(t, service, "owner-missing-rollout", session.ID, chatruntimedomain.SessionStatusFailed)
	if len(retryEntries) < 6 {
		t.Fatalf("expected retry failure entries, got %d", len(retryEntries))
	}
	history, err = os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("read refreshed history: %v", err)
	}
	if strings.Contains(string(history), "底层 Codex 历史线程文件已丢失") {
		t.Fatalf("expected recovery guidance to stay out of exported assistant history, got:\n%s", history)
	}
	if !strings.Contains(string(history), "retry once more") {
		t.Fatalf("expected refreshed history to include the latest user retry, got:\n%s", history)
	}
}

func TestServiceInputWithAttachmentsPassesImageFlagsAndPersistsTurnAttachments(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-images",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	t.Setenv("CHAT_RUNTIME_HELPER_EXPECT_IMAGE_COUNT", "1")
	attachment := execdomain.UserAttachment{
		Name:        "diagram.png",
		ContentType: "image/png",
		DataURL:     "data:image/png;base64,ZmFrZQ==",
	}
	if _, err := service.InputWithAttachments(InputRequest{
		OwnerID:     "owner-images",
		SessionID:   session.ID,
		Input:       "inspect screenshot",
		Attachments: []execdomain.UserAttachment{attachment},
	}); err != nil {
		t.Fatalf("input with attachments: %v", err)
	}

	_, entries := waitForSessionEntries(t, service, "owner-images", session.ID, 2)
	if got := entries[1].Text; got != "mock:inspect screenshot" {
		t.Fatalf("expected attached turn reply, got %q", got)
	}

	turns, err := service.ListTurns("owner-images", session.ID)
	if err != nil {
		t.Fatalf("list turns: %v", err)
	}
	if len(turns) != 1 || len(turns[0].Attachments) != 1 {
		t.Fatalf("expected turn attachments to persist, got %+v", turns)
	}
	if turns[0].Attachments[0].DataURL != attachment.DataURL {
		t.Fatalf("expected persisted attachment data url, got %+v", turns[0].Attachments[0])
	}
}

func TestServiceInputWithRepositoryPreparesCheckoutBeforeCodexAndPersistsBinding(t *testing.T) {
	baseDir := t.TempDir()
	catalog := &stubRepositoryCatalog{
		resolved: chatruntimedomain.Repository{
			Provider:      chatruntimedomain.RepositoryProviderGitHub,
			ID:            "123456789",
			FullName:      "owner/repository",
			Private:       true,
			DefaultBranch: "main",
		},
	}
	preparer := &stubRepositoryWorkspacePreparer{
		checkout: RepositoryCheckout{Branch: "main", HeadSHA: "abc123"},
	}
	service := newTestServiceWithRepositorySupport("success", baseDir, catalog, preparer)
	session, err := service.Create(CreateRequest{OwnerID: "owner-repository"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	started, err := service.InputWithAttachments(InputRequest{
		OwnerID:   "owner-repository",
		SessionID: session.ID,
		Input:     "Update the retry behavior",
		Repository: &chatruntimedomain.RepositoryRef{
			Provider: chatruntimedomain.RepositoryProviderGitHub,
			ID:       "123456789",
			FullName: "untrusted/old-name",
		},
	})
	if err != nil {
		t.Fatalf("input with repository: %v", err)
	}
	if started.Repository == nil || started.Repository.Status != chatruntimedomain.RepositoryPreparationStatusPreparing {
		t.Fatalf("expected preparing binding in immediate snapshot, got %+v", started.Repository)
	}
	if started.Repository.FullName != "owner/repository" {
		t.Fatalf("expected trusted catalog metadata, got %+v", started.Repository)
	}

	finished, entries := waitForSessionEntries(t, service, "owner-repository", session.ID, 2)
	if finished.Repository == nil || finished.Repository.Status != chatruntimedomain.RepositoryPreparationStatusReady {
		t.Fatalf("expected ready repository binding, got %+v", finished.Repository)
	}
	if finished.Repository.Branch != "main" || finished.Repository.HeadSHA != "abc123" {
		t.Fatalf("expected checkout metadata, got %+v", finished.Repository)
	}
	if preparer.calls != 1 || preparer.workspaceDir != session.WorkingDir {
		t.Fatalf("expected one preparation in session workspace, got calls=%d dir=%q", preparer.calls, preparer.workspaceDir)
	}
	if !strings.Contains(entries[1].Text, "Repository context:") || !strings.Contains(entries[1].Text, "owner/repository") {
		t.Fatalf("expected codex to receive repository context, got %q", entries[1].Text)
	}

	restarted := NewService(context.Background(), nil, nil, Options{
		WorkingDir:         baseDir,
		RepositoryCatalog:  catalog,
		RepositoryPreparer: preparer,
	})
	restored, ok := restarted.Get("owner-repository", session.ID)
	if !ok || restored.Repository == nil {
		t.Fatalf("expected persisted repository binding after restart, got %+v", restored.Repository)
	}
	if restored.Repository.Status != chatruntimedomain.RepositoryPreparationStatusReady || restored.Repository.HeadSHA != "abc123" {
		t.Fatalf("expected restored ready checkout metadata, got %+v", restored.Repository)
	}
}

func TestServiceInputRejectsChangingRepositoryAfterBinding(t *testing.T) {
	catalog := &stubRepositoryCatalog{resolved: chatruntimedomain.Repository{
		Provider:      chatruntimedomain.RepositoryProviderGitHub,
		ID:            "123456789",
		FullName:      "owner/repository",
		DefaultBranch: "main",
	}}
	preparer := &stubRepositoryWorkspacePreparer{checkout: RepositoryCheckout{Branch: "main", HeadSHA: "abc123"}}
	service := newTestServiceWithRepositorySupport("success", t.TempDir(), catalog, preparer)
	session, err := service.Create(CreateRequest{OwnerID: "owner-conflict"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.InputWithAttachments(InputRequest{
		OwnerID:   "owner-conflict",
		SessionID: session.ID,
		Input:     "first",
		Repository: &chatruntimedomain.RepositoryRef{
			Provider: chatruntimedomain.RepositoryProviderGitHub,
			ID:       "123456789",
		},
	}); err != nil {
		t.Fatalf("first input: %v", err)
	}
	waitForSessionEntries(t, service, "owner-conflict", session.ID, 2)

	_, err = service.InputWithAttachments(InputRequest{
		OwnerID:   "owner-conflict",
		SessionID: session.ID,
		Input:     "second",
		Repository: &chatruntimedomain.RepositoryRef{
			Provider: chatruntimedomain.RepositoryProviderGitHub,
			ID:       "987654321",
		},
	})
	if !errors.Is(err, ErrRepositoryBindingConflict) {
		t.Fatalf("expected repository binding conflict, got %v", err)
	}
	turns, listErr := service.ListTurns("owner-conflict", session.ID)
	if listErr != nil || len(turns) != 1 {
		t.Fatalf("expected rejected input not to create another turn, got turns=%+v err=%v", turns, listErr)
	}
}

func TestServiceRetryRepositoryResumesPersistedTurnWithoutDuplicateInput(t *testing.T) {
	baseDir := t.TempDir()
	catalog := &stubRepositoryCatalog{resolved: chatruntimedomain.Repository{
		Provider:      chatruntimedomain.RepositoryProviderGitHub,
		ID:            "123456789",
		FullName:      "owner/repository",
		DefaultBranch: "main",
	}}
	failingPreparer := &stubRepositoryWorkspacePreparer{err: errors.New("credential helper detail must stay private")}
	service := newTestServiceWithRepositorySupport("success", baseDir, catalog, failingPreparer)
	session, err := service.Create(CreateRequest{OwnerID: "owner-retry"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	_, err = service.InputWithAttachments(InputRequest{
		OwnerID:   "owner-retry",
		SessionID: session.ID,
		Input:     "Retry the checkout",
		Repository: &chatruntimedomain.RepositoryRef{
			Provider: chatruntimedomain.RepositoryProviderGitHub,
			ID:       "123456789",
		},
		SkillContext: &execdomain.SkillContext{
			Protocol: execdomain.SkillContextProtocolVersion,
			Skills: []execdomain.SkillSpec{{
				ID:    "repository-review",
				Name:  "Repository Review",
				Guide: "Review the selected repository carefully.",
			}},
		},
	})
	if err != nil {
		t.Fatalf("input with failing preparation: %v", err)
	}
	failed, _ := waitForSessionError(t, service, "owner-retry", session.ID)
	if failed.Repository == nil || failed.Repository.Status != chatruntimedomain.RepositoryPreparationStatusFailed {
		t.Fatalf("expected failed repository binding, got %+v", failed.Repository)
	}
	if strings.Contains(failed.Repository.ErrorMessage, "credential helper") {
		t.Fatalf("expected sanitized repository failure, got %+v", failed.Repository)
	}

	successfulPreparer := &stubRepositoryWorkspacePreparer{checkout: RepositoryCheckout{Branch: "main", HeadSHA: "def456"}}
	restarted := newTestServiceWithRepositorySupport("success", baseDir, catalog, successfulPreparer)
	retrying, err := restarted.RetryRepository("owner-retry", session.ID)
	if err != nil {
		t.Fatalf("retry repository: %v", err)
	}
	if retrying.Repository == nil || retrying.Repository.Status != chatruntimedomain.RepositoryPreparationStatusPreparing {
		t.Fatalf("expected retry to return preparing state, got %+v", retrying.Repository)
	}

	finished, entries := waitForSessionEntries(t, restarted, "owner-retry", session.ID, 3)
	if finished.Repository == nil || finished.Repository.Status != chatruntimedomain.RepositoryPreparationStatusReady || finished.Repository.HeadSHA != "def456" {
		t.Fatalf("expected successful retry checkout, got %+v", finished.Repository)
	}
	inputEntries := 0
	for _, entry := range entries {
		if entry.Stream == "input" {
			inputEntries++
		}
	}
	if inputEntries != 1 {
		t.Fatalf("expected retry not to duplicate user input, got %+v", entries)
	}
	turns, err := restarted.ListTurns("owner-retry", session.ID)
	if err != nil || len(turns) != 1 {
		t.Fatalf("expected retry to reuse one turn, got turns=%+v err=%v", turns, err)
	}
	skillContext, err := os.ReadFile(filepath.Join(session.WorkingDir, ".alter0", "codex-runtime", "skills.md"))
	if err != nil {
		t.Fatalf("read retried skill context: %v", err)
	}
	if !strings.Contains(string(skillContext), "Review the selected repository carefully.") {
		t.Fatalf("expected retry to preserve selected skill context, got:\n%s", skillContext)
	}
}

func TestServiceListTurnsIncludesRuntimeTraceEventBlocks(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-step-blocks",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	now := time.Date(2026, 6, 24, 4, 31, 0, 0, time.UTC)
	service.mu.RLock()
	item := service.sessions[session.ID]
	service.mu.RUnlock()
	if item == nil {
		t.Fatalf("expected runtime session")
	}

	item.mu.Lock()
	item.turns = append(item.turns, &runtimeTurn{
		ID:          "turn-1",
		Prompt:      "render process details",
		Status:      "completed",
		StartedAt:   now,
		FinishedAt:  now.Add(2 * time.Second),
		FinalOutput: "done",
		events: []*runtimeEventRecord{{
			ID:         "step-1",
			Type:       "message",
			Title:      "Mixed surfaces",
			Status:     "completed",
			Preview:    "Full detail contains code.",
			StartedAt:  now,
			FinishedAt: now.Add(time.Second),
			Blocks: []RuntimeDetailBlock{{
				Type:     "code",
				Title:    "Fixture",
				Content:  "const stable = true;",
				Language: "ts",
			}},
			Searchable: true,
		}},
	})
	item.mu.Unlock()

	turns, err := service.ListTurns("owner-step-blocks", session.ID)
	if err != nil {
		t.Fatalf("list turns: %v", err)
	}
	if len(turns) != 1 || len(turns[0].RuntimeTraceEvents) != 1 {
		t.Fatalf("expected runtime trace event projection, got %+v", turns[0])
	}
	event := turns[0].RuntimeTraceEvents[0]
	if event.Kind != "assistant_commentary" {
		t.Fatalf("expected message step to become assistant commentary, got %q", event.Kind)
	}
	if event.Raw.Ref != "step-1" || !event.Raw.HasDetail {
		t.Fatalf("expected runtime trace event to keep step detail ref, got %+v", event.Raw)
	}
	if event.DurationMS != int64(time.Second/time.Millisecond) {
		t.Fatalf("expected runtime trace event duration, got %d", event.DurationMS)
	}
	if len(event.Blocks) != 1 || event.Blocks[0].Type != "code" || event.Blocks[0].Content != "const stable = true;" {
		t.Fatalf("expected runtime trace event code block, got %+v", event.Blocks)
	}
}

func TestServiceInputUpgradesAutoTitleWhenLaterPromptIsMoreSpecific(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-title-upgrade",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-title-upgrade", session.ID, "先拉取仓库"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	firstSnapshot, _ := waitForSessionEntries(t, service, "owner-title-upgrade", session.ID, 2)
	if firstSnapshot.Title != "先拉取仓库" {
		t.Fatalf("expected bootstrap title after first input, got %q", firstSnapshot.Title)
	}

	if _, err := service.Input("owner-title-upgrade", session.ID, "修改 chatRuntime 和 chat 的会话标题"); err != nil {
		t.Fatalf("second input: %v", err)
	}
	secondSnapshot, _ := waitForSessionEntries(t, service, "owner-title-upgrade", session.ID, 4)
	if secondSnapshot.Title != "修改 chatRuntime 和 chat 的会话标题" {
		t.Fatalf("expected upgraded title, got %q", secondSnapshot.Title)
	}
}

func TestServiceInputUpgradesStableAutoTitleWhenLaterPromptIsMoreSpecific(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-stable-title-upgrade",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-stable-title-upgrade", session.ID, "排查会话标题逻辑"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	firstSnapshot, _ := waitForSessionEntries(t, service, "owner-stable-title-upgrade", session.ID, 2)
	if firstSnapshot.Title != "排查会话标题逻辑" {
		t.Fatalf("expected stable auto title after first input, got %q", firstSnapshot.Title)
	}

	if _, err := service.Input("owner-stable-title-upgrade", session.ID, "修复多轮沟通后会话标题不刷新"); err != nil {
		t.Fatalf("second input: %v", err)
	}
	secondSnapshot, _ := waitForSessionEntries(t, service, "owner-stable-title-upgrade", session.ID, 4)
	if secondSnapshot.Title != "修复多轮沟通后会话标题不刷新" {
		t.Fatalf("expected later prompt to upgrade stable auto title, got %q", secondSnapshot.Title)
	}
}

func TestServiceInputKeepsTopicTitleWhenLaterPromptIsSupplementalConstraint(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-topic-title-constraint",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-topic-title-constraint", session.ID, "成都旅游攻略"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	firstSnapshot, _ := waitForSessionEntries(t, service, "owner-topic-title-constraint", session.ID, 2)
	if firstSnapshot.Title != "成都旅游攻略" {
		t.Fatalf("expected topic title after first input, got %q", firstSnapshot.Title)
	}

	if _, err := service.Input("owner-topic-title-constraint", session.ID, "图片要用模型生成的，而不是代码绘制的"); err != nil {
		t.Fatalf("second input: %v", err)
	}
	secondSnapshot, _ := waitForSessionEntries(t, service, "owner-topic-title-constraint", session.ID, 4)
	if secondSnapshot.Title != "成都旅游攻略" {
		t.Fatalf("expected supplemental constraint not to replace topic title, got %q", secondSnapshot.Title)
	}
}

func TestServiceInputKeepsManualTitleWhenLaterPromptChanges(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-manual-title",
		Title:   "manual-title",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-manual-title", session.ID, "先拉取仓库"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	_, _ = waitForSessionEntries(t, service, "owner-manual-title", session.ID, 2)
	if _, err := service.Input("owner-manual-title", session.ID, "修改 chatRuntime 和 chat 的会话标题"); err != nil {
		t.Fatalf("second input: %v", err)
	}
	snapshot, _ := waitForSessionEntries(t, service, "owner-manual-title", session.ID, 4)
	if snapshot.Title != "manual-title" {
		t.Fatalf("expected manual title to stay unchanged, got %q", snapshot.Title)
	}
}

func TestServiceInputAppliesExternalThreadTitleFromCodexEvents(t *testing.T) {
	service := newTestService("external-title")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-external-title",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-external-title", session.ID, "first prompt"); err != nil {
		t.Fatalf("input: %v", err)
	}
	snapshot, _ := waitForSessionEntries(t, service, "owner-external-title", session.ID, 2)
	if snapshot.Title != "Codex internal thread title" {
		t.Fatalf("expected external thread title, got %q", snapshot.Title)
	}
}

func TestServiceInputAcceptsLaterExternalThreadTitleUpdates(t *testing.T) {
	service := newTestService("external-title")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-external-title-update",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-external-title-update", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	firstSnapshot, _ := waitForSessionEntries(t, service, "owner-external-title-update", session.ID, 2)
	if firstSnapshot.Title != "Codex internal thread title" {
		t.Fatalf("expected first external thread title, got %q", firstSnapshot.Title)
	}

	secondStart, err := service.Input("owner-external-title-update", session.ID, "second prompt")
	if err != nil {
		t.Fatalf("second input: %v", err)
	}
	if secondStart.Title != "Codex internal thread title" {
		t.Fatalf("expected external title to remain until runtime sends an update, got %q", secondStart.Title)
	}
	secondSnapshot, _ := waitForSessionEntries(t, service, "owner-external-title-update", session.ID, 4)
	if secondSnapshot.Title != "Codex renamed thread title" {
		t.Fatalf("expected later external thread title update, got %q", secondSnapshot.Title)
	}
}

func TestServiceInputPublishesSessionUpdatedForExternalThreadTitle(t *testing.T) {
	service := newTestService("external-title")
	events := make(chan SessionEvent, 16)
	service.SetSessionEventHook(func(event SessionEvent) {
		if event.OwnerID == "owner-external-title-event" {
			events <- event
		}
	})

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-external-title-event",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-external-title-event", session.ID, "first prompt"); err != nil {
		t.Fatalf("input: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.EventType == SessionEventSessionUpdated && event.Session.Title == "Codex internal thread title" {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for external title session.updated event")
		}
	}
}

func TestServiceRecoverRestoresCodexThreadForFollowUpInput(t *testing.T) {
	service := newTestService("success")

	session, err := service.Recover(RecoverRequest{
		OwnerID:          "owner-recover",
		SessionID:        "chat-recover",
		RuntimeSessionID: "thread-recovered",
		Title:            "chat-recover",
		CreatedAt:        time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 3, 19, 10, 5, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}
	if session.RuntimeSessionID != "thread-recovered" {
		t.Fatalf("expected recovered thread id, got %q", session.RuntimeSessionID)
	}

	if _, err := service.Input("owner-recover", session.ID, "follow-up prompt"); err != nil {
		t.Fatalf("recovered input: %v", err)
	}

	snapshot, entries := waitForSessionEntries(t, service, "owner-recover", session.ID, 2)
	if snapshot.RuntimeSessionID != "thread-recovered" {
		t.Fatalf("expected recovered runtime thread id, got %q", snapshot.RuntimeSessionID)
	}
	if got := entries[1].Text; got != "mock:follow-up prompt" {
		t.Fatalf("expected resumed reply, got %q", got)
	}
}

func TestServiceRecoverRejectsSessionOwnedByAnotherOwner(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-original",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-original", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	snapshot, _ := waitForSessionEntries(t, service, "owner-original", session.ID, 2)

	recovered, err := service.Recover(RecoverRequest{
		OwnerID:          "owner-rebound",
		SessionID:        session.ID,
		RuntimeSessionID: snapshot.RuntimeSessionID,
		Title:            snapshot.Title,
		CreatedAt:        snapshot.CreatedAt,
		LastOutputAt:     snapshot.LastOutputAt,
		UpdatedAt:        snapshot.UpdatedAt,
	})
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ownership mismatch to be rejected, got session=%+v err=%v", recovered, err)
	}
	if _, ok := service.Get("owner-original", session.ID); !ok {
		t.Fatalf("expected original owner to keep access")
	}
	if _, ok := service.Get("owner-rebound", session.ID); ok {
		t.Fatalf("expected rebound owner to be unable to access original session")
	}
}

func TestServiceRecoverRejectsEmptySessionOwnedByAnotherOwner(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-original",
		Title:   "empty-session",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	recovered, err := service.Recover(RecoverRequest{
		OwnerID:          "owner-rebound",
		SessionID:        session.ID,
		RuntimeSessionID: session.RuntimeSessionID,
		Title:            session.Title,
		CreatedAt:        session.CreatedAt,
		UpdatedAt:        session.UpdatedAt,
	})
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected empty session ownership mismatch to be rejected, got session=%+v err=%v", recovered, err)
	}
}

func TestServiceRecoverRejectsChatRuntimeIdentityMismatchForAnotherOwner(t *testing.T) {
	service := newTestService("success")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-original",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-original", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	snapshot, _ := waitForSessionEntries(t, service, "owner-original", session.ID, 2)

	recovered, err := service.Recover(RecoverRequest{
		OwnerID:          "owner-wrong",
		SessionID:        session.ID,
		RuntimeSessionID: snapshot.RuntimeSessionID + "-other",
	})
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected chatRuntime identity mismatch owner to be rejected, got session=%+v err=%v", recovered, err)
	}
}

func TestServiceLoadsPersistedSessionsAfterRestart(t *testing.T) {
	baseDir := t.TempDir()
	service := newTestServiceWithBaseDir("success", baseDir)

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-restart",
		Title:   "persisted-session",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-restart", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	firstSnapshot, _ := waitForSessionEntries(t, service, "owner-restart", session.ID, 2)
	if firstSnapshot.RuntimeSessionID != "thread-first-prompt" {
		t.Fatalf("expected persisted thread id, got %q", firstSnapshot.RuntimeSessionID)
	}

	restarted := newTestServiceWithBaseDir("success", baseDir)
	listed := restarted.List("owner-restart")
	if len(listed) != 1 || listed[0].ID != session.ID {
		t.Fatalf("expected persisted session in list after restart, got %+v", listed)
	}
	if listed[0].Title != "persisted-session" {
		t.Fatalf("expected listed persisted title, got %q", listed[0].Title)
	}
	restored, ok := restarted.Get("owner-restart", session.ID)
	if !ok {
		t.Fatalf("expected restored session after restart")
	}
	if restored.Title != "persisted-session" {
		t.Fatalf("expected restored title, got %q", restored.Title)
	}
	if restored.RuntimeSessionID != "thread-first-prompt" {
		t.Fatalf("expected restored thread id, got %q", restored.RuntimeSessionID)
	}

	if _, err := restarted.Input("owner-restart", session.ID, "after restart"); err != nil {
		t.Fatalf("restart input: %v", err)
	}
	snapshot, entries := waitForSessionEntries(t, restarted, "owner-restart", session.ID, 4)
	if snapshot.Title != "persisted-session" {
		t.Fatalf("expected manual title to stay unchanged after restart, got %q", snapshot.Title)
	}
	if snapshot.RuntimeSessionID != "thread-first-prompt" {
		t.Fatalf("expected resumed thread id after restart, got %q", snapshot.RuntimeSessionID)
	}
	if got := entries[3].Text; got != "mock:after restart" {
		t.Fatalf("expected resumed reply after restart, got %q", got)
	}
}

func TestServiceListLoadsPersistedSessionsCreatedAfterServiceStart(t *testing.T) {
	baseDir := t.TempDir()
	service := newTestServiceWithBaseDir("success", baseDir)
	writer := newTestServiceWithBaseDir("success", baseDir)

	session, err := writer.Recover(RecoverRequest{
		OwnerID:   "owner-late-list",
		SessionID: "c_latelist00000000",
		Title:     "late persisted session",
	})
	if err != nil {
		t.Fatalf("recover persisted session: %v", err)
	}

	items := service.List("owner-late-list")
	if len(items) != 1 || items[0].ID != session.ID {
		t.Fatalf("expected list to load late persisted session, got %+v", items)
	}
	if items[0].Title != "late persisted session" {
		t.Fatalf("expected late persisted title, got %q", items[0].Title)
	}
}

func TestServiceIgnoresLegacyPersistedChatSessionIDs(t *testing.T) {
	baseDir := t.TempDir()
	writer := newTestServiceWithBaseDir("success", baseDir)

	if _, err := writer.Recover(RecoverRequest{
		OwnerID:   "owner-legacy",
		SessionID: "chat-20260707T051709.110973500-f01ec2b780bbdb0d",
		Title:     "legacy session",
	}); err != nil {
		t.Fatalf("write legacy persisted session: %v", err)
	}
	if _, err := writer.Recover(RecoverRequest{
		OwnerID:   "owner-legacy",
		SessionID: "c_visible000000000",
		Title:     "visible session",
	}); err != nil {
		t.Fatalf("write compact persisted session: %v", err)
	}

	reloaded := newTestServiceWithBaseDir("success", baseDir)
	items := reloaded.List("owner-legacy")
	if len(items) != 1 {
		t.Fatalf("expected only compact persisted session, got %+v", items)
	}
	if items[0].ID != "c_visible000000000" {
		t.Fatalf("expected compact session id, got %+v", items)
	}
}

func TestServiceLoadPersistedSessionRepairsSupplementalConstraintAutoTitle(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "c_titlefix00000000"
	statePath, err := resolveChatRuntimeSessionStateFilePath(baseDir, sessionID)
	if err != nil {
		t.Fatalf("resolve state path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("prepare state dir: %v", err)
	}
	record := `{
  "summary": {
    "id": "c_titlefix00000000",
    "owner_id": "owner-title-repair",
    "title": "图片要用模型生成的，而不是代码绘制的",
    "status": "ready"
  },
  "title_auto": true,
  "title_score": 3,
  "turns": [
    {
      "id": "turn-1",
      "prompt": "成都旅游攻略",
      "status": "completed",
      "final_output": "已完成成都旅游攻略"
    },
    {
      "id": "turn-2",
      "prompt": "图片要用模型生成的，而不是代码绘制的",
      "status": "completed",
      "final_output": "当前无法替换"
    }
  ]
}`
	if err := os.WriteFile(statePath, []byte(record), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	service := NewService(context.Background(), nil, nil, Options{WorkingDir: baseDir})
	restored, ok := service.Get("owner-title-repair", sessionID)
	if !ok {
		t.Fatalf("expected restored session")
	}
	if restored.Title != "成都旅游攻略" {
		t.Fatalf("expected restored topic title, got %q", restored.Title)
	}
}

func TestServiceKeepsIdleReadySessionReadyAfterRestart(t *testing.T) {
	baseDir := t.TempDir()
	service := newTestServiceWithBaseDir("success", baseDir)

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-idle-restart",
		Title:   "idle-ready",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	restarted := newTestServiceWithBaseDir("success", baseDir)
	restored, ok := restarted.Get("owner-idle-restart", session.ID)
	if !ok {
		t.Fatalf("expected restored idle session after restart")
	}
	if restored.Status != chatruntimedomain.SessionStatusReady {
		t.Fatalf("expected idle session to stay ready after restart, got %q", restored.Status)
	}
}

func TestServiceDeleteRemovesPersistedStateAndWorkspace(t *testing.T) {
	baseDir := t.TempDir()
	service := newTestServiceWithBaseDir("success", baseDir)

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-delete",
		Title:   "delete-me",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-delete", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	snapshot, _ := waitForSessionEntries(t, service, "owner-delete", session.ID, 2)

	statePath, err := resolveChatRuntimeSessionStateFilePath(baseDir, session.ID)
	if err != nil {
		t.Fatalf("resolve state path: %v", err)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected persisted session state, got %v", err)
	}
	if _, err := os.Stat(snapshot.WorkingDir); err != nil {
		t.Fatalf("expected workspace directory, got %v", err)
	}

	deleted, err := service.Delete("owner-delete", session.ID)
	if err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if deleted.ID != session.ID {
		t.Fatalf("expected deleted session id %q, got %q", session.ID, deleted.ID)
	}
	if _, ok := service.Get("owner-delete", session.ID); ok {
		t.Fatalf("expected session to be removed from runtime store")
	}
	if _, err := os.Stat(statePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected state file removed, got %v", err)
	}
	if _, err := os.Stat(snapshot.WorkingDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected workspace removed with session delete, got %v", err)
	}
}

func TestServiceDeleteRunningSessionDoesNotRecreatePersistedState(t *testing.T) {
	baseDir := t.TempDir()
	service := newTestServiceWithBaseDir("sleep", baseDir)

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-delete-running",
		Title:   "delete-running",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-delete-running", session.ID, "long prompt"); err != nil {
		t.Fatalf("input: %v", err)
	}
	statePath, err := resolveChatRuntimeSessionStateFilePath(baseDir, session.ID)
	if err != nil {
		t.Fatalf("resolve state path: %v", err)
	}

	if _, err := service.Delete("owner-delete-running", session.ID); err != nil {
		t.Fatalf("delete running session: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	if _, ok := service.Get("owner-delete-running", session.ID); ok {
		t.Fatalf("expected deleted running session to stay absent")
	}
	if items := service.List("owner-delete-running"); len(items) != 0 {
		t.Fatalf("expected deleted running session to stay out of list, got %+v", items)
	}
	if _, err := os.Stat(statePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected deleted running session state to stay removed, got %v", err)
	}
}

func TestServiceInputRecoversPersistedSessionWhenRuntimeMissing(t *testing.T) {
	baseDir := t.TempDir()
	service := newTestServiceWithBaseDir("success", baseDir)

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-missing-runtime",
		Title:   "empty-before-input",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	service.mu.Lock()
	delete(service.sessions, session.ID)
	_, stillInRuntime := service.sessions[session.ID]
	service.mu.Unlock()

	if stillInRuntime {
		t.Fatalf("expected runtime session to be removed before recovery")
	}

	if _, err := service.Input("owner-missing-runtime", session.ID, "first prompt after restore"); err != nil {
		t.Fatalf("input after runtime loss: %v", err)
	}

	snapshot, entries := waitForSessionEntries(t, service, "owner-missing-runtime", session.ID, 2)
	if snapshot.Title != "empty-before-input" {
		t.Fatalf("expected restored title, got %q", snapshot.Title)
	}
	if snapshot.RuntimeSessionID != "thread-first-prompt-after-restore" {
		t.Fatalf("expected restored thread id, got %q", snapshot.RuntimeSessionID)
	}
	if got := entries[1].Text; got != "mock:first prompt after restore" {
		t.Fatalf("expected recovered reply, got %q", got)
	}
}

func TestServiceInputRejectsConcurrentTurns(t *testing.T) {
	service := newTestService("sleep")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-b",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-b", session.ID, "long prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}
	if _, err := service.Input("owner-b", session.ID, "second prompt"); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("expected busy error, got %v", err)
	}
}

func TestServiceInputReturnsBusySnapshotWhileTurnRuns(t *testing.T) {
	service := newTestService("sleep")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-busy",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	snapshot, err := service.Input("owner-busy", session.ID, "long prompt")
	if err != nil {
		t.Fatalf("input: %v", err)
	}
	if snapshot.Status != chatruntimedomain.SessionStatusBusy {
		t.Fatalf("expected busy snapshot while turn runs, got %q", snapshot.Status)
	}
}

func TestServiceSessionEventHookPublishesTypedTurnLifecycle(t *testing.T) {
	service := newTestService("command")
	events := make(chan SessionEvent, 16)
	service.SetSessionEventHook(func(event SessionEvent) {
		if event.OwnerID == "owner-event" {
			events <- event
		}
	})

	session, err := service.Create(CreateRequest{OwnerID: "owner-event"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-event", session.ID, "first prompt"); err != nil {
		t.Fatalf("input: %v", err)
	}

	deadline := time.After(5 * time.Second)
	seenStarted := false
	seenCommandAppend := false
	seenCommandUpdate := false
	seenCompleted := false
	for !seenStarted || !seenCommandAppend || !seenCommandUpdate || !seenCompleted {
		select {
		case event := <-events:
			if event.SessionID != session.ID {
				t.Fatalf("expected session id %q, got %+v", session.ID, event)
			}
			switch event.EventType {
			case SessionEventTurnStarted:
				if event.Turn == nil || event.Turn.ID == "" || event.Turn.Prompt != "first prompt" || event.Turn.Status != "running" {
					t.Fatalf("expected running turn summary on turn.started, got %+v", event.Turn)
				}
				if event.RuntimeEvent != nil {
					t.Fatalf("turn.started must not carry a runtime event patch: %+v", event.RuntimeEvent)
				}
				seenStarted = true
			case SessionEventTurnEventAppended:
				if event.RuntimeEvent == nil {
					t.Fatalf("turn.event.appended must carry one runtime event patch: %+v", event)
				}
				if event.RuntimeEvent.Kind == "shell_command" && event.RuntimeEvent.Status == "running" {
					if event.Turn != nil && len(event.Turn.RuntimeTraceEvents) != 0 {
						t.Fatalf("runtime event patch must not carry a full turn event list: %+v", event.Turn)
					}
					seenCommandAppend = true
				}
			case SessionEventTurnEventUpdated:
				if event.RuntimeEvent == nil {
					t.Fatalf("turn.event.updated must carry one runtime event patch: %+v", event)
				}
				if event.RuntimeEvent.Kind == "shell_command" && event.RuntimeEvent.Status == "completed" {
					seenCommandUpdate = true
				}
			case SessionEventTurnCompleted:
				if event.RuntimeEvent != nil {
					t.Fatalf("turn.completed must not carry a runtime event patch: %+v", event.RuntimeEvent)
				}
				if event.Turn == nil || event.Turn.Status != "completed" || event.Turn.FinalOutput != "mock:first prompt" {
					t.Fatalf("expected completed turn with final output, got %+v", event.Turn)
				}
				if event.Session.Status != chatruntimedomain.SessionStatusReady {
					t.Fatalf("expected ready session on turn.completed, got %+v", event.Session)
				}
				seenCompleted = true
			}
		case <-deadline:
			t.Fatalf("timed out waiting for typed session events; started=%v append=%v update=%v completed=%v", seenStarted, seenCommandAppend, seenCommandUpdate, seenCompleted)
		}
	}
}

func TestServiceSessionEventHookPublishesCommentaryAsIncrementalRuntimeEvent(t *testing.T) {
	service := newTestService("commentary")
	events := make(chan SessionEvent, 16)
	service.SetSessionEventHook(func(event SessionEvent) {
		if event.OwnerID == "owner-commentary" {
			events <- event
		}
	})

	session, err := service.Create(CreateRequest{OwnerID: "owner-commentary"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-commentary", session.ID, "first prompt"); err != nil {
		t.Fatalf("input: %v", err)
	}

	deadline := time.After(5 * time.Second)
	seenCommentaryAppend := false
	seenCompleted := false
	for !seenCommentaryAppend || !seenCompleted {
		select {
		case event := <-events:
			if event.SessionID != session.ID {
				t.Fatalf("expected session id %q, got %+v", session.ID, event)
			}
			switch event.EventType {
			case SessionEventTurnEventAppended:
				if event.RuntimeEvent == nil {
					t.Fatalf("turn.event.appended must carry a runtime event patch: %+v", event)
				}
				if got := firstRuntimeTextBlockContent(event.RuntimeEvent); got == "final:first prompt" {
					t.Fatalf("final agent message must not be published as a runtime event patch: %+v", event.RuntimeEvent)
				}
				if event.RuntimeEvent.Kind == "assistant_commentary" {
					if event.RuntimeEvent.Status != "completed" {
						t.Fatalf("expected completed commentary runtime event, got %+v", event.RuntimeEvent)
					}
					if got := firstRuntimeTextBlockContent(event.RuntimeEvent); got != "working on first prompt" {
						t.Fatalf("commentary runtime event text = %q, want %q", got, "working on first prompt")
					}
					seenCommentaryAppend = true
				}
			case SessionEventTurnCompleted:
				if event.RuntimeEvent != nil {
					t.Fatalf("turn.completed must not carry a runtime event patch: %+v", event.RuntimeEvent)
				}
				if event.Turn == nil || event.Turn.Status != "completed" || event.Turn.FinalOutput != "final:first prompt" {
					t.Fatalf("expected completed turn with final output only, got %+v", event.Turn)
				}
				seenCompleted = true
			}
		case <-deadline:
			t.Fatalf("timed out waiting for commentary runtime update; commentary=%v completed=%v", seenCommentaryAppend, seenCompleted)
		}
	}
}

func firstRuntimeTextBlockContent(event *RuntimeTraceEvent) string {
	if event == nil || len(event.Blocks) == 0 {
		return ""
	}
	if event.Blocks[0].Text != "" {
		return event.Blocks[0].Text
	}
	return event.Blocks[0].Content
}

func TestServiceSessionEventHookReleasesSessionLockBeforePublishing(t *testing.T) {
	service := newTestService("success")
	hookCanReadSession := make(chan bool, 1)
	service.SetSessionEventHook(func(event SessionEvent) {
		if event.EventType != SessionEventTurnStarted {
			return
		}
		_, ok := service.Get(event.OwnerID, event.SessionID)
		hookCanReadSession <- ok
	})

	session, err := service.Create(CreateRequest{OwnerID: "owner-event-lock"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-event-lock", session.ID, "first prompt"); err != nil {
		t.Fatalf("input: %v", err)
	}

	select {
	case ok := <-hookCanReadSession:
		if !ok {
			t.Fatal("expected event hook to read the session")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for session event hook")
	}
}

func TestServiceShutdownAppendsInterruptedNoticeOnce(t *testing.T) {
	service := newTestService("sleep")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-shutdown",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-shutdown", session.ID, "long prompt"); err != nil {
		t.Fatalf("start input: %v", err)
	}

	service.shutdown()
	time.Sleep(500 * time.Millisecond)

	snapshot, entries := waitForSessionStatus(t, service, "owner-shutdown", session.ID, chatruntimedomain.SessionStatusInterrupted)
	const interruptedMessage = "chatRuntime interrupted: chatRuntime host unavailable"
	count := 0
	for _, entry := range entries {
		if entry.Text == interruptedMessage {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected interrupted message once, got %d entries: %+v", count, entries)
	}
	if snapshot.ErrorMessage != "chatRuntime host unavailable" {
		t.Fatalf("expected interrupted error message, got %q", snapshot.ErrorMessage)
	}
}

func TestServiceInterruptedNoticeAppendsAgainAfterNextTurn(t *testing.T) {
	service := newTestService("sleep")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-repeat-interrupt",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-repeat-interrupt", session.ID, "first long prompt"); err != nil {
		t.Fatalf("start first input: %v", err)
	}
	service.shutdown()
	waitForSessionStatus(t, service, "owner-repeat-interrupt", session.ID, chatruntimedomain.SessionStatusInterrupted)

	if _, err := service.Input("owner-repeat-interrupt", session.ID, "second long prompt"); err != nil {
		t.Fatalf("start second input: %v", err)
	}
	service.shutdown()
	_, entries := waitForSessionStatus(t, service, "owner-repeat-interrupt", session.ID, chatruntimedomain.SessionStatusInterrupted)

	const interruptedMessage = "chatRuntime interrupted: chatRuntime host unavailable"
	count := 0
	for _, entry := range entries {
		if entry.Text == interruptedMessage {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected interrupted message twice across two turns, got %d entries: %+v", count, entries)
	}
}

func TestServiceInputFailsFastOnCodexAuthError(t *testing.T) {
	service := newTestService("auth-error")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-auth",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	startedAt := time.Now()
	if _, err := service.Input("owner-auth", session.ID, "hello"); err != nil {
		t.Fatalf("input: %v", err)
	}

	snapshot, entries := waitForSessionError(t, service, "owner-auth", session.ID)
	if snapshot.Status != chatruntimedomain.SessionStatusFailed {
		t.Fatalf("expected failed session status after auth failure, got %q", snapshot.Status)
	}
	if snapshot.FinishedAt.IsZero() {
		t.Fatal("expected failed session to record finished_at")
	}
	if !strings.Contains(snapshot.ErrorMessage, "codex authentication failed") {
		t.Fatalf("expected auth failure in session error, got %q", snapshot.ErrorMessage)
	}
	if elapsed := time.Since(startedAt); elapsed > 2*time.Second {
		t.Fatalf("expected fast auth failure, got %s", elapsed)
	}
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Text, "codex request failed: codex authentication failed") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected auth failure entry, got %+v", entries)
	}

	if _, err := service.Input("owner-auth", session.ID, "retry after auth failure"); err != nil {
		t.Fatalf("retry failed session: %v", err)
	}
	recovered, recoveredEntries := waitForSessionEntries(t, service, "owner-auth", session.ID, 4)
	if recovered.Status != chatruntimedomain.SessionStatusReady {
		t.Fatalf("expected retry to restore ready status, got %q", recovered.Status)
	}
	if got := recoveredEntries[len(recoveredEntries)-1].Text; got != "mock:retry after auth failure" {
		t.Fatalf("expected retry output, got %q", got)
	}
}

func TestServiceKeepsThreadAfterCodexCompactionFailure(t *testing.T) {
	service := newTestService("compact-error")

	session, err := service.Create(CreateRequest{
		OwnerID: "owner-compact",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.Input("owner-compact", session.ID, "first prompt"); err != nil {
		t.Fatalf("first input: %v", err)
	}

	failedSnapshot, failedEntries := waitForSessionError(t, service, "owner-compact", session.ID)
	if failedSnapshot.RuntimeSessionID != "thread-first-prompt" {
		t.Fatalf("expected failed compaction to keep chatRuntime session id, got %q (error=%q, entries=%+v)", failedSnapshot.RuntimeSessionID, failedSnapshot.ErrorMessage, failedEntries)
	}
	if !strings.Contains(failedSnapshot.ErrorMessage, "continue the previous runtime thread") {
		t.Fatalf("expected compaction recovery message, got %q", failedSnapshot.ErrorMessage)
	}

	foundRecoveryEntry := false
	for _, entry := range failedEntries {
		if strings.Contains(entry.Text, "previous runtime thread retained after context compaction failure") {
			foundRecoveryEntry = true
			break
		}
	}
	if !foundRecoveryEntry {
		t.Fatalf("expected thread retention entry, got %+v", failedEntries)
	}

	if _, err := service.Input("owner-compact", session.ID, "second prompt"); err != nil {
		t.Fatalf("second input: %v", err)
	}

	recoveredSnapshot, entries := waitForSessionEntries(t, service, "owner-compact", session.ID, 4)
	if recoveredSnapshot.RuntimeSessionID != "thread-first-prompt" {
		t.Fatalf("expected previous thread after compaction failure, got %q", recoveredSnapshot.RuntimeSessionID)
	}
	if got := entries[len(entries)-1].Text; got != "mock:second prompt" {
		t.Fatalf("expected second prompt to run on fresh thread, got %q", got)
	}
}

func TestServiceListSortsByConversationUpdatedAt(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	service := &Service{
		sessions: map[string]*runtimeSession{
			"chat-output-newer": {
				summary: chatruntimedomain.Session{
					ID:           "chat-output-newer",
					OwnerID:      chatRuntimeOwnerID,
					CreatedAt:    now.Add(-10 * time.Minute),
					LastOutputAt: now.Add(-2 * time.Minute),
					UpdatedAt:    now.Add(-4 * time.Minute),
				},
			},
			"chat-updated-newer": {
				summary: chatruntimedomain.Session{
					ID:           "chat-updated-newer",
					OwnerID:      chatRuntimeOwnerID,
					CreatedAt:    now.Add(-9 * time.Minute),
					LastOutputAt: now.Add(-3 * time.Minute),
					UpdatedAt:    now.Add(-1 * time.Minute),
				},
			},
		},
	}

	items := service.List(chatRuntimeOwnerID)
	if len(items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(items))
	}
	if items[0].ID != "chat-updated-newer" {
		t.Fatalf("expected updated_at ordering, got first session %q", items[0].ID)
	}
}

func TestServiceListSeparatesChatSessionsFromOtherOwners(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	service := &Service{
		sessions: map[string]*runtimeSession{
			"chat-chat": {
				summary: chatruntimedomain.Session{
					ID:        "chat-chat",
					OwnerID:   "chat",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			"other-standard": {
				summary: chatruntimedomain.Session{
					ID:        "other-standard",
					OwnerID:   "owner-other",
					CreatedAt: now.Add(time.Minute),
					UpdatedAt: now.Add(time.Minute),
				},
			},
		},
	}

	chatItems := service.List("chat")
	if len(chatItems) != 1 || chatItems[0].ID != "chat-chat" {
		t.Fatalf("expected only chat-owned sessions, got %+v", chatItems)
	}
	defaultItems := service.List("")
	if len(defaultItems) != 1 || defaultItems[0].ID != "chat-chat" {
		t.Fatalf("expected empty owner to list chat sessions, got %+v", defaultItems)
	}
	otherItems := service.List("owner-other")
	if len(otherItems) != 1 || otherItems[0].ID != "other-standard" {
		t.Fatalf("expected only other owner sessions, got %+v", otherItems)
	}
	if _, ok := service.Get(chatRuntimeOwnerID, "other-standard"); ok {
		t.Fatalf("expected chat owner to be unable to read other owner session")
	}
}

func TestServiceListReconcilesOrphanedBusySession(t *testing.T) {
	service := newTestService("success")
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	sessionID := "c_orphanlist000000"
	insertOrphanedBusyRuntimeSession(t, service, "owner-orphan-list", sessionID, now)

	items := service.List("owner-orphan-list")
	if len(items) != 1 {
		t.Fatalf("expected orphaned session in list, got %+v", items)
	}
	if items[0].Status != chatruntimedomain.SessionStatusInterrupted {
		t.Fatalf("expected list to reconcile interrupted status, got %q", items[0].Status)
	}
	if items[0].ErrorMessage != chatRuntimeHostUnavailableMessage {
		t.Fatalf("expected chatRuntime host error, got %q", items[0].ErrorMessage)
	}

	turns, err := service.ListTurns("owner-orphan-list", sessionID)
	if err != nil {
		t.Fatalf("list turns: %v", err)
	}
	if len(turns) != 1 || turns[0].Status != "interrupted" {
		t.Fatalf("expected interrupted turn after list reconciliation, got %+v", turns)
	}
	if len(turns[0].RuntimeTraceEvents) == 0 || turns[0].RuntimeTraceEvents[len(turns[0].RuntimeTraceEvents)-1].Title != "Interrupted" {
		t.Fatalf("expected interrupted runtime event, got %+v", turns[0].RuntimeTraceEvents)
	}

	reloaded := NewService(context.Background(), nil, nil, Options{WorkingDir: service.options.WorkingDir})
	restored, ok := reloaded.Get("owner-orphan-list", sessionID)
	if !ok {
		t.Fatalf("expected reconciled session to persist")
	}
	if restored.Status != chatruntimedomain.SessionStatusInterrupted {
		t.Fatalf("expected persisted interrupted status, got %q", restored.Status)
	}
}

func TestServiceInputReconcilesOrphanedBusySessionBeforeBusyCheck(t *testing.T) {
	service := newTestService("success")
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	sessionID := "chat-orphan-input"
	insertOrphanedBusyRuntimeSession(t, service, "owner-orphan-input", sessionID, now)

	snapshot, err := service.Input("owner-orphan-input", sessionID, "continue after restart")
	if err != nil {
		t.Fatalf("expected input to continue reconciled session, got %v", err)
	}
	if snapshot.Status != chatruntimedomain.SessionStatusBusy {
		t.Fatalf("expected new input to start a busy turn, got %q", snapshot.Status)
	}

	final, entries := waitForSessionEntries(t, service, "owner-orphan-input", sessionID, 3)
	if final.Status != chatruntimedomain.SessionStatusReady {
		t.Fatalf("expected continued session to finish ready, got %q", final.Status)
	}
	if len(entries) < 3 {
		t.Fatalf("expected interrupt notice, input and output entries, got %+v", entries)
	}
	foundInterruptedEntry := false
	for _, entry := range entries {
		if entry.Stream == "system" && strings.Contains(entry.Text, "chatRuntime interrupted") {
			foundInterruptedEntry = true
			break
		}
	}
	if !foundInterruptedEntry {
		t.Fatalf("expected entries to record interrupted orphan, got %+v", entries)
	}
}

func TestServiceListDoesNotInterruptLiveWorker(t *testing.T) {
	service := newTestService("sleep")
	session, err := service.Create(CreateRequest{OwnerID: "owner-live-worker"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-live-worker", session.ID, "long prompt"); err != nil {
		t.Fatalf("start input: %v", err)
	}

	items := service.List("owner-live-worker")
	if len(items) != 1 {
		t.Fatalf("expected live session in list, got %+v", items)
	}
	if items[0].Status != chatruntimedomain.SessionStatusBusy {
		t.Fatalf("expected live worker to remain busy, got %q", items[0].Status)
	}

	waitForSessionEntries(t, service, "owner-live-worker", session.ID, 2)
}

func TestServiceDoesNotNormalizeSharedOwnerToChat(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	service := &Service{
		sessions: map[string]*runtimeSession{
			"shared-session": {
				summary: chatruntimedomain.Session{
					ID:        "shared-session",
					OwnerID:   "shared",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
	}

	items := service.List(chatRuntimeOwnerID)
	if len(items) != 0 {
		t.Fatalf("expected chat owner to ignore shared sessions, got %+v", items)
	}
	if _, ok := service.Get("", "shared-session"); ok {
		t.Fatalf("expected empty owner to normalize to chat and ignore shared sessions")
	}
	if _, ok := service.Get("shared", "shared-session"); !ok {
		t.Fatalf("expected explicit shared owner to remain isolated")
	}
}

func TestServiceSetPinnedPersistsAndSortsPinnedSessionsFirst(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: t.TempDir()})
	older, err := service.Recover(RecoverRequest{
		OwnerID:   "owner-pin",
		SessionID: "c_pinolder00000000",
		CreatedAt: now.Add(-20 * time.Minute),
		UpdatedAt: now.Add(-20 * time.Minute),
	})
	if err != nil {
		t.Fatalf("recover older session: %v", err)
	}
	if _, err := service.Recover(RecoverRequest{
		OwnerID:   "owner-pin",
		SessionID: "c_pinnewer00000000",
		CreatedAt: now.Add(-5 * time.Minute),
		UpdatedAt: now.Add(-5 * time.Minute),
	}); err != nil {
		t.Fatalf("recover newer session: %v", err)
	}

	pinned, err := service.SetPinned("owner-pin", older.ID, true)
	if err != nil {
		t.Fatalf("set pinned: %v", err)
	}
	if !pinned.Pinned {
		t.Fatalf("expected pinned snapshot")
	}

	items := service.List("owner-pin")
	if len(items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(items))
	}
	if items[0].ID != older.ID || !items[0].Pinned {
		t.Fatalf("expected pinned older session first, got %+v", items)
	}

	reloaded := NewService(context.Background(), nil, nil, Options{WorkingDir: service.options.WorkingDir})
	restored, ok := reloaded.Get("owner-pin", older.ID)
	if !ok {
		t.Fatalf("expected persisted pinned session to restore")
	}
	if !restored.Pinned {
		t.Fatalf("expected restored pinned session")
	}
}

func TestServiceSetPinnedFalsePersistsAcrossRestore(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	workingDir := t.TempDir()
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: workingDir})
	session, err := service.Recover(RecoverRequest{
		OwnerID:   "owner-unpin",
		SessionID: "c_unpin00000000000",
		CreatedAt: now.Add(-20 * time.Minute),
		UpdatedAt: now.Add(-20 * time.Minute),
	})
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}

	if _, err := service.SetPinned("owner-unpin", session.ID, true); err != nil {
		t.Fatalf("pin session: %v", err)
	}
	unpinned, err := service.SetPinned("owner-unpin", session.ID, false)
	if err != nil {
		t.Fatalf("unpin session: %v", err)
	}
	if unpinned.Pinned {
		t.Fatalf("expected unpinned snapshot")
	}

	reloaded := NewService(context.Background(), nil, nil, Options{WorkingDir: workingDir})
	restored, ok := reloaded.Get("owner-unpin", session.ID)
	if !ok {
		t.Fatalf("expected persisted session to restore")
	}
	if restored.Pinned {
		t.Fatalf("expected restored session to stay unpinned")
	}
}

func TestRuntimeSessionAppendEntryLockedUpdatesLastOutputAtOnlyForRealOutput(t *testing.T) {
	session := &runtimeSession{
		summary: chatruntimedomain.Session{
			ID:        "chat-output-flags",
			OwnerID:   chatRuntimeOwnerID,
			CreatedAt: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
		},
	}

	session.appendEntryLocked("system", "session ready")
	if !session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected system entry to keep last_output_at empty, got %s", session.summary.LastOutputAt)
	}

	session.appendEntryLocked("input", "prompt")
	if !session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected input entry to keep last_output_at empty, got %s", session.summary.LastOutputAt)
	}

	session.appendEntryLocked("stdout", "assistant output")
	if session.summary.LastOutputAt.IsZero() {
		t.Fatalf("expected stdout entry to update last_output_at")
	}

	lastOutputAt := session.summary.LastOutputAt
	session.appendEntryLocked("stderr", "warning")
	if session.summary.LastOutputAt.IsZero() || session.summary.LastOutputAt.Before(lastOutputAt) {
		t.Fatalf("expected stderr entry to preserve or advance last_output_at")
	}
}

func newTestService(mode string) *Service {
	baseDir, err := os.MkdirTemp("", "alter0-chat-service-test-*")
	if err != nil {
		panic(err)
	}
	return newTestServiceWithBaseDir(mode, baseDir)
}

func newTestServiceWithBaseDir(mode string, baseDir string) *Service {
	service := NewService(context.Background(), nil, nil, Options{WorkingDir: baseDir})
	configureTestCodexRunner(service, mode)
	return service
}

type stubRepositoryCatalog struct {
	items      []chatruntimedomain.Repository
	nextCursor string
	listErr    error
	resolved   chatruntimedomain.Repository
	resolveErr error
	resolveRef chatruntimedomain.RepositoryRef
}

func (s *stubRepositoryCatalog) List(_ context.Context, _ string, _ string) (RepositoryPage, error) {
	if s.listErr != nil {
		return RepositoryPage{}, s.listErr
	}
	return RepositoryPage{Items: append([]chatruntimedomain.Repository{}, s.items...), NextCursor: s.nextCursor}, nil
}

func (s *stubRepositoryCatalog) Resolve(_ context.Context, ref chatruntimedomain.RepositoryRef) (chatruntimedomain.Repository, error) {
	s.resolveRef = ref
	if s.resolveErr != nil {
		return chatruntimedomain.Repository{}, s.resolveErr
	}
	return s.resolved, nil
}

type stubRepositoryWorkspacePreparer struct {
	checkout     RepositoryCheckout
	err          error
	calls        int
	repository   chatruntimedomain.Repository
	workspaceDir string
}

func (s *stubRepositoryWorkspacePreparer) Prepare(_ context.Context, repository chatruntimedomain.Repository, workspaceDir string) (RepositoryCheckout, error) {
	s.calls++
	s.repository = repository
	s.workspaceDir = workspaceDir
	if s.err != nil {
		return RepositoryCheckout{}, s.err
	}
	return s.checkout, nil
}

func newTestServiceWithRepositorySupport(mode string, baseDir string, catalog RepositoryCatalog, preparer RepositoryWorkspacePreparer) *Service {
	service := NewService(context.Background(), nil, nil, Options{
		WorkingDir:         baseDir,
		RepositoryCatalog:  catalog,
		RepositoryPreparer: preparer,
	})
	configureTestCodexRunner(service, mode)
	return service
}

func configureTestCodexRunner(service *Service, mode string) {
	service.runner = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmdArgs := append([]string{"-test.run=TestChatRuntimeServiceHelperProcess", "--", name}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
		cmd.Env = append(
			os.Environ(),
			"GO_WANT_CHAT_RUNTIME_HELPER_PROCESS=1",
			"CHAT_RUNTIME_HELPER_MODE="+mode,
		)
		return cmd
	}
}

func insertOrphanedBusyRuntimeSession(t *testing.T, service *Service, ownerID string, sessionID string, now time.Time) {
	t.Helper()

	workspaceDir, err := resolveSessionWorkspaceDir(service.options.WorkingDir, sessionID)
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}
	item := &runtimeSession{
		summary: chatruntimedomain.Session{
			ID:               sessionID,
			RuntimeSessionID: sessionID,
			OwnerID:          ownerID,
			Title:            "Orphaned session",
			Shell:            defaultCodexCommand,
			WorkingDir:       workspaceDir,
			Status:           chatruntimedomain.SessionStatusBusy,
			CreatedAt:        now.Add(-time.Minute),
			UpdatedAt:        now,
		},
		entries: []chatruntimedomain.Entry{
			{
				Cursor:    0,
				Stream:    "input",
				Text:      "long running prompt",
				CreatedAt: now,
			},
		},
		nextID:       1,
		activeTurnID: "turn-1",
		nextTurnID:   2,
		nextEventID:  2,
		turns: []*runtimeTurn{
			{
				ID:        "turn-1",
				Prompt:    "long running prompt",
				Status:    "running",
				StartedAt: now,
				events: []*runtimeEventRecord{
					{
						ID:        "event-1",
						Type:      "reasoning",
						Title:     "Thinking",
						Status:    "running",
						StartedAt: now,
					},
				},
			},
		},
	}

	service.mu.Lock()
	service.sessions[sessionID] = item
	service.mu.Unlock()
	service.persistSession(item)
}

func waitForSessionEntries(t *testing.T, service *Service, ownerID string, sessionID string, want int) (chatruntimedomain.Session, []chatruntimedomain.Entry) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, ok := service.Get(ownerID, sessionID)
		if !ok {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		page, err := service.ListEntries(ownerID, sessionID, 0, 32)
		if err != nil {
			t.Fatalf("list entries: %v", err)
		}
		if len(page.Items) >= want && snapshot.Status == chatruntimedomain.SessionStatusReady {
			return snapshot, page.Items
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d chatRuntime entries", want)
	return chatruntimedomain.Session{}, nil
}

func waitForSessionError(t *testing.T, service *Service, ownerID string, sessionID string) (chatruntimedomain.Session, []chatruntimedomain.Entry) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, ok := service.Get(ownerID, sessionID)
		if !ok {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		page, err := service.ListEntries(ownerID, sessionID, 0, 32)
		if err != nil {
			t.Fatalf("list entries: %v", err)
		}
		if strings.TrimSpace(snapshot.ErrorMessage) != "" && len(page.Items) >= 2 {
			return snapshot, page.Items
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for chatRuntime auth failure")
	return chatruntimedomain.Session{}, nil
}

func waitForSessionStatus(t *testing.T, service *Service, ownerID string, sessionID string, want chatruntimedomain.SessionStatus) (chatruntimedomain.Session, []chatruntimedomain.Entry) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, ok := service.Get(ownerID, sessionID)
		if !ok {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		page, err := service.ListEntries(ownerID, sessionID, 0, 32)
		if err != nil {
			t.Fatalf("list entries: %v", err)
		}
		if snapshot.Status == want {
			return snapshot, page.Items
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for chatRuntime status %q", want)
	return chatruntimedomain.Session{}, nil
}

func TestChatRuntimeServiceHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CHAT_RUNTIME_HELPER_PROCESS") != "1" {
		return
	}

	separatorIndex := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}
	if separatorIndex < 0 || separatorIndex+1 >= len(os.Args) {
		os.Exit(2)
	}

	forwarded := os.Args[separatorIndex+1:]
	if len(forwarded) < 2 || forwarded[0] != defaultCodexCommand {
		os.Exit(2)
	}
	execIndex := -1
	for index, arg := range forwarded {
		if arg == "exec" {
			execIndex = index
			break
		}
	}
	if execIndex < 1 {
		os.Exit(2)
	}
	chatRuntimeArgs := forwarded[execIndex+1:]
	if expectedHome := strings.TrimSpace(os.Getenv("CHAT_RUNTIME_HELPER_EXPECT_CODEX_HOME_SUFFIX")); expectedHome != "" {
		actualHome := filepath.Clean(strings.TrimSpace(os.Getenv("CODEX_HOME")))
		expectedHome = filepath.Clean(expectedHome)
		if actualHome != expectedHome {
			os.Exit(2)
		}
	}
	if expectedImageCount := strings.TrimSpace(os.Getenv("CHAT_RUNTIME_HELPER_EXPECT_IMAGE_COUNT")); expectedImageCount != "" {
		want := 0
		fmt.Sscanf(expectedImageCount, "%d", &want)
		have := 0
		for index := 0; index < len(chatRuntimeArgs)-1; index += 1 {
			if chatRuntimeArgs[index] != "-i" || index+1 >= len(chatRuntimeArgs) {
				continue
			}
			have++
			if _, err := os.Stat(chatRuntimeArgs[index+1]); err != nil {
				os.Exit(2)
			}
		}
		if have != want {
			os.Exit(2)
		}
	}

	mode := os.Getenv("CHAT_RUNTIME_HELPER_MODE")
	if mode == "sleep" {
		time.Sleep(300 * time.Millisecond)
	}

	resumeIndex := -1
	for index, arg := range chatRuntimeArgs {
		if arg == "resume" {
			resumeIndex = index
			break
		}
	}
	if resumeIndex >= 0 {
		if len(chatRuntimeArgs) < resumeIndex+4 {
			os.Exit(2)
		}
		threadID := chatRuntimeArgs[len(chatRuntimeArgs)-2]
		prompt := chatRuntimeArgs[len(chatRuntimeArgs)-1]
		if mode == "missing-rollout" {
			fmt.Fprintf(os.Stderr, "Error: thread/resume: thread/resume failed: no rollout found for thread id %s (code -32600)\n", threadID)
			os.Exit(24)
		}
		if mode == "external-title" {
			fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q,\"title\":\"Codex renamed thread title\"}\n", threadID)
		} else {
			fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", threadID)
		}
		fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
		fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_0\",\"type\":\"agent_message\",\"text\":%q}}\n", "mock:"+prompt)
		fmt.Fprintln(os.Stdout, `{"type":"turn.completed"}`)
		os.Exit(0)
	}

	prompt := chatRuntimeArgs[len(chatRuntimeArgs)-1]
	threadID := "thread-" + strings.ReplaceAll(prompt, " ", "-")
	if mode == "external-title" {
		fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q,\"title\":\"Codex internal thread title\"}\n", threadID)
	} else {
		fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", threadID)
	}
	fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
	if mode == "auth-error" {
		fmt.Fprintln(os.Stdout, `{"type":"error","message":"Reconnecting... 1/5 (unexpected status 401 Unauthorized: Missing bearer or basic authentication in header)"}`)
		time.Sleep(5 * time.Second)
		os.Exit(19)
	}
	if mode == "compact-error" && prompt == "first prompt" {
		fmt.Fprintln(os.Stderr, "2026-04-14T05:19:09.785763Z ERROR codex_core::compact_remote: remote compaction failed turn_id=turn-compact compact_error=stream disconnected before completion")
		fmt.Fprintln(os.Stderr, "2026-04-14T05:19:09.786118Z ERROR codex_core::codex: Failed to run pre-sampling compact")
		os.Exit(23)
	}
	if mode == "command" {
		fmt.Fprintln(os.Stdout, `{"type":"item.started","item":{"id":"item_cmd","type":"command_execution","command":"echo typed-event"}}`)
		fmt.Fprintln(os.Stdout, `{"type":"item.completed","item":{"id":"item_cmd","type":"command_execution","command":"echo typed-event","aggregated_output":"typed-event","status":"completed","exit_code":0}}`)
	}
	if mode == "commentary" {
		fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_commentary\",\"type\":\"agent_message\",\"channel\":\"commentary\",\"text\":%q}}\n", "working on "+prompt)
		fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_final\",\"type\":\"agent_message\",\"channel\":\"final\",\"text\":%q}}\n", "final:"+prompt)
	} else {
		fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_0\",\"type\":\"agent_message\",\"text\":%q}}\n", "mock:"+prompt)
	}
	fmt.Fprintln(os.Stdout, `{"type":"turn.completed"}`)
	os.Exit(0)
}
