# Lightweight Workbench UI/UX Design

This design pass keeps Alter0 as a thin, private wrapper around strong CLI runtimes such as Claude Code and Codex. The interface should feel closer to a focused AI workbench than an admin console: few routes, direct actions, readable state, and low visual noise.

Generated reference boards:

- Chat: `docs/design/references/chat-reference.png`
- Terminal: `docs/design/references/terminal-reference.png`
- Settings: `docs/design/references/settings-reference.png`
- Login: `docs/design/references/login-reference.png`
- Workbench page states: `docs/design/references/workbench-pages-reference.png`
- Terminal overlays: `docs/design/references/terminal-overlays-reference.png`
- Settings page states: `docs/design/references/settings-pages-reference.png`
- Settings detail states: `docs/design/references/settings-detail-reference.png`

## Shared Direction

- Product tone: private, lightweight, operational, high-trust.
- Visual system: off-white page, white surfaces, ink text, thin slate borders, green/teal status accent, restrained shadows, maximum 8px radius for key surfaces.
- Navigation: only Chat, Terminal, Settings are top-level routes. Running-session history lives in the navigation rail for runtime pages.
- Desktop behavior: left rail stays persistent; runtime pages prioritize the active conversation or terminal; settings uses index plus detail/content.
- Mobile behavior: primary content first; navigation and sessions move into drawers; composer stays fixed near the bottom with safe-area padding.

## Chat

Product design:
Chat is the default entry for user intent, memory-aware conversations, and runtime orchestration. It should not look like a management page; it is the place where the user asks for outcomes and the system chooses the right skills.

PC UI:
The desktop layout keeps the left navigation and session rail visible, then gives the timeline a wide, calm reading column. The top area remains compact with model/status/details controls. The composer is the main action surface and uses a simple input plus compact tool/skill controls.

Mobile UI:
The message timeline and composer are first. Sessions are available from the menu drawer. The header hides secondary explanatory copy so vertical space goes to the current conversation.

UX rules:
Session selection should be scannable by title and recency. The model/skill controls should be discoverable but visually secondary to the composer. Empty state prompts should be concise and never feel like a landing page.

## Terminal

Product design:
Terminal is a manual execution surface. It is for directly selected terminal/Codex style work, not for automatic routing from chat. The user should immediately understand which workspace and process status they are controlling.

PC UI:
The desktop page emphasizes the active terminal output with a high-contrast terminal-like reading surface, a stable composer, a session rail, and compact workspace/status metadata.

Mobile UI:
The terminal output takes almost the whole viewport. Sessions and details move to drawers. The composer remains large enough for prompts and commands without requiring precise taps.

UX rules:
Output must remain selectable and readable. Status should be visible but not decorative. Attach/send controls must be touch-safe. Failed, exited, and interrupted states should appear near the composer because that is where recovery happens.

### Terminal Overlays

Product design:
The Terminal composer has two live overlays: skill selection and Codex slash commands. Both overlays should feel like command surfaces, not configuration pages.

PC UI:
The slash command list opens above the composer as a dense single-column command palette with command token, label, hover state, and scroll containment. The skill selector opens as a compact sheet above the composer with selected count implied through checked rows, short descriptions, and touch-safe toggles.

Mobile UI:
Both overlays are viewport-aware fixed panels above the composer and keyboard safe area. They must never cover the full screen unless content requires scrolling.

UX rules:
Typing `/` should immediately narrow commands. Tapping a command inserts it and returns focus to the composer. Skill toggles update the next run only, so the panel should be visually secondary to Send.

## Settings

Product design:
Settings is a compact control surface for the wrapper layer: Codex runtime, model providers, skill repository, memory, workspaces/cleanup, and schedules. It should not behave like a large operations suite.

PC UI:
The desktop settings page uses a narrow section index and a content area. Runtime combines Codex Direct health, model/reasoning configuration, provider registration state, readiness, and diagnostics. Skills, Memory, Workspaces, and Schedules are reachable from the same compact settings page.

Mobile UI:
The section index becomes a compact tab grid. Cards stack vertically, with short labels and visible health/status rows.

UX rules:
Each section should answer: what is configured, is it healthy, what is the next action. Secondary details should sit inside the selected section rather than creating more top-level pages.

### Settings Subpages

Runtime:
Desktop starts with the read-only Codex identity and quota card, then shows the compact overview, model configuration, readiness, and diagnostics. Mobile uses the same order so account and quota state are visible before runtime tuning. Codex authentication is service-account scoped and preserves the active identity display, but does not expose import, login, switching, IDs, or multi-account management controls in the UI.

Skills:
Desktop shows a responsive card grid with id, type, name, scope, and version. Mobile keeps one skill per row/card with field labels visible, because skill identifiers are often long.

Memory:
Desktop prioritizes task history plus the selected memory/task detail pane. Memory tabs stay compact and visible. Mobile presents tabs as a horizontal strip, then stacks history, daily summaries, long-term memory, and spec content.

Workspaces:
Desktop combines session history and task cleanup/history in a single settings section. Mobile keeps filters first, then session/task rows, then detail.

Schedules:
Desktop presents scheduled jobs as operational cards with schedule mode, timezone, prompt preview, and retry policy. Mobile keeps schedule cards single-column with status and next action visible.

Compatibility subpages:
Runtime, Skills, Memory, and Schedules remain supported as settings entries. They share the same narrow settings rail, responsive card grid, form fields, tables, empty states, and status chip rules.

## Login

Product design:
Login is a private workspace gate, not a marketing page. It should make entry fast and communicate readiness/security without extra explanation.

PC UI:
A single centered authentication panel, visible label, password field with show/hide, primary sign-in button, readiness line, and small return-path note.

Mobile UI:
The same form fills the safe viewport with comfortable tap targets and no horizontal overflow. Error space is stable enough that the form does not jump dramatically.

UX rules:
The password label is always visible. Focus rings must be obvious. Error copy is local to the form. The primary action is the only dominant button.
