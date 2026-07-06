package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	chatruntimedomain "alter0/internal/chatruntime/domain"
	execdomain "alter0/internal/execution/domain"
)

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
	})

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
	if !strings.HasPrefix(session.ID, "chat-") {
		t.Fatalf("expected generated chatRuntime session id, got %q", session.ID)
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
		SessionID: "chat-late-list",
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

func TestServiceLoadPersistedSessionRepairsSupplementalConstraintAutoTitle(t *testing.T) {
	baseDir := t.TempDir()
	statePath, err := resolveChatRuntimeSessionStateFilePath(baseDir, "persisted-supplemental-title")
	if err != nil {
		t.Fatalf("resolve state path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("prepare state dir: %v", err)
	}
	record := `{
  "summary": {
    "id": "persisted-supplemental-title",
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
	restored, ok := service.Get("owner-title-repair", "persisted-supplemental-title")
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

func TestServiceSessionUpdateHookPublishesBusyAndFinalSnapshots(t *testing.T) {
	service := newTestService("success")
	statuses := make(chan string, 8)
	service.SetSessionUpdateHook(func(ownerID string, sessionID string, session chatruntimedomain.Session) {
		if ownerID == "owner-hook" && sessionID == session.ID {
			statuses <- string(session.Status)
		}
	})

	session, err := service.Create(CreateRequest{OwnerID: "owner-hook"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := service.Input("owner-hook", session.ID, "first prompt"); err != nil {
		t.Fatalf("input: %v", err)
	}

	deadline := time.After(5 * time.Second)
	seenBusy := false
	seenReady := false
	for !seenBusy || !seenReady {
		select {
		case status := <-statuses:
			if status == string(chatruntimedomain.SessionStatusBusy) {
				seenBusy = true
			}
			if status == string(chatruntimedomain.SessionStatusReady) {
				seenReady = true
			}
		case <-deadline:
			t.Fatalf("timed out waiting for busy and ready hook snapshots; busy=%v ready=%v", seenBusy, seenReady)
		}
	}
}

func TestServiceCreateReleasesGlobalLockBeforeSessionUpdateHook(t *testing.T) {
	service := newTestService("success")
	hookCanReadSession := make(chan bool, 1)
	service.SetSessionUpdateHook(func(ownerID string, sessionID string, session chatruntimedomain.Session) {
		_, ok := service.Get(ownerID, sessionID)
		hookCanReadSession <- ok
	})

	done := make(chan error, 1)
	go func() {
		_, err := service.Create(CreateRequest{OwnerID: "owner-create-hook"})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("create session: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("create session deadlocked while publishing session update hook")
	}

	select {
	case ok := <-hookCanReadSession:
		if !ok {
			t.Fatal("expected update hook to read the newly created session")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for session update hook")
	}
}

func TestServiceRecoverReleasesGlobalLockBeforeSessionUpdateHook(t *testing.T) {
	service := newTestService("success")
	hookCanReadSession := make(chan bool, 1)
	service.SetSessionUpdateHook(func(ownerID string, sessionID string, session chatruntimedomain.Session) {
		_, ok := service.Get(ownerID, sessionID)
		hookCanReadSession <- ok
	})

	done := make(chan error, 1)
	go func() {
		_, err := service.Recover(RecoverRequest{
			OwnerID:   "owner-recover-hook",
			SessionID: "chat-recovered-hook",
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("recover session: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("recover session deadlocked while publishing session update hook")
	}

	select {
	case ok := <-hookCanReadSession:
		if !ok {
			t.Fatal("expected update hook to read the recovered session")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for session update hook")
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

func TestServiceListPrefersLastOutputAtOverUpdatedAt(t *testing.T) {
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
	if items[0].ID != "chat-output-newer" {
		t.Fatalf("expected last output ordering, got first session %q", items[0].ID)
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
	sessionID := "chat-orphan-list"
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
		SessionID: "chat-older",
		CreatedAt: now.Add(-20 * time.Minute),
		UpdatedAt: now.Add(-20 * time.Minute),
	})
	if err != nil {
		t.Fatalf("recover older session: %v", err)
	}
	if _, err := service.Recover(RecoverRequest{
		OwnerID:   "owner-pin",
		SessionID: "chat-newer",
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
		SessionID: "chat-unpin",
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
	return service
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
		fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", threadID)
		fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
		fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_0\",\"type\":\"agent_message\",\"text\":%q}}\n", "mock:"+prompt)
		fmt.Fprintln(os.Stdout, `{"type":"turn.completed"}`)
		os.Exit(0)
	}

	prompt := chatRuntimeArgs[len(chatRuntimeArgs)-1]
	threadID := "thread-" + strings.ReplaceAll(prompt, " ", "-")
	fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", threadID)
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
	fmt.Fprintf(os.Stdout, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_0\",\"type\":\"agent_message\",\"text\":%q}}\n", "mock:"+prompt)
	fmt.Fprintln(os.Stdout, `{"type":"turn.completed"}`)
	os.Exit(0)
}
