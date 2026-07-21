package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	codexapp "alter0/internal/codex/application"
	controlapp "alter0/internal/control/application"
	controldomain "alter0/internal/control/domain"
)

const (
	builtinSkillPriorityKey    = "skill.priority"
	builtinSkillDescriptionKey = "skill.description"
	builtinSkillGuideKey       = "skill.guide"
	builtinSkillFilePathKey    = "skill.file_path"
	builtinSkillWritableKey    = "skill.writable"
	builtinSkillOwnerKey       = "skill.owner"
)

func registerBuiltinSkills(control *controlapp.Service) {
	for _, skill := range builtinSkills() {
		if existing, ok := control.ResolveSkill(skill.ID); ok {
			skill.Enabled = existing.Enabled
		}
		mustUpsertSkill(control, skill)
	}
	for _, retiredSkillID := range append([]string{"memory", "memory-maintenance"}, retiredBuiltinSkillIDs()...) {
		control.DeleteSkill(retiredSkillID)
	}
}

func retiredBuiltinSkillIDs() []string {
	return []string{
		"frontend-design",
		"doc-coauthoring",
		"fullstack-developer",
		"code-reviewer",
		"webapp-testing",
		"find-skills",
		"test-driven-development",
		"ui-ux-pro-max",
		"code-simplifier",
		"code-review",
		"brainstorming",
	}
}

func ensureBuiltinSkillFiles() error {
	repoRoot := resolveBuiltinSkillRepoRoot()
	for _, skill := range builtinSkills() {
		path := strings.TrimSpace(skill.Metadata[builtinSkillFilePathKey])
		if path == "" {
			continue
		}
		resolvedPath := path
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = filepath.Join(repoRoot, resolvedPath)
		}
		if _, err := os.Stat(resolvedPath); err != nil {
			return fmt.Errorf("builtin skill file %s: %w", path, err)
		}
	}
	return nil
}

func configureNativeSkillReconciliation(control *controlapp.Service, logger *slog.Logger) error {
	if control == nil {
		return nil
	}
	destinationRoot, err := codexapp.ResolveNativeSkillRoot()
	if err != nil {
		return err
	}
	reconciler := codexapp.NewNativeSkillReconciler(destinationRoot)
	reconcile := func(capabilities []controldomain.Capability) error {
		result := reconciler.Reconcile(nativeSkillSources(capabilities))
		if len(result.Errors) == 0 {
			if logger != nil {
				logger.Info("native Codex skills reconciled", "installed", len(result.Installed), "removed", len(result.Removed))
			}
			return nil
		}
		messages := make([]string, 0, len(result.Errors))
		for _, item := range result.Errors {
			messages = append(messages, fmt.Sprintf("%s[%s]: %s", item.SkillID, item.Code, item.Message))
		}
		return fmt.Errorf("reconcile native Codex skills: %s", strings.Join(messages, "; "))
	}
	if err := reconcile(control.ListCapabilities()); err != nil {
		if logger != nil {
			logger.Warn("initial native Codex skill reconciliation failed", "error", err.Error())
		}
	}
	control.SetCapabilityChangeHook(reconcile)
	return nil
}

func nativeSkillSources(capabilities []controldomain.Capability) []codexapp.NativeSkillSource {
	repoRoot := resolveBuiltinSkillRepoRoot()
	sources := make([]codexapp.NativeSkillSource, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability.Type != controldomain.CapabilityTypeSkill {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(capability.Metadata[builtinSkillOwnerKey]), "alter0") {
			continue
		}
		filePath := strings.TrimSpace(capability.Metadata[builtinSkillFilePathKey])
		if filePath != "" && !filepath.IsAbs(filePath) {
			filePath = filepath.Join(repoRoot, filePath)
		}
		visibility := strings.ToLower(strings.TrimSpace(capability.Metadata["alter0.skill.visibility"]))
		if visibility == "" {
			visibility = strings.ToLower(strings.TrimSpace(capability.Metadata["skill.visibility"]))
		}
		sources = append(sources, codexapp.NativeSkillSource{
			ID:       capability.ID,
			Enabled:  capability.Enabled && filePath != "",
			Public:   visibility != "private",
			FilePath: filePath,
		})
	}
	return sources
}

func builtinSkills() []controldomain.Skill {
	return []controldomain.Skill{
		{
			ID:      "preview-publish",
			Name:    "Preview Publish",
			Enabled: true,
			Scope:   controldomain.CapabilityScopeGlobal,
			Metadata: map[string]string{
				builtinSkillPriorityKey:    "760",
				builtinSkillDescriptionKey: "Session-scoped preview and test-service deployment playbook for the shared alter0 gateway.",
				builtinSkillGuideKey:       previewPublishSkillGuide(),
				builtinSkillFilePathKey:    filepath.ToSlash(filepath.Join("docs", "skills", "preview-publish", "SKILL.md")),
				builtinSkillOwnerKey:       "alter0",
			},
		},
		fileBackedBuiltinSkill(
			"travel",
			"Travel",
			500,
			"Travel guide workflow for city itineraries, recommendation pools, route cards, generated map assets, and public HTML delivery.",
			travelSkillGuide(),
			filepath.ToSlash(filepath.Join("docs", "skills", "travel", "SKILL.md")),
		),
	}
}

func fileBackedBuiltinSkill(id string, name string, priority int, description string, guide string, filePath string) controldomain.Skill {
	return controldomain.Skill{
		ID:      id,
		Name:    name,
		Enabled: true,
		Scope:   controldomain.CapabilityScopeGlobal,
		Metadata: map[string]string{
			builtinSkillPriorityKey:    fmt.Sprintf("%d", priority),
			builtinSkillDescriptionKey: description,
			builtinSkillGuideKey:       guide,
			builtinSkillFilePathKey:    filePath,
			builtinSkillOwnerKey:       "alter0",
		},
	}
}

func previewPublishSkillGuide() string {
	return strings.Join([]string{
		"# preview publish",
		"",
		"- Use this skill when the user needs a session-scoped preview host, a static artifact preview, or an additional routed test service without changing Nginx.",
		"- For text, image, code, markdown, JSON, screenshots, and standalone HTML artifact previews, stage files in the workspace and publish them with `bash docs/skills/preview-publish/scripts/publish_preview_artifact.sh <session_id> <service_name> --artifact <path> ...`.",
		"- For full applications, routed backends, or frontend build previews, use `deploy_test_service`.",
		"- Registrations land on `/api/control/workspace-services` and are routed by the shared gateway rather than per-service Nginx edits.",
		"- Host routing is fixed at the gateway: `https://<short_hash>.alter0.cn` for the default `web` service, and `https://<service>-<short_hash>.alter0.cn` for additional services such as `api` or `docs`.",
		"- Default `web` deploys should register `service_type=http` and boot the current session backend so the preview host serves both the latest frontend build and `/api/*` from the same branch.",
		"- Use `service_type=frontend_dist` only for static-only UI previews. That mode serves built assets from `internal/interfaces/web/static/dist` but leaves `/api/*` on the shared runtime backend.",
		"- For custom services, deploy `service_type=http` with either an existing `upstream_url` or a `start_command` that boots the service inside the session workspace.",
		"- Keep test-service deployment session-scoped. Reuse the current session's short-hash namespace instead of inventing ad-hoc domains.",
		"- Prefer concise service labels and stable health paths so repeated redeploys land on the same routed host.",
		"- Keep additional services on certificate-safe single-label subdomains under `*.alter0.cn`. Do not generate nested hosts such as `https://<service>.<short_hash>.alter0.cn` or `https://<short_hash>.travel.alter0.cn`.",
		"- Never report server-local artifact links such as `/srv/...`, runtime-root `workspaces/...`, `file://...`, `localhost`, or `127.0.0.1` as user-openable deliverables.",
		"- Do not finish with a local HTML/file path as the primary artifact. Publish it first, then return the deployed `https://*.alter0.cn` URL.",
		"- For public travel guides, deploy `service_name=travel` on `https://travel-<short_hash>.alter0.cn`. Publish only the current session workspace root after the current request's `index.html` has been generated there; do not publish a stale or unrelated page as a fallback.",
	}, "\n")
}

func travelSkillGuide() string {
	return strings.Join([]string{
		"# travel",
		"",
		"- Use this skill when creating or revising a city travel guide, itinerary, recommendation pool, route-map treatment, or public HTML travel page.",
		"- The canonical file-backed skill lives at `docs/skills/travel/SKILL.md`; read it for the city guide workflow, mobile-first page contract, route-card structure, generated itinerary image requirements, and travel service delivery checks.",
	}, "\n")
}

func resolveBuiltinSkillRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return wd
		}
		current = parent
	}
}
