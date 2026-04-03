const appShell = document.getElementById("appShell");
const sessionList = document.getElementById("sessionList");
const sessionEmpty = document.getElementById("sessionEmpty");
const sessionLoadError = document.getElementById("sessionLoadError");
const sessionHistoryPanel = document.getElementById("sessionHistoryPanel");
const sessionHistoryToggle = document.getElementById("sessionHistoryToggle");
const welcomeScreen = document.getElementById("welcomeScreen");
const messageArea = document.getElementById("messageArea");
const chatForm = document.getElementById("chatForm");
const input = document.getElementById("composerInput");
const composerShell = document.querySelector(".composer-shell");
const sendButton = document.getElementById("sendButton");
const charCount = document.getElementById("charCount");
const newChatButton = document.getElementById("newChatButton");
const mobileNewChatButton = document.getElementById("mobileNewChatButton");
const navToggle = document.getElementById("navToggle");
const sessionToggle = document.getElementById("sessionToggle");
const togglePaneButton = document.getElementById("togglePaneButton");
const navCollapseButton = document.getElementById("navCollapseButton");
const mobileBackdrop = document.getElementById("mobileBackdrop");
const sessionHeading = document.getElementById("sessionHeading");
const sessionSubheading = document.getElementById("sessionSubheading");
const welcomeHeading = document.getElementById("welcomeHeading");
const welcomeDescription = document.getElementById("welcomeDescription");
const sessionPane = document.querySelector(".session-pane");
const primaryNav = document.querySelector(".primary-nav");
const chatPane = document.querySelector(".chat-pane");
const chatView = document.getElementById("chatView");
const routeView = document.getElementById("routeView");
const routeTitle = document.getElementById("routeTitle");
const routeSubtitle = document.getElementById("routeSubtitle");
const routeActionButton = document.getElementById("routeActionButton");
const routeBody = document.getElementById("routeBody");
const chatRuntimePanel = document.getElementById("chatRuntimePanel");
const menuRouteItems = document.querySelectorAll(".menu-item[data-route]");
const navTooltipTargets = [...menuRouteItems, navCollapseButton];
const rootStyle = document.documentElement.style;

const MAX_CHARS = 10000;
const DEFAULT_ROUTE = "chat";
const SWIPE_CLOSE_THRESHOLD = 46;
const TERMINAL_SCROLL_STICKY_THRESHOLD = 32;
const TERMINAL_JUMP_BOTTOM_SHOW_THRESHOLD = 240;
const TERMINAL_JUMP_TOP_SHOW_THRESHOLD = 180;
const NAV_TOOLTIP_SHOW_DELAY = 90;
const NAV_TOOLTIP_HIDE_DELAY = 40;
const NAV_TOOLTIP_OFFSET = 12;
const CHAT_TASK_POLL_INTERVAL_MS = 4000;
const CHAT_TASK_POLL_HIDDEN_INTERVAL_MS = 15000;
const MOBILE_VIEWPORT_SYNC_THRESHOLD_PX = 8;
const MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS = 240;
const TERMINAL_POLL_INTERVAL_ACTIVE_MS = 1600;
const TERMINAL_POLL_INTERVAL_IDLE_MS = 4000;
const TERMINAL_POLL_INTERVAL_HIDDEN_MS = 12000;
const TERMINAL_SESSION_LIST_POLL_INTERVAL_MS = 15000;
const TERMINAL_SESSION_LIST_POLL_HIDDEN_INTERVAL_MS = 60000;
const TERMINAL_STORAGE_PERSIST_ACTIVE_DELAY_MS = 320;
const TERMINAL_STORAGE_PERSIST_IDLE_DELAY_MS = 1200;
const TERMINAL_INPUT_PAINT_IDLE_MS = 480;
const STREAM_ENDPOINT = "/api/messages/stream";
const FALLBACK_ENDPOINT = "/api/messages";
const MAIN_AGENT_ID = "main";
const MAIN_AGENT_NAME = "Alter0";
const SESSION_STORAGE_KEY = "alter0.web.sessions.v3";
const LEGACY_CHAT_SESSION_STORAGE_KEY = "alter0.web.sessions.chat.v2";
const LEGACY_AGENT_SESSION_STORAGE_KEY = "alter0.web.sessions.agent.v2";
const LEGACY_SESSION_STORAGE_KEY = "alter0.web.sessions.v1";
const SESSION_HISTORY_PANEL_STORAGE_KEY = "alter0.web.session-history-panel.v1";
const COMPOSER_DRAFT_STORAGE_KEY = "alter0.web.composer.drafts.v1";
const SESSION_ACTIVE_STORAGE_KEY = "alter0.web.session.active.v1";
const TERMINAL_STORAGE_KEY = "alter0.web.terminal.sessions.v2";
const TERMINAL_CLIENT_STORAGE_KEY = "alter0.web.terminal.client.v1";
const RUNTIME_RESTART_NOTICE_STORAGE_KEY = "alter0.web.runtime.restart-notice.v1";
const AVAILABLE_CHAT_TOOLS = [
  {
    id: "search_memory",
    name: "Search Memory",
    description: "Search the resolved memory files by keyword and return matching snippets."
  },
  {
    id: "read_memory",
    name: "Read Memory",
    description: "Read one of the resolved memory files injected into the current agent context."
  },
  {
    id: "write_memory",
    name: "Write Memory",
    description: "Write durable preferences or shorthand mappings back to a resolved memory file."
  },
  {
    id: "codex_exec",
    name: "Codex Exec",
    description: "Allow the agent to hand every concrete execution step to Codex CLI."
  },
  {
    id: "delegate_agent",
    name: "Delegate Agent",
    description: "Allow Alter0 or another delegatable agent to hand off a specialist subtask to a child agent."
  }
];
const AGENT_MEMORY_FILE_OPTIONS = [
  {
    id: "user_md",
    name: "USER.md",
    description: "User profile, collaboration preferences, and stable output conventions."
  },
  {
    id: "soul_md",
    name: "SOUL.md",
    description: "Mandatory long-term rules and hard constraints with highest priority."
  },
  {
    id: "agents_md",
    name: "AGENTS.md",
    description: "Private operating rules for the current agent, resolved to .alter0/agents/<agent_id>/AGENTS.md and not shared across agents."
  },
  {
    id: "memory_long_term",
    name: "MEMORY.md / memory.md",
    description: "Long-term durable memory, including alter0 long-term memory fallback."
  },
  {
    id: "memory_daily_today",
    name: "Daily Memory (Today)",
    description: "Today's daily memory log, aligned with OpenClaw-style daily context."
  },
  {
    id: "memory_daily_yesterday",
    name: "Daily Memory (Yesterday)",
    description: "Yesterday's daily memory log for short-horizon recall on session start."
  }
];
const I18N = {
  en: {
    // Navigation
    "nav.workspace": "Workspace",
    "nav.agent_studio": "Agent Studio",
    "nav.chat": "Chat",
    "nav.coding": "Coding",
    "nav.writing": "Writing",
    "nav.agent_runtime": "Agent",
    "nav.control": "Control",
    "nav.agent": "Profiles",
    "nav.products": "Products",
    "nav.settings": "Settings",
    "nav.channels": "Channels",
    "nav.sessions": "Sessions",
    "nav.tasks": "Tasks",
    "nav.terminal": "Terminal",
    "nav.cron-jobs": "Cron Jobs",
    "nav.memory": "Memory",
    "nav.skills": "Skills",
    "nav.mcp": "MCP",
    "nav.models": "Models",
    "nav.environments": "Environments",
    "nav.expand": "Expand navigation",
    "nav.collapse": "Collapse navigation",
    "composer.unsaved_confirm": "You have unsent content. Leave anyway?",
    
    // Session Pane
    "session.header": "Work with alter0",
    "session.close": "Close",
    "session.new": "New Chat",
    "session.new_agent": "New Agent Run",
    "session.delete": "Delete",
    "session.delete_failed": "Delete session failed: {error}",
    "session.recent": "Recent Sessions",
    "session.history.collapse": "Collapse",
    "session.history.expand": "Expand",
    "session.empty": "No sessions yet. Click New Chat to start.",
    
    // Chat Header
    "chat.menu": "Menu",
    "chat.title": "Chat",
    "chat.subtitle": "Ready to start a new conversation",
    "chat.sessions": "Sessions",
    "chat.lang": "English",
    
    // Welcome Screen
    "welcome.tag": "alter0 assistant",
    "welcome.heading": "Hello, how can I help you today?",
    "welcome.desc": "I am a helpful assistant that can help you with your questions.",
    "welcome.target_title": "Choose who should handle this conversation",
    "welcome.target_hint": "Choose the execution target for this conversation.",
    "welcome.agent_hint": "Choose one of the available Agents to start an execution session.",
    "welcome.model_title": "Choose the model for upcoming messages",
    "welcome.model_hint": "{agent} now handles this workspace by default. Provider, model, tools, and skills apply to upcoming messages.",
    "prompt.journey": "Let's start a new journey!",
    "prompt.skills": "Can you tell me what skills you have?",
    
    // Composer
    "composer.placeholder": "Input your message here...",
    "composer.send": "Send",
    "composer.note": "Works for you, grows with you",
    
    // Dynamic
    "time.just_now": "just now",
    "time.m_ago": "m ago",
    "time.h_ago": "h ago",
    "time.d_ago": "d ago",
    "msg.processing": "Processing...",
    "msg.received_empty": "Received request, but no output content.",
    "msg.stream_error": "Stream connection error",
    "msg.stream_failed": "Stream failed: {error}, please retry.",
    "msg.request_failed": "Request failed: {error}, please retry.",
    "msg.network_error": "Network error: {error}, please retry.",
    "session.new_title": "New Chat",
    "session.new_agent_title": "New Agent Run",
    "session.empty_sub": "Empty session, waiting for your first message",
    "session.empty_agent_sub": "Empty Agent session, choose an Agent and send the first goal",
    "session.no_active": "No active session. Click New Chat to start.",
    "session.no_active_agent": "No active Agent session. Open Agent to start.",
    "session.empty_body": "This session is empty. Type a message to start.",
    "session.empty_agent": "No Agent sessions yet. Open Agent to start.",
    "session.target.raw": "Raw Model",
    "session.target.agent": "Agent",
    "session.target.choose": "Choose a target",
    "session.model.default": "Service Default",
    "status.in_progress": "In Progress",
    "status.failed": "Failed",
    "status.done": "Done",
    "status.enabled": "Enabled",
    "status.disabled": "Disabled",
    "status.queued": "Queued",
    "status.running": "Running",
    "status.starting": "Starting",
    "status.ready": "Ready",
    "status.busy": "Busy",
    "status.success": "Success",
    "action.ok": "OK",
    "action.cancel": "Cancel",
    "status.canceled": "Canceled",
    "status.exited": "Exited",
    "status.interrupted": "Interrupted",
    "field.type": "Type",
    "field.description": "Description",
    "field.name": "Name",
    "field.scope": "Scope",
    "field.version": "Version",
    "field.interval": "Interval",
    "field.cron_expression": "Cron Expression",
    "field.schedule_mode": "Schedule Mode",
    "field.timezone": "Timezone",
    "field.input": "Input",
    "field.retry_limit": "Retry Limit",
    "field.session": "Session",
    "field.id": "ID",
    "field.messages": "Messages",
    "field.created": "Created",
    "field.path": "Path",
    "field.updated": "Updated",
    "field.date": "Date",
    "field.read_only": "Mode",
    "field.status": "Status",
    "field.phase": "Phase",
    "field.task_type": "Task Type",
    "field.trigger_type": "Trigger Type",
    "field.channel_type": "Channel Type",
    "field.channel_id": "Channel ID",
    "field.correlation_id": "Correlation ID",
    "field.agent_id": "Agent ID",
    "field.agent_name": "Agent",
    "field.message_id": "Message ID",
    "field.last_message_id": "Last Message ID",
    "field.request_message_id": "Request Message ID",
    "field.result_message_id": "Result Message ID",
    "field.job_id": "Job ID",
    "field.job_name": "Job Name",
    "field.fired_at": "Fired At",
    "field.goal": "Goal",
    "field.result": "Result",
    "field.finished": "Finished",
    "field.accepted_at": "Accepted At",
    "field.started_at": "Started At",
    "field.last_heartbeat_at": "Last Heartbeat",
    "field.timeout_at": "Timeout Window",
    "field.progress": "Progress",
    "field.queue_position": "Queue Position",
    "field.queue_wait_ms": "Queue Wait",
    "field.retry_count": "Retry Count",
    "field.source_message": "Source Message",
    "field.finished_at": "Finished At",
    "field.tags": "Tags",
    "field.time_range": "Time Range",
    
    // Routes
    "route.chat.title": "Chat",
    "route.chat.subtitle": "Alter0 workspace for general-purpose conversations and orchestration",
    "route.coding.title": "Coding",
    "route.coding.subtitle": "Coding Agent workspace for repository analysis, implementation, and verification",
    "route.writing.title": "Writing",
    "route.writing.subtitle": "Writing Agent workspace for documentation, copy, and structured drafting",
    "route.agent_runtime.title": "Agent",
    "route.agent_runtime.subtitle": "Run conversations through a selected Agent with independent session history",
    "route.agent.title": "Agent 配置",
    "route.agent.subtitle": "Create and configure the Agent Profiles available to Agent conversations. Service-managed ID and version are generated automatically.",
    "route.agent.empty": "No Agent Profiles available.",
    "route.agent.create": "Create New Agent",
    "route.agent.edit": "Edit Agent",
    "route.agent.form.title": "Agent Configuration",
    "route.agent.form.name": "Agent Name",
    "route.agent.form.prompt": "System Prompt",
    "route.agent.form.tools": "Tools",
    "route.agent.process.label": "Process",
    "route.agent.process.steps": "{count} steps",
    "route.agent.process.empty": "No execution details yet.",
    "route.agent.step.observation": "Observation",
    "route.agent.form.skills": "Skills",
    "route.agent.form.mcps": "MCP",
    "route.agent.form.memory_files": "Memory Files",
    "route.agent.form.iterations": "Max Iterations",
    "route.agent.form.enabled": "Enabled",
    "route.agent.form.managed": "Service-managed Fields",
    "route.agent.form.id": "Agent ID",
    "route.agent.form.version": "Version",
    "route.agent.form.scope": "Scope",
    "route.agent.form.pending": "Generated on first save",
    "route.agent.form.save": "Save Agent",
    "route.agent.form.delete": "Delete Agent",
    "route.agent.form.test": "Open Agent",
    "route.agent.form.cancel": "Reset",
    "route.agent.form.new": "Unsaved Agent",
    "route.agent.form.empty": "Select an Agent card or create a new one to configure it.",
    "route.agent.save_failed": "Save Agent failed: {error}",
    "route.agent.delete_failed": "Delete Agent failed: {error}",
    "route.agent.saved": "Agent saved.",
    "route.agent.deleted": "Agent deleted.",
    "route.products.title": "Products",
    "route.products.subtitle": "Manage product workspaces, master agents, and worker-agent matrices",
    "route.products.empty": "No Products available.",
    "route.products.create": "Create Product",
    "route.products.edit": "Edit Product",
    "route.products.panel.workspace": "Workspace",
    "route.products.panel.studio": "Studio",
    "route.products.workspace.title": "Product Workspace",
    "route.products.workspace.subtitle": "Talk to the product master agent and maintain product detail pages.",
    "route.products.workspace.empty": "Select a Product to open its workspace.",
    "route.products.workspace.overview": "Overview",
    "route.products.workspace.master": "Master Agent",
    "route.products.workspace.output": "Artifacts",
    "route.products.workspace.sources": "Knowledge Sources",
    "route.products.workspace.tags": "Tags",
    "route.products.workspace.workers": "Worker Matrix",
    "route.products.workspace.chat_title": "Master Agent Conversation",
    "route.products.workspace.chat_hint": "Use the product master agent to create or revise detail pages.",
    "route.products.workspace.chat_placeholder": "Example: create a 3-day Wuhan page with metro-first travel and local food.",
    "route.products.workspace.chat_send": "Send to Master",
    "route.products.workspace.chat_failed": "Workspace request failed: {error}",
    "route.products.workspace.synced": "Workspace synced.",
    "route.products.workspace.spaces": "Detail Pages",
    "route.products.workspace.space_empty": "No detail pages yet. Ask the master agent to create one.",
    "route.products.workspace.open_new": "New Detail Page",
    "route.products.workspace.detail": "Page Detail",
    "route.products.workspace.detail_empty": "Select a detail page to view its content.",
    "route.products.workspace.space_revision": "Revision",
    "route.products.workspace.updated": "Updated",
    "route.products.workspace.days": "Days",
    "route.products.workspace.content": "Page Content",
    "route.products.workspace.notes": "Notes",
    "route.products.workspace.routes": "Daily Routes",
    "route.products.workspace.layers": "Map Layers",
    "route.products.workspace.open_page": "Open HTML Page",
    "route.products.form.new": "Unsaved Product",
    "route.products.form.id": "Product ID",
    "route.products.form.version": "Version",
    "route.products.form.owner": "Owner Type",
    "route.products.form.name": "Product Name",
    "route.products.form.slug": "Slug",
    "route.products.form.summary": "Summary",
    "route.products.form.status": "Status",
    "route.products.form.visibility": "Visibility",
    "route.products.form.master": "Master Agent",
    "route.products.form.entry_route": "Entry Route",
    "route.products.form.tags": "Tags",
    "route.products.form.artifacts": "Artifact Types",
    "route.products.form.knowledge": "Knowledge Sources",
    "route.products.form.worker_agents": "Worker Agents",
    "route.products.form.managed": "Service-managed Fields",
    "route.products.form.save": "Save Product",
    "route.products.form.delete": "Delete Product",
    "route.products.form.cancel": "Reset",
    "route.products.form.builtin_notice": "Built-in Products are managed by the service and are read-only here.",
    "route.products.saved": "Product saved.",
    "route.products.deleted": "Product deleted.",
    "route.products.save_failed": "Save Product failed: {error}",
    "route.products.delete_failed": "Delete Product failed: {error}",
    "route.products.drafts.title": "Draft Studio",
    "route.products.drafts.subtitle": "Generate product drafts and publish reviewed matrices",
    "route.products.drafts.empty": "No product drafts yet.",
    "route.products.drafts.generated": "Product draft generated.",
    "route.products.drafts.published": "Product draft published.",
    "route.products.drafts.review_saved": "Product draft saved.",
    "route.products.drafts.generate_failed": "Generate draft failed: {error}",
    "route.products.drafts.publish_failed": "Publish draft failed: {error}",
    "route.products.drafts.review_failed": "Save draft failed: {error}",
    "route.products.drafts.form.name": "Draft Name",
    "route.products.drafts.form.goal": "Goal",
    "route.products.drafts.form.target_users": "Target Users",
    "route.products.drafts.form.core": "Core Capabilities",
    "route.products.drafts.form.constraints": "Constraints",
    "route.products.drafts.form.artifacts": "Expected Artifacts",
    "route.products.drafts.form.integrations": "Integrations",
    "route.products.drafts.form.mode": "Mode",
    "route.products.drafts.form.generate": "Generate Draft",
    "route.products.drafts.form.save": "Save Draft",
    "route.products.drafts.form.publish": "Publish Draft",
    "route.products.drafts.form.editor": "Draft JSON",
    "route.products.drafts.form.editor_hint": "Review and adjust the draft JSON before publish.",
    "route.products.drafts.form.expand_disabled": "Expand mode requires a selected managed Product.",
    "route.products.drafts.form.mode.bootstrap": "Bootstrap",
    "route.products.drafts.form.mode.expand": "Expand Selected Product",
    "route.products.drafts.detail.master": "Master Agent",
    "route.products.drafts.detail.workers": "Worker Matrix",
    "route.products.drafts.detail.conflicts": "Conflict Suggestions",
    "route.products.drafts.detail.review": "Review Status",
    "route.products.drafts.detail.empty": "Select a draft to review, edit, and publish.",
    "route.products.drafts.detail.generated": "Generated At",
    "route.products.drafts.detail.updated": "Updated At",
    "route.products.drafts.detail.mode": "Generation Mode",
    "route.products.drafts.detail.product": "Product Draft",
    "route.products.drafts.detail.published": "Published Product",
    "chat.runtime.target": "Conversation Target",
    "chat.runtime.agent": "Agent",
    "chat.runtime.agent_pick": "Choose Agent",
    "chat.runtime.provider": "Provider",
    "chat.runtime.model": "Model",
    "chat.runtime.model_short": "Model",
    "chat.runtime.empty": "No enabled model provider is available yet. Configure one in Models to enable session-level model switching.",
    "chat.runtime.hint": "Applies to upcoming messages in the current chat session.",
    "chat.runtime.tools_mcp": "Tools / MCP",
    "chat.runtime.tools_short": "Tools",
    "chat.runtime.skills": "Skills",
    "chat.runtime.skills_short": "Skills",
    "chat.runtime.target_hint": "Choose the execution target before the first message.",
    "chat.runtime.agent_hint": "Choose the Agent for this session before the first message.",
    "chat.runtime.model_hint": "Switches apply to upcoming messages in this session.",
    "chat.runtime.tools_hint": "Select extra Tools and MCP integrations for upcoming messages.",
    "chat.runtime.skills_hint": "Select extra Skills for upcoming messages.",
    "chat.runtime.mobile": "Session",
    "chat.runtime.mobile_hint": "Choose model, tools, and skills for upcoming messages.",
    "chat.runtime.mobile_meta": "Model {model} · Tools {tools} · Skills {skills}",
    "chat.runtime.active": "Active",
    "chat.runtime.available": "Available",
    "chat.runtime.category.tools": "Tools",
    "chat.runtime.category.mcps": "MCP",
    "chat.runtime.locked": "Conversation target is locked after the first message.",
    "chat.runtime.none": "No items in this section.",
    "route.agent.target_agents": "My Agents",
    "route.agent.pick": "Choose an Agent above to start a new execution session.",
    "route.agent.send_failed": "Agent request failed: {error}",
    "route.channels.title": "Channels",
    "route.channels.subtitle": "Manage connection channels",
    "route.channels.empty": "No Channels available.",
    "route.sessions.title": "Sessions",
    "route.sessions.subtitle": "View archived sessions with source filters",
    "route.sessions.empty": "No sessions found.",
    "route.sessions.filter.trigger_type": "Trigger Type",
    "route.sessions.filter.channel_type": "Channel Type",
    "route.sessions.filter.channel_id": "Channel ID",
    "route.sessions.filter.message_id": "Message ID",
    "route.sessions.filter.job_id": "Job ID",
    "route.sessions.filter.apply": "Apply",
    "route.sessions.filter.reset": "Reset",
    "route.sessions.open_detail": "View Detail",
    "route.copy_value": "Copy value",
    "route.tasks.title": "Tasks",
    "route.tasks.subtitle": "Observe runtime tasks with source, status, and timeline filters",
    "route.tasks.empty": "No tasks found.",
    "route.tasks.empty_hint": "Try adjusting filters or check back in a moment.",
    "route.tasks.filter.session": "Session ID",
    "route.tasks.filter.status": "Status",
    "route.tasks.filter.trigger_type": "Trigger Type",
    "route.tasks.filter.channel_type": "Channel Type",
    "route.tasks.filter.channel_id": "Channel ID",
    "route.tasks.filter.message_id": "Message ID",
    "route.tasks.filter.source_message_id": "Source Message ID",
    "route.tasks.filter.start_at": "Start",
    "route.tasks.filter.end_at": "End",
    "route.tasks.filter.advanced": "Advanced Filters",
    "route.tasks.filter.advanced_show": "Show Advanced Filters",
    "route.tasks.filter.advanced_hide": "Hide Advanced Filters",
    "route.tasks.filter.applying": "Applying filters...",
    "route.tasks.filter.apply": "Apply",
    "route.tasks.filter.reset": "Reset",
    "route.tasks.copy_task_id": "Copy Task ID",
    "route.tasks.open_detail": "Open Drawer",
    "route.tasks.drawer.title": "Task Detail",
    "route.tasks.drawer.empty": "Select a task to view detail.",
    "route.tasks.drawer.close": "Close",
    "route.tasks.drawer.identifiers": "Identifiers",
    "route.tasks.drawer.runtime": "Runtime Details",
    "route.tasks.drawer.quick_actions": "Quick Actions",
    "route.tasks.drawer.quick_timeline": "Generate timeline version",
    "route.tasks.drawer.quick_interview": "Generate interview brief",
    "route.tasks.actions.retry_tip": "Retry: run the task again from scratch.",
    "route.tasks.actions.replay_tip": "Replay: reload and replay current task logs only.",
    "route.tasks.progress": "Progress",
    "route.tasks.page.label": "Page",
    "route.tasks.page.next": "Next Page",
    "route.tasks.logs.title": "Execution Logs",
    "route.tasks.logs.empty": "No execution logs.",
    "route.tasks.logs.streaming": "Streaming latest logs...",
    "route.tasks.logs.done": "Log stream completed.",
    "route.tasks.logs.disconnected": "Log stream disconnected. You can reconnect.",
    "route.tasks.logs.reconnect": "Reconnect",
    "route.tasks.logs.replay": "Replay",
    "route.tasks.terminal.title": "Terminal",
    "route.tasks.terminal.input_placeholder": "Type command or prompt...",
    "route.tasks.terminal.send": "Send",
    "route.tasks.terminal.sending": "Sending...",
    "route.tasks.terminal.hint": "Supports follow-up interaction in the current terminal session.",
    "route.tasks.terminal.followup_note": "Each Send stays in the same Codex session thread.",
    "route.tasks.actions.retry": "Retry",
    "route.tasks.actions.cancel": "Cancel",
    "route.tasks.result.title": "Result Output",
    "route.terminal.title": "Terminal",
    "route.terminal.subtitle": "Persistent Codex CLI sessions with runtime-aligned status",
    "route.terminal.new": "New Codex Session",
    "route.terminal.new_short": "New",
    "route.terminal.empty": "No Codex sessions yet. Create a session to start a CLI thread.",
    "route.terminal.pick": "Select a Codex session to continue.",
    "route.terminal.sessions": "Sessions",
    "route.terminal.session_count": "{count} sessions",
    "route.terminal.hide_sessions": "Hide sessions",
    "route.terminal.details_show": "Details",
    "route.terminal.details_hide": "Hide details",
    "route.terminal.last_output_label": "Last output",
    "route.terminal.input": "Ask Codex...",
    "route.terminal.send": "Send",
    "route.terminal.sending": "Sending...",
    "route.terminal.close": "Close",
    "route.terminal.closing": "Closing...",
    "route.terminal.delete": "Delete",
    "route.terminal.deleting": "Deleting...",
    "route.terminal.session": "Session",
    "route.terminal.shell": "CLI",
    "route.terminal.path": "Path",
    "route.terminal.status": "Status",
    "route.terminal.last_output": "Last output {time}",
    "route.terminal.no_output": "No output yet",
    "route.terminal.logs.heading": "Codex Activity ({session})",
    "route.terminal.logs.empty": "Codex session ready. Send a prompt to start.",
    "route.terminal.busy": "Codex is working on the current turn...",
    "route.terminal.process.label": "Process",
    "route.terminal.process.header": "Processed {duration}",
    "route.terminal.process.steps": "{count} steps",
    "route.terminal.process.empty": "No process steps yet.",
    "route.terminal.process.loading": "Waiting for Codex...",
    "route.terminal.final.heading": "Final Output",
    "route.terminal.output_expand": "Show more",
    "route.terminal.output_collapse": "Show less",
    "route.terminal.jump_bottom": "Latest",
    "route.terminal.jump_top": "Top",
    "route.terminal.jump_prev": "Previous",
    "route.terminal.jump_next": "Next",
    "route.terminal.navigation": "Turn navigation",
    "route.terminal.step.loading": "Loading step details...",
    "route.terminal.step.error": "Load step failed: {error}",
    "route.terminal.step.search": "Search in output",
    "route.terminal.send_failed": "Send failed: {error}",
    "route.terminal.logs_failed": "Load terminal output failed: {error}",
    "route.terminal.close_failed": "Close terminal failed: {error}",
    "route.terminal.delete_failed": "Delete terminal failed: {error}",
    "route.terminal.delete_confirm": "Delete this Codex session and its workspace files?",
    "route.terminal.loading": "Loading terminal session...",
    "route.terminal.interrupted": "Codex runtime exited. Send a new input to recover this session.",
    "route.terminal.closed": "Codex session exited. Send a new input to continue in this session.",
    "trigger.user": "User",
    "trigger.cron": "Cron",
    "trigger.system": "System",
    "channel.cli": "CLI",
    "channel.web": "Web",
    "channel.scheduler": "Scheduler",
    "route.cron.title": "Cron Jobs",
    "route.cron.subtitle": "Configure schedules and trace fired sessions",
    "route.cron.empty": "No Cron Jobs available.",
    "route.cron.form.title": "Create / Update Cron Job",
    "route.cron.form.job_id": "Job ID",
    "route.cron.form.name": "Task Name",
    "route.cron.form.input": "Prompt / Input",
    "route.cron.form.retry": "Retry Limit",
    "route.cron.form.timezone": "Timezone",
    "route.cron.form.mode": "Schedule Mode",
    "route.cron.form.every": "Every",
    "route.cron.form.unit": "Unit",
    "route.cron.form.time": "Time",
    "route.cron.form.weekday": "Weekday",
    "route.cron.form.expression": "Cron Expression",
    "route.cron.form.enabled": "Enabled",
    "route.cron.form.submit": "Save Job",
    "route.cron.form.reset": "Reset",
    "route.cron.list.title": "Configured Jobs",
    "route.cron.mode.every": "Every",
    "route.cron.mode.daily": "Daily",
    "route.cron.mode.weekly": "Weekly",
    "route.cron.unit.minute": "Minute",
    "route.cron.unit.hour": "Hour",
    "route.cron.unit.day": "Day",
    "route.cron.weekday.0": "Sunday",
    "route.cron.weekday.1": "Monday",
    "route.cron.weekday.2": "Tuesday",
    "route.cron.weekday.3": "Wednesday",
    "route.cron.weekday.4": "Thursday",
    "route.cron.weekday.5": "Friday",
    "route.cron.weekday.6": "Saturday",
    "route.cron.action.edit": "Edit",
    "route.cron.action.delete": "Delete",
    "route.cron.action.runs": "Runs",
    "route.cron.runs.empty": "No runs yet.",
    "route.cron.runs.open_sessions": "View Sessions",
    "route.cron.expression_invalid": "Expression is not in supported visual pattern.",
    "route.cron.save_failed": "Save failed: {error}",
    "route.cron.delete_failed": "Delete failed: {error}",
    "route.cron.runs_failed": "Load runs failed: {error}",
    "route.memory.title": "Memory",
    "route.memory.subtitle": "Summary-first memory view with task history drill-down",
    "route.memory.tab.long_term": "Long-Term",
    "route.memory.tab.daily": "Daily",
    "route.memory.tab.tasks": "Task History",
    "route.memory.tab.mandatory": "SOUL.md",
    "route.memory.tab.specification": "Specification",
    "route.memory.empty.long_term": "No long-term memory file available.",
    "route.memory.empty.daily": "No daily memory files available.",
    "route.memory.empty.mandatory": "No SOUL.md file available.",
    "route.memory.empty.specification": "No memory specification document available.",
    "route.memory.read_only": "Read-only",
    "route.memory.daily.source": "Source directory",
    "route.memory.daily.summary": "Summary",
    "route.memory.spec.section.default": "Document",
    "route.memory.tasks.empty": "No task history.",
    "route.memory.tasks.filter.status": "Status",
    "route.memory.tasks.filter.task_type": "Task Type",
    "route.memory.tasks.filter.start_at": "Start",
    "route.memory.tasks.filter.end_at": "End",
    "route.memory.tasks.filter.apply": "Apply",
    "route.memory.tasks.filter.reset": "Reset",
    "route.memory.tasks.open_detail": "View Detail",
    "route.memory.tasks.detail.title": "Task Detail",
    "route.memory.tasks.detail.empty": "Select a task summary to view detail.",
    "route.memory.tasks.logs.load": "Load Logs",
    "route.memory.tasks.logs.more": "Load More Logs",
    "route.memory.tasks.page.label": "Page",
    "route.memory.tasks.page.next": "Next Page",
    "route.memory.tasks.logs.empty": "No logs available.",
    "route.memory.tasks.artifacts.load": "Load Artifacts",
    "route.memory.tasks.artifacts.empty": "No artifacts available.",
    "route.memory.tasks.artifacts.download": "Download",
    "route.memory.tasks.artifacts.preview": "Preview",
    "route.memory.tasks.artifacts.download_fail": "Download failed: {error}",
    "route.memory.tasks.back": "Back to Summary",
    "route.memory.tasks.rebuild": "Rebuild Summary",
    "route.memory.tasks.rebuild_ok": "Summary rebuilt.",
    "route.memory.tasks.rebuild_fail": "Summary rebuild failed: {error}",
    "route.memory.tasks.logs_hint": "Logs unavailable. You can rebuild summary.",
    "route.skills.title": "Skills",
    "route.skills.subtitle": "Skills configuration",
    "route.skills.empty": "No Skills available.",
    "route.mcp.title": "MCP",
    "route.mcp.subtitle": "Model Context Protocol configuration",
    "route.mcp.empty": "No MCP available.",
    "route.models.title": "Models",
    "route.models.subtitle": "Model capabilities",
    "route.envs.title": "Environments",
    "route.envs.subtitle": "Environment and deployment settings",
    "route.envs.save": "Save Changes",
    "route.envs.restart_service": "Restart Service",
    "route.envs.restarting": "Restarting service...",
    "route.envs.restarting_sync": "Syncing remote master and restarting service...",
    "route.envs.restart_failed": "Restart failed: {error}",
    "route.envs.restart_confirm": "Restart the service now?",
    "route.envs.restart_confirm_desc": "The page will reload automatically after the new runtime passes health checks.",
    "route.envs.restart_sync_master": "Sync remote master changes before restart",
    "route.envs.restart_sync_master_hint": "Recommended. Requires local branch master and a clean tracked working tree.",
    "route.envs.restart_wait_timeout": "Restart is taking longer than expected. Refresh and retry in a moment.",
    "route.envs.restart_success": "Service restart completed. The page is now connected to the latest runtime.",
    "route.envs.runtime.last_restart_at": "Last Restart",
    "route.envs.runtime.commit_hash": "Commit Hash",
    "route.envs.refresh": "Reload",
    "route.envs.show_sensitive": "Reveal Sensitive",
    "route.envs.hide_sensitive": "Hide Sensitive",
    "route.envs.current_value": "Configured",
    "route.envs.default_value": "Default",
    "route.envs.effective_value": "Effective",
    "route.envs.value_type": "Value Type",
    "route.envs.apply_mode": "Apply Mode",
    "route.envs.source": "Source",
    "route.envs.validation": "Validation",
    "route.envs.details": "More Details",
    "route.envs.pending_restart": "Pending Restart",
    "route.envs.hot_reload": "Hot Reload",
    "route.envs.no_changes": "No configuration changes.",
    "route.envs.saved": "Environment configuration saved.",
    "route.envs.save_failed": "Save failed: {error}",
    "route.envs.loading": "Loading environments...",
    "route.envs.audit.title": "Change Audits",
    "route.envs.audit.empty": "No environment audits.",
    "route.envs.audit.operator": "Operator",
    "route.envs.audit.at": "Changed At",
    "route.envs.audit.requires_restart": "Requires Restart",
    "route.envs.audit.change": "{key}: {old} → {new} ({mode})",
    "route.envs.restart_notice": "Some changes require restart: {keys}",
    "route.envs.apply.immediate": "Immediate",
    "route.envs.apply.restart": "Restart",
    "route.envs.source.default": "Default",
    "route.envs.source.runtime": "Runtime",
    "route.envs.source.persisted": "Persisted",
    "route.envs.type.integer": "Integer",
    "route.envs.type.duration": "Duration, e.g. 5s / 2m / 1h",
    "route.envs.type.string": "Text",
    "route.envs.type.enum": "Enumerated option",
    "route.envs.type.unknown": "Unknown",
    "route.envs.validation.none": "No constraints",
    "route.envs.hidden": "Hidden value",
    "route.connected": "Page Connected",
    "route.connected_desc": "This page route is active. Content can be expanded by module.",
    "loading": "Loading...",
    "load_failed": "Load failed: {error}"
  },
  zh: {
    // Navigation
    "nav.workspace": "工作区",
    "nav.agent_studio": "Agent Studio",
    "nav.chat": "对话",
    "nav.coding": "Coding",
    "nav.writing": "Writing",
    "nav.agent_runtime": "Agent",
    "nav.control": "控制台",
    "nav.agent": "配置",
    "nav.products": "产品",
    "nav.settings": "设置",
    "nav.channels": "通道",
    "nav.sessions": "会话列表",
    "nav.tasks": "任务观测",
    "nav.terminal": "终端代理",
    "nav.cron-jobs": "定时任务",
    "nav.memory": "记忆",
    "nav.skills": "技能",
    "nav.mcp": "MCP 协议",
    "nav.models": "模型",
    "nav.environments": "环境",
    "nav.expand": "展开导航",
    "nav.collapse": "收起导航",
    "composer.unsaved_confirm": "当前有未发送内容，仍要离开吗？",
    
    // Session Pane
    "session.header": "与 alter0 协作",
    "session.close": "关闭",
    "session.new": "新对话",
    "session.new_agent": "新 Agent 会话",
    "session.delete": "删除",
    "session.delete_failed": "删除会话失败：{error}",
    "session.recent": "最近会话",
    "session.history.collapse": "折叠",
    "session.history.expand": "展开",
    "session.empty": "暂无会话，点击“新对话”开始。",
    
    // Chat Header
    "chat.menu": "菜单",
    "chat.title": "对话",
    "chat.subtitle": "准备好开始新的对话",
    "chat.sessions": "会话",
    "chat.lang": "中文",
    
    // Welcome Screen
    "welcome.tag": "alter0 助手",
    "welcome.heading": "你好，今天有什么可以帮你？",
    "welcome.desc": "我是你的全能助手，随时准备回答你的问题。",
    "welcome.target_title": "选择这段会话由谁来处理",
    "welcome.target_hint": "为当前会话选择执行目标。",
    "welcome.agent_hint": "选择一个可用 Agent，开始独立的执行会话。",
    "welcome.model_title": "为后续消息选择模型",
    "welcome.model_hint": "当前工作区默认由 {agent} 处理，后续消息仍可继续调整 Provider、Model、Tools 与 Skills。",
    "prompt.journey": "让我们开启一段新的旅程吧！",
    "prompt.skills": "能告诉我你有哪些技能吗？",
    
    // Composer
    "composer.placeholder": "在此输入消息...",
    "composer.send": "发送",
    "composer.note": "为你效劳，伴你成长",
    
    // Dynamic
    "time.just_now": "刚刚",
    "time.m_ago": "分钟前",
    "time.h_ago": "小时前",
    "time.d_ago": "天前",
    "msg.processing": "处理中...",
    "msg.received_empty": "已收到请求，但没有输出内容。",
    "msg.stream_error": "流式连接错误",
    "msg.stream_failed": "流式请求失败：{error}，请重试。",
    "msg.request_failed": "请求失败：{error}，请重试。",
    "msg.network_error": "网络错误：{error}，请重试。",
    "session.new_title": "新对话",
    "session.new_agent_title": "新 Agent 会话",
    "session.empty_sub": "空会话，等待你的第一条消息",
    "session.empty_agent_sub": "空 Agent 会话，先选择 Agent，再发送第一条目标",
    "session.no_active": "没有活动会话。点击“新对话”开始。",
    "session.no_active_agent": "当前没有活动 Agent 会话。请前往 Agent 页面开始。",
    "session.empty_body": "当前会话为空。输入消息开始对话。",
    "session.empty_agent": "当前还没有 Agent 会话。请前往 Agent 页面开始。",
    "session.target.raw": "Raw Model",
    "session.target.agent": "Agent",
    "session.target.choose": "选择目标",
    "session.model.default": "服务默认",
    "status.in_progress": "进行中",
    "status.failed": "失败",
    "status.done": "完成",
    "status.enabled": "启用",
    "status.disabled": "停用",
    "status.queued": "排队中",
    "status.running": "运行中",
    "status.starting": "启动中",
    "status.ready": "就绪",
    "status.busy": "忙碌中",
    "status.success": "成功",
    "action.ok": "确定",
    "action.cancel": "取消",
    "status.canceled": "已取消",
    "status.exited": "已退出",
    "status.interrupted": "已中断",
    "field.type": "类型",
    "field.description": "描述",
    "field.name": "名称",
    "field.scope": "范围",
    "field.version": "版本",
    "field.interval": "间隔",
    "field.cron_expression": "Cron 表达式",
    "field.schedule_mode": "调度模式",
    "field.timezone": "时区",
    "field.input": "输入",
    "field.retry_limit": "重试次数",
    "field.session": "会话",
    "field.id": "ID",
    "field.messages": "消息数",
    "field.created": "创建时间",
    "field.path": "路径",
    "field.updated": "更新时间",
    "field.date": "日期",
    "field.read_only": "模式",
    "field.status": "状态",
    "field.phase": "阶段",
    "field.task_type": "任务类型",
    "field.trigger_type": "触发类型",
    "field.channel_type": "通道类型",
    "field.channel_id": "通道 ID",
    "field.correlation_id": "关联 ID",
    "field.agent_id": "Agent ID",
    "field.agent_name": "Agent",
    "field.message_id": "消息 ID",
    "field.last_message_id": "最近消息 ID",
    "field.request_message_id": "请求消息 ID",
    "field.result_message_id": "结果消息 ID",
    "field.job_id": "作业 ID",
    "field.job_name": "作业名称",
    "field.fired_at": "触发时间",
    "field.goal": "目标",
    "field.result": "结果",
    "field.finished": "完成时间",
    "field.accepted_at": "受理时间",
    "field.started_at": "开始时间",
    "field.last_heartbeat_at": "最近心跳",
    "field.timeout_at": "超时窗口",
    "field.progress": "进度",
    "field.queue_position": "排队位次",
    "field.queue_wait_ms": "排队耗时",
    "field.retry_count": "重试次数",
    "field.source_message": "源消息",
    "field.finished_at": "完成于",
    "field.tags": "标签",
    "field.time_range": "时间范围",
    
    // Routes
    "route.chat.title": "对话",
    "route.chat.subtitle": "默认 Alter0 对话工作区，适合通用任务与子 Agent 编排",
    "route.coding.title": "Coding",
    "route.coding.subtitle": "Coding Agent 工作区，面向仓库分析、实现与验证",
    "route.writing.title": "Writing",
    "route.writing.subtitle": "Writing Agent 工作区，面向文档、文案与结构化写作",
    "route.agent_runtime.title": "Agent",
    "route.agent_runtime.subtitle": "通过选定 Agent 执行会话，并维护独立的会话历史",
    "route.agent.title": "Agent Profiles",
    "route.agent.subtitle": "维护可在 Agent 会话中使用的 Agent Profile，ID 与版本由服务自动生成和管理。",
    "route.agent.empty": "暂无可用 Agent Profile。",
    "route.agent.create": "创建 Agent",
    "route.agent.edit": "编辑 Agent",
    "route.agent.form.title": "Agent 配置",
    "route.agent.form.name": "Agent 名称",
    "route.agent.form.prompt": "System Prompt",
    "route.agent.form.tools": "Tools",
    "route.agent.process.label": "过程",
    "route.agent.process.steps": "{count} 步",
    "route.agent.process.empty": "暂无执行细节。",
    "route.agent.step.observation": "观察",
    "route.agent.form.skills": "Skills",
    "route.agent.form.mcps": "MCP",
    "route.agent.form.memory_files": "Memory Files",
    "route.agent.form.iterations": "最大迭代次数",
    "route.agent.form.enabled": "启用",
    "route.agent.form.managed": "服务托管字段",
    "route.agent.form.id": "Agent ID",
    "route.agent.form.version": "版本",
    "route.agent.form.scope": "作用域",
    "route.agent.form.pending": "首次保存后自动生成",
    "route.agent.form.save": "保存 Agent",
    "route.agent.form.delete": "删除 Agent",
    "route.agent.form.test": "打开 Agent",
    "route.agent.form.cancel": "重置",
    "route.agent.form.new": "未保存 Agent",
    "route.agent.form.empty": "选择一个 Agent 卡片，或创建一个新的 Agent 后再配置。",
    "route.agent.save_failed": "保存 Agent 失败：{error}",
    "route.agent.delete_failed": "删除 Agent 失败：{error}",
    "route.agent.saved": "Agent 已保存。",
    "route.agent.deleted": "Agent 已删除。",
    "route.products.title": "Products",
    "route.products.subtitle": "管理 Product Workspace、主 Agent 与子 Agent 矩阵",
    "route.products.empty": "暂无 Product。",
    "route.products.create": "创建 Product",
    "route.products.edit": "编辑 Product",
    "route.products.panel.workspace": "Workspace",
    "route.products.panel.studio": "Studio",
    "route.products.workspace.title": "Product Workspace",
    "route.products.workspace.subtitle": "通过主 Agent 对话维护产品详情页与具体空间页面。",
    "route.products.workspace.empty": "选择一个 Product 后进入对应 Workspace。",
    "route.products.workspace.overview": "概览",
    "route.products.workspace.master": "主 Agent",
    "route.products.workspace.output": "产物",
    "route.products.workspace.sources": "知识源",
    "route.products.workspace.tags": "标签",
    "route.products.workspace.workers": "子 Agent 矩阵",
    "route.products.workspace.chat_title": "主 Agent 对话",
    "route.products.workspace.chat_hint": "通过主 Agent 创建或修改具体详情页。",
    "route.products.workspace.chat_placeholder": "例如：给武汉创建一个三天地铁优先、带夜宵推荐的城市页。",
    "route.products.workspace.chat_send": "发送给主 Agent",
    "route.products.workspace.chat_failed": "Workspace 请求失败：{error}",
    "route.products.workspace.synced": "Workspace 已同步。",
    "route.products.workspace.spaces": "详情页空间",
    "route.products.workspace.space_empty": "还没有详情页，可直接让主 Agent 创建。",
    "route.products.workspace.open_new": "新建详情页",
    "route.products.workspace.detail": "页面详情",
    "route.products.workspace.detail_empty": "选择一个详情页后查看页面内容。",
    "route.products.workspace.space_revision": "版本修订",
    "route.products.workspace.updated": "更新时间",
    "route.products.workspace.days": "天数",
    "route.products.workspace.content": "页面正文",
    "route.products.workspace.notes": "补充说明",
    "route.products.workspace.routes": "每日路线",
    "route.products.workspace.layers": "地图图层",
    "route.products.workspace.open_page": "打开 HTML 页面",
    "route.products.form.new": "未保存 Product",
    "route.products.form.id": "Product ID",
    "route.products.form.version": "版本",
    "route.products.form.owner": "归属类型",
    "route.products.form.name": "Product 名称",
    "route.products.form.slug": "Slug",
    "route.products.form.summary": "摘要",
    "route.products.form.status": "状态",
    "route.products.form.visibility": "可见性",
    "route.products.form.master": "总 Agent",
    "route.products.form.entry_route": "入口路由",
    "route.products.form.tags": "标签",
    "route.products.form.artifacts": "产物类型",
    "route.products.form.knowledge": "知识源",
    "route.products.form.worker_agents": "子 Agent",
    "route.products.form.managed": "服务端维护字段",
    "route.products.form.save": "保存 Product",
    "route.products.form.delete": "删除 Product",
    "route.products.form.cancel": "重置",
    "route.products.form.builtin_notice": "内置 Product 由服务维护，此处只读。",
    "route.products.saved": "Product 已保存。",
    "route.products.deleted": "Product 已删除。",
    "route.products.save_failed": "保存 Product 失败：{error}",
    "route.products.delete_failed": "删除 Product 失败：{error}",
    "route.products.drafts.title": "Draft Studio",
    "route.products.drafts.subtitle": "生成 Product 草稿并发布审核后的矩阵",
    "route.products.drafts.empty": "暂无 Product 草稿。",
    "route.products.drafts.generated": "Product 草稿已生成。",
    "route.products.drafts.published": "Product 草稿已发布。",
    "route.products.drafts.review_saved": "Product 草稿已保存。",
    "route.products.drafts.generate_failed": "生成 Product 草稿失败：{error}",
    "route.products.drafts.publish_failed": "发布 Product 草稿失败：{error}",
    "route.products.drafts.review_failed": "保存 Product 草稿失败：{error}",
    "route.products.drafts.form.name": "草稿名称",
    "route.products.drafts.form.goal": "目标",
    "route.products.drafts.form.target_users": "目标用户",
    "route.products.drafts.form.core": "核心能力",
    "route.products.drafts.form.constraints": "约束",
    "route.products.drafts.form.artifacts": "预期产物",
    "route.products.drafts.form.integrations": "集成要求",
    "route.products.drafts.form.mode": "模式",
    "route.products.drafts.form.generate": "生成草稿",
    "route.products.drafts.form.save": "保存草稿",
    "route.products.drafts.form.publish": "发布草稿",
    "route.products.drafts.form.editor": "草稿 JSON",
    "route.products.drafts.form.editor_hint": "发布前可直接审核并调整整份草稿 JSON。",
    "route.products.drafts.form.expand_disabled": "扩展模式需要先选中一个可编辑的 Product。",
    "route.products.drafts.form.mode.bootstrap": "新建矩阵",
    "route.products.drafts.form.mode.expand": "扩展当前 Product",
    "route.products.drafts.detail.master": "总 Agent",
    "route.products.drafts.detail.workers": "子 Agent 矩阵",
    "route.products.drafts.detail.conflicts": "冲突建议",
    "route.products.drafts.detail.review": "审核状态",
    "route.products.drafts.detail.empty": "选择一个草稿后，可在这里审核、编辑并发布。",
    "route.products.drafts.detail.generated": "生成时间",
    "route.products.drafts.detail.updated": "更新时间",
    "route.products.drafts.detail.mode": "生成模式",
    "route.products.drafts.detail.product": "Product 草稿",
    "route.products.drafts.detail.published": "已发布 Product",
    "chat.runtime.target": "会话目标",
    "chat.runtime.agent": "Agent",
    "chat.runtime.agent_pick": "选择 Agent",
    "chat.runtime.provider": "提供方",
    "chat.runtime.model": "模型",
    "chat.runtime.model_short": "模型",
    "chat.runtime.empty": "当前还没有可用的启用模型 Provider。请先在 Models 页面完成配置。",
    "chat.runtime.hint": "会作用于当前会话后续发送的消息。",
    "chat.runtime.tools_mcp": "工具 / MCP",
    "chat.runtime.tools_short": "工具",
    "chat.runtime.skills": "技能",
    "chat.runtime.skills_short": "技能",
    "chat.runtime.target_hint": "请在发送第一条消息前确定当前会话目标。",
    "chat.runtime.agent_hint": "请在发送第一条消息前为当前会话选择 Agent。",
    "chat.runtime.model_hint": "切换后会作用于当前会话后续发送的消息。",
    "chat.runtime.tools_hint": "为后续消息选择额外启用的工具与 MCP。",
    "chat.runtime.skills_hint": "为后续消息选择额外启用的技能。",
    "chat.runtime.mobile": "会话设置",
    "chat.runtime.mobile_hint": "为后续消息集中选择模型、工具与技能。",
    "chat.runtime.mobile_meta": "模型 {model} · 工具 {tools} · 技能 {skills}",
    "chat.runtime.active": "已启用",
    "chat.runtime.available": "可启用",
    "chat.runtime.category.tools": "工具",
    "chat.runtime.category.mcps": "MCP 服务",
    "chat.runtime.locked": "发送第一条消息后，会话目标不可切换。",
    "chat.runtime.none": "该分区暂无项目。",
    "route.agent.target_agents": "我的 Agents",
    "route.agent.pick": "请先在上方选择 Agent，再开始新的执行会话。",
    "route.agent.send_failed": "Agent 请求失败：{error}",
    "route.channels.title": "通道",
    "route.channels.subtitle": "管理连接通道",
    "route.channels.empty": "暂无可用通道。",
    "route.sessions.title": "会话列表",
    "route.sessions.subtitle": "查看归档会话并按来源筛选",
    "route.sessions.empty": "暂无会话记录。",
    "route.sessions.filter.trigger_type": "触发类型",
    "route.sessions.filter.channel_type": "通道类型",
    "route.sessions.filter.channel_id": "通道 ID",
    "route.sessions.filter.message_id": "消息 ID",
    "route.sessions.filter.job_id": "任务 ID",
    "route.sessions.filter.apply": "应用",
    "route.sessions.filter.reset": "重置",
    "route.sessions.open_detail": "查看详情",
    "route.copy_value": "复制内容",
    "route.tasks.title": "任务观测",
    "route.tasks.subtitle": "基于来源、状态和时间范围观测运行任务",
    "route.tasks.empty": "暂无任务记录。",
    "route.tasks.empty_hint": "试试调整筛选条件，或稍后再试。",
    "route.tasks.filter.session": "会话 ID",
    "route.tasks.filter.status": "状态",
    "route.tasks.filter.trigger_type": "触发类型",
    "route.tasks.filter.channel_type": "通道类型",
    "route.tasks.filter.channel_id": "通道 ID",
    "route.tasks.filter.message_id": "消息 ID",
    "route.tasks.filter.source_message_id": "源消息 ID",
    "route.tasks.filter.start_at": "开始时间",
    "route.tasks.filter.end_at": "结束时间",
    "route.tasks.filter.advanced": "高级筛选",
    "route.tasks.filter.advanced_show": "展开高级筛选",
    "route.tasks.filter.advanced_hide": "收起高级筛选",
    "route.tasks.filter.applying": "筛选中...",
    "route.tasks.filter.apply": "筛选",
    "route.tasks.filter.reset": "重置",
    "route.tasks.copy_task_id": "复制任务 ID",
    "route.tasks.open_detail": "查看详情",
    "route.tasks.drawer.title": "任务详情",
    "route.tasks.drawer.empty": "选择任务后展示详情。",
    "route.tasks.drawer.close": "关闭",
    "route.tasks.drawer.identifiers": "标识信息",
    "route.tasks.drawer.runtime": "运行信息",
    "route.tasks.drawer.quick_actions": "快捷操作",
    "route.tasks.drawer.quick_timeline": "生成时间轴版",
    "route.tasks.drawer.quick_interview": "生成面试速记版",
    "route.tasks.actions.retry_tip": "重试：从头重新执行该任务。",
    "route.tasks.actions.replay_tip": "回放：仅重新拉取并播放当前任务日志。",
    "route.tasks.progress": "进度",
    "route.tasks.page.label": "页码",
    "route.tasks.page.next": "下一页",
    "route.tasks.logs.title": "执行日志",
    "route.tasks.logs.empty": "暂无执行日志。",
    "route.tasks.logs.streaming": "日志实时拉取中...",
    "route.tasks.logs.done": "日志流已结束。",
    "route.tasks.logs.disconnected": "日志流已断开，可手动重连。",
    "route.tasks.logs.reconnect": "重连",
    "route.tasks.logs.replay": "回放",
    "route.tasks.terminal.title": "终端",
    "route.tasks.terminal.input_placeholder": "输入命令或追问继续交互...",
    "route.tasks.terminal.send": "发送",
    "route.tasks.terminal.sending": "发送中...",
    "route.tasks.terminal.hint": "支持在当前终端会话中继续交互。",
    "route.tasks.terminal.followup_note": "每次发送都会继续复用同一个 Codex 会话线程。",
    "route.tasks.actions.retry": "重试",
    "route.tasks.actions.cancel": "取消",
    "route.tasks.result.title": "终态输出",
    "route.terminal.title": "终端",
    "route.terminal.subtitle": "独立终端会话，状态与实际 shell 进程保持一致",
    "route.terminal.new": "新建终端会话",
    "route.terminal.new_short": "新建",
    "route.terminal.empty": "暂无终端会话。创建后即可启动独立 shell。",
    "route.terminal.pick": "选择一个终端会话开始交互。",
    "route.terminal.sessions": "会话列表",
    "route.terminal.session_count": "{count} 个会话",
    "route.terminal.hide_sessions": "收起会话",
    "route.terminal.details_show": "查看详情",
    "route.terminal.details_hide": "收起详情",
    "route.terminal.last_output_label": "最近输出",
    "route.terminal.input": "输入命令...",
    "route.terminal.send": "发送",
    "route.terminal.sending": "发送中...",
    "route.terminal.close": "关闭",
    "route.terminal.closing": "关闭中...",
    "route.terminal.delete": "删除",
    "route.terminal.deleting": "删除中...",
    "route.terminal.session": "会话",
    "route.terminal.shell": "Shell",
    "route.terminal.path": "路径",
    "route.terminal.status": "状态",
    "route.terminal.logs.heading": "终端输出（{session}）",
    "route.terminal.logs.empty": "暂无终端输出。",
    "route.terminal.busy": "Codex 正在处理当前这一轮...",
    "route.terminal.process.label": "过程",
    "route.terminal.process.steps": "{count} 步",
    "route.terminal.output_expand": "展开更多",
    "route.terminal.output_collapse": "收起",
    "route.terminal.jump_bottom": "回到底部",
    "route.terminal.jump_top": "回到顶部",
    "route.terminal.jump_prev": "上一条",
    "route.terminal.jump_next": "下一条",
    "route.terminal.navigation": "终端导航",
    "route.terminal.send_failed": "发送失败：{error}",
    "route.terminal.logs_failed": "终端输出加载失败：{error}",
    "route.terminal.close_failed": "终端关闭失败：{error}",
    "route.terminal.delete_failed": "终端删除失败：{error}",
    "route.terminal.delete_confirm": "删除当前终端会话及其工作区文件？",
    "route.terminal.loading": "正在加载终端会话...",
    "route.terminal.interrupted": "Codex 运行态已退出，继续发送即可恢复当前会话。",
    "route.terminal.closed": "Codex 会话已退出，继续发送即可在当前会话内恢复。",
    "trigger.user": "用户触发",
    "trigger.cron": "定时触发",
    "trigger.system": "系统触发",
    "channel.cli": "CLI",
    "channel.web": "Web",
    "channel.scheduler": "Scheduler",
    "route.cron.title": "定时任务",
    "route.cron.subtitle": "配置调度并追踪触发会话",
    "route.cron.empty": "暂无定时任务。",
    "route.cron.form.title": "创建 / 更新定时任务",
    "route.cron.form.job_id": "任务 ID",
    "route.cron.form.name": "任务名称",
    "route.cron.form.input": "提示词 / 输入",
    "route.cron.form.retry": "重试次数",
    "route.cron.form.timezone": "时区",
    "route.cron.form.mode": "调度模式",
    "route.cron.form.every": "每隔",
    "route.cron.form.unit": "单位",
    "route.cron.form.time": "时间",
    "route.cron.form.weekday": "星期",
    "route.cron.form.expression": "Cron 表达式",
    "route.cron.form.enabled": "启用",
    "route.cron.form.submit": "保存任务",
    "route.cron.form.reset": "重置",
    "route.cron.list.title": "已配置任务",
    "route.cron.mode.every": "周期执行",
    "route.cron.mode.daily": "每日固定时间",
    "route.cron.mode.weekly": "每周固定时间",
    "route.cron.unit.minute": "分钟",
    "route.cron.unit.hour": "小时",
    "route.cron.unit.day": "天",
    "route.cron.weekday.0": "周日",
    "route.cron.weekday.1": "周一",
    "route.cron.weekday.2": "周二",
    "route.cron.weekday.3": "周三",
    "route.cron.weekday.4": "周四",
    "route.cron.weekday.5": "周五",
    "route.cron.weekday.6": "周六",
    "route.cron.action.edit": "编辑",
    "route.cron.action.delete": "删除",
    "route.cron.action.runs": "触发记录",
    "route.cron.runs.empty": "暂无触发记录。",
    "route.cron.runs.open_sessions": "查看会话",
    "route.cron.expression_invalid": "该表达式不在可视化支持范围内。",
    "route.cron.save_failed": "保存失败：{error}",
    "route.cron.delete_failed": "删除失败：{error}",
    "route.cron.runs_failed": "加载触发记录失败：{error}",
    "route.memory.title": "记忆",
    "route.memory.subtitle": "任务摘要优先展示，支持按需下钻日志与产物",
    "route.memory.tab.long_term": "长期记忆",
    "route.memory.tab.daily": "天级记忆",
    "route.memory.tab.tasks": "任务历史",
    "route.memory.tab.mandatory": "SOUL.md",
    "route.memory.tab.specification": "说明文档",
    "route.memory.empty.long_term": "暂无长期记忆文件。",
    "route.memory.empty.daily": "暂无天级记忆文件。",
    "route.memory.empty.mandatory": "暂无 SOUL.md 文件。",
    "route.memory.empty.specification": "暂无记忆模块说明文档。",
    "route.memory.read_only": "只读",
    "route.memory.daily.source": "来源目录",
    "route.memory.daily.summary": "摘要",
    "route.memory.spec.section.default": "文档内容",
    "route.memory.tasks.empty": "暂无任务历史。",
    "route.memory.tasks.filter.status": "状态",
    "route.memory.tasks.filter.task_type": "任务类型",
    "route.memory.tasks.filter.start_at": "开始时间",
    "route.memory.tasks.filter.end_at": "结束时间",
    "route.memory.tasks.filter.apply": "筛选",
    "route.memory.tasks.filter.reset": "重置",
    "route.memory.tasks.open_detail": "查看详情",
    "route.memory.tasks.detail.title": "任务详情",
    "route.memory.tasks.detail.empty": "选择一条任务摘要查看详情。",
    "route.memory.tasks.logs.load": "加载日志",
    "route.memory.tasks.logs.more": "加载更多日志",
    "route.memory.tasks.page.label": "页码",
    "route.memory.tasks.page.next": "下一页",
    "route.memory.tasks.logs.empty": "暂无日志。",
    "route.memory.tasks.artifacts.load": "加载产物",
    "route.memory.tasks.artifacts.empty": "暂无产物。",
    "route.memory.tasks.artifacts.download": "下载",
    "route.memory.tasks.artifacts.preview": "预览",
    "route.memory.tasks.artifacts.download_fail": "下载失败：{error}",
    "route.memory.tasks.back": "返回摘要",
    "route.memory.tasks.rebuild": "重建摘要",
    "route.memory.tasks.rebuild_ok": "摘要重建完成。",
    "route.memory.tasks.rebuild_fail": "摘要重建失败：{error}",
    "route.memory.tasks.logs_hint": "日志不可用，可执行摘要重建。",
    "route.skills.title": "技能",
    "route.skills.subtitle": "技能配置",
    "route.skills.empty": "暂无可用技能。",
    "route.mcp.title": "MCP 协议",
    "route.mcp.subtitle": "Model Context Protocol 配置",
    "route.mcp.empty": "暂无 MCP 配置。",
    "route.models.title": "模型",
    "route.models.subtitle": "模型能力",
    "route.envs.title": "环境",
    "route.envs.subtitle": "环境与部署设置",
    "route.envs.save": "保存变更",
    "route.envs.restart_service": "重启服务",
    "route.envs.restarting": "服务正在重启...",
    "route.envs.restarting_sync": "正在同步远端 master 并重启服务...",
    "route.envs.restart_failed": "重启失败：{error}",
    "route.envs.restart_confirm": "现在重启服务吗？",
    "route.envs.restart_confirm_desc": "新实例探活通过后，当前页面会自动刷新并重新连接。",
    "route.envs.restart_sync_master": "重启前同步远端 master 最新改动",
    "route.envs.restart_sync_master_hint": "默认开启。要求当前本地分支为 master，且已跟踪工作区保持干净。",
    "route.envs.restart_wait_timeout": "服务重启时间超出预期，请稍后刷新后重试。",
    "route.envs.restart_success": "服务重启已完成，当前页面已连接到最新运行实例。",
    "route.envs.runtime.last_restart_at": "最近重启时间",
    "route.envs.runtime.commit_hash": "Commit Hash",
    "route.envs.refresh": "重新加载",
    "route.envs.show_sensitive": "显示敏感项",
    "route.envs.hide_sensitive": "隐藏敏感项",
    "route.envs.current_value": "配置值",
    "route.envs.default_value": "默认值",
    "route.envs.effective_value": "生效值",
    "route.envs.value_type": "值类型",
    "route.envs.apply_mode": "生效方式",
    "route.envs.source": "来源",
    "route.envs.validation": "校验规则",
    "route.envs.details": "更多信息",
    "route.envs.pending_restart": "待重启生效",
    "route.envs.hot_reload": "热更新",
    "route.envs.no_changes": "没有配置变更。",
    "route.envs.saved": "环境配置已保存。",
    "route.envs.save_failed": "保存失败：{error}",
    "route.envs.loading": "正在加载环境配置...",
    "route.envs.audit.title": "变更审计",
    "route.envs.audit.empty": "暂无环境配置审计。",
    "route.envs.audit.operator": "操作人",
    "route.envs.audit.at": "变更时间",
    "route.envs.audit.requires_restart": "需要重启",
    "route.envs.audit.change": "{key}: {old} → {new}（{mode}）",
    "route.envs.restart_notice": "以下配置需重启后生效：{keys}",
    "route.envs.apply.immediate": "即时生效",
    "route.envs.apply.restart": "重启生效",
    "route.envs.source.default": "默认值",
    "route.envs.source.runtime": "运行时",
    "route.envs.source.persisted": "持久化",
    "route.envs.type.integer": "整数",
    "route.envs.type.duration": "时长，例如 5s / 2m / 1h",
    "route.envs.type.string": "文本",
    "route.envs.type.enum": "枚举选项",
    "route.envs.type.unknown": "未知",
    "route.envs.validation.none": "无约束",
    "route.envs.hidden": "隐藏值",
    "route.connected": "页面已连接",
    "route.connected_desc": "该页面路由已激活。内容可按模块扩展。",
    "loading": "加载中...",
    "load_failed": "加载失败：{error}"
  }
};

const ROUTES = {
  chat: {
    key: "chat",
    mode: "chat",
    conversation: "agent",
    defaultTarget: {
      type: "agent",
      id: MAIN_AGENT_ID,
      name: MAIN_AGENT_NAME
    }
  },
  "agent-runtime": {
    key: "agent_runtime",
    mode: "chat",
    conversation: "agent",
    targetPicker: true
  },
  agent: {
    key: "agent",
    mode: "page",
    loader: loadAgentView
  },
  products: {
    key: "products",
    mode: "page",
    loader: loadProductsView
  },
  channels: {
    key: "channels",
    mode: "page",
    loader: loadChannelsView
  },
  sessions: {
    key: "sessions",
    mode: "page",
    loader: loadSessionsView
  },
  tasks: {
    key: "tasks",
    mode: "page",
    loader: loadControlTasksView
  },
  terminal: {
    key: "terminal",
    mode: "page",
    loader: loadTerminalView
  },
  "cron-jobs": {
    key: "cron",
    mode: "page",
    loader: loadCronJobsView
  },
  memory: {
    key: "memory",
    mode: "page",
    loader: loadMemoryView
  },
  skills: {
    key: "skills",
    mode: "page",
    loader: loadSkillsView
  },
  mcp: {
    key: "mcp",
    mode: "page",
    loader: loadMCPView
  },
  models: {
    key: "models",
    mode: "page",
    loader: loadModelsView
  },
  environments: {
    key: "envs",
    mode: "page",
    loader: loadEnvironmentsView
  }
};

const state = {
  currentRoute: DEFAULT_ROUTE,
  activeSessionByBucket: {},
  sessions: [],
  sessionLoadError: "",
  chatCatalog: {
    agents: [],
    providers: [],
    skills: [],
    mcps: [],
    loading: false,
    loaded: false,
    error: "",
    providerLoading: false,
    providerLoaded: false,
    providerError: "",
    capabilityLoading: false,
    capabilityLoaded: false,
    capabilityError: ""
  },
  chatRuntime: {
    openPopover: ""
  },
  agentRuntimeState: {
    selectedAgentID: ""
  },
  agentRouteState: {
    selectedAgentID: "",
    activeSessionByAgent: {}
  },
  productRouteState: {
    selectedProductID: "",
    selectedDraftID: "",
    activePanel: "workspace",
    selectedSpaceID: "",
    workspaceSessionByProduct: {}
  },
  sessionRouteFilters: {
    triggerType: "",
    channelType: "",
    channelID: "",
    messageID: "",
    jobID: ""
  },
  sessionHistoryCollapsed: false,
  mobileViewport: {
    baselineHeight: 0,
    width: 0,
    syncFrame: 0,
    alignFocusedInput: false,
    height: 0,
    keyboardOffset: 0,
    lastAlignedAt: 0
  },
  pending: false,
  pendingCount: 0,
  pageRenderToken: 0,
  messageRenderFrame: 0,
  pendingMessageRenderPreserveScroll: false,
  messageRenderSignatures: new Map(),
  navCollapsed: false,
  suppressHashRouteConfirm: "",
  lang: "en" // default
};

let navTooltipNode = null;
let navTooltipTarget = null;
let navTooltipShowTimer = 0;
let navTooltipHideTimer = 0;
let chatTaskPollTimer = 0;
let chatTaskPollPending = false;

function isDocumentVisible() {
  return typeof document === "undefined" || document.visibilityState !== "hidden";
}

function t(key, params = {}) {
  const dict = I18N[state.lang] || I18N.en;
  let val = dict[key] || I18N.en[key] || key;
  for (const [k, v] of Object.entries(params)) {
    val = val.replace(`{${k}}`, v);
  }
  return val;
}

function navCollapseLabel() {
  return state.navCollapsed ? t("nav.expand") : t("nav.collapse");
}

function sessionHistoryToggleLabel() {
  return state.sessionHistoryCollapsed ? t("session.history.expand") : t("session.history.collapse");
}

function syncSessionHistoryPanel() {
  if (!sessionHistoryPanel || !sessionHistoryToggle) {
    return;
  }
  const collapsed = state.sessionHistoryCollapsed;
  if (sessionPane) {
    sessionPane.classList.toggle("history-collapsed", collapsed);
  }
  sessionHistoryPanel.hidden = collapsed;
  sessionHistoryToggle.dataset.collapsedState = collapsed ? "collapsed" : "expanded";
  sessionHistoryToggle.setAttribute("aria-expanded", collapsed ? "false" : "true");
  const label = sessionHistoryToggleLabel();
  sessionHistoryToggle.textContent = label;
  sessionHistoryToggle.setAttribute("aria-label", label);
}

function setSessionHistoryCollapsed(collapsed) {
  state.sessionHistoryCollapsed = collapsed;
  syncSessionHistoryPanel();
}

function syncMenuItemTooltips() {
  for (const node of menuRouteItems) {
    const label = node.querySelector(".menu-label");
    const text = label && label.textContent ? label.textContent.trim() : "";
    if (!text) {
      node.removeAttribute("data-tooltip");
      node.removeAttribute("title");
      node.removeAttribute("aria-label");
      continue;
    }
    node.setAttribute("data-tooltip", text);
    node.removeAttribute("title");
    node.setAttribute("aria-label", text);
  }

  navCollapseButton.setAttribute("data-tooltip", navCollapseLabel());
  navCollapseButton.removeAttribute("title");

  if (!navTooltipTarget) {
    return;
  }
  if (!shouldShowNavTooltipFor(navTooltipTarget)) {
    hideNavTooltip(true);
    return;
  }
  const text = tooltipTextForNode(navTooltipTarget);
  if (!text) {
    hideNavTooltip(true);
    return;
  }
  const tooltip = ensureNavTooltipNode();
  tooltip.textContent = text;
  positionNavTooltip(navTooltipTarget);
}

function ensureNavTooltipNode() {
  if (navTooltipNode) {
    return navTooltipNode;
  }
  const node = document.createElement("div");
  node.className = "nav-tooltip";
  node.setAttribute("role", "tooltip");
  node.setAttribute("aria-hidden", "true");
  document.body.appendChild(node);
  navTooltipNode = node;
  return node;
}

function tooltipTextForNode(node) {
  if (!node) {
    return "";
  }
  if (node === navCollapseButton) {
    return navCollapseLabel();
  }
  const label = node.querySelector(".menu-label");
  return label && label.textContent ? label.textContent.trim() : "";
}

function shouldShowNavTooltipFor(node) {
  if (!node || isMobileViewport()) {
    return false;
  }
  if (node === navCollapseButton) {
    return true;
  }
  return state.navCollapsed;
}

function positionNavTooltip(target) {
  if (!navTooltipNode || !target) {
    return;
  }
  const rect = target.getBoundingClientRect();
  const viewportMargin = 8;
  const top = Math.min(
    Math.max(rect.top + (rect.height / 2), viewportMargin),
    window.innerHeight - viewportMargin
  );
  const maxLeft = Math.max(viewportMargin, window.innerWidth - navTooltipNode.offsetWidth - viewportMargin);
  const left = Math.min(Math.max(rect.right + NAV_TOOLTIP_OFFSET, viewportMargin), maxLeft);
  navTooltipNode.style.top = `${top}px`;
  navTooltipNode.style.left = `${left}px`;
}

function showNavTooltip(target) {
  if (!shouldShowNavTooltipFor(target)) {
    hideNavTooltip(true);
    return;
  }
  const text = tooltipTextForNode(target);
  if (!text) {
    hideNavTooltip(true);
    return;
  }
  const tooltip = ensureNavTooltipNode();
  tooltip.textContent = text;
  tooltip.classList.add("visible");
  tooltip.setAttribute("aria-hidden", "false");
  navTooltipTarget = target;
  positionNavTooltip(target);
}

function queueNavTooltip(target, immediate = false) {
  if (!target || !shouldShowNavTooltipFor(target)) {
    hideNavTooltip(true);
    return;
  }
  if (navTooltipHideTimer) {
    window.clearTimeout(navTooltipHideTimer);
    navTooltipHideTimer = 0;
  }
  if (immediate) {
    if (navTooltipShowTimer) {
      window.clearTimeout(navTooltipShowTimer);
      navTooltipShowTimer = 0;
    }
    showNavTooltip(target);
    return;
  }
  if (navTooltipShowTimer) {
    window.clearTimeout(navTooltipShowTimer);
  }
  navTooltipShowTimer = window.setTimeout(() => {
    navTooltipShowTimer = 0;
    showNavTooltip(target);
  }, NAV_TOOLTIP_SHOW_DELAY);
}

function hideNavTooltip(immediate = false) {
  if (navTooltipShowTimer) {
    window.clearTimeout(navTooltipShowTimer);
    navTooltipShowTimer = 0;
  }
  if (!navTooltipNode) {
    navTooltipTarget = null;
    return;
  }
  const close = () => {
    if (!navTooltipNode) {
      return;
    }
    navTooltipNode.classList.remove("visible");
    navTooltipNode.setAttribute("aria-hidden", "true");
    navTooltipTarget = null;
    navTooltipHideTimer = 0;
  };
  if (immediate) {
    if (navTooltipHideTimer) {
      window.clearTimeout(navTooltipHideTimer);
      navTooltipHideTimer = 0;
    }
    close();
    return;
  }
  if (navTooltipHideTimer) {
    window.clearTimeout(navTooltipHideTimer);
  }
  navTooltipHideTimer = window.setTimeout(close, NAV_TOOLTIP_HIDE_DELAY);
}

function bindNavTooltipEvents() {
  for (const node of navTooltipTargets) {
    if (!node) {
      continue;
    }
    node.addEventListener("mouseenter", () => {
      queueNavTooltip(node);
    });
    node.addEventListener("mouseleave", () => {
      hideNavTooltip();
    });
    node.addEventListener("focus", () => {
      queueNavTooltip(node, true);
    });
    node.addEventListener("blur", () => {
      hideNavTooltip(true);
    });
    node.addEventListener("pointerdown", () => {
      hideNavTooltip(true);
    });
  }
  window.addEventListener("scroll", () => {
    if (!navTooltipTarget) {
      return;
    }
    if (!shouldShowNavTooltipFor(navTooltipTarget)) {
      hideNavTooltip(true);
      return;
    }
    positionNavTooltip(navTooltipTarget);
  }, true);
}

function setLanguage(lang) {
  if (!I18N[lang]) return;
  state.lang = lang;
  document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
  
  // Update static elements
  document.querySelectorAll("[data-i18n]").forEach(el => {
    const key = el.getAttribute("data-i18n");
    if (key) {
      if (el.tagName === "INPUT" || el.tagName === "TEXTAREA") {
        el.placeholder = t(key);
      } else {
        // preserve child elements if any (like icons) - wait, icons are usually separate.
        // If element has only text, textContent is fine.
        // If element has icon + text, we need to be careful.
        // For menu items, the text is in .menu-label
        el.textContent = t(key);
      }
    }
  });

  // Update dynamic views
  renderSessions();
  syncHeader();
  syncWelcomeCopy();
  renderChatRuntimePanel();
  renderWelcomeTargetPicker();
  
  // Re-render current route if it's a page
  if ((ROUTES[state.currentRoute] || ROUTES.chat).mode !== "chat") {
    renderRoute(state.currentRoute);
  }
  
  // Update button text
  const localeBtn = document.querySelector(".locale");
  if (localeBtn) {
    localeBtn.textContent = state.lang === "en" ? "English" : "中文";
    localeBtn.setAttribute("data-short-lang", state.lang === "en" ? "EN" : "中");
  }
  navCollapseButton.setAttribute("aria-label", navCollapseLabel());
  syncMenuItemTooltips();
  syncSessionHistoryPanel();
}

function toggleLanguage() {
  const next = state.lang === "en" ? "zh" : "en";
  setLanguage(next);
}

function makeID() {
  if (window.crypto && typeof window.crypto.randomUUID === "function") {
    return window.crypto.randomUUID();
  }
  return `session-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

function routeConversationMode(route = state.currentRoute) {
  const config = ROUTES[route] || ROUTES.chat;
  return config.conversation === "agent" ? "agent" : "chat";
}

function isAgentConversationRoute(route = state.currentRoute) {
  return routeConversationMode(route) === "agent";
}

function routeDefaultTarget(route = state.currentRoute) {
  const config = ROUTES[route] || ROUTES.chat;
  if (config.defaultTarget) {
    return normalizeChatTarget(config.defaultTarget);
  }
  return config.conversation === "agent" ? defaultAgentRuntimeTarget() : defaultChatTarget();
}

function routeAllowsTargetPicker(route = state.currentRoute) {
  const config = ROUTES[route] || ROUTES.chat;
  return Boolean(config.targetPicker);
}

function storedConversationSessions() {
  return Array.isArray(state.sessions) ? state.sessions : [];
}

function setStoredConversationSessions(items) {
  state.sessions = Array.isArray(items) ? items : [];
  sortSessionsByCreatedAtDesc(state.sessions);
}

function conversationSessions(route = state.currentRoute) {
  const bucket = conversationHistoryBucket(route);
  return storedConversationSessions().filter((item) => sessionHistoryBucket(item) === bucket);
}

function setConversationSessions(items, route = state.currentRoute) {
  const bucket = conversationHistoryBucket(route);
  const preserved = storedConversationSessions().filter((item) => sessionHistoryBucket(item) !== bucket);
  const nextItems = (Array.isArray(items) ? items : []).map((item) => ({
    ...item,
    historyBucket: bucket
  }));
  setStoredConversationSessions([...preserved, ...nextItems]);
}

function activeConversationSessionID(route = state.currentRoute) {
  const bucket = conversationHistoryBucket(route);
  return normalizeText(state.activeSessionByBucket?.[bucket]);
}

function setActiveConversationSessionID(sessionID, route = state.currentRoute) {
  const bucket = conversationHistoryBucket(route);
  const normalized = normalizeText(sessionID);
  if (!bucket) {
    return;
  }
  if (!normalized) {
    delete state.activeSessionByBucket[bucket];
    return;
  }
  state.activeSessionByBucket[bucket] = normalized;
}

function conversationSessionLoadError() {
  return state.sessionLoadError;
}

function setConversationSessionLoadError(message) {
  state.sessionLoadError = normalizeText(message);
}

function timeLabel(epochMillis = Date.now()) {
  return new Date(epochMillis).toLocaleTimeString(state.lang === "zh" ? "zh-CN" : "en-US", {
    hour: "2-digit",
    minute: "2-digit"
  });
}

function shorten(text, maxLength) {
  if (text.length <= maxLength) {
    return text;
  }
  return `${text.slice(0, maxLength - 1)}…`;
}

const AUTO_SESSION_TITLE_STABLE_SCORE = 5;
const SESSION_TITLE_PLACEHOLDERS = new Set(["New Chat", "新对话", "New Agent Run", "新 Agent 会话"]);
const AUTO_SESSION_TITLE_POLITE_PREFIXES = [
  "请帮我",
  "麻烦帮我",
  "麻烦先帮我",
  "先帮我",
  "帮我先",
  "帮我",
  "麻烦先",
  "麻烦",
  "请先",
  "请",
  "please",
  "can you",
  "could you",
  "help me"
];
const AUTO_SESSION_TITLE_BOOTSTRAP_VERBS = [
  "拉取",
  "同步",
  "clone",
  "pull",
  "checkout",
  "fetch",
  "查看",
  "看下",
  "看看",
  "分析",
  "熟悉",
  "inspect",
  "analyze",
  "analyse",
  "read",
  "open",
  "explore"
];
const AUTO_SESSION_TITLE_REPO_WORDS = [
  "仓库",
  "代码库",
  "repo",
  "repository",
  "codebase",
  "branch"
];
const AUTO_SESSION_TITLE_CONNECTORS = [
  "然后",
  "之后",
  "后再",
  "再",
  "then",
  "and then",
  "after that"
];
const AUTO_SESSION_TITLE_ACTION_WORDS = [
  "修复",
  "修改",
  "实现",
  "新增",
  "添加",
  "删除",
  "优化",
  "重构",
  "调整",
  "排查",
  "补齐",
  "支持",
  "更新",
  "rename",
  "fix",
  "change",
  "update",
  "implement",
  "add",
  "remove",
  "optimize",
  "optimise",
  "refactor",
  "debug",
  "support"
];
const AUTO_SESSION_TITLE_TECHNICAL_WORDS = [
  "terminal",
  "agent",
  "chat",
  "session",
  "title",
  "name",
  "logic",
  "api",
  "route",
  "ui",
  "ux",
  "test",
  "readme",
  "docs",
  "会话",
  "标题",
  "命名",
  "逻辑",
  "接口",
  "路由",
  "页面",
  "前端",
  "后端",
  "文档",
  "测试",
  "工作区"
];
const AUTO_SESSION_TITLE_FOLLOW_UPS = ["继续", "继续处理", "接着", "然后呢", "next", "continue"];

function normalizeSessionTitleValue(value) {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim().split(/\s+/u).filter(Boolean).join(" ");
}

function containsAutoSessionTitleWord(value, words) {
  return words.some((item) => value.includes(item));
}

function stripAutoSessionTitlePrefix(value) {
  let trimmed = normalizeSessionTitleValue(value);
  let lower = trimmed.toLowerCase();
  let updated = true;
  while (updated) {
    updated = false;
    AUTO_SESSION_TITLE_POLITE_PREFIXES.forEach((prefix) => {
      if (!lower.startsWith(prefix)) {
        return;
      }
      trimmed = normalizeSessionTitleValue(trimmed.slice(prefix.length));
      lower = trimmed.toLowerCase();
      updated = true;
    });
  }
  return trimmed.replace(/^[- ]+/u, "");
}

function isBootstrapSessionTitle(value) {
  const lower = stripAutoSessionTitlePrefix(value).toLowerCase();
  if (!lower) {
    return false;
  }
  return containsAutoSessionTitleWord(lower, AUTO_SESSION_TITLE_BOOTSTRAP_VERBS)
    && containsAutoSessionTitleWord(lower, AUTO_SESSION_TITLE_REPO_WORDS);
}

function isFollowUpSessionTitle(value) {
  return AUTO_SESSION_TITLE_FOLLOW_UPS.includes(normalizeSessionTitleValue(value).toLowerCase());
}

function splitAutoSessionTitleSegments(value) {
  return normalizeSessionTitleValue(value)
    .replace(/[，,。.;；:\r]+/gu, "\n")
    .split("\n")
    .map((item) => normalizeSessionTitleValue(item))
    .filter(Boolean);
}

function extractBootstrapSessionTitleRemainder(value) {
  const trimmed = stripAutoSessionTitlePrefix(value);
  if (!isBootstrapSessionTitle(trimmed)) {
    return "";
  }
  const lower = trimmed.toLowerCase();
  for (const connector of AUTO_SESSION_TITLE_CONNECTORS) {
    const index = lower.indexOf(connector);
    if (index < 0) {
      continue;
    }
    const remainder = stripAutoSessionTitlePrefix(trimmed.slice(index + connector.length));
    if (remainder) {
      return remainder;
    }
  }
  return "";
}

function trimBootstrapSessionTitlePrefix(value) {
  const normalized = normalizeSessionTitleValue(value);
  if (!normalized) {
    return "";
  }
  const segments = splitAutoSessionTitleSegments(normalized);
  if (!segments.length) {
    return normalized;
  }
  const first = stripAutoSessionTitlePrefix(segments[0]);
  const remainder = extractBootstrapSessionTitleRemainder(first);
  if (remainder) {
    return normalizeSessionTitleValue([remainder, ...segments.slice(1)].join(" "));
  }
  if (isBootstrapSessionTitle(first) && segments.length > 1) {
    return normalizeSessionTitleValue(segments.slice(1).join(" "));
  }
  return normalized;
}

function scoreAutoSessionTitle(value) {
  const normalized = normalizeSessionTitleValue(value);
  if (!normalized) {
    return 0;
  }
  let score = 1;
  const length = [...normalized].length;
  if (length >= 10) {
    score += 2;
  } else if (length >= 6) {
    score += 1;
  }
  const lower = normalized.toLowerCase();
  if (containsAutoSessionTitleWord(lower, AUTO_SESSION_TITLE_ACTION_WORDS)) {
    score += 2;
  }
  if (containsAutoSessionTitleWord(lower, AUTO_SESSION_TITLE_TECHNICAL_WORDS)) {
    score += 2;
  }
  if (/[\/_.#]/u.test(normalized)) {
    score += 1;
  }
  if (isBootstrapSessionTitle(normalized)) {
    score -= 4;
  }
  if (isFollowUpSessionTitle(normalized)) {
    score -= 3;
  }
  if (length <= 4) {
    score -= 2;
  }
  return Math.max(score, 0);
}

function isDefaultSessionTitle(value) {
  return SESSION_TITLE_PLACEHOLDERS.has(normalizeSessionTitleValue(value));
}

function inferStoredSessionTitleState(title, fallbackTitle = "") {
  const normalizedTitle = normalizeSessionTitleValue(title);
  const normalizedFallback = normalizeSessionTitleValue(fallbackTitle);
  if (!normalizedTitle || isDefaultSessionTitle(normalizedTitle) || (normalizedFallback && normalizedTitle.toLowerCase() === normalizedFallback.toLowerCase())) {
    return { titleAuto: true, titleScore: 0 };
  }
  const trimmed = trimBootstrapSessionTitlePrefix(normalizedTitle);
  const titleScore = scoreAutoSessionTitle(trimmed);
  if (titleScore < AUTO_SESSION_TITLE_STABLE_SCORE
    && (trimmed !== normalizedTitle || isBootstrapSessionTitle(normalizedTitle) || isFollowUpSessionTitle(normalizedTitle))) {
    return { titleAuto: true, titleScore };
  }
  return { titleAuto: false, titleScore };
}

function buildAutoSessionTitle(value, maxLength) {
  const normalized = normalizeSessionTitleValue(value);
  if (!normalized) {
    return { title: "", titleAuto: true, titleScore: 0 };
  }
  const trimmed = trimBootstrapSessionTitlePrefix(normalized);
  const titleScore = scoreAutoSessionTitle(trimmed);
  const candidate = titleScore > 0 ? trimmed : normalized;
  return {
    title: shorten(candidate, maxLength),
    titleAuto: titleScore < AUTO_SESSION_TITLE_STABLE_SCORE,
    titleScore,
  };
}

function getSession(id = activeConversationSessionID(), route = state.currentRoute) {
  const normalizedID = normalizeText(id);
  return conversationSessions(route).find((item) => item.id === normalizedID) || null;
}

function defaultChatTarget() {
  return {
    type: "model",
    id: "raw-model",
    name: t("session.target.raw")
  };
}

function defaultAgentRuntimeTarget() {
  return resolveAgentRuntimeTarget();
}

function fixedAgentRouteTarget(route = state.currentRoute) {
  const config = ROUTES[route] || ROUTES.chat;
  if (config.conversation !== "agent" || config.targetPicker) {
    return null;
  }
  return normalizeChatTarget(config.defaultTarget || routeDefaultTarget(route));
}

function currentConversationTarget(route = state.currentRoute) {
  const fixedTarget = fixedAgentRouteTarget(route);
  if (fixedTarget) {
    return fixedTarget;
  }
  const config = ROUTES[route] || ROUTES.chat;
  if (config.conversation === "agent" && config.targetPicker) {
    return syncAgentRuntimeTarget();
  }
  return routeDefaultTarget(route);
}

function conversationHistoryBucketForTarget(target = {}) {
  const normalizedTarget = normalizeChatTarget(target);
  if (normalizedTarget.type === "agent") {
    const agentID = normalizeText(normalizedTarget.id);
    return agentID ? `agent:${agentID}` : "agent:unassigned";
  }
  if (normalizeText(normalizedTarget.id) === "raw-model") {
    return `agent:${MAIN_AGENT_ID}`;
  }
  return `model:${normalizeText(normalizedTarget.id) || "default"}`;
}

function conversationHistoryBucket(route = state.currentRoute) {
  return conversationHistoryBucketForTarget(currentConversationTarget(route));
}

function sessionHistoryBucket(session) {
  return normalizeText(session?.historyBucket) || conversationHistoryBucketForTarget({
    type: session?.targetType,
    id: session?.targetID,
    name: session?.targetName
  });
}

function ensureActiveConversationSession(route = state.currentRoute) {
  const sessions = conversationSessions(route);
  const activeID = activeConversationSessionID(route);
  const active = sessions.find((item) => item.id === activeID) || null;
  if (active) {
    return active;
  }
  if (!sessions.length) {
    setActiveConversationSessionID("", route);
    return null;
  }
  setActiveConversationSessionID(sessions[0].id, route);
  return sessions[0];
}

function enabledModelsForProvider(provider) {
  const models = Array.isArray(provider?.models) ? provider.models : [];
  return models.filter((item) => item && item.is_enabled !== false);
}

function enabledChatProviders() {
  const providers = Array.isArray(state.chatCatalog.providers) ? state.chatCatalog.providers : [];
  return providers.filter((provider) => Boolean(provider?.is_enabled) && enabledModelsForProvider(provider).length > 0);
}

function findChatProvider(providerID) {
  const normalizedID = normalizeText(providerID);
  if (!normalizedID) {
    return null;
  }
  return enabledChatProviders().find((provider) => normalizeText(provider?.id) === normalizedID) || null;
}

function findChatModel(providerID, modelID) {
  const provider = findChatProvider(providerID);
  if (!provider) {
    return null;
  }
  const normalizedModelID = normalizeText(modelID);
  return enabledModelsForProvider(provider).find((item) => normalizeText(item?.id) === normalizedModelID) || null;
}

function defaultChatModelSelection() {
  const providers = enabledChatProviders();
  const provider = providers.find((item) => Boolean(item?.is_default)) || providers[0] || null;
  if (!provider) {
    return {
      providerID: "",
      modelID: ""
    };
  }
  const models = enabledModelsForProvider(provider);
  const providerDefaultModel = normalizeText(provider?.default_model);
  return {
    providerID: normalizeText(provider.id),
    modelID: models.some((item) => normalizeText(item?.id) === providerDefaultModel)
      ? providerDefaultModel
      : normalizeText(models[0]?.id)
  };
}

function normalizeChatModelSelection(selection = {}) {
  return {
    providerID: normalizeText(selection?.providerID || selection?.provider_id || ""),
    modelID: normalizeText(selection?.modelID || selection?.model_id || "")
  };
}

function sessionModelSelection(session) {
  if (!session || typeof session !== "object") {
    return defaultChatModelSelection();
  }
  return normalizeChatModelSelection({
    providerID: session.modelProviderID,
    modelID: session.modelID
  });
}

function resolveEffectiveChatModelSelection(session) {
  const current = sessionModelSelection(session);
  const fallback = defaultChatModelSelection();
  let providerID = current.providerID;
  if (!findChatProvider(providerID)) {
    providerID = fallback.providerID;
  }
  const provider = findChatProvider(providerID);
  const models = enabledModelsForProvider(provider);
  const providerDefaultModel = normalizeText(provider?.default_model);
  let modelID = current.modelID;
  if (!models.some((item) => normalizeText(item?.id) === modelID)) {
    modelID = models.some((item) => normalizeText(item?.id) === providerDefaultModel)
      ? providerDefaultModel
      : normalizeText(models[0]?.id);
  }
  return {
    providerID,
    modelID
  };
}

function updateSessionModelSelection(session, selection) {
  if (!session) {
    return;
  }
  const normalized = normalizeChatModelSelection(selection);
  session.modelProviderID = normalized.providerID;
  session.modelID = normalized.modelID;
}

function syncSessionModelSelection(session) {
  if (!session) {
    return false;
  }
  const resolved = resolveEffectiveChatModelSelection(session);
  if (session.modelProviderID === resolved.providerID && session.modelID === resolved.modelID) {
    return false;
  }
  updateSessionModelSelection(session, resolved);
  return true;
}

function reconcileChatModelSelections() {
  let changed = false;
  conversationSessions().forEach((session) => {
    if (syncSessionModelSelection(session)) {
      changed = true;
    }
  });
  if (changed) {
    persistSessions();
    renderSessions();
    syncHeader();
  }
  return changed;
}

function normalizeSelectionIDs(values) {
  return Array.from(new Set((Array.isArray(values) ? values : []).map((item) => normalizeText(item)).filter(Boolean)));
}

function findChatAgent(agentID) {
  const normalizedID = normalizeText(agentID);
  if (!normalizedID) {
    return null;
  }
  return (Array.isArray(state.chatCatalog.agents) ? state.chatCatalog.agents : []).find((item) => normalizeText(item?.id) === normalizedID) || null;
}

function agentConversationRoute(agent = {}) {
  const route = normalizeText(agent?.ui_route || agent?.uiRoute);
  if (!route || route === "agent-runtime") {
    return "";
  }
  const config = ROUTES[route];
  if (!config || config.mode !== "chat") {
    return "";
  }
  return route;
}

function genericRuntimeConversationAgents() {
  return (Array.isArray(state.chatCatalog.agents) ? state.chatCatalog.agents : []).filter((item) => {
    const agentID = normalizeText(item?.id);
    return agentID && !agentConversationRoute(item);
  });
}

function resolveAgentRuntimeTarget(target = {}) {
  const requestedID = normalizeText(target?.id || state.agentRuntimeState.selectedAgentID);
  const agents = genericRuntimeConversationAgents();
  const selected = agents.find((item) => normalizeText(item?.id) === requestedID) || agents[0] || null;
  if (!selected) {
    return {
      type: "agent",
      id: "",
      name: t("session.target.agent")
    };
  }
  return normalizeChatTarget({
    type: "agent",
    id: selected.id,
    name: selected.name || selected.id
  });
}

function syncAgentRuntimeTarget(target = {}) {
  const resolved = resolveAgentRuntimeTarget(target);
  state.agentRuntimeState.selectedAgentID = resolved.id;
  return resolved;
}

function reconcileAgentRuntimeTarget() {
  const previousID = normalizeText(state.agentRuntimeState.selectedAgentID);
  const resolved = syncAgentRuntimeTarget();
  return previousID !== resolved.id;
}

function defaultRuntimeSelectionsForTarget(target) {
  const normalizedTarget = normalizeChatTarget(target);
  if (normalizedTarget.type !== "agent") {
    return {
      toolIDs: [],
      skillIDs: [],
      mcpIDs: []
    };
  }
  const agent = findChatAgent(normalizedTarget.id);
  return {
    toolIDs: normalizeSelectionIDs(agent?.tools),
    skillIDs: normalizeSelectionIDs(agent?.skills),
    mcpIDs: normalizeSelectionIDs(agent?.mcps)
  };
}

function sessionRuntimeSelections(session) {
  const defaults = defaultRuntimeSelectionsForTarget(sessionTarget(session));
  if (!session || typeof session !== "object") {
    return defaults;
  }
  return {
    toolIDs: normalizeSelectionIDs("toolIDs" in session ? session.toolIDs : defaults.toolIDs),
    skillIDs: normalizeSelectionIDs("skillIDs" in session ? session.skillIDs : defaults.skillIDs),
    mcpIDs: normalizeSelectionIDs("mcpIDs" in session ? session.mcpIDs : defaults.mcpIDs)
  };
}

function updateSessionRuntimeSelections(session, selection = {}) {
  if (!session) {
    return;
  }
  const current = sessionRuntimeSelections(session);
  session.toolIDs = normalizeSelectionIDs("toolIDs" in selection ? selection.toolIDs : current.toolIDs);
  session.skillIDs = normalizeSelectionIDs("skillIDs" in selection ? selection.skillIDs : current.skillIDs);
  session.mcpIDs = normalizeSelectionIDs("mcpIDs" in selection ? selection.mcpIDs : current.mcpIDs);
}

function runtimeOptionCount(session, type) {
  const selection = sessionRuntimeSelections(session);
  if (type === "skills") {
    return selection.skillIDs.length;
  }
  return selection.toolIDs.length + selection.mcpIDs.length;
}

function targetLocked(session) {
  return Boolean(session && Array.isArray(session.messages) && session.messages.length > 0);
}

function normalizeChatTarget(target = {}) {
  const normalizedType = String(target?.type || "").trim().toLowerCase() === "agent" ? "agent" : "model";
  const normalizedID = String(target?.id || "").trim() || (normalizedType === "agent" ? "" : "raw-model");
  const fallbackName = normalizedType === "agent" ? normalizedID : t("session.target.raw");
  const normalizedName = String(target?.name || "").trim() || fallbackName;
  return {
    type: normalizedType,
    id: normalizedID,
    name: normalizedName
  };
}

function resolveSessionTargetForRoute(session, route = state.currentRoute) {
  const fixedTarget = fixedAgentRouteTarget(route);
  if (fixedTarget) {
    return fixedTarget;
  }
  if (!session || typeof session !== "object") {
    return currentConversationTarget(route);
  }
  const target = normalizeChatTarget({
    type: session?.targetType,
    id: session?.targetID,
    name: session?.targetName
  });
  if (routeConversationMode(route) !== "agent") {
    return target;
  }
  if (routeAllowsTargetPicker(route)) {
    const agent = findChatAgent(target.id);
    if (target.type === "agent" && target.id && agent && !agentConversationRoute(agent)) {
      return target;
    }
    return currentConversationTarget(route);
  }
  if (target.type !== "agent" || !target.id) {
    return currentConversationTarget(route);
  }
  return target;
}

function sessionTarget(session, route = state.currentRoute) {
  if (!session || typeof session !== "object") {
    return routeDefaultTarget(route);
  }
  return resolveSessionTargetForRoute(session, route);
}

function sessionTargetLabel(session) {
  const target = sessionTarget(session);
  if (target.type === "agent") {
    return target.name || t("session.target.agent");
  }
  return t("session.target.raw");
}

function sessionTargetBadgeLabel(session) {
  const target = sessionTarget(session);
  if (target.type === "agent") {
    return `${t("session.target.agent")} · ${target.name}`;
  }
  return t("session.target.raw");
}

function syncSessionTargetForRoute(session, route = state.currentRoute) {
  if (!session || typeof session !== "object") {
    return false;
  }
  const resolved = resolveSessionTargetForRoute(session, route);
  if (session.targetType === resolved.type && session.targetID === resolved.id && session.targetName === resolved.name) {
    return false;
  }
  updateSessionTarget(session, resolved);
  return true;
}

function sessionModelLabel(session) {
  const selection = resolveEffectiveChatModelSelection(session);
  const provider = findChatProvider(selection.providerID);
  const model = findChatModel(selection.providerID, selection.modelID);
  if (!provider || !model) {
    return t("session.model.default");
  }
  return `${provider.name || provider.id} / ${model.name || model.id}`;
}

function updateSessionTarget(session, target) {
  if (!session) {
    return;
  }
  const normalizedTarget = normalizeChatTarget(target);
  session.targetType = normalizedTarget.type;
  session.targetID = normalizedTarget.id;
  session.targetName = normalizedTarget.name;
  const defaults = defaultRuntimeSelectionsForTarget(normalizedTarget);
  updateSessionRuntimeSelections(session, defaults);
}

function sortSessionsByCreatedAtDesc(items) {
  items.sort((left, right) => right.createdAt - left.createdAt);
}

function isBlankSession(item) {
  return Boolean(item) && Array.isArray(item.messages) && item.messages.length === 0;
}

function getLatestBlankSession(route = state.currentRoute) {
  const blankSessions = conversationSessions(route).filter((item) => isBlankSession(item));
  if (!blankSessions.length) {
    return null;
  }
  sortSessionsByCreatedAtDesc(blankSessions);
  return blankSessions[0];
}

function enforceSingleBlankSession(route = state.currentRoute) {
  const latestBlank = getLatestBlankSession(route);
  if (!latestBlank) {
    return false;
  }
  const sessions = conversationSessions(route);
  const originalCount = sessions.length;
  setConversationSessions(sessions.filter((item) => !isBlankSession(item) || item.id === latestBlank.id), route);
  const activeID = activeConversationSessionID(route);
  if (activeID && !getSession(activeID, route)) {
    setActiveConversationSessionID(latestBlank.id, route);
  }
  return conversationSessions(route).length !== originalCount;
}

function focusSession(sessionID) {
  if (!getSession(sessionID)) {
    return;
  }
  const switchingSession = Boolean(activeConversationSessionID() && activeConversationSessionID() !== sessionID);
  if (switchingSession && !confirmComposerNavigation()) {
    return;
  }
  setActiveConversationSessionID(sessionID);
  syncMainChatComposerDraft(sessionID);
  navigateToRoute(state.currentRoute, { skipConfirm: switchingSession });
  renderSessions();
  renderMessages();
  syncHeader();
  renderChatRuntimePanel();
  renderWelcomeTargetPicker();
  closeTransientPanels();
}

function formatSince(epochMillis) {
  const delta = Date.now() - epochMillis;
  const minutes = Math.floor(delta / 60000);
  if (minutes < 1) {
    return t("time.just_now");
  }
  if (minutes < 60) {
    return `${minutes} ${t("time.m_ago")}`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours} ${t("time.h_ago")}`;
  }
  const days = Math.floor(hours / 24);
  return `${days} ${t("time.d_ago")}`;
}

function getSessionStorage() {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

function getBrowserSessionStorage() {
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}

function readComposerDraftValue(storage, key) {
  if (!storage || !key) {
    return "";
  }
  try {
    const raw = storage.getItem(COMPOSER_DRAFT_STORAGE_KEY);
    if (!raw) {
      return "";
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return "";
    }
    const value = parsed[key];
    return typeof value === "string" ? value : "";
  } catch {
    return "";
  }
}

function writeComposerDraftValue(storage, key, value) {
  if (!storage || !key) {
    return;
  }
  try {
    const raw = storage.getItem(COMPOSER_DRAFT_STORAGE_KEY);
    const parsed = raw ? JSON.parse(raw) : {};
    const next = parsed && typeof parsed === "object" ? parsed : {};
    const normalized = String(value || "");
    if (normalized) {
      next[key] = normalized;
    } else {
      delete next[key];
    }
    storage.setItem(COMPOSER_DRAFT_STORAGE_KEY, JSON.stringify(next));
  } catch {
  }
}

function persistRuntimeRestartNotice() {
  const storage = getBrowserSessionStorage();
  if (!storage) {
    return;
  }
  try {
    storage.setItem(RUNTIME_RESTART_NOTICE_STORAGE_KEY, JSON.stringify({
      status: "success",
      created_at: Date.now()
    }));
  } catch {
  }
}

function consumeRuntimeRestartNotice() {
  const storage = getBrowserSessionStorage();
  if (!storage) {
    return null;
  }
  try {
    const raw = storage.getItem(RUNTIME_RESTART_NOTICE_STORAGE_KEY);
    storage.removeItem(RUNTIME_RESTART_NOTICE_STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    return parsed.status === "success" ? parsed : null;
  } catch {
    return null;
  }
}

let globalModalHost = null;

function ensureGlobalModalHost() {
  if (globalModalHost instanceof HTMLElement && globalModalHost.isConnected) {
    return globalModalHost;
  }
  const existing = document.querySelector("[data-global-modal-root]");
  if (existing instanceof HTMLElement) {
    globalModalHost = existing;
    return globalModalHost;
  }
  const node = document.createElement("div");
  node.setAttribute("data-global-modal-root", "system");
  document.body.appendChild(node);
  globalModalHost = node;
  return node;
}

function activateModalBackdrop(modalBackdrop) {
  if (!(modalBackdrop instanceof HTMLElement) || modalBackdrop.dataset.modalReady === "true") {
    return;
  }
  modalBackdrop.dataset.modalReady = "true";
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      if (!modalBackdrop.isConnected) {
        return;
      }
      modalBackdrop.setAttribute("data-modal-state", "open");
    });
  });
}

function hideGlobalModal() {
  const host = ensureGlobalModalHost();
  host.innerHTML = "";
}

function showGlobalModal(options = {}) {
  const host = ensureGlobalModalHost();
  const title = normalizeText(options.title || "");
  const message = normalizeText(options.message || "");
  const confirmLabel = normalizeText(options.confirmLabel || "") || t("action.ok");
  host.innerHTML = `<div class="modal-backdrop" data-modal data-modal-state="enter">
    <div class="modal-dialog">
      <div class="modal-header">
        <h3>${escapeHTML(title)}</h3>
        <button type="button" data-global-modal-close aria-label="${escapeHTML(confirmLabel)}">&times;</button>
      </div>
      <div class="modal-body">
        <p>${escapeHTML(message)}</p>
      </div>
      <div class="modal-footer">
        <button type="button" data-global-modal-close>${escapeHTML(confirmLabel)}</button>
      </div>
    </div>
  </div>`;
  host.querySelectorAll("[data-global-modal-close]").forEach((element) => {
    element.addEventListener("click", () => {
      hideGlobalModal();
    });
  });
  const modalBackdrop = host.querySelector("[data-modal]");
  if (modalBackdrop instanceof HTMLElement) {
    modalBackdrop.addEventListener("click", (event) => {
      if (event.target && event.target.hasAttribute("data-modal")) {
        hideGlobalModal();
      }
    });
    activateModalBackdrop(modalBackdrop);
  }
}

function showGlobalConfirmModal(options = {}) {
  const host = ensureGlobalModalHost();
  const title = normalizeText(options.title || "");
  const message = normalizeText(options.message || "");
  const description = normalizeText(options.description || "");
  const confirmLabel = normalizeText(options.confirmLabel || "") || t("action.ok");
  const cancelLabel = normalizeText(options.cancelLabel || "") || t("action.cancel");
  const checkboxLabel = normalizeText(options.checkboxLabel || "");
  const checkboxHint = normalizeText(options.checkboxHint || "");
  const checkboxChecked = options.checkboxChecked !== false;
  const hasCheckbox = Boolean(checkboxLabel);
  host.innerHTML = `<div class="modal-backdrop" data-modal data-modal-state="enter">
    <div class="modal-dialog">
      <div class="modal-header">
        <h3>${escapeHTML(title)}</h3>
        <button type="button" data-global-modal-cancel aria-label="${escapeHTML(cancelLabel)}">&times;</button>
      </div>
      <div class="modal-body">
        ${message ? `<p class="environment-restart-confirm-copy">${escapeHTML(message)}</p>` : ""}
        ${description ? `<p class="environment-restart-confirm-hint">${escapeHTML(description)}</p>` : ""}
        ${hasCheckbox ? `<label class="environment-restart-confirm-option">
          <input type="checkbox" data-global-modal-checkbox ${checkboxChecked ? "checked" : ""}>
          <span>
            ${escapeHTML(checkboxLabel)}
            ${checkboxHint ? `<small>${escapeHTML(checkboxHint)}</small>` : ""}
          </span>
        </label>` : ""}
      </div>
      <div class="modal-footer">
        <button type="button" data-global-modal-cancel data-variant="secondary">${escapeHTML(cancelLabel)}</button>
        <button type="button" data-global-modal-confirm>${escapeHTML(confirmLabel)}</button>
      </div>
    </div>
  </div>`;
  return new Promise((resolve) => {
    let settled = false;
    const settle = (confirmed) => {
      if (settled) {
        return;
      }
      settled = true;
      const checkbox = host.querySelector("[data-global-modal-checkbox]");
      hideGlobalModal();
      resolve({
        confirmed,
        checked: checkbox instanceof HTMLInputElement ? checkbox.checked : false
      });
    };
    host.querySelectorAll("[data-global-modal-cancel]").forEach((element) => {
      element.addEventListener("click", () => settle(false));
    });
    const confirmButton = host.querySelector("[data-global-modal-confirm]");
    if (confirmButton instanceof HTMLElement) {
      confirmButton.addEventListener("click", () => settle(true));
    }
    const modalBackdrop = host.querySelector("[data-modal]");
    if (modalBackdrop instanceof HTMLElement) {
      modalBackdrop.addEventListener("click", (event) => {
        if (event.target && event.target.hasAttribute("data-modal")) {
          settle(false);
        }
      });
      activateModalBackdrop(modalBackdrop);
    }
    requestAnimationFrame(() => {
      if (confirmButton instanceof HTMLElement && confirmButton.isConnected) {
        confirmButton.focus();
      }
    });
  });
}

function showPendingRuntimeRestartNotice() {
  const notice = consumeRuntimeRestartNotice();
  if (!notice) {
    return;
  }
  showGlobalModal({
    title: t("route.envs.restart_service"),
    message: t("route.envs.restart_success")
  });
}

function getMainChatDraftKey(sessionID = activeConversationSessionID(), mode = routeConversationMode()) {
  return `${mode}.main:${normalizeText(sessionID || "default")}`;
}

function clearMainChatDraft(sessionID) {
  writeComposerDraftValue(getBrowserSessionStorage(), getMainChatDraftKey(sessionID), "");
}

const composerNavigationRegistry = new Map();

function registerComposerNavigationGuard(key, controller) {
  const guardKey = String(key || "").trim();
  if (!guardKey || !controller || typeof controller.hasDraft !== "function") {
    return;
  }
  composerNavigationRegistry.set(guardKey, controller);
}

function unregisterComposerNavigationGuard(key) {
  const guardKey = String(key || "").trim();
  if (!guardKey) {
    return;
  }
  composerNavigationRegistry.delete(guardKey);
}

function hasBlockingComposerDraft() {
  for (const controller of composerNavigationRegistry.values()) {
    if (!controller || typeof controller.hasDraft !== "function") {
      continue;
    }
    if (controller.hasDraft()) {
      return true;
    }
  }
  return false;
}

function syncComposerGuardState() {
  if (!document.body) {
    return;
  }
  const hasDraft = hasBlockingComposerDraft();
  document.body.setAttribute("data-composer-unsaved-state", hasDraft ? "dirty" : "clean");
}

function setComposerConfirmState(state) {
  if (!document.body) {
    return;
  }
  document.body.setAttribute("data-composer-unsaved-confirm", String(state || "idle"));
}

function confirmComposerNavigation() {
  if (!hasBlockingComposerDraft()) {
    setComposerConfirmState("idle");
    return true;
  }
  setComposerConfirmState("pending");
  const confirmed = window.confirm(t("composer.unsaved_confirm"));
  setComposerConfirmState(confirmed ? "accepted" : "dismissed");
  return confirmed;
}

function createReusableComposer() {
  const state = {
    cleanup: null,
    inputNode: null,
    formNode: null,
    composing: false,
    submitting: false,
    disabled: false,
    submitNodes: [],
    counterNode: null,
    maxLength: 0,
    draftStorage: null,
    draftKey: "",
    navigationGuardKey: ""
  };

  const syncNodeState = () => {
    if (!state.inputNode) {
      syncComposerGuardState();
      return;
    }
    const draftState = String(state.inputNode.value || "").trim() ? "dirty" : "empty";
    state.inputNode.setAttribute("data-composer-draft-state", draftState);
    state.inputNode.setAttribute("data-composer-composing", state.composing ? "true" : "false");
    state.inputNode.setAttribute("data-composer-pending", state.submitting ? "true" : "false");
    state.inputNode.setAttribute("data-composer-disabled", state.inputNode.disabled ? "true" : "false");
    syncComposerGuardState();
  };

  const reset = () => {
    state.inputNode = null;
    state.formNode = null;
    state.composing = false;
    state.submitting = false;
    state.disabled = false;
    state.submitNodes = [];
    state.counterNode = null;
    state.maxLength = 0;
    state.draftStorage = null;
    state.draftKey = "";
    state.navigationGuardKey = "";
  };

  const unbind = () => {
    unregisterComposerNavigationGuard(state.navigationGuardKey);
    if (typeof state.cleanup === "function") {
      state.cleanup();
    }
    state.cleanup = null;
    if (state.inputNode) {
      state.inputNode.removeAttribute("data-composer-draft-state");
      state.inputNode.removeAttribute("data-composer-composing");
      state.inputNode.removeAttribute("data-composer-pending");
      state.inputNode.removeAttribute("data-composer-disabled");
    }
    reset();
    syncComposerGuardState();
  };

  const invokeSubmit = async (hooks, event) => {
    if (state.submitting || state.composing || typeof hooks?.onSubmit !== "function" || !state.inputNode) {
      return;
    }
    const submittedValue = String(state.inputNode.value || "");
    const submittedInputNode = state.inputNode;
    const submittedDraftStorage = state.draftStorage;
    const submittedDraftKey = state.draftKey;
    if (submittedValue.trim() === "") {
      return;
    }
    const shouldClearDraft = hooks.clearDraftOnSubmit && submittedValue.trim() !== "";
    state.submitting = true;
    if (typeof hooks.onPendingChange === "function") {
      hooks.onPendingChange(true, state.inputNode);
    } else {
      updateDisabledState(true);
    }
    try {
      await hooks.onSubmit(state.inputNode, event, {
        isComposing: () => state.composing
      });
      const sameDraftTarget = state.inputNode === submittedInputNode && state.draftKey === submittedDraftKey;
      if (shouldClearDraft) {
        const currentSubmittedDraftValue = readComposerDraftValue(submittedDraftStorage, submittedDraftKey);
        const hasNewDraftValue = currentSubmittedDraftValue !== "" && currentSubmittedDraftValue !== submittedValue;
        if (!hasNewDraftValue) {
          writeComposerDraftValue(submittedDraftStorage, submittedDraftKey, "");
          if (sameDraftTarget) {
            state.inputNode.value = String(state.inputNode.value || "");
          }
        }
      } else if (sameDraftTarget && (submittedValue.trim() !== "" || String(state.inputNode.value || "").trim() !== "")) {
        writeComposerDraftValue(submittedDraftStorage, submittedDraftKey, state.inputNode.value);
      }
      syncCounter();
    } finally {
      state.submitting = false;
      if (typeof hooks.onPendingChange === "function") {
        hooks.onPendingChange(false, state.inputNode);
      } else {
        updateDisabledState(false);
      }
    }
  };

  const resolveStorage = (mode) => {
    if (mode === "local") {
      return getSessionStorage();
    }
    if (mode === "session") {
      return getBrowserSessionStorage();
    }
    return null;
  };

  const persistDraft = (value) => {
    writeComposerDraftValue(state.draftStorage, state.draftKey, value);
  };

  const restoreDraft = () => {
    if (!state.inputNode) {
      return "";
    }
    const restoredDraft = readComposerDraftValue(state.draftStorage, state.draftKey);
    state.inputNode.value = restoredDraft;
    syncCounter();
    return restoredDraft;
  };

  const syncCounter = () => {
    if (!state.counterNode || !state.inputNode || !state.maxLength) {
      return;
    }
    state.counterNode.textContent = `${String(state.inputNode.value || "").length}/${state.maxLength}`;
  };

  const updateDisabledState = (pending) => {
    const disabled = Boolean(pending || state.disabled);
    if (state.inputNode) {
      state.inputNode.disabled = disabled;
    }
    state.submitNodes.forEach((node) => {
      if (node) {
        node.disabled = disabled;
      }
    });
    syncNodeState();
  };

  return {
    bind(inputNode, formNode, hooks = {}) {
      unbind();
      if (!inputNode) {
        return;
      }
      state.inputNode = inputNode;
      state.formNode = formNode || null;
      state.counterNode = hooks.counterNode || null;
      state.maxLength = Number(hooks.maxLength || inputNode.maxLength || 0);
      state.submitNodes = Array.isArray(hooks.submitNodes)
        ? hooks.submitNodes.filter(Boolean)
        : [hooks.submitNode || (formNode ? formNode.querySelector("[type=\"submit\"]") : null)].filter(Boolean);
      state.draftStorage = resolveStorage(String(hooks.draftStorage || "").trim().toLowerCase());
      state.draftKey = typeof hooks.draftKey === "function" ? String(hooks.draftKey() || "").trim() : String(hooks.draftKey || "").trim();
      state.navigationGuardKey = String(hooks.navigationGuardKey || state.draftKey || "").trim();
      state.disabled = Boolean(hooks.disabled);
      const stableName = String(hooks.stableName || "").trim();
      if (stableName) {
        inputNode.setAttribute("data-composer-input", stableName);
        if (formNode) {
          formNode.setAttribute("data-composer-form", stableName);
        }
        if (state.counterNode) {
          state.counterNode.setAttribute("data-composer-counter", stableName);
        }
        state.submitNodes.forEach((node) => node.setAttribute("data-composer-submit", stableName));
      }
      inputNode.setAttribute("data-composer-ready", "true");
      const submitOnEnter = hooks.submitOnEnter !== false;
      const allowShiftEnter = Boolean(hooks.allowShiftEnter);
      const submitStrategy = hooks.submitStrategy === "keydown" ? "keydown" : "form";

      const restoredDraft = readComposerDraftValue(state.draftStorage, state.draftKey);
      if (!String(inputNode.value || "") && restoredDraft) {
        inputNode.value = restoredDraft;
        if (typeof hooks.onDraftRestore === "function") {
          hooks.onDraftRestore(inputNode, restoredDraft);
        }
      }
      registerComposerNavigationGuard(state.navigationGuardKey, {
        hasDraft: () => {
          if (!state.inputNode || !document.body.contains(state.inputNode)) {
            return false;
          }
          return String(state.inputNode.value || "").trim() !== "";
        }
      });
      syncCounter();
      updateDisabledState(false);
      syncNodeState();

      const handleInput = (event) => {
        persistDraft(inputNode.value);
        syncCounter();
        syncNodeState();
        if (typeof hooks.onInput === "function") {
          hooks.onInput(inputNode, event, {
            isComposing: () => state.composing
          });
        }
      };

      const handleFocus = (event) => {
        if (typeof hooks.onFocus === "function") {
          hooks.onFocus(inputNode, event, {
            isComposing: () => state.composing
          });
        }
      };

      const handleBlur = (event) => {
        window.setTimeout(() => {
          if (document.activeElement === inputNode) {
            return;
          }
          if (typeof hooks.onBlur === "function") {
            hooks.onBlur(inputNode, event, {
              isComposing: () => state.composing
            });
          }
        }, 0);
      };

      const handleCompositionStart = (event) => {
        state.composing = true;
        syncNodeState();
        if (typeof hooks.onCompositionStart === "function") {
          hooks.onCompositionStart(inputNode, event, {
            isComposing: () => state.composing
          });
        }
      };

      const handleCompositionEnd = (event) => {
        window.setTimeout(() => {
          state.composing = false;
          persistDraft(inputNode.value);
          syncCounter();
          syncNodeState();
          if (typeof hooks.onCompositionEnd === "function") {
            hooks.onCompositionEnd(inputNode, event, {
              isComposing: () => state.composing
            });
          }
        }, 0);
      };

      const handleKeyDown = (event) => {
        if (event.key !== "Enter" || !submitOnEnter) {
          return;
        }
        if (event.isComposing || state.composing) {
          event.preventDefault();
          return;
        }
        if (allowShiftEnter && event.shiftKey) {
          return;
        }
        if (submitStrategy !== "keydown") {
          return;
        }
        event.preventDefault();
        void invokeSubmit(hooks, event);
      };

      const handleSubmit = (event) => {
        event.preventDefault();
        if (state.composing) {
          return;
        }
        if (submitStrategy !== "form") {
          return;
        }
        void invokeSubmit(hooks, event);
      };

      const handleSubmitClick = (event) => {
        if (submitStrategy !== "keydown") {
          return;
        }
        event.preventDefault();
        if (state.composing) {
          return;
        }
        void invokeSubmit(hooks, event);
      };

      inputNode.addEventListener("input", handleInput);
      inputNode.addEventListener("focus", handleFocus);
      inputNode.addEventListener("blur", handleBlur);
      inputNode.addEventListener("compositionstart", handleCompositionStart);
      inputNode.addEventListener("compositionend", handleCompositionEnd);
      inputNode.addEventListener("keydown", handleKeyDown);
      if (formNode) {
        formNode.addEventListener("submit", handleSubmit);
      }
      state.submitNodes.forEach((node) => {
        node.addEventListener("click", handleSubmitClick);
      });

      state.cleanup = () => {
        inputNode.removeEventListener("input", handleInput);
        inputNode.removeEventListener("focus", handleFocus);
        inputNode.removeEventListener("blur", handleBlur);
        inputNode.removeEventListener("compositionstart", handleCompositionStart);
        inputNode.removeEventListener("compositionend", handleCompositionEnd);
        inputNode.removeEventListener("keydown", handleKeyDown);
        if (formNode) {
          formNode.removeEventListener("submit", handleSubmit);
        }
        state.submitNodes.forEach((node) => {
          node.removeEventListener("click", handleSubmitClick);
        });
      };
    },
    isComposing() {
      return state.composing;
    },
    setDisabled(flag) {
      state.disabled = Boolean(flag);
      updateDisabledState(false);
    },
    syncDraft(value = null) {
      if (!state.inputNode) {
        return;
      }
      persistDraft(value === null ? state.inputNode.value : value);
    },
    clearDraft() {
      if (!state.inputNode) {
        persistDraft("");
        syncComposerGuardState();
        return;
      }
      state.inputNode.value = "";
      persistDraft("");
      syncCounter();
      syncNodeState();
    },
    hasDraft() {
      if (!state.inputNode || !document.body.contains(state.inputNode)) {
        return false;
      }
      return String(state.inputNode.value || "").trim() !== "";
    },
    getDraftKey() {
      return state.draftKey;
    },
    switchDraftKey(nextKey, options = {}) {
      const resolvedKey = String(nextKey || "").trim();
      const preserveCurrent = options.preserveCurrent !== false;
      if (preserveCurrent && state.inputNode && state.draftKey) {
        persistDraft(state.inputNode.value);
      }
      state.draftKey = resolvedKey;
      restoreDraft();
      syncNodeState();
    },
    syncCounter,
    unbind
  };
}

const mainChatComposer = createReusableComposer();

function syncMainChatComposerDraft(sessionID = activeConversationSessionID(), options = {}) {
  mainChatComposer.switchDraftKey(getMainChatDraftKey(sessionID), options);
}

function loadSessionHistoryCollapsedState() {
  const storage = getBrowserSessionStorage();
  if (!storage) {
    return false;
  }
  const raw = storage.getItem(SESSION_HISTORY_PANEL_STORAGE_KEY);
  if (!raw) {
    return false;
  }

  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed.collapsed_state === "boolean") {
      return parsed.collapsed_state;
    }
  } catch {
    return raw === "1";
  }
  return false;
}

function persistSessionHistoryCollapsedState() {
  const storage = getBrowserSessionStorage();
  if (!storage) {
    return;
  }
  try {
    storage.setItem(SESSION_HISTORY_PANEL_STORAGE_KEY, JSON.stringify({
      collapsed_state: state.sessionHistoryCollapsed
    }));
  } catch {
  }
}

function loadActiveConversationState() {
  const storage = getSessionStorage();
  if (!storage) {
    return {};
  }
  const raw = storage.getItem(SESSION_ACTIVE_STORAGE_KEY);
  if (!raw) {
    return {};
  }
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    return Object.entries(parsed).reduce((acc, [key, value]) => {
      const bucket = normalizeText(key);
      const sessionID = normalizeText(value);
      if (bucket && sessionID) {
        acc[bucket] = sessionID;
      }
      return acc;
    }, {});
  } catch {
    return {};
  }
}

function persistActiveConversationState() {
  const storage = getSessionStorage();
  if (!storage) {
    return;
  }
  try {
    storage.setItem(SESSION_ACTIVE_STORAGE_KEY, JSON.stringify(state.activeSessionByBucket || {}));
  } catch {
  }
}

function normalizeStoredMessage(item, fallbackAt) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const text = typeof item.text === "string" ? item.text : "";
  if (!text) {
    return null;
  }
  const role = item.role === "assistant" ? "assistant" : "user";
  const at = Number.isFinite(item.at) ? item.at : fallbackAt;
  const status = typeof item.status === "string" && item.status ? item.status : "done";
  return {
    id: typeof item.id === "string" && item.id ? item.id : makeID(),
    role,
    text,
    at,
    route: typeof item.route === "string" ? item.route : "",
    source: typeof item.source === "string" ? item.source : "",
    error: Boolean(item.error),
    status,
    retryable: Boolean(item.retryable),
    task_id: typeof item.task_id === "string" ? item.task_id : "",
    task_status: typeof item.task_status === "string" ? item.task_status : "",
    task_pending: Boolean(item.task_pending),
    task_result_delivered: Boolean(item.task_result_delivered),
    task_result_for: typeof item.task_result_for === "string" ? item.task_result_for : "",
    task_completed_at: Number.isFinite(item.task_completed_at) ? item.task_completed_at : 0,
    agent_process_collapsed: typeof item.agent_process_collapsed === "boolean" ? item.agent_process_collapsed : undefined
  };
}

function normalizeStoredSession(item, mode = routeConversationMode()) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const id = typeof item.id === "string" && item.id ? item.id : makeID();
  const defaultTitle = mode === "agent" ? "New Agent Run" : "New Chat";
  const title = typeof item.title === "string" && item.title.trim() ? item.title.trim() : defaultTitle;
  const createdAt = Number.isFinite(item.createdAt) ? item.createdAt : Date.now();
  const target = normalizeChatTarget({
    type: item.targetType || (mode === "agent" ? "agent" : "model"),
    id: item.targetID || (mode === "agent" ? "" : "raw-model"),
    name: item.targetName || (mode === "agent" ? t("session.target.agent") : t("session.target.raw"))
  });
  const rawMessages = Array.isArray(item.messages) ? item.messages : [];
  const messages = [];
  for (const raw of rawMessages) {
    const normalized = normalizeStoredMessage(raw, createdAt);
    if (normalized) {
      messages.push(normalized);
    }
  }
  const inferredTitleState = inferStoredSessionTitleState(title, defaultTitle);
  return {
    id,
    title,
    titleAuto: typeof item.titleAuto === "boolean" ? item.titleAuto : inferredTitleState.titleAuto,
    titleScore: Number.isFinite(item.titleScore) ? Math.max(Number(item.titleScore), 0) : inferredTitleState.titleScore,
    createdAt,
    messages,
    historyBucket: normalizeText(item.historyBucket) || conversationHistoryBucketForTarget(target),
    targetType: target.type,
    targetID: target.id,
    targetName: target.name,
    modelProviderID: normalizeText(item.modelProviderID),
    modelID: normalizeText(item.modelID),
    toolIDs: normalizeSelectionIDs(item.toolIDs),
    skillIDs: normalizeSelectionIDs(item.skillIDs),
    mcpIDs: normalizeSelectionIDs(item.mcpIDs)
  };
}

function normalizeLegacyStoredSession(item, mode = "") {
  if (!item || typeof item !== "object") {
    return null;
  }
  const legacyTarget = normalizeChatTarget({
    type: item.targetType,
    id: item.targetID,
    name: item.targetName
  });
  if (mode === "agent" && legacyTarget.type !== "agent") {
    return null;
  }
  if (mode === "chat" && legacyTarget.type === "agent") {
    return null;
  }
  const normalized = normalizeStoredSession(item, mode);
  if (!normalized) {
    return null;
  }
  if (legacyTarget.type === "agent") {
    normalized.targetType = legacyTarget.type;
    normalized.targetID = legacyTarget.id;
    normalized.targetName = legacyTarget.name;
  }
  return normalized;
}

function parseStoredSessionArray(raw) {
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error("local_storage_corrupted");
  }
  if (!Array.isArray(parsed)) {
    throw new Error("local_storage_invalid");
  }
  return parsed;
}

function mergeNormalizedSessions(target, items) {
  const index = new Map(target.map((item) => [item.id, item]));
  items.forEach((item) => {
    if (!item || !item.id) {
      return;
    }
    const existing = index.get(item.id);
    if (!existing || Number(item.createdAt || 0) >= Number(existing.createdAt || 0)) {
      index.set(item.id, item);
    }
  });
  return Array.from(index.values());
}

function loadSessionsFromStorage(mode = routeConversationMode()) {
  const storage = getSessionStorage();
  if (!storage) {
    return [];
  }

  const sharedRaw = storage.getItem(SESSION_STORAGE_KEY);
  const sessions = [];
  if (sharedRaw) {
    const parsed = parseStoredSessionArray(sharedRaw);
    parsed.forEach((entry) => {
      const normalized = normalizeStoredSession(entry);
      if (normalized) {
        sessions.push(normalized);
      }
    });
    sortSessionsByCreatedAtDesc(sessions);
    return sessions;
  }

  const legacyBuckets = [
    {
      raw: storage.getItem(LEGACY_CHAT_SESSION_STORAGE_KEY),
      normalize: (entry) => normalizeStoredSession(entry, "chat")
    },
    {
      raw: storage.getItem(LEGACY_AGENT_SESSION_STORAGE_KEY),
      normalize: (entry) => normalizeStoredSession(entry, "agent")
    },
    {
      raw: storage.getItem(LEGACY_SESSION_STORAGE_KEY),
      normalize: (entry) => normalizeLegacyStoredSession(entry)
    }
  ];
  legacyBuckets.forEach((bucket) => {
    if (!bucket.raw) {
      return;
    }
    const parsed = parseStoredSessionArray(bucket.raw);
    const normalizedItems = [];
    parsed.forEach((entry) => {
      const normalized = bucket.normalize(entry);
      if (normalized) {
        normalizedItems.push(normalized);
      }
    });
    const merged = mergeNormalizedSessions(sessions, normalizedItems);
    sessions.length = 0;
    sessions.push(...merged);
  });
  sortSessionsByCreatedAtDesc(sessions);
  return sessions;
}

function persistSessions(mode = routeConversationMode()) {
  const storage = getSessionStorage();
  if (!storage) {
    return;
  }
  try {
    storage.setItem(SESSION_STORAGE_KEY, JSON.stringify(storedConversationSessions()));
    persistActiveConversationState();
    setConversationSessionLoadError("", mode);
  } catch {
    setConversationSessionLoadError("session_save_failed", mode);
  }
  syncSessionLoadHint();
}

function bootstrapSessions(mode = routeConversationMode()) {
  setConversationSessionLoadError("", mode);
  setStoredConversationSessions([]);
  state.activeSessionByBucket = loadActiveConversationState();

  try {
    const sessions = loadSessionsFromStorage(mode);
    setStoredConversationSessions(sessions);
    reconcileAgentRuntimeTarget();
    ensureActiveConversationSession("chat");
    ensureActiveConversationSession("agent-runtime");
  } catch (err) {
    const message = err instanceof Error ? err.message : "unknown_error";
    setConversationSessionLoadError(message, mode);
  }
}

async function refreshChatAgentCatalog() {
  if (state.chatCatalog.loading) {
    return;
  }
  state.chatCatalog.loading = true;
  let targetChanged = false;
  try {
    const payload = await fetchJSON("/api/agents");
    state.chatCatalog.agents = Array.isArray(payload?.items)
      ? payload.items.filter((item) => Boolean(item?.enabled))
      : [];
    targetChanged = reconcileAgentRuntimeTarget();
    state.chatCatalog.error = "";
    state.chatCatalog.loaded = true;
  } catch (error) {
    state.chatCatalog.error = error instanceof Error ? error.message : "unknown_error";
    state.chatCatalog.loaded = true;
  } finally {
    state.chatCatalog.loading = false;
    if (targetChanged || state.currentRoute === "agent-runtime") {
      ensureActiveConversationSession("agent-runtime");
      renderSessions();
      renderMessages();
      syncHeader();
      syncWelcomeCopy();
    }
    renderChatRuntimePanel();
    renderWelcomeTargetPicker();
  }
}

async function refreshChatProviderCatalog() {
  if (state.chatCatalog.providerLoading) {
    return;
  }
  state.chatCatalog.providerLoading = true;
  try {
    const payload = await fetchJSON("/api/control/llm/providers");
    state.chatCatalog.providers = Array.isArray(payload?.items) ? payload.items : [];
    state.chatCatalog.providerError = "";
    state.chatCatalog.providerLoaded = true;
    reconcileChatModelSelections();
  } catch (error) {
    state.chatCatalog.providerError = error instanceof Error ? error.message : "unknown_error";
    state.chatCatalog.providerLoaded = true;
  } finally {
    state.chatCatalog.providerLoading = false;
    renderChatRuntimePanel();
    renderWelcomeTargetPicker();
    renderSessions();
    syncHeader();
  }
}

async function refreshChatCapabilityCatalog() {
  if (state.chatCatalog.capabilityLoading) {
    return;
  }
  state.chatCatalog.capabilityLoading = true;
  try {
    const [skillPayload, mcpPayload] = await Promise.all([
      fetchJSON("/api/control/skills"),
      fetchJSON("/api/control/mcps")
    ]);
    state.chatCatalog.skills = Array.isArray(skillPayload?.items)
      ? skillPayload.items.filter((item) => Boolean(item?.enabled))
      : [];
    state.chatCatalog.mcps = Array.isArray(mcpPayload?.items)
      ? mcpPayload.items.filter((item) => Boolean(item?.enabled))
      : [];
    state.chatCatalog.capabilityError = "";
    state.chatCatalog.capabilityLoaded = true;
  } catch (error) {
    state.chatCatalog.capabilityError = error instanceof Error ? error.message : "unknown_error";
    state.chatCatalog.capabilityLoaded = true;
  } finally {
    state.chatCatalog.capabilityLoading = false;
    renderChatRuntimePanel();
  }
}

function renderRuntimeGroupSection(titleKey, items, renderItem) {
  if (!items.length) {
    return `<p class="composer-runtime-empty">${escapeHTML(t("chat.runtime.none"))}</p>`;
  }
  return `<section class="composer-runtime-section">
    <div class="composer-runtime-section-title">
      <strong>${escapeHTML(t(titleKey))}</strong>
      <span>${escapeHTML(String(items.length))}</span>
    </div>
    ${items.map(renderItem).join("")}
  </section>`;
}

function renderRuntimeCapabilityGroup(items, selectedSet, toggleKey) {
  if (!items.length) {
    return "";
  }
  const groupedTools = items.filter((item) => item.kind === "tool");
  const groupedMcps = items.filter((item) => item.kind === "mcp");
  const groups = [
    { key: "chat.runtime.category.tools", items: groupedTools },
    { key: "chat.runtime.category.mcps", items: groupedMcps }
  ].filter((group) => group.items.length);
  return groups.map((group) => `<section class="composer-runtime-section">
    <div class="composer-runtime-group-title">
      <strong>${escapeHTML(t(group.key))}</strong>
      <span>${escapeHTML(String(group.items.length))}</span>
    </div>
    <div class="composer-runtime-checkbox-group">
      ${group.items.map((item) => `<label class="composer-runtime-checkbox">
        <input type="checkbox" data-runtime-toggle-item="${escapeHTML(toggleKey)}" data-runtime-item-kind="${escapeHTML(item.kind)}" value="${escapeHTML(item.id)}" ${selectedSet.has(item.id) ? "checked" : ""}>
        <span class="composer-runtime-checkbox-copy">
          <strong>${escapeHTML(item.name)}</strong>
          <span>${escapeHTML(item.description)}</span>
        </span>
      </label>`).join("")}
    </div>
  </section>`).join("");
}

function renderChatRuntimeCompactPopover({
  agentRuntime,
  target,
  locked,
  targetOptions,
  provider,
  model,
  modelGroups,
  modelSelection,
  capabilityActive,
  capabilityAvailable,
  capabilitySelected,
  activeSkills,
  availableSkills,
  skillSelected,
  note
}) {
  const summaryChips = [];
  if (agentRuntime && target?.id) {
    summaryChips.push(`<span class="composer-runtime-chip">${escapeHTML(`${t("session.target.agent")} · ${target.name}`)}</span>`);
  }
  summaryChips.push(`<span class="composer-runtime-chip">${escapeHTML(provider && model ? `${t("chat.runtime.model_short")} · ${model.name || model.id}` : `${t("chat.runtime.model_short")} · ${t("session.model.default")}`)}</span>`);
  summaryChips.push(`<span class="composer-runtime-chip">${escapeHTML(`${t("chat.runtime.tools_short")} ${String(capabilitySelected.size)}`)}</span>`);
  summaryChips.push(`<span class="composer-runtime-chip">${escapeHTML(`${t("chat.runtime.skills_short")} ${String(skillSelected.size)}`)}</span>`);

  const targetSection = agentRuntime ? `<section class="composer-runtime-section">
    <div class="composer-runtime-popover-head">
      <strong>${escapeHTML(t("chat.runtime.agent"))}</strong>
      <p>${escapeHTML(locked ? t("chat.runtime.locked") : t("chat.runtime.agent_hint"))}</p>
    </div>
    <div class="composer-runtime-option-list">
      ${targetOptions.length ? targetOptions.map((item) => {
        const optionTarget = normalizeChatTarget(item.target);
        const activeClass = optionTarget.id === target.id ? " is-active" : "";
        return `<button class="composer-runtime-option${activeClass}" type="button" data-runtime-target-type="${escapeHTML(optionTarget.type)}" data-runtime-target-id="${escapeHTML(optionTarget.id)}" data-runtime-target-name="${escapeHTML(optionTarget.name)}" ${locked ? "disabled" : ""}>
          <strong>${escapeHTML(`${t("session.target.agent")} · ${optionTarget.name}`)}</strong>
          <span>${escapeHTML(item.subtitle)}</span>
        </button>`;
      }).join("") : `<p class="composer-runtime-empty">${escapeHTML(t("chat.runtime.none"))}</p>`}
    </div>
  </section>` : "";

  const modelSection = `<section class="composer-runtime-section">
    <div class="composer-runtime-popover-head">
      <strong>${escapeHTML(`${t("chat.runtime.provider")} / ${t("chat.runtime.model")}`)}</strong>
      <p>${escapeHTML(state.chatCatalog.providerError || t("chat.runtime.model_hint"))}</p>
    </div>
    ${modelGroups.length ? modelGroups.map((providerItem) => {
      const models = enabledModelsForProvider(providerItem);
      return `<section class="composer-runtime-provider-group">
        <div class="composer-runtime-group-title">
          <strong>${escapeHTML(providerItem.name || providerItem.id)}</strong>
          <span>${escapeHTML(String(models.length))}</span>
        </div>
        <div class="composer-runtime-option-list">
          ${models.map((modelItem) => {
            const activeClass = normalizeText(providerItem.id) === modelSelection.providerID && normalizeText(modelItem.id) === modelSelection.modelID ? " is-active" : "";
            return `<button class="composer-runtime-model-option${activeClass}" type="button" data-runtime-provider-id="${escapeHTML(providerItem.id)}" data-runtime-model-id="${escapeHTML(modelItem.id)}">
              <strong>${escapeHTML(modelItem.name || modelItem.id)}</strong>
              <span>${escapeHTML(providerItem.name || providerItem.id)}</span>
            </button>`;
          }).join("")}
        </div>
      </section>`;
    }).join("") : `<p class="composer-runtime-empty">${escapeHTML(t("chat.runtime.empty"))}</p>`}
  </section>`;

  const capabilitySection = `<section class="composer-runtime-section">
    <div class="composer-runtime-popover-head">
      <strong>${escapeHTML(t("chat.runtime.tools_mcp"))}</strong>
      <p>${escapeHTML(state.chatCatalog.capabilityError || t("chat.runtime.tools_hint"))}</p>
    </div>
    ${renderRuntimeGroupSection("chat.runtime.active", capabilityActive, () => "")}
    ${capabilityActive.length ? renderRuntimeCapabilityGroup(capabilityActive, capabilitySelected, "capabilities") : ""}
    <div class="composer-runtime-separator"></div>
    ${renderRuntimeGroupSection("chat.runtime.available", capabilityAvailable, () => "")}
    ${capabilityAvailable.length ? renderRuntimeCapabilityGroup(capabilityAvailable, capabilitySelected, "capabilities") : ""}
  </section>`;

  const skillsSection = `<section class="composer-runtime-section">
    <div class="composer-runtime-popover-head">
      <strong>${escapeHTML(t("chat.runtime.skills"))}</strong>
      <p>${escapeHTML(state.chatCatalog.capabilityError || t("chat.runtime.skills_hint"))}</p>
    </div>
    ${renderRuntimeGroupSection("chat.runtime.active", activeSkills, (item) => `<label class="composer-runtime-checkbox">
      <input type="checkbox" data-runtime-toggle-item="skills" value="${escapeHTML(item.id)}" checked>
      <span class="composer-runtime-checkbox-copy">
        <strong>${escapeHTML(item.name)}</strong>
        <span>${escapeHTML(item.description)}</span>
      </span>
    </label>`)}
    <div class="composer-runtime-separator"></div>
    ${renderRuntimeGroupSection("chat.runtime.available", availableSkills, (item) => `<label class="composer-runtime-checkbox">
      <input type="checkbox" data-runtime-toggle-item="skills" value="${escapeHTML(item.id)}">
      <span class="composer-runtime-checkbox-copy">
        <strong>${escapeHTML(item.name)}</strong>
        <span>${escapeHTML(item.description)}</span>
      </span>
    </label>`)}
  </section>`;

  return `<div class="composer-runtime-popover composer-runtime-popover-mobile is-wide">
    <div class="composer-runtime-popover-head">
      <strong>${escapeHTML(t("chat.runtime.mobile"))}</strong>
      <p>${escapeHTML(note || t("chat.runtime.mobile_hint"))}</p>
    </div>
    <div class="composer-runtime-summary">${summaryChips.join("")}</div>
    ${targetSection ? `${targetSection}<div class="composer-runtime-separator"></div>` : ""}
    ${modelSection}
    <div class="composer-runtime-separator"></div>
    ${capabilitySection}
    <div class="composer-runtime-separator"></div>
    ${skillsSection}
  </div>`;
}

function renderChatRuntimePanel() {
  if (!chatRuntimePanel) {
    return;
  }
  const mode = routeConversationMode();
  const initialTarget = currentConversationTarget();
  const activeSession = getSession() || ((mode === "agent" && !initialTarget.id) ? null : createSession(initialTarget, mode, state.currentRoute));
  const target = sessionTarget(activeSession);
  const locked = targetLocked(activeSession);
  const modelSelection = resolveEffectiveChatModelSelection(activeSession);
  const provider = findChatProvider(modelSelection.providerID);
  const model = findChatModel(modelSelection.providerID, modelSelection.modelID);
  const selections = sessionRuntimeSelections(activeSession);
  const toolMCPCount = selections.toolIDs.length + selections.mcpIDs.length;
  const skillsCount = selections.skillIDs.length;
  const openPopover = state.chatRuntime.openPopover;
  const agentRuntime = mode === "agent";
  const targetLabel = agentRuntime
    ? (target.id ? `${t("chat.runtime.agent")} · ${target.name}` : t("chat.runtime.agent_pick"))
    : t("session.target.raw");
  const modelLabel = provider && model
    ? `${t("chat.runtime.model_short")} · ${model.name || model.id}`
    : `${t("chat.runtime.model_short")} · ${t("session.model.default")}`;
  const toolLabel = `${t("chat.runtime.tools_short")} · ${String(toolMCPCount)}`;
  const skillLabel = `${t("chat.runtime.skills_short")} · ${String(skillsCount)}`;
  const runtimeErrorNote = [state.chatCatalog.providerError, state.chatCatalog.capabilityError].filter(Boolean).join(" | ");
  const compactRuntime = isTerminalSessionSheetViewport();
  const compactMeta = agentRuntime && !target.id
    ? t("chat.runtime.agent_pick")
    : t("chat.runtime.mobile_meta", {
        model: model ? (model.name || model.id) : t("session.model.default"),
        tools: String(toolMCPCount),
        skills: String(skillsCount)
      });
  const targetOptions = agentRuntime
    ? genericRuntimeConversationAgents().map((agent) => ({
        target: {
          type: "agent",
          id: normalizeText(agent?.id),
          name: String(agent?.name || agent?.id || "").trim()
        },
        subtitle: String(agent?.system_prompt || agent?.id || "").trim() || t("session.target.agent")
      })).filter((item) => item.target.id)
    : [];
  const modelGroups = enabledChatProviders();
  const capabilityItems = [
    ...AVAILABLE_CHAT_TOOLS.map((item) => ({ ...item, kind: "tool" })),
    ...((Array.isArray(state.chatCatalog.mcps) ? state.chatCatalog.mcps : []).map((item) => ({
      id: normalizeText(item?.id),
      name: String(item?.name || item?.id || "").trim(),
      description: String(item?.description || item?.scope || "MCP").trim(),
      kind: "mcp"
    })).filter((item) => item.id))
  ];
  const capabilitySelected = new Set([...selections.toolIDs, ...selections.mcpIDs]);
  const capabilityActive = capabilityItems.filter((item) => capabilitySelected.has(item.id));
  const capabilityAvailable = capabilityItems.filter((item) => !capabilitySelected.has(item.id));
  const skillItems = (Array.isArray(state.chatCatalog.skills) ? state.chatCatalog.skills : []).map((item) => ({
    id: normalizeText(item?.id),
    name: String(item?.name || item?.id || "").trim(),
    description: String(item?.description || item?.scope || "Skill").trim()
  })).filter((item) => item.id);
  const skillSelected = new Set(selections.skillIDs);
  const activeSkills = skillItems.filter((item) => skillSelected.has(item.id));
  const availableSkills = skillItems.filter((item) => !skillSelected.has(item.id));

  if (compactRuntime) {
    chatRuntimePanel.innerHTML = `<div class="composer-runtime-group composer-runtime-group-compact">
      <div class="composer-runtime-control composer-runtime-control-compact">
        <button class="composer-runtime-trigger composer-runtime-trigger-compact${openPopover === "mobile" ? " is-open" : ""}" type="button" data-runtime-toggle="mobile" title="${escapeHTML(t("chat.runtime.mobile_hint"))}">
          <span class="composer-runtime-trigger-icon">⚙️</span>
          <span class="composer-runtime-trigger-copy">
            <strong>${escapeHTML(t("chat.runtime.mobile"))}</strong>
            <span>${escapeHTML(compactMeta)}</span>
          </span>
          <span class="composer-runtime-trigger-caret">▾</span>
        </button>
        ${openPopover === "mobile" ? renderChatRuntimeCompactPopover({
          agentRuntime,
          target,
          locked,
          targetOptions,
          provider,
          model,
          modelGroups,
          modelSelection,
          capabilityActive,
          capabilityAvailable,
          capabilitySelected,
          activeSkills,
          availableSkills,
          skillSelected,
          note: state.chatCatalog.providerError || state.chatCatalog.capabilityError || t("chat.runtime.mobile_hint")
        }) : ""}
      </div>
    </div>
    ${runtimeErrorNote ? `<p class="chat-runtime-note chat-runtime-error">${escapeHTML(runtimeErrorNote)}</p>` : ""}`;
  } else {
  chatRuntimePanel.innerHTML = `<div class="composer-runtime-group">
    ${agentRuntime ? `<div class="composer-runtime-control">
      <button class="composer-runtime-trigger${openPopover === "target" ? " is-open" : ""}${locked ? " is-disabled" : ""}" type="button" data-runtime-toggle="target" aria-disabled="${locked ? "true" : "false"}" title="${escapeHTML(locked ? t("chat.runtime.locked") : t("chat.runtime.agent_hint"))}">
        <span class="composer-runtime-trigger-icon">🎯</span>
        <span class="composer-runtime-trigger-label">${escapeHTML(targetLabel)}</span>
        <span class="composer-runtime-trigger-caret">▾</span>
      </button>
      ${openPopover === "target" ? `<div class="composer-runtime-popover">
        <div class="composer-runtime-popover-head">
          <strong>${escapeHTML(t("chat.runtime.agent"))}</strong>
          <p>${escapeHTML(locked ? t("chat.runtime.locked") : t("chat.runtime.agent_hint"))}</p>
        </div>
        <div class="composer-runtime-option-list">
          ${targetOptions.map((item) => {
            const optionTarget = normalizeChatTarget(item.target);
            const activeClass = optionTarget.id === target.id ? " is-active" : "";
            return `<button class="composer-runtime-option${activeClass}" type="button" data-runtime-target-type="${escapeHTML(optionTarget.type)}" data-runtime-target-id="${escapeHTML(optionTarget.id)}" data-runtime-target-name="${escapeHTML(optionTarget.name)}" ${locked ? "disabled" : ""}>
              <strong>${escapeHTML(`${t("session.target.agent")} · ${optionTarget.name}`)}</strong>
              <span>${escapeHTML(item.subtitle)}</span>
            </button>`;
          }).join("")}
        </div>
      </div>` : ""}
    </div>` : `<div class="composer-runtime-chip">${escapeHTML(targetLabel)}</div>`}
    <div class="composer-runtime-control">
      <button class="composer-runtime-trigger${openPopover === "model" ? " is-open" : ""}" type="button" data-runtime-toggle="model" title="${escapeHTML(t("chat.runtime.model_hint"))}">
        <span class="composer-runtime-trigger-icon">✨</span>
        <span class="composer-runtime-trigger-label">${escapeHTML(modelLabel)}</span>
        <span class="composer-runtime-trigger-caret">▾</span>
      </button>
      ${openPopover === "model" ? `<div class="composer-runtime-popover is-wide">
        <div class="composer-runtime-popover-head">
          <strong>${escapeHTML(`${t("chat.runtime.provider")} / ${t("chat.runtime.model")}`)}</strong>
          <p>${escapeHTML(state.chatCatalog.providerError || t("chat.runtime.model_hint"))}</p>
        </div>
        ${modelGroups.length ? modelGroups.map((providerItem) => {
          const models = enabledModelsForProvider(providerItem);
          return `<section class="composer-runtime-provider-group">
            <div class="composer-runtime-group-title">
              <strong>${escapeHTML(providerItem.name || providerItem.id)}</strong>
              <span>${escapeHTML(String(models.length))}</span>
            </div>
            <div class="composer-runtime-option-list">
              ${models.map((modelItem) => {
                const activeClass = normalizeText(providerItem.id) === modelSelection.providerID && normalizeText(modelItem.id) === modelSelection.modelID ? " is-active" : "";
                return `<button class="composer-runtime-model-option${activeClass}" type="button" data-runtime-provider-id="${escapeHTML(providerItem.id)}" data-runtime-model-id="${escapeHTML(modelItem.id)}">
                  <strong>${escapeHTML(modelItem.name || modelItem.id)}</strong>
                  <span>${escapeHTML(providerItem.name || providerItem.id)}</span>
                </button>`;
              }).join("")}
            </div>
          </section>`;
        }).join("") : `<p class="composer-runtime-empty">${escapeHTML(t("chat.runtime.empty"))}</p>`}
      </div>` : ""}
    </div>
    <div class="composer-runtime-divider" aria-hidden="true"></div>
    <div class="composer-runtime-control">
      <button class="composer-runtime-trigger${openPopover === "capabilities" ? " is-open" : ""}" type="button" data-runtime-toggle="capabilities" title="${escapeHTML(t("chat.runtime.tools_hint"))}">
        <span class="composer-runtime-trigger-icon">🛠️</span>
        <span class="composer-runtime-trigger-label">${escapeHTML(toolLabel)}</span>
        <span class="composer-runtime-trigger-caret">▾</span>
      </button>
      ${openPopover === "capabilities" ? `<div class="composer-runtime-popover is-wide">
        <div class="composer-runtime-popover-head">
          <strong>${escapeHTML(t("chat.runtime.tools_mcp"))}</strong>
          <p>${escapeHTML(state.chatCatalog.capabilityError || t("chat.runtime.tools_hint"))}</p>
        </div>
        ${renderRuntimeGroupSection("chat.runtime.active", capabilityActive, () => "")}
        ${capabilityActive.length ? renderRuntimeCapabilityGroup(capabilityActive, capabilitySelected, "capabilities") : ""}
        <div class="composer-runtime-separator"></div>
        ${renderRuntimeGroupSection("chat.runtime.available", capabilityAvailable, () => "")}
        ${capabilityAvailable.length ? renderRuntimeCapabilityGroup(capabilityAvailable, capabilitySelected, "capabilities") : ""}
      </div>` : ""}
    </div>
    <div class="composer-runtime-control">
      <button class="composer-runtime-trigger${openPopover === "skills" ? " is-open" : ""}" type="button" data-runtime-toggle="skills" title="${escapeHTML(t("chat.runtime.skills_hint"))}">
        <span class="composer-runtime-trigger-icon">⚡</span>
        <span class="composer-runtime-trigger-label">${escapeHTML(skillLabel)}</span>
        <span class="composer-runtime-trigger-caret">▾</span>
      </button>
      ${openPopover === "skills" ? `<div class="composer-runtime-popover">
        <div class="composer-runtime-popover-head">
          <strong>${escapeHTML(t("chat.runtime.skills"))}</strong>
          <p>${escapeHTML(state.chatCatalog.capabilityError || t("chat.runtime.skills_hint"))}</p>
        </div>
        ${renderRuntimeGroupSection("chat.runtime.active", activeSkills, (item) => `<label class="composer-runtime-checkbox">
          <input type="checkbox" data-runtime-toggle-item="skills" value="${escapeHTML(item.id)}" checked>
          <span class="composer-runtime-checkbox-copy">
            <strong>${escapeHTML(item.name)}</strong>
            <span>${escapeHTML(item.description)}</span>
          </span>
        </label>`)}
        <div class="composer-runtime-separator"></div>
        ${renderRuntimeGroupSection("chat.runtime.available", availableSkills, (item) => `<label class="composer-runtime-checkbox">
          <input type="checkbox" data-runtime-toggle-item="skills" value="${escapeHTML(item.id)}">
          <span class="composer-runtime-checkbox-copy">
            <strong>${escapeHTML(item.name)}</strong>
            <span>${escapeHTML(item.description)}</span>
          </span>
        </label>`)}
      </div>` : ""}
    </div>
  </div>
  ${runtimeErrorNote ? `<p class="chat-runtime-note chat-runtime-error">${escapeHTML(runtimeErrorNote)}</p>` : ""}`;
  }

  chatRuntimePanel.querySelectorAll("[data-runtime-toggle]").forEach((node) => {
    node.addEventListener("click", () => {
      const nextPopover = String(node.getAttribute("data-runtime-toggle") || "").trim();
      if (!nextPopover) {
        return;
      }
      if (nextPopover === "target" && locked) {
        return;
      }
      state.chatRuntime.openPopover = state.chatRuntime.openPopover === nextPopover ? "" : nextPopover;
      renderChatRuntimePanel();
    });
  });

  chatRuntimePanel.querySelectorAll("[data-runtime-target-type]").forEach((node) => {
    node.addEventListener("click", () => {
      if (locked) {
        return;
      }
      const nextTarget = {
        type: node.getAttribute("data-runtime-target-type") || "model",
        id: node.getAttribute("data-runtime-target-id") || "",
        name: node.getAttribute("data-runtime-target-name") || ""
      };
      syncAgentRuntimeTarget(nextTarget);
      let session = ensureActiveConversationSession(state.currentRoute);
      if (!session) {
        session = createSession(nextTarget, mode, state.currentRoute);
      }
      state.chatRuntime.openPopover = "";
      syncMainChatComposerDraft(session.id, { preserveCurrent: false });
      renderSessions();
      renderMessages();
      syncHeader();
      syncWelcomeCopy();
      renderChatRuntimePanel();
      renderWelcomeTargetPicker();
    });
  });

  chatRuntimePanel.querySelectorAll("[data-runtime-provider-id]").forEach((node) => {
    node.addEventListener("click", () => {
      const session = getSession() || createSession(defaultChatTarget());
      updateSessionModelSelection(session, {
        providerID: node.getAttribute("data-runtime-provider-id") || "",
        modelID: node.getAttribute("data-runtime-model-id") || ""
      });
      state.chatRuntime.openPopover = "";
      persistSessions();
      renderSessions();
      syncHeader();
      renderChatRuntimePanel();
    });
  });

  chatRuntimePanel.querySelectorAll("[data-runtime-toggle-item]").forEach((inputNode) => {
    inputNode.addEventListener("change", () => {
      const session = getSession() || createSession(defaultChatTarget());
      const mode = String(inputNode.getAttribute("data-runtime-toggle-item") || "").trim();
      const value = normalizeText(inputNode.value);
      const current = sessionRuntimeSelections(session);
      if (mode === "skills") {
        const nextSkills = inputNode.checked
          ? normalizeSelectionIDs([...current.skillIDs, value])
          : current.skillIDs.filter((item) => item !== value);
        updateSessionRuntimeSelections(session, { skillIDs: nextSkills });
      } else {
        const itemKind = String(inputNode.getAttribute("data-runtime-item-kind") || "").trim();
        if (itemKind === "tool") {
          const nextTools = inputNode.checked
            ? normalizeSelectionIDs([...current.toolIDs, value])
            : current.toolIDs.filter((item) => item !== value);
          updateSessionRuntimeSelections(session, { toolIDs: nextTools });
        } else {
          const nextMcps = inputNode.checked
            ? normalizeSelectionIDs([...current.mcpIDs, value])
            : current.mcpIDs.filter((item) => item !== value);
          updateSessionRuntimeSelections(session, { mcpIDs: nextMcps });
        }
      }
      persistSessions();
      renderSessions();
      syncHeader();
      renderChatRuntimePanel();
    });
  });
}

function closeChatRuntimePopover() {
  if (!state.chatRuntime.openPopover) {
    return;
  }
  state.chatRuntime.openPopover = "";
  renderChatRuntimePanel();
}

function renderWelcomeTargetPicker() {
  const targetList = document.getElementById("welcomeTargetList");
  if (!targetList) {
    return;
  }
  const active = getSession();
  const currentTarget = sessionTarget(active);
  const agentRuntime = isAgentConversationRoute();
  const allowPicker = routeAllowsTargetPicker();
  const agents = agentRuntime ? genericRuntimeConversationAgents() : [];
  const buttons = [];
  if (agentRuntime && allowPicker) {
    agents.forEach((agent) => {
      const agentID = String(agent?.id || "").trim();
      if (!agentID) {
        return;
      }
      const agentName = String(agent?.name || agentID).trim() || agentID;
      const activeClassName = currentTarget.type === "agent" && currentTarget.id === agentID ? " active" : "";
      buttons.push(`<button class="welcome-target-card${activeClassName}" type="button" data-chat-target-type="agent" data-chat-target-id="${escapeHTML(agentID)}" data-chat-target-name="${escapeHTML(agentName)}">
        <strong>${escapeHTML(agentName)}</strong>
        <span>${escapeHTML(agentID)}</span>
      </button>`);
    });
  } else if (agentRuntime) {
    const fallbackTarget = routeDefaultTarget();
    const targetName = currentTarget.name || fallbackTarget.name || t("session.target.agent");
    const targetID = currentTarget.id || fallbackTarget.id || "";
    buttons.push(`<div class="welcome-target-card active is-static">
      <strong>${escapeHTML(targetName)}</strong>
      <span>${escapeHTML(targetID || t("session.target.agent"))}</span>
    </div>`);
  } else {
    buttons.push(`<div class="welcome-target-card active is-static">
      <strong>${escapeHTML(t("session.target.raw"))}</strong>
      <span>${escapeHTML(t("route.chat.subtitle"))}</span>
    </div>`);
  }
  if (agentRuntime && allowPicker && state.chatCatalog.error) {
    buttons.push(`<p class="welcome-target-error">${escapeHTML(state.chatCatalog.error)}</p>`);
  }
  targetList.innerHTML = buttons.join("");
  targetList.querySelectorAll("[data-chat-target-type]").forEach((node) => {
    node.addEventListener("click", () => {
      const target = {
        type: node.getAttribute("data-chat-target-type") || "model",
        id: node.getAttribute("data-chat-target-id") || "",
        name: node.getAttribute("data-chat-target-name") || ""
      };
      syncAgentRuntimeTarget(target);
      let session = ensureActiveConversationSession(state.currentRoute);
      if (!session) {
        session = createSession(target, routeConversationMode(state.currentRoute), state.currentRoute);
      }
      syncMainChatComposerDraft(session.id, { preserveCurrent: false });
      renderSessions();
      renderMessages();
      syncHeader();
      syncWelcomeCopy();
      renderChatRuntimePanel();
      renderWelcomeTargetPicker();
    });
  });
}

function openAgentRuntimeWithTarget(target) {
  const normalizedTarget = normalizeChatTarget(target);
  const agent = findChatAgent(normalizedTarget.id);
  const dedicatedRoute = agentConversationRoute(agent);
  if (dedicatedRoute) {
    navigateToRoute(dedicatedRoute, { skipConfirm: true });
    window.requestAnimationFrame(() => {
      input.focus();
    });
    return;
  }
  syncAgentRuntimeTarget(normalizedTarget);
  let session = ensureActiveConversationSession("agent-runtime");
  if (!session) {
    session = createSession(normalizedTarget, "agent", "agent-runtime");
  }
  setActiveConversationSessionID(session.id, "agent-runtime");
  navigateToRoute("agent-runtime", { skipConfirm: true });
  window.requestAnimationFrame(() => {
    syncMainChatComposerDraft(session.id, { preserveCurrent: false });
    input.focus();
  });
}

function createSession(target = null, mode = routeConversationMode(), route = state.currentRoute) {
  const latestBlank = getLatestBlankSession(route);
  if (latestBlank) {
    if (target) {
      updateSessionTarget(latestBlank, target);
    }
    setActiveConversationSessionID(latestBlank.id, route);
    syncMainChatComposerDraft(latestBlank.id);
    renderSessions();
    renderMessages();
    syncHeader();
    renderWelcomeTargetPicker();
    persistSessions(routeConversationMode(route));
    return latestBlank;
  }

  const createdAt = Date.now();
  const defaultTarget = routeDefaultTarget(route);
  const normalizedTarget = normalizeChatTarget(target || defaultTarget);
  const runtimeDefaults = defaultRuntimeSelectionsForTarget(normalizedTarget);
  const item = {
    id: makeID(),
    title: t(routeAllowsTargetPicker(route) ? "session.new_agent_title" : "session.new_title"),
    titleAuto: true,
    titleScore: 0,
    createdAt,
    messages: [],
    targetType: normalizedTarget.type,
    targetID: normalizedTarget.id,
    targetName: normalizedTarget.name,
    modelProviderID: defaultChatModelSelection().providerID,
    modelID: defaultChatModelSelection().modelID,
    toolIDs: runtimeDefaults.toolIDs,
    skillIDs: runtimeDefaults.skillIDs,
    mcpIDs: runtimeDefaults.mcpIDs,
    historyBucket: conversationHistoryBucketForTarget(normalizedTarget)
  };
  const sessions = conversationSessions(route).slice();
  sessions.unshift(item);
  setConversationSessions(sessions, route);
  setActiveConversationSessionID(item.id, route);
  syncMainChatComposerDraft(item.id);
  renderSessions();
  renderMessages();
  syncHeader();
  syncWelcomeCopy();
  renderChatRuntimePanel();
  renderWelcomeTargetPicker();
  persistSessions(routeConversationMode(route));
  return item;
}

async function removeSession(sessionID) {
  const activeSessionID = activeConversationSessionID();
  const removedActiveSession = activeSessionID === sessionID;
  if (activeSessionID === sessionID && !confirmComposerNavigation()) {
    return;
  }
  const currentSessions = conversationSessions();
  const nextSessions = currentSessions.filter((item) => item.id !== sessionID);
  if (nextSessions.length === currentSessions.length) {
    return;
  }

  try {
    await deleteServerSession(sessionID);
  } catch (error) {
    const message = error instanceof Error ? error.message : "unknown_error";
    window.alert(t("session.delete_failed", { error: message }));
    return;
  }

  setConversationSessions(nextSessions);
  if (activeSessionID === sessionID || !getSession(activeSessionID)) {
    const latestBlank = getLatestBlankSession();
    if (latestBlank) {
      setActiveConversationSessionID(latestBlank.id);
    } else if (conversationSessions().length) {
      setActiveConversationSessionID(conversationSessions()[0].id);
    } else {
      setActiveConversationSessionID("");
    }
  }

  clearMainChatDraft(sessionID);
  if (removedActiveSession) {
    syncMainChatComposerDraft(activeConversationSessionID(), { preserveCurrent: false });
  }
  enforceSingleBlankSession();
  renderSessions();
  renderMessages();
  syncHeader();
  renderChatRuntimePanel();
  renderWelcomeTargetPicker();
  persistSessions();
}

function syncHeader() {
  const route = ROUTES[state.currentRoute] || ROUTES.chat;
  const newSessionLabel = routeAllowsTargetPicker() ? t("session.new_agent") : t("session.new");
  newChatButton.textContent = newSessionLabel;
  sessionToggle.hidden = false;
  if (mobileNewChatButton) {
    if (route.key === "terminal") {
      mobileNewChatButton.textContent = t("route.terminal.new_short");
      mobileNewChatButton.setAttribute("aria-label", t("route.terminal.new_short"));
      mobileNewChatButton.hidden = false;
      sessionToggle.textContent = t("route.terminal.sessions");
      sessionToggle.setAttribute("aria-label", t("route.terminal.sessions"));
    } else {
      mobileNewChatButton.textContent = newSessionLabel;
      mobileNewChatButton.setAttribute("aria-label", newSessionLabel);
      mobileNewChatButton.hidden = false;
      sessionToggle.textContent = t("chat.sessions");
      sessionToggle.setAttribute("aria-label", t("chat.sessions"));
    }
  }

  if (route.mode !== "chat") {
    sessionHeading.textContent = "alter0";
    sessionSubheading.textContent = t("chat.menu");
    return;
  }

  const routeKey = route.key || "chat";
  const titleKey = `route.${routeKey}.title`;
  const subtitleKey = `route.${routeKey}.subtitle`;
  const active = getSession();
  if (!active) {
    sessionHeading.textContent = t(titleKey);
    sessionSubheading.textContent = t(subtitleKey);
    return;
  }
  sessionHeading.textContent = active.title;
  const targetLabel = sessionTargetLabel(active);
  const modelLabel = sessionModelLabel(active);
  if (active.messages.length === 0) {
    sessionSubheading.textContent = routeAllowsTargetPicker()
      ? `${targetLabel} · ${modelLabel} · ${t("session.empty_agent_sub")}`
      : `${targetLabel} · ${modelLabel} · ${t("session.empty_sub")}`;
    return;
  }
  sessionSubheading.textContent = `${targetLabel} · ${modelLabel} · ${active.messages.length} messages`;
}

function syncWelcomeCopy() {
  const active = getSession();
  const workspaceTarget = sessionTarget(active || null);
  const workspaceAgentName = workspaceTarget.type === "agent" && workspaceTarget.name
    ? workspaceTarget.name
    : MAIN_AGENT_NAME;
  if (!active) {
    welcomeHeading.textContent = t("welcome.heading");
    welcomeDescription.textContent = routeAllowsTargetPicker() ? t("session.no_active_agent") : t("session.no_active");
    return;
  }
  welcomeHeading.textContent = t("welcome.heading");
  welcomeDescription.textContent = routeAllowsTargetPicker()
    ? `${t("welcome.desc")} ${t("welcome.agent_hint")}`
    : `${t("welcome.desc")} ${t("welcome.model_hint", { agent: workspaceAgentName })}`;
}

function syncSessionLoadHint() {
  const message = conversationSessionLoadError();
  sessionLoadError.textContent = message;
  sessionLoadError.style.display = message ? "block" : "none";
}

function renderSessions() {
  sessionList.innerHTML = "";
  syncSessionLoadHint();
  const sessions = conversationSessions();
  const activeSessionID = activeConversationSessionID();
  if (!sessions.length) {
    sessionEmpty.textContent = routeAllowsTargetPicker() ? t("session.empty_agent") : t("session.empty");
    sessionEmpty.style.display = "block";
    return;
  }
  sessionEmpty.style.display = "none";

  for (const item of sessions) {
    const row = document.createElement("div");
    row.className = "session-card-row";

    const card = document.createElement("button");
    card.type = "button";
    card.className = "session-card";
    card.dataset.sessionId = item.id;
    card.setAttribute("role", "option");
    card.setAttribute("aria-selected", item.id === activeSessionID ? "true" : "false");
    if (item.id === activeSessionID) {
      card.classList.add("active");
    }

    const title = document.createElement("p");
    title.className = "session-card-title";
    title.textContent = item.title;

    const meta = document.createElement("p");
    meta.className = "session-card-meta";
    meta.textContent = `${sessionTargetBadgeLabel(item)} · ${sessionModelLabel(item)} · ${item.messages.length} messages · ${formatSince(item.createdAt)}`;

    card.appendChild(title);
    card.appendChild(meta);
    card.addEventListener("click", () => {
      focusSession(item.id);
    });

    const deleteButton = document.createElement("button");
    deleteButton.type = "button";
    deleteButton.className = "session-card-delete";
    deleteButton.textContent = t("session.delete");
    deleteButton.setAttribute("aria-label", t("session.delete"));
    deleteButton.addEventListener("click", async (event) => {
      event.stopPropagation();
      await removeSession(item.id);
    });

    row.appendChild(card);
    row.appendChild(deleteButton);
    sessionList.appendChild(row);
  }
}

function updateSessionTitle(session, fallbackText) {
  const currentState = inferStoredSessionTitleState(session.title, "");
  const currentAuto = typeof session.titleAuto === "boolean" ? session.titleAuto : currentState.titleAuto;
  const currentScore = Number.isFinite(session.titleScore) ? Math.max(Number(session.titleScore), 0) : currentState.titleScore;
  if (!currentAuto && !isDefaultSessionTitle(session.title)) {
    return;
  }
  const nextTitle = buildAutoSessionTitle(fallbackText, 18);
  if (!nextTitle.title) {
    return;
  }
  const currentTitle = normalizeSessionTitleValue(session.title);
  const canReplacePlaceholder = (isDefaultSessionTitle(currentTitle) || !currentTitle) && nextTitle.title !== session.title;
  if (!canReplacePlaceholder && nextTitle.titleScore <= currentScore) {
    session.titleAuto = currentAuto;
    session.titleScore = currentScore;
    return;
  }
  session.title = nextTitle.title;
  session.titleAuto = nextTitle.titleAuto;
  session.titleScore = nextTitle.titleScore;
  persistSessions();
}

function appendMessageToSession(session, role, text, options = {}) {
  if (!session) {
    return null;
  }
  if (role === "user") {
    updateSessionTitle(session, text);
  }
  const message = {
    id: makeID(),
    role,
    text,
    at: Date.now(),
    route: options.route || "",
    source: typeof options.source === "string" ? options.source : "",
    error: Boolean(options.error),
    status: options.status || (options.error ? "error" : "done"),
    retryable: Boolean(options.retryable),
    task_id: typeof options.task_id === "string" ? options.task_id : "",
    task_status: typeof options.task_status === "string" ? options.task_status : "",
    task_pending: Boolean(options.task_pending),
    task_result_delivered: Boolean(options.task_result_delivered),
    task_result_for: typeof options.task_result_for === "string" ? options.task_result_for : "",
    task_completed_at: Number.isFinite(options.task_completed_at) ? options.task_completed_at : 0,
    agent_process_collapsed: typeof options.agent_process_collapsed === "boolean" ? options.agent_process_collapsed : undefined
  };
  session.messages.push(message);
  enforceSingleBlankSession();
  renderSessions();
  scheduleMessagesRender();
  syncHeader();
  persistSessions();
  return message;
}

function appendMessage(role, text, options = {}) {
  let session = getSession();
  if (!session) {
    session = getLatestBlankSession();
    if (session) {
      setActiveConversationSessionID(session.id);
    } else {
      session = createSession();
    }
  }
  return appendMessageToSession(session, role, text, options);
}

function updateMessage(message, patch = {}) {
  if (!message) {
    return;
  }
  Object.assign(message, patch);
  scheduleMessagesRender();
  syncHeader();
  persistSessions();
}

function assistantStatusLabel(status) {
  if (status === "streaming") {
    return t("status.in_progress");
  }
  if (status === "queued") {
    return t("status.queued");
  }
  if (status === "running") {
    return t("status.running");
  }
  if (status === "canceled") {
    return t("status.canceled");
  }
  if (status === "success") {
    return t("status.success");
  }
  if (status === "failed") {
    return t("status.failed");
  }
  if (status === "error") {
    return t("status.failed");
  }
  return t("status.done");
}

function messageSourceLabel(source) {
  const normalized = normalizeText(source).toLowerCase();
  if (normalized === "model") {
    return "MODEL";
  }
  if (normalized === "codex_cli") {
    return "CODEX CLI";
  }
  return "";
}

function normalizeTaskStatus(status) {
  return normalizeText(status).toLowerCase();
}

function isTerminalTaskStatus(status) {
  const normalized = normalizeTaskStatus(status);
  return normalized === "success" || normalized === "failed" || normalized === "canceled";
}

function applyAsyncTaskStateToMessage(message, payload = {}) {
  if (!message) {
    return;
  }
  const taskID = normalizeText(payload.task_id || payload.taskID || message.task_id);
  if (!taskID) {
    return;
  }
  const taskStatus = normalizeTaskStatus(payload.task_status || payload.status || message.task_status || "queued");
  updateMessage(message, {
    task_id: taskID,
    task_status: taskStatus,
    task_pending: !isTerminalTaskStatus(taskStatus),
    status: taskStatus || message.status || "done",
    error: taskStatus === "failed" || taskStatus === "canceled" ? Boolean(message.error) : false
  });
}

function taskCompletionText(task) {
  const status = normalizeTaskStatus(task?.status);
  if (status === "success") {
    return normalizeText(task?.summary) || normalizeText(task?.task_summary?.result) || t("msg.received_empty");
  }
  if (status === "canceled") {
    return normalizeText(task?.summary) || `Async task ${normalizeText(task?.id)} canceled`;
  }
  return normalizeText(task?.summary) || normalizeText(task?.task_summary?.result) || normalizeText(task?.result?.error_code) || `Async task ${normalizeText(task?.id)} failed`;
}

function notifyAsyncTaskCompletion(session, task, text) {
  if (!session || !task || !document.hidden || typeof window.Notification === "undefined") {
    return;
  }
  if (window.Notification.permission !== "granted") {
    return;
  }
  const title = session.title || t("chat.title");
  try {
    new window.Notification(title, {
      body: shorten(normalizeText(text), 120)
    });
  } catch {
  }
}

function deliverAsyncTaskResult(session, message, task) {
  if (!session || !message || !task) {
    return;
  }
  const taskID = normalizeText(task.id || message.task_id);
  if (!taskID || message.task_result_delivered) {
    return;
  }
  const taskStatus = normalizeTaskStatus(task.status);
  const text = taskCompletionText(task);
  updateMessage(message, {
    task_id: taskID,
    task_status: taskStatus,
    task_pending: false,
    task_result_delivered: true,
    task_completed_at: Date.now(),
    status: taskStatus || "done"
  });
  const alreadyDelivered = session.messages.some((item) => item !== message && normalizeText(item.task_result_for) === taskID);
  if (!alreadyDelivered) {
    appendMessageToSession(session, "assistant", text, {
      route: typeof task?.result?.route === "string" ? task.result.route : "",
      source: normalizeText(task?.result?.metadata?.["alter0.execution.source"]),
      error: taskStatus === "failed" || taskStatus === "canceled",
      status: taskStatus === "failed" || taskStatus === "canceled" ? "error" : "done",
      retryable: taskStatus === "failed",
      task_result_for: taskID,
      task_completed_at: Date.now()
    });
  }
  notifyAsyncTaskCompletion(session, task, text);
}

function collectPendingTaskBindings() {
  const bindings = [];
  for (const session of storedConversationSessions()) {
    const messages = Array.isArray(session?.messages) ? session.messages : [];
    for (const message of messages) {
      const taskID = normalizeText(message?.task_id);
      if (!taskID || message?.task_result_delivered) {
        continue;
      }
      bindings.push({ session, message, taskID });
    }
  }
  return bindings;
}

function stopChatTaskPolling() {
  if (!chatTaskPollTimer) {
    return;
  }
  window.clearTimeout(chatTaskPollTimer);
  chatTaskPollTimer = 0;
}

function scheduleChatTaskPolling(options = {}) {
  stopChatTaskPolling();
  if (!collectPendingTaskBindings().length) {
    return;
  }
  const delay = options.immediate
    ? 0
    : (isDocumentVisible() ? CHAT_TASK_POLL_INTERVAL_MS : CHAT_TASK_POLL_HIDDEN_INTERVAL_MS);
  chatTaskPollTimer = window.setTimeout(async () => {
    chatTaskPollTimer = 0;
    await pollChatTaskUpdates();
    scheduleChatTaskPolling();
  }, delay);
}

async function pollChatTaskUpdates() {
  if (chatTaskPollPending) {
    return;
  }
  const bindings = collectPendingTaskBindings();
  if (!bindings.length) {
    return;
  }
  chatTaskPollPending = true;
  try {
    const taskMap = new Map();
    for (const binding of bindings) {
      if (taskMap.has(binding.taskID)) {
        continue;
      }
      try {
        const task = await fetchJSON(`/api/tasks/${encodeURIComponent(binding.taskID)}`);
        taskMap.set(binding.taskID, task || null);
      } catch {
        taskMap.set(binding.taskID, null);
      }
    }
    bindings.forEach(({ session, message, taskID }) => {
      const task = taskMap.get(taskID);
      if (!task) {
        return;
      }
      applyAsyncTaskStateToMessage(message, {
        task_id: taskID,
        task_status: task.status
      });
      if (isTerminalTaskStatus(task.status)) {
        deliverAsyncTaskResult(session, message, task);
      }
    });
  } finally {
    chatTaskPollPending = false;
  }
}

function ensureChatTaskPolling() {
  scheduleChatTaskPolling({ immediate: true });
}

function extractAsyncTaskPayload(payload) {
  const taskID = normalizeText(payload?.task_id);
  if (!taskID) {
    return null;
  }
  return {
    task_id: taskID,
    task_status: normalizeTaskStatus(payload?.task_status || "queued")
  };
}

function extractMessageSource(result) {
  return normalizeText(result?.metadata?.["alter0.execution.source"]);
}

function renderMessages(options = {}) {
  if (state.messageRenderFrame) {
    window.cancelAnimationFrame(state.messageRenderFrame);
    state.messageRenderFrame = 0;
    state.pendingMessageRenderPreserveScroll = false;
  }
  const active = getSession();
  const hasMessages = Boolean(active && active.messages.length);
  welcomeScreen.style.display = hasMessages ? "none" : "block";
  messageArea.style.display = hasMessages ? "block" : "none";
  chatPane.classList.toggle("empty-state", !hasMessages);
  const preserveScrollPosition = Boolean(options.preserveScrollPosition);
  const previousScrollTop = messageArea.scrollTop;
  const previousScrollHeight = messageArea.scrollHeight;

  if (!hasMessages) {
    syncWelcomeCopy();
    renderWelcomeTargetPicker();
    messageArea.innerHTML = "";
    state.messageRenderSignatures.clear();
    return;
  }
  syncMessageList(active);
  if (preserveScrollPosition) {
    messageArea.scrollTop = Math.max(0, previousScrollTop + (messageArea.scrollHeight - previousScrollHeight));
  } else {
    messageArea.scrollTop = messageArea.scrollHeight;
  }
}

function toggleAgentProcessMessage(messageID) {
  const active = getSession();
  const normalizedMessageID = normalizeText(messageID);
  if (!active || !normalizedMessageID) {
    return;
  }
  const message = active.messages.find((item) => normalizeText(item?.id) === normalizedMessageID);
  if (!message) {
    return;
  }
  const parsed = parseAgentExecutionText(message.text || "");
  if (!parsed.steps.length) {
    return;
  }
  message.agent_process_collapsed = !resolveAgentProcessCollapsed(message, parsed);
  scheduleMessagesRender({ preserveScrollPosition: true });
  persistSessions();
}

function scheduleMessagesRender(options = {}) {
  state.pendingMessageRenderPreserveScroll = state.pendingMessageRenderPreserveScroll || Boolean(options.preserveScrollPosition);
  if (state.messageRenderFrame) {
    return;
  }
  state.messageRenderFrame = window.requestAnimationFrame(() => {
    const preserveScrollPosition = Boolean(state.pendingMessageRenderPreserveScroll);
    state.messageRenderFrame = 0;
    state.pendingMessageRenderPreserveScroll = false;
    renderMessages({ preserveScrollPosition });
  });
}

function captureMessageRenderSignature(message) {
  if (!message) {
    return "";
  }
  return [
    normalizeText(message.id),
    normalizeText(message.role),
    String(message.text || ""),
    normalizeText(message.route),
    normalizeText(message.source),
    String(Boolean(message.error)),
    normalizeText(message.status),
    String(Number(message.at || 0)),
    String(Boolean(message.retryable)),
    String(Boolean(message.task_pending)),
    normalizeText(message.task_id),
    normalizeText(message.task_status),
    String(Boolean(message.task_result_delivered)),
    normalizeText(message.task_result_for),
    String(Number(message.task_completed_at || 0)),
    String(message.agent_process_collapsed)
  ].join("|");
}

function renderMessageMetaHTML(message) {
  const segments = [];
  if (message.route && message.role === "assistant") {
    segments.push(`<span class="route-pill">${escapeHTML(String(message.route || "").toUpperCase())}</span>`);
  }
  if (message.role === "assistant") {
    const sourceLabel = messageSourceLabel(message.source);
    if (sourceLabel) {
      segments.push(`<span class="source-pill">${escapeHTML(sourceLabel)}</span>`);
    }
    segments.push(`<span class="status-pill ${escapeHTML(message.status || "done")}">${escapeHTML(assistantStatusLabel(message.status))}</span>`);
  }
  segments.push(`<span>${escapeHTML(timeLabel(message.at))}</span>`);
  return segments.join("");
}

function renderMessageArticleHTML(message) {
  const classes = ["msg", escapeHTML(message.role)];
  if (message.error) {
    classes.push("error");
  }
  if (message.status === "streaming") {
    classes.push("streaming");
  }
  const bubbleHTML = message.role === "assistant"
    ? renderAgentExecutionMessage(message)
    : renderMarkdownToHTML(message.text);
  return `<article class="${classes.join(" ")}" data-message-id="${escapeHTML(message.id)}">
    <div class="msg-bubble">${bubbleHTML}</div>
    <div class="msg-meta">${renderMessageMetaHTML(message)}</div>
  </article>`;
}

function findMessageArticleNode(listNode, messageID) {
  if (!(listNode instanceof HTMLElement) || !messageID) {
    return null;
  }
  const escapedMessageID = window.CSS && typeof window.CSS.escape === "function"
    ? window.CSS.escape(messageID)
    : messageID.replace(/["\\]/g, "\\$&");
  return listNode.querySelector(`[data-message-id="${escapedMessageID}"]`);
}

function ensureMessageListNode(sessionID) {
  let listNode = messageArea.querySelector(".message-list");
  const normalizedSessionID = normalizeText(sessionID);
  if (!(listNode instanceof HTMLElement)) {
    messageArea.innerHTML = `<div class="message-list" data-message-session-id="${escapeHTML(normalizedSessionID)}"></div>`;
    listNode = messageArea.querySelector(".message-list");
  }
  if (!(listNode instanceof HTMLElement)) {
    return null;
  }
  if (normalizeText(listNode.getAttribute("data-message-session-id")) !== normalizedSessionID) {
    listNode.innerHTML = "";
    listNode.setAttribute("data-message-session-id", normalizedSessionID);
    state.messageRenderSignatures.clear();
  }
  return listNode;
}

function syncMessageList(session) {
  const messages = Array.isArray(session?.messages) ? session.messages : [];
  const listNode = ensureMessageListNode(session?.id || "");
  if (!(listNode instanceof HTMLElement)) {
    return;
  }
  const existingByID = new Map(
    [...listNode.querySelectorAll("[data-message-id]")].map((node) => [normalizeText(node.getAttribute("data-message-id")), node])
  );
  const nextIDs = new Set();
  let previousNode = null;

  messages.forEach((message) => {
    const messageID = normalizeText(message?.id);
    if (!messageID || messageID === "-") {
      return;
    }
    nextIDs.add(messageID);
    const signature = captureMessageRenderSignature(message);
    let node = existingByID.get(messageID) || null;
    if (!node) {
      const html = renderMessageArticleHTML(message);
      if (previousNode instanceof HTMLElement) {
        previousNode.insertAdjacentHTML("afterend", html);
        node = previousNode.nextElementSibling;
      } else {
        listNode.insertAdjacentHTML("afterbegin", html);
        node = listNode.firstElementChild;
      }
      state.messageRenderSignatures.set(messageID, signature);
    } else if (state.messageRenderSignatures.get(messageID) !== signature) {
      node.outerHTML = renderMessageArticleHTML(message);
      node = findMessageArticleNode(listNode, messageID);
      state.messageRenderSignatures.set(messageID, signature);
    }
    if (!(node instanceof HTMLElement)) {
      return;
    }
    const expectedNode = previousNode instanceof HTMLElement ? previousNode.nextElementSibling : listNode.firstElementChild;
    if (node !== expectedNode) {
      listNode.insertBefore(node, expectedNode || null);
    }
    previousNode = node;
  });

  existingByID.forEach((node, messageID) => {
    if (nextIDs.has(messageID)) {
      return;
    }
    state.messageRenderSignatures.delete(messageID);
    if (node instanceof HTMLElement) {
      node.remove();
    }
  });
}

function setPending(flag) {
  state.pendingCount = Math.max(0, Number(state.pendingCount || 0) + (flag ? 1 : -1));
  state.pending = state.pendingCount > 0;
  if (mainChatComposer) {
    mainChatComposer.setDisabled(false);
    mainChatComposer.syncCounter();
    return;
  }
  sendButton.disabled = false;
  input.disabled = false;
}

function updateCharCount() {
  if (mainChatComposer) {
    mainChatComposer.syncCounter();
    return;
  }
  const value = input.value.slice(0, MAX_CHARS);
  if (value.length !== input.value.length) {
    input.value = value;
  }
  charCount.textContent = `${value.length}/${MAX_CHARS}`;
}

async function safeReadJSON(response) {
  try {
    return await response.json();
  } catch {
    return {};
  }
}

function parseSSEBlock(block) {
  const lines = block.split("\n");
  let event = "message";
  const dataLines = [];
  for (const line of lines) {
    if (!line || line.startsWith(":")) {
      continue;
    }
    if (line.startsWith("event:")) {
      event = line.slice(6).trim();
      continue;
    }
    if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  }

  if (!dataLines.length) {
    return null;
  }

  const rawData = dataLines.join("\n");
  try {
    return { event, data: JSON.parse(rawData) };
  } catch {
    return { event, data: { raw: rawData } };
  }
}

async function sendMessageStream(payload, assistantMessage, endpoints = {}) {
  let sawEvent = false;
  let sawDone = false;
  let routeHint = "";
  let output = "";
  const streamEndpoint = String(endpoints.stream || STREAM_ENDPOINT);

  try {
    const response = await fetch(streamEndpoint, {
      method: "POST",
      headers: {
        Accept: "text/event-stream",
        "Content-Type": "application/json"
      },
      body: JSON.stringify(payload)
    });

    if (!response.ok) {
      const body = await safeReadJSON(response);
      const failure = body.error || body?.result?.error_code || `HTTP ${response.status}`;
      return { ok: false, canFallback: true, error: failure };
    }

    const contentType = response.headers.get("content-type") || "";
    if (!contentType.includes("text/event-stream") || !response.body) {
      return { ok: false, canFallback: true, error: "streaming endpoint unavailable" };
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";
    let streamError = "";

    while (true) {
      const { value, done } = await reader.read();
      if (done) {
        buffer += decoder.decode();
        buffer += "\n\n";
      } else {
        buffer += decoder.decode(value, { stream: true });
      }

      let splitAt = buffer.indexOf("\n\n");
      while (splitAt >= 0) {
        const block = buffer.slice(0, splitAt).replace(/\r/g, "");
        buffer = buffer.slice(splitAt + 2);

        const parsed = parseSSEBlock(block);
        if (parsed) {
          sawEvent = true;
          if (parsed.event === "start") {
            updateMessage(assistantMessage, {
              status: "streaming",
              error: false,
              retryable: false,
              text: output || t("msg.processing")
            });
          } else if (parsed.event === "delta") {
            const delta = typeof parsed.data.delta === "string" ? parsed.data.delta : "";
            if (typeof parsed.data.route === "string" && parsed.data.route) {
              routeHint = parsed.data.route;
            }
            if (delta) {
              output += delta;
              updateMessage(assistantMessage, {
                text: output,
                route: routeHint,
                status: "streaming",
                at: Date.now()
              });
            }
          } else if (parsed.event === "done") {
            const result = parsed.data && typeof parsed.data === "object" ? parsed.data.result || {} : {};
            const route = typeof result.route === "string" && result.route ? result.route : routeHint;
            const resultOutput = typeof result.output === "string" ? result.output : "";
            const finalOutput = resultOutput.trim() ? resultOutput : output;
            const asyncTask = extractAsyncTaskPayload(parsed.data);
            const source = extractMessageSource(result);
            updateMessage(assistantMessage, {
              text: finalOutput.trim() || t("msg.received_empty"),
              route,
              source: source || assistantMessage.source,
              error: false,
              status: asyncTask ? asyncTask.task_status : "done",
              retryable: false,
              at: Date.now(),
              task_id: asyncTask ? asyncTask.task_id : assistantMessage.task_id,
              task_status: asyncTask ? asyncTask.task_status : assistantMessage.task_status,
              task_pending: Boolean(asyncTask),
              task_result_delivered: false
            });
            if (asyncTask) {
              ensureChatTaskPolling();
            }
            sawDone = true;
          } else if (parsed.event === "error") {
            const result = parsed.data && typeof parsed.data === "object" ? parsed.data.result || {} : {};
            if (typeof result.route === "string" && result.route) {
              routeHint = result.route;
            }
            const message = typeof parsed.data.error === "string" && parsed.data.error ? parsed.data.error : t("msg.stream_error");
            updateMessage(assistantMessage, {
              text: t("msg.stream_failed", { error: message }),
              route: routeHint,
              error: true,
              status: "error",
              retryable: true,
              at: Date.now()
            });
            streamError = message;
          }
        }

        splitAt = buffer.indexOf("\n\n");
      }

      if (streamError) {
        await reader.cancel();
        return { ok: false, canFallback: false, error: streamError };
      }
      if (done) {
        break;
      }
    }

    if (sawDone) {
      return { ok: true, canFallback: false, error: "" };
    }

    return {
      ok: false,
      canFallback: !sawEvent,
      error: sawEvent ? "Stream connection interrupted" : "streaming endpoint unavailable"
    };
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown network error";
    return { ok: false, canFallback: !sawEvent, error: message };
  }
}

async function sendMessageFallback(payload, assistantMessage, endpoints = {}) {
  const fallbackEndpoint = String(endpoints.fallback || FALLBACK_ENDPOINT);
  const response = await fetch(fallbackEndpoint, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });

  const body = await safeReadJSON(response);
  if (!response.ok) {
    const failure = body.error || body?.result?.error_code || `HTTP ${response.status}`;
    updateMessage(assistantMessage, {
      text: t("msg.request_failed", { error: failure }),
      route: body?.result?.route || "",
      error: true,
      status: "error",
      retryable: true,
      at: Date.now()
    });
    return;
  }

  const output = (body?.result?.output || "").trim() || t("msg.received_empty");
  const asyncTask = extractAsyncTaskPayload(body);
  const source = extractMessageSource(body?.result || {});
  updateMessage(assistantMessage, {
    text: output,
    route: body?.result?.route || "",
    source: source || assistantMessage.source,
    error: false,
    status: asyncTask ? asyncTask.task_status : "done",
    retryable: false,
    at: Date.now(),
    task_id: asyncTask ? asyncTask.task_id : assistantMessage.task_id,
    task_status: asyncTask ? asyncTask.task_status : assistantMessage.task_status,
    task_pending: Boolean(asyncTask),
    task_result_delivered: false
  });
  if (asyncTask) {
    ensureChatTaskPolling();
  }
}

async function sendMessage(rawContent) {
  const route = (ROUTES[state.currentRoute] || ROUTES.chat).mode === "chat" ? state.currentRoute : "chat";
  if (state.currentRoute !== route) {
    navigateToRoute(route);
  }
  const content = rawContent.trim();
  if (!content) {
    return;
  }

  const active = getSession();
  if (syncSessionTargetForRoute(active, route)) {
    persistSessions();
    renderSessions();
    syncHeader();
    renderWelcomeTargetPicker();
  }
  const target = sessionTarget(active, route);
  const isAgentSession = isAgentConversationRoute();
  if (isAgentSession && !target.id) {
    appendMessage("assistant", t("route.agent.pick"), {
      error: true,
      status: "error",
      retryable: false
    });
    return;
  }

  appendMessage("user", content);
  input.value = "";
  mainChatComposer.clearDraft();
  updateCharCount();
  closeChatRuntimePopover();
  setPending(true);

  const activeSession = getSession();
  const selection = resolveEffectiveChatModelSelection(activeSession);
  const runtimeSelections = sessionRuntimeSelections(activeSession);
  const metadata = {};
  if (selection.providerID) {
    metadata["alter0.llm.provider_id"] = selection.providerID;
  }
  if (selection.modelID) {
    metadata["alter0.llm.model"] = selection.modelID;
  }
  metadata["alter0.agent.tools"] = JSON.stringify(runtimeSelections.toolIDs);
  metadata["alter0.skills.include"] = JSON.stringify(runtimeSelections.skillIDs);
  metadata["alter0.mcp.request.enable"] = JSON.stringify(runtimeSelections.mcpIDs);
  const payload = isAgentSession
    ? {
      agent_id: target.id,
      session_id: activeSession ? activeSession.id : "",
      channel_id: "web-default",
      content,
      metadata
    }
    : {
      session_id: activeSession ? activeSession.id : "",
      channel_id: "web-default",
      content,
      metadata
    };
  const endpoints = isAgentSession
    ? {
      stream: "/api/agent/messages/stream",
      fallback: "/api/agent/messages"
    }
    : {
      stream: STREAM_ENDPOINT,
      fallback: FALLBACK_ENDPOINT
    };
  const assistantMessage = appendMessage("assistant", t("msg.processing"), { status: "streaming" });

  try {
    const streamResult = await sendMessageStream(payload, assistantMessage, endpoints);
    if (streamResult.ok) {
      return;
    }

    if (streamResult.canFallback) {
      await sendMessageFallback(payload, assistantMessage, endpoints);
      return;
    }

    if (assistantMessage.status !== "error") {
      updateMessage(assistantMessage, {
        text: t("msg.stream_failed", { error: streamResult.error || "unknown" }),
        error: true,
        status: "error",
        retryable: true,
        at: Date.now()
      });
    }
  } catch (err) {
    const message = err instanceof Error ? err.message : "unknown_network_error";
    updateMessage(assistantMessage, {
      text: t("msg.network_error", { error: message }),
      error: true,
      status: "error",
      retryable: true,
      at: Date.now()
    });
  } finally {
    setPending(false);
    input.focus();
  }
}

function activeMenuRoute(route) {
  for (const node of menuRouteItems) {
    node.classList.toggle("active", node.dataset.route === route);
  }
}

function parseHashRoute() {
  const raw = window.location.hash.replace(/^#\/?/, "").trim().toLowerCase();
  if (!raw) {
    return DEFAULT_ROUTE;
  }
  return ROUTES[raw] ? raw : DEFAULT_ROUTE;
}

function isMobileViewport() {
  return window.matchMedia("(max-width: 1100px)").matches;
}

function isTerminalSessionSheetViewport() {
  return window.matchMedia("(max-width: 760px)").matches;
}

function setSidebarCollapsed(collapsed) {
  state.navCollapsed = collapsed;
  appShell.classList.toggle("nav-collapsed", collapsed);
  navCollapseButton.setAttribute("aria-expanded", collapsed ? "false" : "true");
  navCollapseButton.setAttribute("aria-label", navCollapseLabel());
  syncMenuItemTooltips();
  if (navTooltipTarget) {
    if (!shouldShowNavTooltipFor(navTooltipTarget)) {
      hideNavTooltip(true);
      return;
    }
    showNavTooltip(navTooltipTarget);
  }
}

function syncOverlayState() {
  const opened = appShell.classList.contains("nav-open") || appShell.classList.contains("panel-open");
  appShell.classList.toggle("overlay-open", opened);
}

function closeTransientPanels() {
  hideNavTooltip(true);
  appShell.classList.remove("nav-open");
  appShell.classList.remove("panel-open");
  if (state.chatRuntime.openPopover) {
    state.chatRuntime.openPopover = "";
    if (state.currentRoute === "chat") {
      renderChatRuntimePanel();
    }
  }
  syncOverlayState();
}

function collapseMobileSidebar() {
  if (!isMobileViewport()) {
    return;
  }
  closeTransientPanels();
}

function isEditableViewportNode(node) {
  if (!(node instanceof HTMLElement)) {
    return false;
  }
  if (node instanceof HTMLTextAreaElement) {
    return !node.disabled && !node.readOnly;
  }
  if (node instanceof HTMLInputElement) {
    const type = String(node.type || "text").toLowerCase();
    const blockedTypes = new Set(["button", "checkbox", "color", "file", "hidden", "image", "radio", "range", "reset", "submit"]);
    return !node.disabled && !node.readOnly && !blockedTypes.has(type);
  }
  return node.isContentEditable;
}

function activeViewportInput() {
  return isEditableViewportNode(document.activeElement) ? document.activeElement : null;
}

function updateKeyboardInset(options = {}) {
  if (!isMobileViewport()) {
    state.mobileViewport.baselineHeight = 0;
    state.mobileViewport.width = 0;
    state.mobileViewport.height = 0;
    state.mobileViewport.keyboardOffset = 0;
    rootStyle.setProperty("--mobile-viewport-height", "100dvh");
    rootStyle.setProperty("--keyboard-offset", "0px");
    return;
  }

  const viewport = window.visualViewport;
  const activeInput = activeViewportInput();
  const effectiveHeight = Math.max(
    0,
    Math.round(viewport ? viewport.height + Math.max(viewport.offsetTop, 0) : window.innerHeight)
  );
  const viewportWidth = Math.max(
    0,
    Math.round(viewport ? viewport.width : window.innerWidth)
  );
  const widthChanged = Math.abs(viewportWidth - state.mobileViewport.width) > 48;

  if (!state.mobileViewport.baselineHeight || widthChanged) {
    state.mobileViewport.baselineHeight = effectiveHeight;
  }
  if (!activeInput || effectiveHeight >= state.mobileViewport.baselineHeight - 2) {
    state.mobileViewport.baselineHeight = effectiveHeight;
  } else {
    state.mobileViewport.baselineHeight = Math.max(state.mobileViewport.baselineHeight, effectiveHeight);
  }

  const keyboardOffset = Math.max(0, state.mobileViewport.baselineHeight - effectiveHeight);
  const heightChanged = Math.abs(effectiveHeight - state.mobileViewport.height) >= MOBILE_VIEWPORT_SYNC_THRESHOLD_PX;
  const offsetChanged = Math.abs(keyboardOffset - state.mobileViewport.keyboardOffset) >= MOBILE_VIEWPORT_SYNC_THRESHOLD_PX;

  state.mobileViewport.width = viewportWidth;
  if (heightChanged || state.mobileViewport.height === 0) {
    rootStyle.setProperty("--mobile-viewport-height", `${effectiveHeight}px`);
  }
  if (offsetChanged || state.mobileViewport.height === 0) {
    rootStyle.setProperty("--keyboard-offset", `${keyboardOffset}px`);
  }
  state.mobileViewport.height = effectiveHeight;
  state.mobileViewport.keyboardOffset = keyboardOffset;

  if (options.alignFocusedInput && activeInput instanceof HTMLElement) {
    const now = Date.now();
    if (now - Number(state.mobileViewport.lastAlignedAt || 0) < MOBILE_VIEWPORT_ALIGN_COOLDOWN_MS) {
      return;
    }
    state.mobileViewport.lastAlignedAt = now;
    window.requestAnimationFrame(() => {
      if (document.activeElement !== activeInput) {
        return;
      }
      activeInput.scrollIntoView({ block: "nearest" });
      if (composerShell instanceof HTMLElement) {
        composerShell.scrollIntoView({ block: "end" });
      }
    });
  }
}

function scheduleViewportInsetSync(options = {}) {
  state.mobileViewport.alignFocusedInput = state.mobileViewport.alignFocusedInput || Boolean(options.alignFocusedInput);
  if (state.mobileViewport.syncFrame) {
    return;
  }
  state.mobileViewport.syncFrame = window.requestAnimationFrame(() => {
    const alignFocusedInput = state.mobileViewport.alignFocusedInput;
    state.mobileViewport.syncFrame = 0;
    state.mobileViewport.alignFocusedInput = false;
    updateKeyboardInset({ alignFocusedInput });
  });
}

function bindSwipeClose(panel, panelClassName) {
  if (!panel) {
    return;
  }

  let startX = 0;
  let tracking = false;

  panel.addEventListener("touchstart", (event) => {
    if (!isMobileViewport() || !appShell.classList.contains(panelClassName)) {
      return;
    }
    tracking = true;
    startX = event.changedTouches[0].clientX;
  }, { passive: true });

  panel.addEventListener("touchend", (event) => {
    if (!tracking) {
      return;
    }
    tracking = false;
    const deltaX = event.changedTouches[0].clientX - startX;
    if (deltaX < -SWIPE_CLOSE_THRESHOLD) {
      closeTransientPanels();
    }
  }, { passive: true });
}

function setMainContentMode(mode) {
  const infoMode = mode === "page";
  appShell.classList.toggle("info-mode", infoMode);
  chatPane.classList.toggle("page-mode", infoMode);
  chatView.hidden = infoMode;
  routeView.hidden = !infoMode;
}

function navigateToRoute(route, options = {}) {
  const safe = ROUTES[route] ? route : DEFAULT_ROUTE;
  if (!options.skipConfirm && safe !== state.currentRoute && !confirmComposerNavigation()) {
    return;
  }
  collapseMobileSidebar();
  const targetHash = `#${safe}`;
  if (window.location.hash !== targetHash) {
    state.suppressHashRouteConfirm = safe;
    window.location.hash = targetHash;
    return;
  }
  void renderRoute(safe);
}

function startNewChatSession() {
  const existingBlank = getLatestBlankSession("chat");
  if (existingBlank) {
    updateSessionTarget(existingBlank, routeDefaultTarget("chat"));
    setActiveConversationSessionID(existingBlank.id, "chat");
    persistSessions();
    focusSession(existingBlank.id);
  } else {
    if (!confirmComposerNavigation()) {
      return;
    }
    createSession(routeDefaultTarget("chat"), routeConversationMode("chat"), "chat");
  }
  navigateToRoute("chat", { skipConfirm: true });
  closeTransientPanels();
  window.requestAnimationFrame(() => {
    renderWelcomeTargetPicker();
    input.focus();
  });
}

function startNewAgentSession() {
  const currentRoute = state.currentRoute;
  const target = routeAllowsTargetPicker(currentRoute) ? defaultAgentRuntimeTarget() : routeDefaultTarget(currentRoute);
  const existingBlank = getLatestBlankSession();
  if (existingBlank) {
    updateSessionTarget(existingBlank, target);
    setActiveConversationSessionID(existingBlank.id, "agent");
    persistSessions();
    renderSessions();
    renderMessages();
    syncHeader();
    renderWelcomeTargetPicker();
    if (routeAllowsTargetPicker(currentRoute)) {
      navigateToRoute("agent-runtime", { skipConfirm: true });
    }
  } else {
    if (!confirmComposerNavigation()) {
      return;
    }
    createSession(target, "agent", currentRoute);
    if (routeAllowsTargetPicker(currentRoute)) {
      navigateToRoute("agent-runtime", { skipConfirm: true });
    }
  }
  closeTransientPanels();
  window.requestAnimationFrame(() => {
    renderWelcomeTargetPicker();
    input.focus();
  });
}

function renderRouteCards(items, emptyText, renderItem) {
  if (!items.length) {
    return `<p class="route-empty">${emptyText}</p>`;
  }
  return items.map((item) => renderItem(item)).join("");
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => {
    if (char === "&") return "&amp;";
    if (char === "<") return "&lt;";
    if (char === ">") return "&gt;";
    if (char === '"') return "&quot;";
    return "&#39;";
  });
}

function renderMarkdownToHTML(value) {
  const normalized = String(value ?? "").replace(/\r\n?/g, "\n");
  if (!normalized.trim()) {
    return "";
  }
  const tokens = [];
  const fencePattern = /```([a-zA-Z0-9_-]+)?\n([\s\S]*?)```/g;
  let cursor = 0;
  let match = fencePattern.exec(normalized);
  while (match) {
    if (match.index > cursor) {
      tokens.push({ type: "markdown", content: normalized.slice(cursor, match.index) });
    }
    tokens.push({
      type: "code",
      language: String(match[1] || "").trim().toLowerCase(),
      content: String(match[2] || "").replace(/\n$/, "")
    });
    cursor = match.index + match[0].length;
    match = fencePattern.exec(normalized);
  }
  if (cursor < normalized.length) {
    tokens.push({ type: "markdown", content: normalized.slice(cursor) });
  }
  return tokens.map((token) => {
    if (token.type === "code") {
      const languageClass = token.language ? ` class="language-${escapeHTML(token.language)}"` : "";
      return `<pre class="chat-md-pre"><code${languageClass}>${escapeHTML(token.content)}</code></pre>`;
    }
    return renderMarkdownBlocks(token.content);
  }).join("");
}

function renderMarkdownBlocks(content) {
  const lines = String(content || "").split("\n");
  const html = [];
  let paragraphLines = [];
  let quoteLines = [];
  let listType = "";
  let listItems = [];

  const flushParagraph = () => {
    if (!paragraphLines.length) {
      return;
    }
    html.push(`<p>${paragraphLines.map((line) => renderMarkdownInline(line)).join("<br>")}</p>`);
    paragraphLines = [];
  };

  const flushQuote = () => {
    if (!quoteLines.length) {
      return;
    }
    html.push(`<blockquote>${renderMarkdownBlocks(quoteLines.join("\n"))}</blockquote>`);
    quoteLines = [];
  };

  const flushList = () => {
    if (!listType || !listItems.length) {
      listType = "";
      listItems = [];
      return;
    }
    html.push(`<${listType}>${listItems.map((item) => `<li>${renderMarkdownInline(item)}</li>`).join("")}</${listType}>`);
    listType = "";
    listItems = [];
  };

  const flushAll = () => {
    flushParagraph();
    flushQuote();
    flushList();
  };

  for (const rawLine of lines) {
    const trimmed = rawLine.trim();
    if (!trimmed) {
      flushAll();
      continue;
    }

    if (/^>\s?/.test(trimmed)) {
      flushParagraph();
      flushList();
      quoteLines.push(trimmed.replace(/^>\s?/, ""));
      continue;
    }
    flushQuote();

    const unorderedMatch = /^[-*+]\s+(.+)$/.exec(trimmed);
    if (unorderedMatch) {
      flushParagraph();
      if (listType && listType !== "ul") {
        flushList();
      }
      listType = "ul";
      listItems.push(unorderedMatch[1]);
      continue;
    }

    const orderedMatch = /^(\d+)\.\s+(.+)$/.exec(trimmed);
    if (orderedMatch) {
      flushParagraph();
      if (listType && listType !== "ol") {
        flushList();
      }
      listType = "ol";
      listItems.push(orderedMatch[2]);
      continue;
    }

    flushList();

    const headingMatch = /^(#{1,6})\s+(.+)$/.exec(trimmed);
    if (headingMatch) {
      flushParagraph();
      const level = headingMatch[1].length;
      html.push(`<h${level}>${renderMarkdownInline(headingMatch[2])}</h${level}>`);
      continue;
    }

    if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph();
      html.push("<hr>");
      continue;
    }

    paragraphLines.push(trimmed);
  }

  flushAll();
  return html.join("");
}

function renderMarkdownInline(content) {
  let rendered = escapeHTML(String(content ?? ""));
  const placeholders = [];
  const reserve = (html) => {
    const token = `\u0000${placeholders.length}\u0000`;
    placeholders.push(html);
    return token;
  };

  rendered = rendered.replace(/`([^`]+)`/g, (_, code) => reserve(`<code class="chat-md-inline-code">${escapeHTML(code)}</code>`));
  rendered = rendered.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label, url) => {
    const href = sanitizeMarkdownURL(url);
    if (!href) {
      return renderMarkdownInline(label);
    }
    return reserve(`<a href="${href}" target="_blank" rel="noreferrer noopener">${renderMarkdownInline(label)}</a>`);
  });
  rendered = rendered.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  rendered = rendered.replace(/__([^_]+)__/g, "<strong>$1</strong>");
  rendered = rendered.replace(/(^|[\s(>])\*([^*\n]+)\*(?=$|[\s).,!?:;<])/g, "$1<em>$2</em>");
  rendered = rendered.replace(/(^|[\s(>])_([^_\n]+)_(?=$|[\s).,!?:;<])/g, "$1<em>$2</em>");

  return rendered.replace(/\u0000(\d+)\u0000/g, (_, index) => placeholders[Number(index)] || "");
}

function sanitizeMarkdownURL(rawURL) {
  const value = String(rawURL || "").trim();
  if (!value) {
    return "";
  }
  const normalized = value.replace(/^<|>$/g, "");
  if (/^(https?:|mailto:)/i.test(normalized) || normalized.startsWith("/") || normalized.startsWith("#")) {
    return escapeHTML(normalized);
  }
  return "";
}

function parseAgentExecutionText(value) {
  const normalized = String(value ?? "").replace(/\r\n?/g, "\n");
  if (!normalized.trim()) {
    return { steps: [], answer: "" };
  }

  const lines = normalized.split("\n");
  const steps = [];
  const answerLines = [];
  let currentStep = null;
  let index = 0;

  const pushCurrentStep = () => {
    if (!currentStep) {
      return;
    }
    const title = String(currentStep.title || "").trim();
    const detail = String(currentStep.detail || "").trim();
    if (!title && !detail) {
      currentStep = null;
      return;
    }
    steps.push({
      kind: String(currentStep.kind || "action").trim() || "action",
      title,
      detail
    });
    currentStep = null;
  };

  while (index < lines.length) {
    const line = lines[index];
    const trimmed = line.trim();
    if (trimmed.startsWith("[agent] action:")) {
      pushCurrentStep();
      currentStep = {
        kind: "action",
        title: trimmed.slice("[agent] action:".length).trim(),
        detail: ""
      };
      index += 1;
      continue;
    }
    if (trimmed === "[agent] observation:") {
      const detailLines = [];
      index += 1;
      while (index < lines.length) {
        const nextLine = lines[index];
        const nextTrimmed = nextLine.trim();
        if (nextTrimmed.startsWith("[agent] action:") || nextTrimmed === "[agent] observation:") {
          break;
        }
        detailLines.push(nextLine);
        index += 1;
      }
      const detail = detailLines.join("\n").trim();
      if (currentStep) {
        currentStep.detail = detail;
      } else {
        currentStep = {
          kind: "observation",
          title: t("route.agent.step.observation"),
          detail
        };
        pushCurrentStep();
      }
      continue;
    }
    answerLines.push(line);
    index += 1;
  }

  pushCurrentStep();
  return {
    steps,
    answer: answerLines.join("\n").trim()
  };
}

function resolveAgentProcessCollapsed(message, parsed) {
  if (message && typeof message.agent_process_collapsed === "boolean") {
    return message.agent_process_collapsed;
  }
  return Boolean(String(parsed?.answer || "").trim()) && String(message?.status || "").trim() !== "streaming";
}

function renderAgentProcessStep(step, index) {
  const title = String(step?.title || "").trim() || `${t("route.agent.process.label")} ${String(index + 1)}`;
  const detail = String(step?.detail || "").trim();
  return `<article class="agent-process-step">
    <div class="agent-process-step-head">
      <span class="agent-process-step-index">${escapeHTML(String(index + 1))}</span>
      <span class="agent-process-step-title">${escapeHTML(title)}</span>
    </div>
    ${detail ? `<div class="agent-process-step-body">${renderMarkdownToHTML(detail)}</div>` : ""}
  </article>`;
}

function renderAssistantCopyButton(copyValue, className = "") {
  const content = String(copyValue || "");
  if (!content.trim()) {
    return "";
  }
  const classes = ["route-field-copy", "assistant-message-copy"];
  if (className) {
    classes.push(className);
  }
  return `<button class="${classes.join(" ")}" type="button" data-copy-value="${escapeHTML(content)}" title="${escapeHTML(t("route.copy_value"))}" aria-label="${escapeHTML(t("route.copy_value"))}">${renderCopyIcon()}</button>`;
}

function renderAssistantFinalBody(contentHTML, copyValue, className = "") {
  if (!String(contentHTML || "").trim()) {
    return "";
  }
  return `<div class="assistant-message-shell${className ? ` ${className}` : ""}">
    <div class="assistant-message-toolbar">
      ${renderAssistantCopyButton(copyValue)}
    </div>
    <div class="assistant-message-body">${contentHTML}</div>
  </div>`;
}

function renderAgentExecutionMessage(message) {
  const parsed = parseAgentExecutionText(message?.text || "");
  const status = String(message?.status || "").trim();
  if (!parsed.steps.length) {
    if (status === "streaming") {
      return renderMarkdownToHTML(message?.text || "");
    }
    return renderAssistantFinalBody(renderMarkdownToHTML(message?.text || ""), String(message?.text || "").trim());
  }

  const messageID = normalizeText(message?.id);
  const collapsed = resolveAgentProcessCollapsed(message, parsed);
  const answerHTML = String(parsed.answer || "").trim()
    ? renderAssistantFinalBody(`<div class="agent-process-answer">${renderMarkdownToHTML(parsed.answer)}</div>`, parsed.answer, "agent-process-answer-shell")
    : "";

  return `${parsed.steps.length ? `<section class="agent-process-shell ${collapsed ? "is-collapsed" : ""}" data-agent-process-shell="${escapeHTML(messageID)}">
    <button class="agent-process-toggle" type="button" data-agent-process-toggle="${escapeHTML(messageID)}" aria-expanded="${collapsed ? "false" : "true"}">
      <span class="agent-process-toggle-icon">${collapsed ? ">" : "v"}</span>
      <span class="agent-process-copy">
        <span class="agent-process-title">${escapeHTML(t("route.agent.process.label"))}</span>
        <span class="agent-process-summary">${escapeHTML(t("route.agent.process.steps", { count: String(parsed.steps.length) }))}</span>
      </span>
    </button>
    <div class="agent-process-body" ${collapsed ? "hidden" : ""}>
      ${parsed.steps.map((step, index) => renderAgentProcessStep(step, index)).join("") || `<div class="agent-process-empty">${escapeHTML(t("route.agent.process.empty"))}</div>`}
    </div>
  </section>` : ""}
  ${answerHTML}`.trim();
}

function normalizeText(value) {
  if (value === null || value === undefined) {
    return "-";
  }
  const text = String(value).trim();
  return text || "-";
}

function routeTypeIcon(type) {
  const normalized = String(type || "").toLowerCase();
  if (normalized.includes("cron") || normalized.includes("time")) {
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"></circle><polyline points="12 7 12 12 15.5 14"></polyline></svg>`;
  }
  if (normalized.includes("http") || normalized.includes("web")) {
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"></circle><path d="M3 12h18"></path><path d="M12 3a15 15 0 0 1 0 18"></path><path d="M12 3a15 15 0 0 0 0 18"></path></svg>`;
  }
  if (normalized.includes("mcp") || normalized.includes("proto")) {
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3 4 7.5v9L12 21l8-4.5v-9L12 3z"></path><path d="m4 7.5 8 4.5 8-4.5"></path><path d="M12 12v9"></path></svg>`;
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 17h16"></path><path d="M7 17V7h10v10"></path><path d="m9.5 10.5 1.8 1.8-1.8 1.8"></path><path d="M13.2 14.1h2.3"></path></svg>`;
}

function routeFieldRow(labelKey, value, options = {}) {
  const safeValue = normalizeText(value);
  const copyable = Boolean(options?.copyable) && safeValue !== "-";
  const multiline = Boolean(options?.multiline);
  const mono = Boolean(options?.mono);
  const preview = Boolean(options?.preview);
  const clampLines = Number.isFinite(Number(options?.clampLines))
    ? Math.max(2, Number(options.clampLines))
    : 0;
  const classNames = ["route-field-value"];
  if (multiline) {
    classNames.push("is-multiline");
  }
  if (mono) {
    classNames.push("is-mono");
  }
  if (preview || clampLines > 0) {
    classNames.push("is-preview");
  }
  const copyButton = copyable
    ? `<button class="route-field-copy" type="button" data-copy-value="${escapeHTML(safeValue)}" title="${escapeHTML(t("route.copy_value"))}" aria-label="${escapeHTML(t("route.copy_value"))}">${renderCopyIcon()}</button>`
    : "";
  const clampStyle = clampLines > 0 ? ` style="--line-clamp:${clampLines}"` : "";
  return `<p class="route-field-row">
    <span>${t(labelKey)}</span>
    <span class="route-field-value-wrap">
      <strong class="${classNames.join(" ")}" title="${escapeHTML(safeValue)}"${clampStyle}>${escapeHTML(safeValue)}</strong>
      ${copyButton}
    </span>
  </p>`;
}

async function copyTextValue(value) {
  const text = normalizeText(value);
  if (text === "-") {
    return false;
  }
  if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
    await navigator.clipboard.writeText(text);
    return true;
  }
  const fallback = document.createElement("textarea");
  fallback.value = text;
  fallback.setAttribute("readonly", "readonly");
  fallback.style.position = "absolute";
  fallback.style.left = "-9999px";
  document.body.appendChild(fallback);
  fallback.select();
  const copied = document.execCommand("copy");
  fallback.remove();
  return copied;
}

function routeStatusBadge(enabled) {
  const active = Boolean(enabled);
  return `<div class="status-badge ${active ? "" : "disabled"}">
    <span class="status-dot"></span>
    <span>${active ? t("status.enabled") : t("status.disabled")}</span>
  </div>`;
}

function renderRouteTagList(tags) {
  const safeTags = Array.isArray(tags)
    ? tags.map((item) => String(item || "").trim()).filter((item) => item && item !== "-")
    : [];
  if (!safeTags.length) {
    return `<span class="route-tag-placeholder">-</span>`;
  }
  return `<div class="route-tag-list">
    ${safeTags.map((tag) => `<span class="route-tag">${escapeHTML(tag)}</span>`).join("")}
  </div>`;
}

function renderRouteTagSection(labelKey, tags, className = "") {
  const classNames = ["route-card-tag-section"];
  if (className) {
    classNames.push(className);
  }
  return `<div class="${classNames.join(" ")}">
    <span>${t(labelKey)}</span>
    ${renderRouteTagList(tags)}
  </div>`;
}

function renderRouteSection(title, body, options = {}) {
  const safeOptions = options && typeof options === "object" ? options : {};
  const classNames = ["route-section"];
  if (safeOptions.className) {
    classNames.push(safeOptions.className);
  }
  const titleTag = safeOptions.titleTag || "h6";
  return `<section class="${classNames.join(" ")}">
    <${titleTag} class="route-section-title">${escapeHTML(normalizeText(title))}</${titleTag}>
    ${body}
  </section>`;
}

function routeCardTemplate(title, type, fields = [], enabled = false, body = "") {
  const options = arguments.length > 5 ? arguments[5] : {};
  const safeOptions = options && typeof options === "object" ? options : {};
  const classNames = ["route-card"];
  if (safeOptions.className) {
    classNames.push(safeOptions.className);
  }
  const badgeHTML = typeof safeOptions.badgeHTML === "string"
    ? safeOptions.badgeHTML
    : routeStatusBadge(enabled);
  const bodyClassName = safeOptions.bodyClassName || "route-card-body";
  const footerClassName = safeOptions.footerClassName
    ? ` route-card-footer ${safeOptions.footerClassName}`
    : " route-card-footer";
  const fieldRows = Array.isArray(fields) ? fields.filter((item) => typeof item === "string" && item.trim()) : [];
  return `<article class="${classNames.join(" ")}">
    <div class="route-card-head">
      <div class="route-card-title-wrap">
      <div class="route-card-icon" aria-hidden="true">${routeTypeIcon(type)}</div>
        <div class="route-card-title-copy">
          <h4 title="${escapeHTML(normalizeText(title))}">${escapeHTML(normalizeText(title))}</h4>
          ${typeof safeOptions.titleMetaHTML === "string" ? safeOptions.titleMetaHTML : ""}
        </div>
      </div>
      ${badgeHTML}
    </div>
    ${fieldRows.length ? `<div class="route-meta">
      ${fieldRows.join("")}
    </div>` : ""}
    ${body ? `<div class="${bodyClassName}">${body}</div>` : ""}
    ${safeOptions.footer ? `<footer class="${footerClassName.trim()}">${safeOptions.footer}</footer>` : ""}
  </article>`;
}

function syncRouteAction(route) {
  if (!routeActionButton) {
    return;
  }
  routeActionButton.hidden = true;
  routeActionButton.dataset.route = "";
}

async function fetchJSON(path) {
  const response = await fetch(path, { method: "GET" });
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  return response.json();
}

async function deleteServerSession(sessionID) {
  const response = await fetch(`/api/sessions/${encodeURIComponent(sessionID)}`, { method: "DELETE" });
  if (!response.ok) {
    const body = await safeReadJSON(response);
    throw new Error(body.error || body.error_code || `HTTP ${response.status}`);
  }
  return safeReadJSON(response);
}

function resolveDownloadFilename(headerValue, fallbackName) {
  const raw = String(headerValue || "");
  const encodedMatch = raw.match(/filename\*=UTF-8''([^;]+)/i);
  if (encodedMatch && encodedMatch[1]) {
    try {
      return decodeURIComponent(encodedMatch[1]);
    } catch {
    }
  }
  const plainMatch = raw.match(/filename=\"?([^\";]+)\"?/i);
  if (plainMatch && plainMatch[1]) {
    return plainMatch[1];
  }
  return normalizeText(fallbackName || "artifact.bin");
}

async function downloadTaskArtifact(downloadURL, fallbackName) {
  const response = await fetch(downloadURL, { method: "GET" });
  if (!response.ok) {
    const body = await safeReadJSON(response);
    throw new Error(body.error || body.error_code || `HTTP ${response.status}`);
  }
  const blob = await response.blob();
  const objectURL = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = objectURL;
  anchor.download = resolveDownloadFilename(response.headers.get("Content-Disposition"), fallbackName);
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(objectURL);
}

async function loadChannelsView(container) {
  const data = await fetchJSON("/api/control/channels");
  const items = Array.isArray(data.items) ? data.items : [];
  container.innerHTML = renderRouteCards(
    items,
    t("route.channels.empty"),
    (item) => routeCardTemplate(
      item.id,
      item.type,
      [
        routeFieldRow("field.id", item.id, { copyable: true, mono: true }),
        routeFieldRow("field.type", item.type),
        routeFieldRow("field.description", item.description, { multiline: true, preview: true, clampLines: 3 })
      ],
      item.enabled
    )
  );
}

async function loadSkillsView(container) {
  const data = await fetchJSON("/api/control/skills");
  const items = Array.isArray(data.items) ? data.items : [];
  container.innerHTML = renderRouteCards(
    items,
    t("route.skills.empty"),
    (item) => routeCardTemplate(
      item.id,
      item.type,
      [
        routeFieldRow("field.id", item.id, { copyable: true, mono: true }),
        routeFieldRow("field.type", item.type),
        routeFieldRow("field.name", item.name),
        routeFieldRow("field.scope", item.scope),
        routeFieldRow("field.version", item.version)
      ],
      item.enabled
    )
  );
}

async function loadMCPView(container) {
  const data = await fetchJSON("/api/control/mcps");
  const items = Array.isArray(data.items) ? data.items : [];
  container.innerHTML = renderRouteCards(
    items,
    t("route.mcp.empty"),
    (item) => routeCardTemplate(
      item.id,
      item.type,
      [
        routeFieldRow("field.id", item.id, { copyable: true, mono: true }),
        routeFieldRow("field.type", item.type),
        routeFieldRow("field.name", item.name),
        routeFieldRow("field.scope", item.scope),
        routeFieldRow("field.version", item.version)
      ],
      item.enabled
    )
  );
}

function normalizeProductRouteState(routeState = {}) {
  return {
    selectedProductID: String(routeState?.selectedProductID || "").trim(),
    selectedDraftID: String(routeState?.selectedDraftID || "").trim(),
    activePanel: String(routeState?.activePanel || "workspace").trim() || "workspace",
    selectedSpaceID: String(routeState?.selectedSpaceID || "").trim(),
    workspaceSessionByProduct: routeState?.workspaceSessionByProduct && typeof routeState.workspaceSessionByProduct === "object"
      ? { ...routeState.workspaceSessionByProduct }
      : {}
  };
}

function normalizeProductBuilderDraft(product = {}) {
  return {
    id: String(product?.id || "").trim(),
    name: String(product?.name || "").trim(),
    slug: String(product?.slug || "").trim(),
    summary: String(product?.summary || "").trim(),
    status: String(product?.status || "draft").trim() || "draft",
    visibility: String(product?.visibility || "private").trim() || "private",
    owner_type: String(product?.owner_type || "managed").trim() || "managed",
    version: String(product?.version || "").trim(),
    master_agent_id: String(product?.master_agent_id || "").trim(),
    entry_route: String(product?.entry_route || "products").trim() || "products",
    tags: Array.isArray(product?.tags) ? product.tags.map((item) => String(item || "").trim()).filter(Boolean) : [],
    artifact_types: Array.isArray(product?.artifact_types) ? product.artifact_types.map((item) => String(item || "").trim()).filter(Boolean) : [],
    knowledge_sources: Array.isArray(product?.knowledge_sources) ? product.knowledge_sources.map((item) => String(item || "").trim()).filter(Boolean) : [],
    worker_agents: Array.isArray(product?.worker_agents) ? product.worker_agents.map((item) => ({
      agent_id: String(item?.agent_id || "").trim(),
      role: String(item?.role || "").trim(),
      responsibility: String(item?.responsibility || "").trim(),
      capabilities: Array.isArray(item?.capabilities) ? item.capabilities.map((cap) => String(cap || "").trim()).filter(Boolean) : [],
      enabled: item?.enabled !== false
    })).filter((item) => item.agent_id) : []
  };
}

function normalizeProductDraftAgent(agent = {}) {
  return {
    agent_id: String(agent?.agent_id || "").trim(),
    name: String(agent?.name || "").trim(),
    description: String(agent?.description || "").trim(),
    system_prompt: String(agent?.system_prompt || "").trim(),
    max_iterations: Number.isFinite(Number(agent?.max_iterations)) ? Math.max(0, Number(agent.max_iterations)) : 0,
    tools: Array.isArray(agent?.tools) ? agent.tools.map((item) => String(item || "").trim()).filter(Boolean) : [],
    skills: Array.isArray(agent?.skills) ? agent.skills.map((item) => String(item || "").trim()).filter(Boolean) : [],
    mcps: Array.isArray(agent?.mcps) ? agent.mcps.map((item) => String(item || "").trim()).filter(Boolean) : [],
    memory_files: Array.isArray(agent?.memory_files) ? agent.memory_files.map((item) => String(item || "").trim()).filter(Boolean) : [],
    capabilities: Array.isArray(agent?.capabilities) ? agent.capabilities.map((item) => String(item || "").trim()).filter(Boolean) : [],
    allowed_delegate_targets: Array.isArray(agent?.allowed_delegate_targets) ? agent.allowed_delegate_targets.map((item) => String(item || "").trim()).filter(Boolean) : [],
    enabled: agent?.enabled !== false,
    delegatable: agent?.delegatable !== false
  };
}

function normalizeProductDraftWorker(worker = {}) {
  return {
    agent_id: String(worker?.agent_id || "").trim(),
    name: String(worker?.name || "").trim(),
    role: String(worker?.role || "").trim(),
    responsibility: String(worker?.responsibility || "").trim(),
    description: String(worker?.description || "").trim(),
    system_prompt: String(worker?.system_prompt || "").trim(),
    input_contract: String(worker?.input_contract || "").trim(),
    output_contract: String(worker?.output_contract || "").trim(),
    allowed_tools: Array.isArray(worker?.allowed_tools) ? worker.allowed_tools.map((item) => String(item || "").trim()).filter(Boolean) : [],
    allowed_delegate_targets: Array.isArray(worker?.allowed_delegate_targets) ? worker.allowed_delegate_targets.map((item) => String(item || "").trim()).filter(Boolean) : [],
    dependencies: Array.isArray(worker?.dependencies) ? worker.dependencies.map((item) => String(item || "").trim()).filter(Boolean) : [],
    skills: Array.isArray(worker?.skills) ? worker.skills.map((item) => String(item || "").trim()).filter(Boolean) : [],
    mcps: Array.isArray(worker?.mcps) ? worker.mcps.map((item) => String(item || "").trim()).filter(Boolean) : [],
    memory_files: Array.isArray(worker?.memory_files) ? worker.memory_files.map((item) => String(item || "").trim()).filter(Boolean) : [],
    capabilities: Array.isArray(worker?.capabilities) ? worker.capabilities.map((item) => String(item || "").trim()).filter(Boolean) : [],
    priority: Number.isFinite(Number(worker?.priority)) ? Math.max(0, Number(worker.priority)) : 0,
    max_iterations: Number.isFinite(Number(worker?.max_iterations)) ? Math.max(0, Number(worker.max_iterations)) : 0,
    enabled: worker?.enabled !== false
  };
}

function normalizeProductStudioDraft(draft = {}) {
  return {
    draft_id: String(draft?.draft_id || "").trim(),
    mode: String(draft?.mode || "bootstrap").trim() || "bootstrap",
    review_status: String(draft?.review_status || "draft").trim() || "draft",
    generated_by: String(draft?.generated_by || "").trim(),
    generated_at: String(draft?.generated_at || "").trim(),
    updated_at: String(draft?.updated_at || "").trim(),
    goal: String(draft?.goal || "").trim(),
    target_users: Array.isArray(draft?.target_users) ? draft.target_users.map((item) => String(item || "").trim()).filter(Boolean) : [],
    core_capabilities: Array.isArray(draft?.core_capabilities) ? draft.core_capabilities.map((item) => String(item || "").trim()).filter(Boolean) : [],
    constraints: Array.isArray(draft?.constraints) ? draft.constraints.map((item) => String(item || "").trim()).filter(Boolean) : [],
    expected_artifacts: Array.isArray(draft?.expected_artifacts) ? draft.expected_artifacts.map((item) => String(item || "").trim()).filter(Boolean) : [],
    integration_requirements: Array.isArray(draft?.integration_requirements) ? draft.integration_requirements.map((item) => String(item || "").trim()).filter(Boolean) : [],
    conflict_suggestions: Array.isArray(draft?.conflict_suggestions) ? draft.conflict_suggestions.map((item) => String(item || "").trim()).filter(Boolean) : [],
    published_product_id: String(draft?.published_product_id || "").trim(),
    product: normalizeProductBuilderDraft(draft?.product || {}),
    master_agent: normalizeProductDraftAgent(draft?.master_agent || {}),
    worker_matrix: Array.isArray(draft?.worker_matrix) ? draft.worker_matrix.map((item) => normalizeProductDraftWorker(item)).filter((item) => item.agent_id) : []
  };
}

function normalizeProductWorkspaceMasterAgent(agent = {}) {
  return {
    agent_id: String(agent?.agent_id || "").trim(),
    name: String(agent?.name || "").trim(),
    description: String(agent?.description || "").trim(),
    capabilities: Array.isArray(agent?.capabilities) ? agent.capabilities.map((item) => String(item || "").trim()).filter(Boolean) : [],
    tools: Array.isArray(agent?.tools) ? agent.tools.map((item) => String(item || "").trim()).filter(Boolean) : [],
    skills: Array.isArray(agent?.skills) ? agent.skills.map((item) => String(item || "").trim()).filter(Boolean) : [],
    mcps: Array.isArray(agent?.mcps) ? agent.mcps.map((item) => String(item || "").trim()).filter(Boolean) : [],
    memory_files: Array.isArray(agent?.memory_files) ? agent.memory_files.map((item) => String(item || "").trim()).filter(Boolean) : []
  };
}

function normalizeProductWorkspaceSpaceSummary(space = {}) {
  return {
    space_id: String(space?.space_id || "").trim(),
    title: String(space?.title || "").trim(),
    slug: String(space?.slug || "").trim(),
    html_path: String(space?.html_path || "").trim(),
    summary: String(space?.summary || "").trim(),
    type: String(space?.type || "").trim(),
    status: String(space?.status || "").trim() || "active",
    revision: Number.isFinite(Number(space?.revision)) ? Math.max(0, Number(space.revision)) : 0,
    updated_at: String(space?.updated_at || "").trim(),
    tags: Array.isArray(space?.tags) ? space.tags.map((item) => String(item || "").trim()).filter(Boolean) : []
  };
}

function normalizeProductWorkspace(payload = {}) {
  return {
    product: normalizeProductBuilderDraft(payload?.product || {}),
    master_agent: payload?.master_agent ? normalizeProductWorkspaceMasterAgent(payload.master_agent) : null,
    space_type: String(payload?.space_type || "").trim(),
    space_label: String(payload?.space_label || "").trim(),
    workspace_hint: String(payload?.workspace_hint || "").trim(),
    spaces: Array.isArray(payload?.spaces) ? payload.spaces.map((item) => normalizeProductWorkspaceSpaceSummary(item)).filter((item) => item.space_id) : []
  };
}

function normalizeProductWorkspaceSpaceDetail(payload = {}) {
  return {
    space: normalizeProductWorkspaceSpaceSummary(payload?.space || {}),
    guide: payload?.guide && typeof payload.guide === "object" ? {
      id: String(payload.guide?.id || "").trim(),
      city: String(payload.guide?.city || "").trim(),
      days: Number.isFinite(Number(payload.guide?.days)) ? Math.max(0, Number(payload.guide.days)) : 0,
      travel_style: String(payload.guide?.travel_style || "").trim(),
      budget: String(payload.guide?.budget || "").trim(),
      companions: Array.isArray(payload.guide?.companions) ? payload.guide.companions.map((item) => String(item || "").trim()).filter(Boolean) : [],
      must_visit: Array.isArray(payload.guide?.must_visit) ? payload.guide.must_visit.map((item) => String(item || "").trim()).filter(Boolean) : [],
      avoid: Array.isArray(payload.guide?.avoid) ? payload.guide.avoid.map((item) => String(item || "").trim()).filter(Boolean) : [],
      additional_requirements: Array.isArray(payload.guide?.additional_requirements) ? payload.guide.additional_requirements.map((item) => String(item || "").trim()).filter(Boolean) : [],
      keep_conditions: Array.isArray(payload.guide?.keep_conditions) ? payload.guide.keep_conditions.map((item) => String(item || "").trim()).filter(Boolean) : [],
      replace_conditions: Array.isArray(payload.guide?.replace_conditions) ? payload.guide.replace_conditions.map((item) => String(item || "").trim()).filter(Boolean) : [],
      notes: Array.isArray(payload.guide?.notes) ? payload.guide.notes.map((item) => String(item || "").trim()).filter(Boolean) : [],
      daily_routes: Array.isArray(payload.guide?.daily_routes) ? payload.guide.daily_routes.map((item) => ({
        day: Number.isFinite(Number(item?.day)) ? Math.max(0, Number(item.day)) : 0,
        theme: String(item?.theme || "").trim(),
        stops: Array.isArray(item?.stops) ? item.stops.map((value) => String(value || "").trim()).filter(Boolean) : [],
        transit: Array.isArray(item?.transit) ? item.transit.map((value) => String(value || "").trim()).filter(Boolean) : [],
        dining_plan: Array.isArray(item?.dining_plan) ? item.dining_plan.map((value) => String(value || "").trim()).filter(Boolean) : []
      })) : [],
      map_layers: Array.isArray(payload.guide?.map_layers) ? payload.guide.map_layers.map((item) => ({
        id: String(item?.id || "").trim(),
        label: String(item?.label || "").trim(),
        description: String(item?.description || "").trim()
      })) : [],
      content: String(payload.guide?.content || "").trim(),
      revision: Number.isFinite(Number(payload.guide?.revision)) ? Math.max(0, Number(payload.guide.revision)) : 0,
      updated_at: String(payload.guide?.updated_at || "").trim()
    } : null
  };
}

function createProductWorkspaceMessage(role, text, options = {}) {
  return {
    id: "product-workspace-" + Math.random().toString(16).slice(2),
    role: role === "assistant" ? "assistant" : "user",
    text: String(text || "").trim(),
    status: String(options.status || "done").trim() || "done",
    error: Boolean(options.error),
    at: options.at || Date.now()
  };
}

function renderProductWorkspaceMessageList(items) {
  if (!Array.isArray(items) || !items.length) {
    return `<p class="route-empty">${escapeHTML(t("route.products.workspace.chat_hint"))}</p>`;
  }
  return items.map((item) => `<article class="product-workspace-message is-${escapeHTML(item.role || "assistant")} ${item.error ? "is-error" : ""}">
    <div class="product-workspace-message-meta">
      <strong>${escapeHTML(item.role === "user" ? "You" : "Agent")}</strong>
      <span>${escapeHTML(item.status === "streaming" ? t("msg.processing") : formatDateTime(item.at) || "")}</span>
    </div>
    <p>${escapeHTML(String(item.text || "").trim() || "-")}</p>
  </article>`).join("");
}

function renderProductWorkspaceSpaceCards(items, selectedSpaceID) {
  if (!Array.isArray(items) || !items.length) {
    return `<p class="route-empty">${escapeHTML(t("route.products.workspace.space_empty"))}</p>`;
  }
  return items.map((item) => {
    const activeClassName = item.space_id === selectedSpaceID ? " is-active" : "";
    const openPageLink = item.html_path
      ? `<a class="task-filter-reset" href="${escapeHTML(item.html_path)}" target="_blank" rel="noreferrer">${escapeHTML(t("route.products.workspace.open_page"))}</a>`
      : "";
    return `<article class="agent-route-card${activeClassName}">
      <button class="agent-route-card-button" type="button" data-product-space-select="${escapeHTML(item.space_id)}">
      <div class="agent-route-card-head">
        <div class="agent-route-card-copy">
          <strong title="${escapeHTML(item.title || item.space_id)}">${escapeHTML(item.title || item.space_id)}</strong>
          <span title="${escapeHTML(item.slug || item.space_id)}">${escapeHTML(item.slug || item.space_id)}</span>
        </div>
        <span class="agent-route-state ${String(item.status || "").trim() === "active" ? "is-enabled" : "is-disabled"}">${escapeHTML(normalizeText(item.status || "active"))}</span>
      </div>
      <p class="agent-route-card-prompt">${escapeHTML(item.summary || "-")}</p>
      <div class="agent-route-card-tags">${item.tags.length ? item.tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("") : ""}</div>
      </button>
      ${openPageLink ? `<div class="task-filter-actions">${openPageLink}</div>` : ""}
    </article>`;
  }).join("");
}

function createProductDraftRequest(mode = "bootstrap") {
  return {
    name: "",
    goal: "",
    target_users: "",
    core_capabilities: "",
    constraints: "",
    expected_artifacts: "",
    integration_requirements: "",
    mode: String(mode || "bootstrap").trim() || "bootstrap"
  };
}

function serializeProductWorkerAgents(items) {
  if (!Array.isArray(items) || !items.length) {
    return "";
  }
  return items.map((item) => {
    const segments = [String(item?.agent_id || "").trim(), String(item?.role || "").trim(), String(item?.responsibility || "").trim()].filter(Boolean);
    return segments.join(" | ");
  }).filter(Boolean).join("\n");
}

function parseProductWorkerAgentInput(value) {
  return String(value || "").split(/\r?\n/).map((line) => line.trim()).filter(Boolean).map((line) => {
    const parts = line.split("|").map((item) => item.trim()).filter(Boolean);
    return {
      agent_id: String(parts[0] || "").trim(),
      role: String(parts[1] || "").trim(),
      responsibility: String(parts.slice(2).join(" | ") || "").trim(),
      enabled: true
    };
  }).filter((item) => item.agent_id);
}

function renderProductBuilderCards(items, selectedProductID) {
  if (!items.length) {
    return `<p class="route-empty">${t("route.products.empty")}</p>`;
  }
  return items.map((item) => {
    const productID = String(item?.id || "").trim();
    const productName = String(item?.name || productID || "").trim() || productID;
    const activeClassName = productID === selectedProductID ? " is-active" : "";
    const tags = [normalizeText(item?.owner_type), normalizeText(item?.status), `${Number(item?.worker_agents?.length || 0)} agents`].filter((tag) => tag !== "-");
    return `<button class="agent-route-card${activeClassName}" type="button" data-product-select="${escapeHTML(productID)}">
      <div class="agent-route-card-head">
        <div class="agent-route-card-copy">
          <h4>${escapeHTML(productName)}</h4>
          <span title="${escapeHTML(productID)}">${escapeHTML(productID)}</span>
        </div>
        <span class="agent-route-state ${(String(item?.status || "").trim() === "active") ? "is-enabled" : "is-disabled"}">${escapeHTML(normalizeText(item?.status || "draft"))}</span>
      </div>
      <p class="agent-route-card-prompt">${escapeHTML(String(item?.summary || "").trim() || t("route.products.empty"))}</p>
      <div class="agent-route-card-tags">${tags.length ? tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("") : ""}</div>
    </button>`;
  }).join("");
}

function renderProductDraftCards(items, selectedDraftID) {
  if (!items.length) {
    return `<p class="route-empty">${t("route.products.drafts.empty")}</p>`;
  }
  return items.map((item) => {
    const draftID = String(item?.draft_id || "").trim();
    const productName = String(item?.product?.name || item?.product?.id || draftID || "").trim() || draftID;
    const reviewStatus = String(item?.review_status || "draft").trim() || "draft";
    const activeClassName = draftID === selectedDraftID ? " is-active" : "";
    const tags = [normalizeText(item?.mode), normalizeText(reviewStatus), normalizeText(item?.product?.id), `${Number(item?.worker_matrix?.length || 0)} workers`].filter((tag) => tag !== "-");
    return `<button class="agent-route-card${activeClassName}" type="button" data-product-draft-select="${escapeHTML(draftID)}">
      <div class="agent-route-card-head">
        <div class="agent-route-card-copy">
          <strong title="${escapeHTML(productName)}">${escapeHTML(productName)}</strong>
          <span title="${escapeHTML(draftID)}">${escapeHTML(draftID)}</span>
        </div>
        <span class="agent-route-state ${(reviewStatus === "published" || reviewStatus === "reviewed") ? "is-enabled" : "is-disabled"}">${escapeHTML(normalizeText(reviewStatus))}</span>
      </div>
      <p class="agent-route-card-prompt">${escapeHTML(String(item?.goal || item?.product?.summary || "").trim() || t("route.products.drafts.empty"))}</p>
      <div class="agent-route-card-tags">${tags.length ? tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("") : ""}</div>
    </button>`;
  }).join("");
}

function renderProductDraftWorkerCards(items) {
  if (!items.length) {
    return `<p class="route-empty">${t("route.products.drafts.empty")}</p>`;
  }
  return items.map((item) => {
    const workerName = String(item?.name || item?.agent_id || "").trim() || String(item?.agent_id || "").trim();
    const tags = [normalizeText(item?.role), `${Number(item?.allowed_tools?.length || 0)} tools`, `${Number(item?.dependencies?.length || 0)} deps`].filter((tag) => tag !== "-");
    const prompt = [
      String(item?.responsibility || "").trim(),
      String(item?.input_contract || "").trim(),
      String(item?.output_contract || "").trim()
    ].filter(Boolean).join(" | ");
    return `<article class="agent-route-card">
      <div class="agent-route-card-head">
        <div class="agent-route-card-copy">
          <strong title="${escapeHTML(workerName)}">${escapeHTML(workerName)}</strong>
          <span title="${escapeHTML(String(item?.agent_id || "").trim())}">${escapeHTML(String(item?.agent_id || "").trim())}</span>
        </div>
        <span class="agent-route-state ${item?.enabled !== false ? "is-enabled" : "is-disabled"}">${escapeHTML(item?.enabled !== false ? t("status.enabled") : t("status.disabled"))}</span>
      </div>
      <p class="agent-route-card-prompt">${escapeHTML(prompt || String(item?.description || "").trim() || "-")}</p>
      <div class="agent-route-card-tags">${tags.length ? tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("") : ""}</div>
    </article>`;
  }).join("");
}
function normalizeAgentRouteState(routeState = {}) {
  return {
    selectedAgentID: String(routeState?.selectedAgentID || "").trim(),
    activeSessionByAgent: routeState?.activeSessionByAgent && typeof routeState.activeSessionByAgent === "object"
      ? { ...routeState.activeSessionByAgent }
      : {}
  };
}

function normalizeAgentBuilderDraft(agent = {}) {
  return {
    id: String(agent?.id || "").trim(),
    name: String(agent?.name || "").trim(),
    enabled: agent?.enabled !== false,
    scope: String(agent?.scope || "global").trim() || "global",
    version: String(agent?.version || "").trim(),
    system_prompt: String(agent?.system_prompt || "").trim(),
    max_iterations: Number.isFinite(Number(agent?.max_iterations)) ? Math.max(0, Number(agent.max_iterations)) : 0,
    tools: Array.isArray(agent?.tools) && agent.tools.length ? agent.tools.map((item) => String(item || "").trim()).filter(Boolean) : ["codex_exec", "search_memory", "read_memory", "write_memory"],
    skills: Array.isArray(agent?.skills) ? agent.skills.map((item) => String(item || "").trim()).filter(Boolean) : [],
    mcps: Array.isArray(agent?.mcps) ? agent.mcps.map((item) => String(item || "").trim()).filter(Boolean) : [],
    memory_files: Array.isArray(agent?.memory_files) ? agent.memory_files.map((item) => String(item || "").trim()).filter(Boolean) : []
  };
}

function parseAgentListInput(value) {
  return Array.from(new Set(String(value || "").split(",").map((item) => item.trim()).filter(Boolean)));
}

function renderAgentBuilderCards(items, selectedAgentID) {
  if (!items.length) {
    return `<p class="route-empty">${t("route.agent.empty")}</p>`;
  }
  return items.map((item) => {
    const agentID = String(item?.id || "").trim();
    const agentName = String(item?.name || agentID || "").trim() || agentID;
    const enabled = Boolean(item?.enabled);
    const activeClassName = agentID === selectedAgentID ? " is-active" : "";
    const tags = [
      normalizeText(item?.version),
      `${Number(item?.skills?.length || 0)} skills`
    ].filter((tag) => tag !== "-");
    return `<button class="agent-route-card${activeClassName}" type="button" data-agent-select="${escapeHTML(agentID)}" aria-pressed="${agentID === selectedAgentID ? "true" : "false"}">
      <div class="agent-route-card-head">
        <div class="agent-route-card-copy">
          <strong title="${escapeHTML(agentName)}">${escapeHTML(agentName)}</strong>
          <span title="${escapeHTML(agentID)}">${escapeHTML(agentID)}</span>
        </div>
        <span class="agent-route-state ${enabled ? "is-enabled" : "is-disabled"}">${escapeHTML(enabled ? t("status.enabled") : t("status.disabled"))}</span>
      </div>
      <p class="agent-route-card-prompt">${escapeHTML(String(item?.system_prompt || "").trim() || t("route.agent.form.empty"))}</p>
      <div class="agent-route-card-tags">${tags.length ? tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("") : ""}</div>
    </button>`;
  }).join("");
}

function renderAgentOptionList(items, selectedValues, fieldName) {
  if (!items.length) {
    return `<p class="route-empty">${escapeHTML(t("route.connected_desc"))}</p>`;
  }
  const selected = new Set((Array.isArray(selectedValues) ? selectedValues : []).map((item) => String(item || "").trim()));
  return items.map((item) => {
    const optionID = String(item?.id || "").trim();
    if (!optionID) {
      return "";
    }
    const optionName = String(item?.name || optionID).trim() || optionID;
    return `<label class="agent-builder-option">
      <input type="checkbox" name="${escapeHTML(fieldName)}" value="${escapeHTML(optionID)}" ${selected.has(optionID) ? "checked" : ""}>
      <span title="${escapeHTML(String(item?.description || "").trim())}">${escapeHTML(optionName)}</span>
    </label>`;
  }).join("");
}

async function loadProductsView(container) {
  const localState = {
    routeState: normalizeProductRouteState(state.productRouteState),
    products: [],
    drafts: [],
    draft: normalizeProductBuilderDraft(),
    draftRequest: createProductDraftRequest(),
    selectedDraft: normalizeProductStudioDraft(),
    workspace: normalizeProductWorkspace(),
    workspaceDetail: normalizeProductWorkspaceSpaceDetail(),
    workspaceMessagesByProduct: {},
    workspaceLoading: false,
    workspacePending: false,
    workspaceError: "",
    draftEditorText: "",
    statusMessage: "",
    statusKind: "",
    loading: true
  };

  const persistRouteState = () => {
    state.productRouteState = {
      selectedProductID: localState.routeState.selectedProductID,
      selectedDraftID: localState.routeState.selectedDraftID,
      activePanel: localState.routeState.activePanel,
      selectedSpaceID: localState.routeState.selectedSpaceID,
      workspaceSessionByProduct: { ...localState.routeState.workspaceSessionByProduct }
    };
  };

  const requestJSON = async (path, options = {}) => {
    const headers = new Headers(options.headers || {});
    if (options.body && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }
    const response = await fetch(path, {
      method: options.method || "GET",
      headers,
      body: options.body
    });
    const payload = await safeReadJSON(response);
    if (!response.ok) {
      throw new Error(typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`);
    }
    return payload;
  };

  const findSelectedProduct = () => {
    return localState.products.find((item) => String(item?.id || "").trim() === localState.routeState.selectedProductID) || null;
  };

  const findSelectedDraft = () => {
    return localState.drafts.find((item) => String(item?.draft_id || "").trim() === localState.routeState.selectedDraftID) || null;
  };

  const findSelectedSpaceSummary = () => {
    return localState.workspace.spaces.find((item) => String(item?.space_id || "").trim() === localState.routeState.selectedSpaceID) || null;
  };

  const currentWorkspaceMessages = () => {
    const productID = String(localState.routeState.selectedProductID || "").trim();
    return productID ? (localState.workspaceMessagesByProduct[productID] || []) : [];
  };

  const setCurrentWorkspaceMessages = (items) => {
    const productID = String(localState.routeState.selectedProductID || "").trim();
    if (!productID) {
      return;
    }
    localState.workspaceMessagesByProduct[productID] = Array.isArray(items) ? items : [];
  };

  const syncDraftFromSelection = () => {
    localState.draft = normalizeProductBuilderDraft(findSelectedProduct() || {});
  };

  const syncSelectedDraft = (draft = null) => {
    const selected = normalizeProductStudioDraft(draft || findSelectedDraft() || {});
    localState.selectedDraft = selected;
    localState.routeState.selectedDraftID = selected.draft_id;
    localState.draftEditorText = selected.draft_id ? JSON.stringify(selected, null, 2) : "";
  };

  const syncDraftRequest = (form) => {
    if (!form) {
      return;
    }
    const formData = new FormData(form);
    localState.draftRequest = {
      name: String(formData.get("name") || "").trim(),
      goal: String(formData.get("goal") || "").trim(),
      target_users: String(formData.get("target_users") || "").trim(),
      core_capabilities: String(formData.get("core_capabilities") || "").trim(),
      constraints: String(formData.get("constraints") || "").trim(),
      expected_artifacts: String(formData.get("expected_artifacts") || "").trim(),
      integration_requirements: String(formData.get("integration_requirements") || "").trim(),
      mode: String(formData.get("mode") || "bootstrap").trim() || "bootstrap"
    };
  };

  const renderChipList = (items) => {
    const values = Array.isArray(items) ? items.filter(Boolean) : [];
    if (!values.length) {
      return `<p class="route-empty">-</p>`;
    }
    return `<div class="product-workspace-chip-list">${values.map((item) => `<span>${escapeHTML(String(item || "").trim())}</span>`).join("")}</div>`;
  };

  const renderWorkspaceDetail = () => {
    const selectedSpace = findSelectedSpaceSummary();
    const guide = localState.workspaceDetail?.guide;
    if (!selectedSpace || !guide) {
      return `<p class="route-empty">${escapeHTML(t("route.products.workspace.detail_empty"))}</p>`;
    }
    const routeRows = guide.daily_routes.length
      ? guide.daily_routes.map((item) => `<article class="product-workspace-detail-card">
        <strong>${escapeHTML(`Day ${item.day}`)}</strong>
        <p>${escapeHTML(item.theme || item.stops.join(" -> ") || "-")}</p>
        ${item.stops.length ? `<span>${escapeHTML(item.stops.join(" -> "))}</span>` : ""}
        ${item.transit.length ? `<span>${escapeHTML(item.transit.join(" / "))}</span>` : ""}
      </article>`).join("")
      : `<p class="route-empty">-</p>`;
    const layerRows = guide.map_layers.length
      ? guide.map_layers.map((item) => `<article class="product-workspace-detail-card">
        <strong>${escapeHTML(item.label || item.id || "-")}</strong>
        <p>${escapeHTML(item.description || "-")}</p>
      </article>`).join("")
      : `<p class="route-empty">-</p>`;
    const openPageLink = selectedSpace?.html_path
      ? `<a class="task-filter-apply" href="${escapeHTML(selectedSpace.html_path)}" target="_blank" rel="noreferrer">${escapeHTML(t("route.products.workspace.open_page"))}</a>`
      : "";

    return `
      ${openPageLink ? `<div class="task-filter-actions">${openPageLink}</div>` : ""}
      <div class="agent-builder-managed">
        <div class="agent-builder-managed-item">
          <span>${escapeHTML(t("route.products.workspace.days"))}</span>
          <strong>${escapeHTML(String(guide.days || 0))}</strong>
        </div>
        <div class="agent-builder-managed-item">
          <span>${escapeHTML(t("route.products.workspace.space_revision"))}</span>
          <strong>${escapeHTML(String(guide.revision || 0))}</strong>
        </div>
        <div class="agent-builder-managed-item">
          <span>${escapeHTML(t("route.products.workspace.updated"))}</span>
          <strong>${escapeHTML(formatDateTime(guide.updated_at) || "-")}</strong>
        </div>
      </div>
      <div class="product-workspace-detail-grid">
        <section class="product-workspace-detail-section">
          <h5>${escapeHTML(t("route.products.workspace.tags"))}</h5>
          ${renderChipList([guide.city, guide.travel_style, guide.budget].filter(Boolean))}
        </section>
        <section class="product-workspace-detail-section">
          <h5>${escapeHTML("Companions / Must Visit")}</h5>
          ${renderChipList([...(guide.companions || []), ...(guide.must_visit || [])])}
        </section>
        <section class="product-workspace-detail-section">
          <h5>${escapeHTML("Avoid / Additional")}</h5>
          ${renderChipList([...(guide.avoid || []), ...(guide.additional_requirements || []), ...(guide.keep_conditions || []), ...(guide.replace_conditions || [])])}
        </section>
      </div>
      <section class="product-workspace-detail-section product-workspace-detail-content">
        <h5>${escapeHTML(t("route.products.workspace.content"))}</h5>
        <pre>${escapeHTML(guide.content || "-")}</pre>
      </section>
      <section class="product-workspace-detail-section">
        <h5>${escapeHTML(t("route.products.workspace.notes"))}</h5>
        ${renderChipList(guide.notes)}
      </section>
      <section class="product-workspace-detail-section">
        <h5>${escapeHTML(t("route.products.workspace.routes"))}</h5>
        <div class="product-workspace-detail-stack">${routeRows}</div>
      </section>
      <section class="product-workspace-detail-section">
        <h5>${escapeHTML(t("route.products.workspace.layers"))}</h5>
        <div class="product-workspace-detail-stack">${layerRows}</div>
      </section>`;
  };

  const renderWorkspacePanel = (selectedProduct) => {
    if (!selectedProduct?.id) {
      return `<p class="route-empty">${escapeHTML(t("route.products.workspace.empty"))}</p>`;
    }
    if (localState.workspaceLoading) {
      return `<p class="route-loading">${escapeHTML(t("loading"))}</p>`;
    }
    if (localState.workspaceError) {
      return `<p class="route-error">${escapeHTML(localState.workspaceError)}</p>`;
    }
    const workspace = localState.workspace;
    const master = workspace.master_agent;
    const selectedSpace = findSelectedSpaceSummary();
    const chatPlaceholder = selectedSpace
      ? `${t("route.products.workspace.chat_placeholder")} (${selectedSpace.title})`
      : t("route.products.workspace.chat_placeholder");
    return `<section class="product-workspace-grid">
      <section class="route-surface product-workspace-panel">
        <div class="agent-route-pane-head">
          <div class="agent-route-pane-copy">
            <h4>${escapeHTML(t("route.products.workspace.title"))}</h4>
            <p>${escapeHTML(workspace.workspace_hint || t("route.products.workspace.subtitle"))}</p>
          </div>
        </div>
        <div class="agent-builder-managed">
          <div class="agent-builder-managed-item">
            <span>${escapeHTML(t("route.products.form.id"))}</span>
            <strong>${escapeHTML(selectedProduct.id || "-")}</strong>
          </div>
          <div class="agent-builder-managed-item">
            <span>${escapeHTML(t("route.products.form.status"))}</span>
            <strong>${escapeHTML(selectedProduct.status || "-")}</strong>
          </div>
          <div class="agent-builder-managed-item">
            <span>${escapeHTML(t("route.products.form.visibility"))}</span>
            <strong>${escapeHTML(selectedProduct.visibility || "-")}</strong>
          </div>
          <div class="agent-builder-managed-item">
            <span>${escapeHTML(t("route.products.form.version"))}</span>
            <strong>${escapeHTML(selectedProduct.version || "-")}</strong>
          </div>
        </div>
        <div class="product-workspace-overview-grid">
          <section class="product-workspace-detail-section">
            <h5>${escapeHTML(t("route.products.workspace.overview"))}</h5>
            <p class="product-workspace-summary">${escapeHTML(selectedProduct.summary || "-")}</p>
          </section>
          <section class="product-workspace-detail-section">
            <h5>${escapeHTML(t("route.products.workspace.master"))}</h5>
            ${master ? `
              <div class="product-workspace-master">
                <strong>${escapeHTML(master.name || master.agent_id || "-")}</strong>
                <span>${escapeHTML(master.agent_id || "-")}</span>
                <p>${escapeHTML(master.description || "-")}</p>
              </div>
              ${renderChipList([...(master.capabilities || []), ...(master.tools || [])])}
            ` : `<p class="route-empty">-</p>`}
          </section>
          <section class="product-workspace-detail-section">
            <h5>${escapeHTML(t("route.products.workspace.output"))}</h5>
            ${renderChipList(selectedProduct.artifact_types)}
          </section>
          <section class="product-workspace-detail-section">
            <h5>${escapeHTML(t("route.products.workspace.sources"))}</h5>
            ${renderChipList(selectedProduct.knowledge_sources)}
          </section>
          <section class="product-workspace-detail-section">
            <h5>${escapeHTML(t("route.products.workspace.tags"))}</h5>
            ${renderChipList(selectedProduct.tags)}
          </section>
          <section class="product-workspace-detail-section">
            <h5>${escapeHTML(t("route.products.workspace.workers"))}</h5>
            ${renderChipList((selectedProduct.worker_agents || []).map((item) => item.role || item.agent_id))}
          </section>
        </div>
      </section>
      <section class="route-surface product-workspace-panel">
        <div class="agent-route-pane-head">
          <div class="agent-route-pane-copy">
            <h4>${escapeHTML(t("route.products.workspace.chat_title"))}</h4>
            <p>${escapeHTML(t("route.products.workspace.chat_hint"))}</p>
          </div>
        </div>
        <div class="product-workspace-chat-log">${renderProductWorkspaceMessageList(currentWorkspaceMessages())}</div>
        <form class="product-workspace-chat-form" data-product-workspace-chat-form>
          <textarea name="content" rows="4" placeholder="${escapeHTML(chatPlaceholder)}" ${localState.workspacePending ? "disabled" : ""}></textarea>
          <div class="task-filter-actions">
            <button class="task-filter-reset" type="button" data-product-space-clear ${selectedSpace ? "" : "disabled"}>${escapeHTML(t("route.products.workspace.open_new"))}</button>
            <button class="task-filter-apply" type="submit" ${localState.workspacePending ? "disabled" : ""}>${escapeHTML(t("route.products.workspace.chat_send"))}</button>
          </div>
        </form>
      </section>
      <section class="route-surface product-workspace-panel">
        <div class="agent-route-pane-head">
          <div class="agent-route-pane-copy">
            <h4>${escapeHTML(workspace.space_label || t("route.products.workspace.spaces"))}</h4>
            <p>${escapeHTML(t("route.products.workspace.space_empty"))}</p>
          </div>
        </div>
        <div class="agent-route-list">${renderProductWorkspaceSpaceCards(workspace.spaces, localState.routeState.selectedSpaceID)}</div>
      </section>
      <section class="route-surface product-workspace-panel product-workspace-panel-wide">
        <div class="agent-route-pane-head">
          <div class="agent-route-pane-copy">
            <h4>${escapeHTML(selectedSpace?.title || t("route.products.workspace.detail"))}</h4>
            <p>${escapeHTML(selectedSpace?.summary || t("route.products.workspace.detail_empty"))}</p>
          </div>
        </div>
        ${renderWorkspaceDetail()}
      </section>
    </section>`;
  };

  const renderStudioPanel = (selectedProduct) => {
    const selectedDraft = localState.selectedDraft;
    const isBuiltin = String(selectedProduct?.owner_type || "").trim() === "builtin";
    const canDelete = Boolean(selectedProduct?.id) && !isBuiltin;
    const canSave = !isBuiltin;
    const managedID = selectedProduct?.id || "-";
    const managedVersion = selectedProduct?.version || "-";
    const managedOwner = selectedProduct?.owner_type || localState.draft.owner_type || "managed";
    const workerAgentValue = serializeProductWorkerAgents(localState.draft.worker_agents);
    const currentDraftMode = String(localState.draftRequest.mode || "bootstrap").trim() || "bootstrap";
    const canExpandDraft = Boolean(selectedProduct?.id) && String(selectedProduct?.owner_type || "").trim() !== "builtin";
    const canPublishDraft = Boolean(selectedDraft?.draft_id) && !selectedDraft?.published_product_id && String(selectedDraft?.product?.owner_type || "managed").trim() !== "builtin";
    const masterTags = [
      `${Number(selectedDraft?.master_agent?.tools?.length || 0)} tools`,
      `${Number(selectedDraft?.master_agent?.skills?.length || 0)} skills`,
      `${Number(selectedDraft?.master_agent?.allowed_delegate_targets?.length || 0)} delegates`
    ].filter((tag) => !tag.startsWith("0 "));
    return `
      <div class="agent-route-pane-head">
        <div class="agent-route-pane-copy">
          <h4>${escapeHTML(localState.draft.id ? t("route.products.edit") : t("route.products.form.new"))}</h4>
          <p>${escapeHTML(t("route.products.form.managed"))}</p>
        </div>
        <div class="agent-builder-actions">
          <button type="button" data-product-reset>${escapeHTML(t("route.products.form.cancel"))}</button>
        </div>
      </div>
      ${isBuiltin ? `<p class="agent-builder-status">${escapeHTML(t("route.products.form.builtin_notice"))}</p>` : ""}
      <div class="agent-builder-managed">
        <div class="agent-builder-managed-item">
          <span>${escapeHTML(t("route.products.form.id"))}</span>
          <strong>${escapeHTML(managedID)}</strong>
        </div>
        <div class="agent-builder-managed-item">
          <span>${escapeHTML(t("route.products.form.version"))}</span>
          <strong>${escapeHTML(managedVersion)}</strong>
        </div>
        <div class="agent-builder-managed-item">
          <span>${escapeHTML(t("route.products.form.owner"))}</span>
          <strong>${escapeHTML(managedOwner)}</strong>
        </div>
      </div>
      <form class="agent-builder-form" data-product-form>
        <label><span>${t("route.products.form.name")}</span><input type="text" name="name" value="${escapeHTML(localState.draft.name)}" placeholder="Travel"></label>
        <label><span>${t("route.products.form.slug")}</span><input type="text" name="slug" value="${escapeHTML(localState.draft.slug)}" placeholder="travel"></label>
        <label><span>${t("route.products.form.master")}</span><input type="text" name="master_agent_id" value="${escapeHTML(localState.draft.master_agent_id)}" placeholder="travel-master"></label>
        <label><span>${t("route.products.form.status")}</span><select name="status">
          ${["draft", "active", "disabled", "archived"].map((option) => `<option value="${escapeHTML(option)}" ${option === localState.draft.status ? "selected" : ""}>${escapeHTML(option)}</option>`).join("")}
        </select></label>
        <label><span>${t("route.products.form.visibility")}</span><select name="visibility">
          ${["private", "public"].map((option) => `<option value="${escapeHTML(option)}" ${option === localState.draft.visibility ? "selected" : ""}>${escapeHTML(option)}</option>`).join("")}
        </select></label>
        <label><span>${t("route.products.form.entry_route")}</span><input type="text" name="entry_route" value="${escapeHTML(localState.draft.entry_route)}" placeholder="products"></label>
        <label class="agent-builder-wide"><span>${t("route.products.form.summary")}</span><textarea name="summary" rows="4">${escapeHTML(localState.draft.summary)}</textarea></label>
        <label class="agent-builder-wide"><span>${t("route.products.form.tags")}</span><input type="text" name="tags" value="${escapeHTML(localState.draft.tags.join(", "))}" placeholder="travel, itinerary"></label>
        <label class="agent-builder-wide"><span>${t("route.products.form.artifacts")}</span><input type="text" name="artifact_types" value="${escapeHTML(localState.draft.artifact_types.join(", "))}" placeholder="city_guide, itinerary, map_layers"></label>
        <label class="agent-builder-wide"><span>${t("route.products.form.knowledge")}</span><input type="text" name="knowledge_sources" value="${escapeHTML(localState.draft.knowledge_sources.join(", "))}" placeholder="poi_catalog, metro_network"></label>
        <label class="agent-builder-wide"><span>${t("route.products.form.worker_agents")}</span><textarea name="worker_agents" rows="6" placeholder="travel-route-planner | route-planner | Plan daily routes">${escapeHTML(workerAgentValue)}</textarea></label>
        <div class="task-filter-actions">
          <button class="task-filter-apply" type="submit" ${canSave ? "" : "disabled"}>${escapeHTML(t("route.products.form.save"))}</button>
          <button class="task-filter-reset" type="button" data-product-delete ${canDelete ? "" : "disabled"}>${escapeHTML(t("route.products.form.delete"))}</button>
        </div>
      </form>
      <div class="agent-builder-section">
        <div class="agent-route-pane-head">
          <div class="agent-route-pane-copy">
            <h4>${escapeHTML(t("route.products.drafts.title"))}</h4>
            <p>${escapeHTML(t("route.products.drafts.subtitle"))}</p>
          </div>
        </div>
        ${currentDraftMode === "expand" && !canExpandDraft ? `<p class="agent-builder-status">${escapeHTML(t("route.products.drafts.form.expand_disabled"))}</p>` : ""}
        ${currentDraftMode === "expand" && canExpandDraft ? `<p class="agent-builder-status">${escapeHTML(`${t("route.products.form.id")}: ${selectedProduct.id}`)}</p>` : ""}
        <form class="agent-builder-form" data-product-draft-generate-form>
          <label><span>${t("route.products.drafts.form.name")}</span><input type="text" name="name" value="${escapeHTML(localState.draftRequest.name)}" placeholder="Travel Premium"></label>
          <label><span>${t("route.products.drafts.form.mode")}</span><select name="mode">
            <option value="bootstrap" ${currentDraftMode === "bootstrap" ? "selected" : ""}>${escapeHTML(t("route.products.drafts.form.mode.bootstrap"))}</option>
            <option value="expand" ${currentDraftMode === "expand" ? "selected" : ""}>${escapeHTML(t("route.products.drafts.form.mode.expand"))}</option>
          </select></label>
          <label class="agent-builder-wide"><span>${t("route.products.drafts.form.goal")}</span><textarea name="goal" rows="3">${escapeHTML(localState.draftRequest.goal)}</textarea></label>
          <label class="agent-builder-wide"><span>${t("route.products.drafts.form.target_users")}</span><input type="text" name="target_users" value="${escapeHTML(localState.draftRequest.target_users)}" placeholder="travellers, families"></label>
          <label class="agent-builder-wide"><span>${t("route.products.drafts.form.core")}</span><input type="text" name="core_capabilities" value="${escapeHTML(localState.draftRequest.core_capabilities)}" placeholder="city guide, itinerary, metro, food, map"></label>
          <label class="agent-builder-wide"><span>${t("route.products.drafts.form.constraints")}</span><input type="text" name="constraints" value="${escapeHTML(localState.draftRequest.constraints)}" placeholder="budget-sensitive, kid-friendly"></label>
          <label class="agent-builder-wide"><span>${t("route.products.drafts.form.artifacts")}</span><input type="text" name="expected_artifacts" value="${escapeHTML(localState.draftRequest.expected_artifacts)}" placeholder="city_guide, itinerary, map_layers"></label>
          <label class="agent-builder-wide"><span>${t("route.products.drafts.form.integrations")}</span><input type="text" name="integration_requirements" value="${escapeHTML(localState.draftRequest.integration_requirements)}" placeholder="maps, metro, food catalog"></label>
          <div class="task-filter-actions">
            <button class="task-filter-apply" type="submit" ${(currentDraftMode === "expand" && !canExpandDraft) ? "disabled" : ""}>${escapeHTML(t("route.products.drafts.form.generate"))}</button>
          </div>
        </form>
        <div class="agent-route-list">${renderProductDraftCards(localState.drafts, localState.routeState.selectedDraftID)}</div>
        ${selectedDraft?.draft_id ? `
          <div class="agent-builder-managed">
            <div class="agent-builder-managed-item">
              <span>${escapeHTML(t("route.products.form.id"))}</span>
              <strong>${escapeHTML(selectedDraft.draft_id)}</strong>
            </div>
            <div class="agent-builder-managed-item">
              <span>${escapeHTML(t("route.products.drafts.detail.review"))}</span>
              <strong>${escapeHTML(normalizeText(selectedDraft.review_status || "draft"))}</strong>
            </div>
            <div class="agent-builder-managed-item">
              <span>${escapeHTML(t("route.products.drafts.detail.mode"))}</span>
              <strong>${escapeHTML(normalizeText(selectedDraft.mode || "bootstrap"))}</strong>
            </div>
            <div class="agent-builder-managed-item">
              <span>${escapeHTML(t("route.products.drafts.detail.product"))}</span>
              <strong>${escapeHTML(selectedDraft.product.id || "-")}</strong>
            </div>
            <div class="agent-builder-managed-item">
              <span>${escapeHTML(t("route.products.drafts.detail.generated"))}</span>
              <strong>${escapeHTML(formatDateTime(selectedDraft.generated_at) || "-")}</strong>
            </div>
            <div class="agent-builder-managed-item">
              <span>${escapeHTML(t("route.products.drafts.detail.updated"))}</span>
              <strong>${escapeHTML(formatDateTime(selectedDraft.updated_at) || "-")}</strong>
            </div>
            <div class="agent-builder-managed-item">
              <span>${escapeHTML(t("route.products.drafts.detail.published"))}</span>
              <strong>${escapeHTML(selectedDraft.published_product_id || "-")}</strong>
            </div>
          </div>
          <div class="agent-builder-section">
            <h5>${escapeHTML(t("route.products.drafts.detail.master"))}</h5>
            <article class="agent-route-card">
              <div class="agent-route-card-head">
                <div class="agent-route-card-copy">
                  <strong title="${escapeHTML(selectedDraft.master_agent.name || selectedDraft.master_agent.agent_id)}">${escapeHTML(selectedDraft.master_agent.name || selectedDraft.master_agent.agent_id || "-")}</strong>
                  <span title="${escapeHTML(selectedDraft.master_agent.agent_id)}">${escapeHTML(selectedDraft.master_agent.agent_id || "-")}</span>
                </div>
                <span class="agent-route-state ${selectedDraft.master_agent.enabled ? "is-enabled" : "is-disabled"}">${escapeHTML(selectedDraft.master_agent.enabled ? t("status.enabled") : t("status.disabled"))}</span>
              </div>
              <p class="agent-route-card-prompt">${escapeHTML(selectedDraft.master_agent.description || selectedDraft.master_agent.system_prompt || "-")}</p>
              <div class="agent-route-card-tags">${masterTags.length ? masterTags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("") : ""}</div>
            </article>
          </div>
          <div class="agent-builder-section">
            <h5>${escapeHTML(t("route.products.drafts.detail.workers"))}</h5>
            <div class="agent-route-list">${renderProductDraftWorkerCards(selectedDraft.worker_matrix)}</div>
          </div>
          ${selectedDraft.conflict_suggestions.length ? `
            <div class="agent-builder-section">
              <h5>${escapeHTML(t("route.products.drafts.detail.conflicts"))}</h5>
              <div class="agent-route-list">
                ${selectedDraft.conflict_suggestions.map((item) => `<article class="agent-route-card"><p class="agent-route-card-prompt">${escapeHTML(item)}</p></article>`).join("")}
              </div>
            </div>
          ` : ""}
          <form class="agent-builder-form" data-product-draft-editor-form>
            <label class="agent-builder-wide"><span>${t("route.products.drafts.form.editor")}</span><textarea name="draft_json" rows="18">${escapeHTML(localState.draftEditorText)}</textarea></label>
            <div class="agent-builder-wide">
              <p class="agent-route-pane-copy"><span>${escapeHTML(t("route.products.drafts.form.editor_hint"))}</span></p>
            </div>
            <div class="task-filter-actions">
              <button class="task-filter-apply" type="submit">${escapeHTML(t("route.products.drafts.form.save"))}</button>
              <button class="task-filter-reset" type="button" data-product-draft-publish ${canPublishDraft ? "" : "disabled"}>${escapeHTML(t("route.products.drafts.form.publish"))}</button>
            </div>
          </form>
        ` : `<p class="route-empty">${escapeHTML(t("route.products.drafts.detail.empty"))}</p>`}
      </div>`;
  };

  const paint = () => {
    if (localState.loading) {
      container.innerHTML = `<p class="route-loading">${t("loading")}</p>`;
      return;
    }
    const selectedProduct = findSelectedProduct();
    container.innerHTML = `<section class="agent-studio-view">
      <aside class="route-surface agent-studio-list-pane">
        <div class="agent-route-pane-head">
          <div class="agent-route-pane-copy">
            <h4>${escapeHTML(t("route.products.title"))}</h4>
            <p>${escapeHTML(t("route.products.subtitle"))}</p>
          </div>
          <button class="route-primary-button" type="button" data-product-create>${escapeHTML(t("route.products.create"))}</button>
        </div>
        <div class="agent-route-list">${renderProductBuilderCards(localState.products, localState.routeState.selectedProductID)}</div>
      </aside>
      <section class="route-surface agent-studio-form-pane">
        ${localState.statusMessage ? `<p class="agent-builder-status ${localState.statusKind === "error" ? "is-error" : "is-success"}">${escapeHTML(localState.statusMessage)}</p>` : ""}
        <div class="product-workspace-tabs">
          <button class="memory-tab ${localState.routeState.activePanel === "workspace" ? "active" : ""}" type="button" data-product-panel="workspace">${escapeHTML(t("route.products.panel.workspace"))}</button>
          <button class="memory-tab ${localState.routeState.activePanel === "studio" ? "active" : ""}" type="button" data-product-panel="studio">${escapeHTML(t("route.products.panel.studio"))}</button>
        </div>
        ${localState.routeState.activePanel === "workspace" ? renderWorkspacePanel(selectedProduct) : renderStudioPanel(selectedProduct)}
      </section>
    </section>`;
  };

  const loadWorkspace = async (productID, options = {}) => {
    const selectedID = String(productID || "").trim();
    if (!selectedID) {
      localState.workspace = normalizeProductWorkspace();
      localState.workspaceDetail = normalizeProductWorkspaceSpaceDetail();
      localState.workspaceLoading = false;
      localState.workspaceError = "";
      if (!options.silent) {
        persistRouteState();
        paint();
        bind();
      }
      return;
    }
    localState.workspaceLoading = true;
    localState.workspaceError = "";
    if (!options.silent) {
      paint();
    }
    try {
      const payload = await requestJSON(`/api/products/${encodeURIComponent(selectedID)}/workspace`);
      if (selectedID !== String(localState.routeState.selectedProductID || "").trim()) {
        return;
      }
      localState.workspace = normalizeProductWorkspace(payload);
      const validSpaceIDs = new Set(localState.workspace.spaces.map((item) => item.space_id));
      if (!validSpaceIDs.has(localState.routeState.selectedSpaceID)) {
        localState.routeState.selectedSpaceID = localState.workspace.spaces[0]?.space_id || "";
      }
      if (localState.routeState.selectedSpaceID) {
        const detailPayload = await requestJSON(`/api/products/${encodeURIComponent(selectedID)}/workspace/spaces/${encodeURIComponent(localState.routeState.selectedSpaceID)}`);
        if (selectedID !== String(localState.routeState.selectedProductID || "").trim()) {
          return;
        }
        localState.workspaceDetail = normalizeProductWorkspaceSpaceDetail(detailPayload);
      } else {
        localState.workspaceDetail = normalizeProductWorkspaceSpaceDetail();
      }
    } catch (error) {
      localState.workspace = normalizeProductWorkspace();
      localState.workspaceDetail = normalizeProductWorkspaceSpaceDetail();
      localState.workspaceError = error instanceof Error ? error.message : "unknown_error";
    } finally {
      if (selectedID === String(localState.routeState.selectedProductID || "").trim()) {
        localState.workspaceLoading = false;
      }
      if (!options.silent) {
        persistRouteState();
        paint();
        bind();
      }
    }
  };

  const reload = async (statusMessage = "", statusKind = "") => {
    localState.loading = true;
    localState.statusMessage = statusMessage;
    localState.statusKind = statusKind;
    paint();
    const [productsPayload, draftsPayload] = await Promise.all([
      fetchJSON("/api/control/products"),
      requestJSON("/api/control/products/drafts")
    ]);
    localState.products = Array.isArray(productsPayload?.items) ? productsPayload.items : [];
    localState.drafts = Array.isArray(draftsPayload?.items) ? draftsPayload.items.map((item) => normalizeProductStudioDraft(item)) : [];
    if (!findSelectedProduct()) {
      localState.routeState.selectedProductID = localState.products[0]?.id || "";
    }
    syncDraftFromSelection();
    if (!findSelectedDraft()) {
      localState.routeState.selectedDraftID = localState.drafts[0]?.draft_id || "";
    }
    syncSelectedDraft();
    await loadWorkspace(localState.routeState.selectedProductID, { silent: true });
    localState.loading = false;
    persistRouteState();
    paint();
    bind();
  };

  const saveProduct = async (form) => {
    const selectedProduct = findSelectedProduct();
    if (String(selectedProduct?.owner_type || "").trim() === "builtin") {
      return;
    }
    const formData = new FormData(form);
    const payload = {
      name: String(formData.get("name") || "").trim(),
      slug: String(formData.get("slug") || "").trim(),
      summary: String(formData.get("summary") || "").trim(),
      status: String(formData.get("status") || "draft").trim(),
      visibility: String(formData.get("visibility") || "private").trim(),
      master_agent_id: String(formData.get("master_agent_id") || "").trim(),
      entry_route: String(formData.get("entry_route") || "").trim(),
      tags: parseAgentListInput(formData.get("tags") || ""),
      artifact_types: parseAgentListInput(formData.get("artifact_types") || ""),
      knowledge_sources: parseAgentListInput(formData.get("knowledge_sources") || ""),
      worker_agents: parseProductWorkerAgentInput(formData.get("worker_agents") || "")
    };
    try {
      const saved = await requestJSON(selectedProduct?.id ? `/api/control/products/${encodeURIComponent(selectedProduct.id)}` : "/api/control/products", {
        method: selectedProduct?.id ? "PUT" : "POST",
        body: JSON.stringify(payload)
      });
      localState.routeState.selectedProductID = String(saved?.id || "").trim();
      await reload(t("route.products.saved"), "success");
    } catch (error) {
      localState.statusMessage = t("route.products.save_failed", {
        error: error instanceof Error ? error.message : "unknown_error"
      });
      localState.statusKind = "error";
      paint();
      bind();
    }
  };

  const deleteProduct = async () => {
    const selected = findSelectedProduct();
    if (!selected || !selected.id || String(selected?.owner_type || "").trim() === "builtin") {
      return;
    }
    try {
      await requestJSON(`/api/control/products/${encodeURIComponent(selected.id)}`, {
        method: "DELETE"
      });
      localState.routeState.selectedProductID = "";
      await reload(t("route.products.deleted"), "success");
    } catch (error) {
      localState.statusMessage = t("route.products.delete_failed", {
        error: error instanceof Error ? error.message : "unknown_error"
      });
      localState.statusKind = "error";
      paint();
      bind();
    }
  };

  const generateDraft = async (form) => {
    syncDraftRequest(form);
    const mode = String(localState.draftRequest.mode || "bootstrap").trim() || "bootstrap";
    const selectedProduct = findSelectedProduct();
    if (mode === "expand" && (!selectedProduct?.id || String(selectedProduct?.owner_type || "").trim() === "builtin")) {
      localState.statusMessage = t("route.products.drafts.form.expand_disabled");
      localState.statusKind = "error";
      paint();
      bind();
      return;
    }
    const payload = {
      name: localState.draftRequest.name,
      goal: localState.draftRequest.goal,
      target_users: parseAgentListInput(localState.draftRequest.target_users),
      core_capabilities: parseAgentListInput(localState.draftRequest.core_capabilities),
      constraints: parseAgentListInput(localState.draftRequest.constraints),
      expected_artifacts: parseAgentListInput(localState.draftRequest.expected_artifacts),
      integration_requirements: parseAgentListInput(localState.draftRequest.integration_requirements),
      mode
    };
    const path = mode === "expand"
      ? `/api/control/products/${encodeURIComponent(selectedProduct.id)}/matrix/generate`
      : "/api/control/products/generate";
    try {
      const created = await requestJSON(path, {
        method: "POST",
        body: JSON.stringify(payload)
      });
      localState.routeState.selectedDraftID = String(created?.draft_id || "").trim();
      localState.draftRequest = createProductDraftRequest(mode);
      await reload(t("route.products.drafts.generated"), "success");
    } catch (error) {
      localState.statusMessage = t("route.products.drafts.generate_failed", {
        error: error instanceof Error ? error.message : "unknown_error"
      });
      localState.statusKind = "error";
      paint();
      bind();
    }
  };

  const saveDraftReview = async (form) => {
    const selectedDraft = localState.selectedDraft;
    if (!selectedDraft?.draft_id) {
      return;
    }
    const formData = new FormData(form);
    const rawJSON = String(formData.get("draft_json") || "").trim();
    localState.draftEditorText = rawJSON;
    try {
      const payload = JSON.parse(rawJSON || "{}");
      if (!payload || Array.isArray(payload) || typeof payload !== "object") {
        throw new Error("draft json must be an object");
      }
      await requestJSON(`/api/control/products/drafts/${encodeURIComponent(selectedDraft.draft_id)}`, {
        method: "PUT",
        body: JSON.stringify(payload)
      });
      await reload(t("route.products.drafts.review_saved"), "success");
    } catch (error) {
      localState.statusMessage = t("route.products.drafts.review_failed", {
        error: error instanceof Error ? error.message : "unknown_error"
      });
      localState.statusKind = "error";
      paint();
      bind();
    }
  };

  const publishDraft = async () => {
    const selectedDraft = localState.selectedDraft;
    if (!selectedDraft?.draft_id) {
      return;
    }
    try {
      const payload = await requestJSON(`/api/control/products/drafts/${encodeURIComponent(selectedDraft.draft_id)}/publish`, {
        method: "POST"
      });
      localState.routeState.selectedDraftID = String(payload?.draft?.draft_id || selectedDraft.draft_id).trim();
      localState.routeState.selectedProductID = String(payload?.product?.id || localState.routeState.selectedProductID).trim();
      await reload(t("route.products.drafts.published"), "success");
    } catch (error) {
      localState.statusMessage = t("route.products.drafts.publish_failed", {
        error: error instanceof Error ? error.message : "unknown_error"
      });
      localState.statusKind = "error";
      paint();
      bind();
    }
  };

  const sendWorkspaceMessage = async (form) => {
    const selectedProduct = findSelectedProduct();
    if (!selectedProduct?.id || localState.workspacePending) {
      return;
    }
    const formData = new FormData(form);
    const content = String(formData.get("content") || "").trim();
    if (!content) {
      return;
    }
    const assistantMessage = createProductWorkspaceMessage("assistant", t("msg.processing"), { status: "streaming" });
    setCurrentWorkspaceMessages([
      ...currentWorkspaceMessages(),
      createProductWorkspaceMessage("user", content),
      assistantMessage
    ]);
    localState.workspacePending = true;
    paint();
    bind();
    try {
      if (selectedProduct.id === "travel") {
        const payload = await requestJSON(`/api/products/${encodeURIComponent(selectedProduct.id)}/workspace/chat`, {
          method: "POST",
          body: JSON.stringify({
            session_id: localState.routeState.workspaceSessionByProduct[selectedProduct.id] || "",
            space_id: localState.routeState.selectedSpaceID,
            content
          })
        });
        assistantMessage.text = String(payload?.reply || "").trim() || t("msg.received_empty");
        assistantMessage.status = "done";
        if (payload?.guide?.id || payload?.space_id) {
          localState.routeState.selectedSpaceID = String(payload?.guide?.id || payload?.space_id || "").trim();
          await loadWorkspace(selectedProduct.id, { silent: true });
        }
      } else {
        const payload = await requestJSON(`/api/products/${encodeURIComponent(selectedProduct.id)}/messages`, {
          method: "POST",
          body: JSON.stringify({
            session_id: localState.routeState.workspaceSessionByProduct[selectedProduct.id] || "",
            content
          })
        });
        assistantMessage.text = String(payload?.result?.output || "").trim() || t("msg.received_empty");
        assistantMessage.status = "done";
        const nextSessionID = String(payload?.result?.session_id || "").trim();
        if (nextSessionID) {
          localState.routeState.workspaceSessionByProduct[selectedProduct.id] = nextSessionID;
        }
      }
      localState.statusMessage = t("route.products.workspace.synced");
      localState.statusKind = "success";
    } catch (error) {
      assistantMessage.text = t("route.products.workspace.chat_failed", {
        error: error instanceof Error ? error.message : "unknown_error"
      });
      assistantMessage.status = "error";
      assistantMessage.error = true;
      localState.statusMessage = assistantMessage.text;
      localState.statusKind = "error";
    } finally {
      localState.workspacePending = false;
      persistRouteState();
      paint();
      bind();
    }
  };

  const bind = () => {
    container.querySelectorAll("[data-product-select]").forEach((node) => {
      node.addEventListener("click", async () => {
        const nextID = String(node.getAttribute("data-product-select") || "").trim();
        if (!nextID || nextID === localState.routeState.selectedProductID) {
          return;
        }
        localState.routeState.selectedProductID = nextID;
        localState.routeState.selectedSpaceID = "";
        syncDraftFromSelection();
        persistRouteState();
        paint();
        bind();
        await loadWorkspace(nextID);
      });
    });

    container.querySelectorAll("[data-product-panel]").forEach((node) => {
      node.addEventListener("click", () => {
        const nextPanel = String(node.getAttribute("data-product-panel") || "workspace").trim() || "workspace";
        if (nextPanel === localState.routeState.activePanel) {
          return;
        }
        localState.routeState.activePanel = nextPanel;
        persistRouteState();
        paint();
        bind();
      });
    });

    container.querySelectorAll("[data-product-draft-select]").forEach((node) => {
      node.addEventListener("click", () => {
        const nextID = String(node.getAttribute("data-product-draft-select") || "").trim();
        if (!nextID || nextID === localState.routeState.selectedDraftID) {
          return;
        }
        localState.routeState.selectedDraftID = nextID;
        syncSelectedDraft();
        persistRouteState();
        paint();
        bind();
      });
    });

    const createButton = container.querySelector("[data-product-create]");
    if (createButton) {
      createButton.addEventListener("click", () => {
        localState.routeState.selectedProductID = "";
        localState.routeState.activePanel = "studio";
        localState.routeState.selectedSpaceID = "";
        localState.draft = normalizeProductBuilderDraft();
        localState.workspace = normalizeProductWorkspace();
        localState.workspaceDetail = normalizeProductWorkspaceSpaceDetail();
        localState.statusMessage = "";
        localState.statusKind = "";
        persistRouteState();
        paint();
        bind();
      });
    }

    const resetButton = container.querySelector("[data-product-reset]");
    if (resetButton) {
      resetButton.addEventListener("click", () => {
        syncDraftFromSelection();
        localState.statusMessage = "";
        localState.statusKind = "";
        paint();
        bind();
      });
    }

    const deleteButton = container.querySelector("[data-product-delete]");
    if (deleteButton) {
      deleteButton.addEventListener("click", async () => {
        await deleteProduct();
      });
    }

    const form = container.querySelector("[data-product-form]");
    if (form) {
      form.addEventListener("submit", async (event) => {
        event.preventDefault();
        await saveProduct(form);
      });
    }

    const generateForm = container.querySelector("[data-product-draft-generate-form]");
    if (generateForm) {
      generateForm.addEventListener("input", () => {
        syncDraftRequest(generateForm);
      });
      generateForm.addEventListener("change", () => {
        syncDraftRequest(generateForm);
      });
      generateForm.addEventListener("submit", async (event) => {
        event.preventDefault();
        await generateDraft(generateForm);
      });
    }

    const draftEditorForm = container.querySelector("[data-product-draft-editor-form]");
    if (draftEditorForm) {
      const editor = draftEditorForm.querySelector("textarea[name='draft_json']");
      if (editor) {
        editor.addEventListener("input", () => {
          localState.draftEditorText = String(editor.value || "");
        });
      }
      draftEditorForm.addEventListener("submit", async (event) => {
        event.preventDefault();
        await saveDraftReview(draftEditorForm);
      });
    }

    const publishButton = container.querySelector("[data-product-draft-publish]");
    if (publishButton) {
      publishButton.addEventListener("click", async () => {
        await publishDraft();
      });
    }

    container.querySelectorAll("[data-product-space-select]").forEach((node) => {
      node.addEventListener("click", async () => {
        const nextID = String(node.getAttribute("data-product-space-select") || "").trim();
        if (!nextID || nextID === localState.routeState.selectedSpaceID) {
          return;
        }
        localState.routeState.selectedSpaceID = nextID;
        await loadWorkspace(String(localState.routeState.selectedProductID || "").trim());
      });
    });

    const clearSpaceButton = container.querySelector("[data-product-space-clear]");
    if (clearSpaceButton) {
      clearSpaceButton.addEventListener("click", () => {
        localState.routeState.selectedSpaceID = "";
        localState.workspaceDetail = normalizeProductWorkspaceSpaceDetail();
        persistRouteState();
        paint();
        bind();
      });
    }

    const workspaceChatForm = container.querySelector("[data-product-workspace-chat-form]");
    if (workspaceChatForm) {
      workspaceChatForm.addEventListener("submit", async (event) => {
        event.preventDefault();
        await sendWorkspaceMessage(workspaceChatForm);
      });
    }
  };

  await reload("", "");
}
async function loadAgentView(container) {
  const localState = {
    routeState: normalizeAgentRouteState(state.agentRouteState),
    agents: [],
    skills: [],
    mcps: [],
    memoryFiles: AGENT_MEMORY_FILE_OPTIONS,
    draft: normalizeAgentBuilderDraft(),
    statusMessage: "",
    statusKind: "",
    loading: true
  };

  const persistRouteState = () => {
    state.agentRouteState = {
      selectedAgentID: localState.routeState.selectedAgentID,
      activeSessionByAgent: { ...localState.routeState.activeSessionByAgent }
    };
  };

  const requestJSON = async (path, options = {}) => {
    const headers = new Headers(options.headers || {});
    if (options.body && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }
    const response = await fetch(path, {
      method: options.method || "GET",
      headers,
      body: options.body
    });
    const payload = await safeReadJSON(response);
    if (!response.ok) {
      throw new Error(typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`);
    }
    return payload;
  };

  const findSelectedAgent = () => {
    return localState.agents.find((item) => String(item?.id || "").trim() === localState.routeState.selectedAgentID) || null;
  };

  const syncDraftFromSelection = () => {
    localState.draft = normalizeAgentBuilderDraft(findSelectedAgent() || {});
  };

  const paint = () => {
    if (localState.loading) {
      container.innerHTML = `<p class="route-loading">${t("loading")}</p>`;
      return;
    }
    const selectedAgent = findSelectedAgent();
    const canDelete = Boolean(selectedAgent && selectedAgent.id);
    const canTest = Boolean(localState.draft.id);
    const managedID = selectedAgent?.id || t("route.agent.form.pending");
    const managedVersion = selectedAgent?.version || t("route.agent.form.pending");
    const managedScope = selectedAgent?.scope || "global";
    container.innerHTML = `<section class="agent-studio-view">
      <aside class="route-surface agent-studio-list-pane">
        <div class="agent-route-pane-head">
          <div class="agent-route-pane-copy">
            <h4>${escapeHTML(t("route.agent.title"))}</h4>
            <p>${escapeHTML(t("route.agent.subtitle"))}</p>
          </div>
          <button class="route-primary-button" type="button" data-agent-create>${escapeHTML(t("route.agent.create"))}</button>
        </div>
        <div class="agent-route-list">${renderAgentBuilderCards(localState.agents, localState.routeState.selectedAgentID)}</div>
      </aside>
      <section class="route-surface agent-studio-form-pane">
        <div class="agent-route-pane-head">
          <div class="agent-route-pane-copy">
            <h4>${escapeHTML(localState.draft.id ? t("route.agent.edit") : t("route.agent.form.new"))}</h4>
            <p>${escapeHTML(t("route.agent.form.managed"))}</p>
          </div>
          <div class="agent-builder-actions">
            <button type="button" data-agent-chat-now ${canTest ? "" : "disabled"}>${escapeHTML(t("route.agent.form.test"))}</button>
            <button type="button" data-agent-reset>${escapeHTML(t("route.agent.form.cancel"))}</button>
          </div>
        </div>
        ${localState.statusMessage ? `<p class="agent-builder-status ${localState.statusKind === "error" ? "is-error" : "is-success"}">${escapeHTML(localState.statusMessage)}</p>` : ""}
        <div class="agent-builder-managed">
          <div class="agent-builder-managed-item">
            <span>${escapeHTML(t("route.agent.form.id"))}</span>
            <strong>${escapeHTML(managedID)}</strong>
          </div>
          <div class="agent-builder-managed-item">
            <span>${escapeHTML(t("route.agent.form.version"))}</span>
            <strong>${escapeHTML(managedVersion)}</strong>
          </div>
          <div class="agent-builder-managed-item">
            <span>${escapeHTML(t("route.agent.form.scope"))}</span>
            <strong>${escapeHTML(managedScope)}</strong>
          </div>
        </div>
        <form class="agent-builder-form" data-agent-form>
          <label><span>${t("route.agent.form.name")}</span><input type="text" name="name" value="${escapeHTML(localState.draft.name)}" placeholder="Researcher"></label>
          <label><span>${t("route.agent.form.iterations")}</span><input type="number" min="0" name="max_iterations" value="${escapeHTML(localState.draft.max_iterations)}"></label>
          <label class="agent-builder-toggle"><span>${t("route.agent.form.enabled")}</span><input type="checkbox" name="enabled" ${localState.draft.enabled ? "checked" : ""}></label>
          <label class="agent-builder-wide"><span>${t("route.agent.form.prompt")}</span><textarea name="system_prompt" rows="6">${escapeHTML(localState.draft.system_prompt)}</textarea></label>
          <label class="agent-builder-wide"><span>${t("route.agent.form.tools")}</span><input type="text" name="tools" value="${escapeHTML(localState.draft.tools.join(", "))}" placeholder="codex_exec, search_memory, read_memory, write_memory, delegate_agent"></label>
          <div class="agent-builder-wide agent-builder-section">
            <h5>${escapeHTML(t("route.agent.form.skills"))}</h5>
            <div class="agent-builder-options">${renderAgentOptionList(localState.skills, localState.draft.skills, "skills")}</div>
          </div>
          <div class="agent-builder-wide agent-builder-section">
            <h5>${escapeHTML(t("route.agent.form.mcps"))}</h5>
            <div class="agent-builder-options">${renderAgentOptionList(localState.mcps, localState.draft.mcps, "mcps")}</div>
          </div>
          <div class="agent-builder-wide agent-builder-section">
            <h5>${escapeHTML(t("route.agent.form.memory_files"))}</h5>
            <div class="agent-builder-options">${renderAgentOptionList(localState.memoryFiles, localState.draft.memory_files, "memory_files")}</div>
          </div>
          <div class="task-filter-actions">
            <button class="task-filter-apply" type="submit">${escapeHTML(t("route.agent.form.save"))}</button>
            <button class="task-filter-reset" type="button" data-agent-delete ${canDelete ? "" : "disabled"}>${escapeHTML(t("route.agent.form.delete"))}</button>
          </div>
        </form>
      </section>
    </section>`;
  };

  const reload = async (statusMessage = "", statusKind = "") => {
    localState.loading = true;
    localState.statusMessage = statusMessage;
    localState.statusKind = statusKind;
    paint();
    const [agentPayload, skillPayload, mcpPayload] = await Promise.all([
      fetchJSON("/api/control/agents"),
      fetchJSON("/api/control/skills"),
      fetchJSON("/api/control/mcps")
    ]);
    localState.agents = Array.isArray(agentPayload?.items) ? agentPayload.items : [];
    localState.skills = Array.isArray(skillPayload?.items) ? skillPayload.items : [];
    localState.mcps = Array.isArray(mcpPayload?.items) ? mcpPayload.items : [];
    if (!findSelectedAgent()) {
      localState.routeState.selectedAgentID = localState.agents[0]?.id || "";
    }
    syncDraftFromSelection();
    localState.loading = false;
    persistRouteState();
    paint();
    bind();
  };

  const saveAgent = async (form) => {
    const formData = new FormData(form);
    const selectedAgent = findSelectedAgent();
    const payload = {
      name: String(formData.get("name") || "").trim(),
      enabled: formData.get("enabled") === "on",
      system_prompt: String(formData.get("system_prompt") || "").trim(),
      max_iterations: Number(formData.get("max_iterations") || 0),
      tools: parseAgentListInput(formData.get("tools") || ""),
      skills: formData.getAll("skills").map((item) => String(item || "").trim()).filter(Boolean),
      mcps: formData.getAll("mcps").map((item) => String(item || "").trim()).filter(Boolean),
      memory_files: formData.getAll("memory_files").map((item) => String(item || "").trim()).filter(Boolean)
    };
    try {
      const saved = await requestJSON(selectedAgent?.id ? `/api/control/agents/${encodeURIComponent(selectedAgent.id)}` : "/api/control/agents", {
        method: selectedAgent?.id ? "PUT" : "POST",
        body: JSON.stringify(payload)
      });
      localState.routeState.selectedAgentID = normalizeText(saved?.id);
      await reload(t("route.agent.saved"), "success");
      void refreshChatAgentCatalog();
    } catch (error) {
      localState.statusMessage = t("route.agent.save_failed", {
        error: error instanceof Error ? error.message : "unknown_error"
      });
      localState.statusKind = "error";
      paint();
      bind();
    }
  };

  const deleteAgent = async () => {
    const selected = findSelectedAgent();
    if (!selected || !selected.id) {
      return;
    }
    try {
      await requestJSON(`/api/control/agents/${encodeURIComponent(selected.id)}`, {
        method: "DELETE"
      });
      localState.routeState.selectedAgentID = "";
      await reload(t("route.agent.deleted"), "success");
      void refreshChatAgentCatalog();
    } catch (error) {
      localState.statusMessage = t("route.agent.delete_failed", {
        error: error instanceof Error ? error.message : "unknown_error"
      });
      localState.statusKind = "error";
      paint();
      bind();
    }
  };

  const bind = () => {
    container.querySelectorAll("[data-agent-select]").forEach((node) => {
      node.addEventListener("click", () => {
        const nextID = String(node.getAttribute("data-agent-select") || "").trim();
        if (!nextID || nextID === localState.routeState.selectedAgentID) {
          return;
        }
        localState.routeState.selectedAgentID = nextID;
        syncDraftFromSelection();
        persistRouteState();
        paint();
        bind();
      });
    });

    const createButton = container.querySelector("[data-agent-create]");
    if (createButton) {
      createButton.addEventListener("click", () => {
        localState.routeState.selectedAgentID = "";
        localState.draft = normalizeAgentBuilderDraft();
        persistRouteState();
        paint();
        bind();
      });
    }

    const resetButton = container.querySelector("[data-agent-reset]");
    if (resetButton) {
      resetButton.addEventListener("click", () => {
        syncDraftFromSelection();
        localState.statusMessage = "";
        localState.statusKind = "";
        paint();
        bind();
      });
    }

    const chatNowButton = container.querySelector("[data-agent-chat-now]");
    if (chatNowButton) {
      chatNowButton.addEventListener("click", () => {
        if (!localState.draft.id) {
          return;
        }
        openAgentRuntimeWithTarget({
          type: "agent",
          id: localState.draft.id,
          name: localState.draft.name || localState.draft.id
        });
      });
    }

    const deleteButton = container.querySelector("[data-agent-delete]");
    if (deleteButton) {
      deleteButton.addEventListener("click", async () => {
        await deleteAgent();
      });
    }

    const form = container.querySelector("[data-agent-form]");
    if (form) {
      form.addEventListener("submit", async (event) => {
        event.preventDefault();
        await saveAgent(form);
      });
    }
  };

  await reload();
}

function normalizeSessionRouteFilters(filters = {}) {
  const triggerType = String(filters.triggerType || "").trim().toLowerCase();
  const channelType = String(filters.channelType || "").trim().toLowerCase();
  const channelID = String(filters.channelID || "").trim();
  const messageID = String(filters.messageID || "").trim();
  const jobID = String(filters.jobID || "").trim();
  return {
    triggerType,
    channelType,
    channelID,
    messageID,
    jobID
  };
}

function sessionListQuery(filters = {}, page = 1, pageSize = 50) {
  const normalized = normalizeSessionRouteFilters(filters);
  const params = [];
  params.push(`page=${Math.max(page, 1)}`);
  params.push(`page_size=${Math.max(pageSize, 1)}`);
  if (normalized.triggerType) {
    params.push(`trigger_type=${escapeQueryValue(normalized.triggerType)}`);
  }
  if (normalized.channelType) {
    params.push(`channel_type=${escapeQueryValue(normalized.channelType)}`);
  }
  if (normalized.channelID) {
    params.push(`channel_id=${escapeQueryValue(normalized.channelID)}`);
  }
  if (normalized.messageID) {
    params.push(`message_id=${escapeQueryValue(normalized.messageID)}`);
  }
  if (normalized.jobID) {
    params.push(`job_id=${escapeQueryValue(normalized.jobID)}`);
  }
  return `/api/sessions?${params.join("&")}`;
}

function parseCronExpressionVisual(expression) {
  const normalized = String(expression || "").trim().replace(/\s+/g, " ");
  if (!normalized) {
    return null;
  }
  const parts = normalized.split(" ");
  if (parts.length !== 5) {
    return null;
  }
  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts;
  if (month !== "*") {
    return null;
  }

  const parseStep = (value) => {
    const match = /^\*\/(\d+)$/.exec(value);
    if (!match) {
      return null;
    }
    const parsed = Number.parseInt(match[1], 10);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return null;
    }
    return parsed;
  };

  const parseRangeInt = (value, min, max) => {
    if (!/^\d+$/.test(value)) {
      return null;
    }
    const parsed = Number.parseInt(value, 10);
    if (!Number.isFinite(parsed) || parsed < min || parsed > max) {
      return null;
    }
    return parsed;
  };

  const minuteStep = parseStep(minute);
  if (minuteStep !== null && hour === "*" && dayOfMonth === "*" && dayOfWeek === "*") {
    return {
      mode: "every",
      every: minuteStep,
      unit: "minute",
      time: "09:00",
      weekday: 1
    };
  }

  if (minute === "0") {
    const hourStep = parseStep(hour);
    if (hourStep !== null && dayOfMonth === "*" && dayOfWeek === "*") {
      return {
        mode: "every",
        every: hourStep,
        unit: "hour",
        time: "09:00",
        weekday: 1
      };
    }
    if (hour === "0") {
      const dayStep = parseStep(dayOfMonth);
      if (dayStep !== null && dayOfWeek === "*") {
        return {
          mode: "every",
          every: dayStep,
          unit: "day",
          time: "09:00",
          weekday: 1
        };
      }
    }
  }

  const parsedMinute = parseRangeInt(minute, 0, 59);
  const parsedHour = parseRangeInt(hour, 0, 23);
  if (parsedMinute === null || parsedHour === null) {
    return null;
  }

  const timeValue = `${String(parsedHour).padStart(2, "0")}:${String(parsedMinute).padStart(2, "0")}`;
  if (dayOfMonth === "*" && dayOfWeek === "*") {
    return {
      mode: "daily",
      every: 1,
      unit: "day",
      time: timeValue,
      weekday: 1
    };
  }
  if (dayOfMonth === "*") {
    const parsedWeekday = parseRangeInt(dayOfWeek, 0, 7);
    if (parsedWeekday === null) {
      return null;
    }
    return {
      mode: "weekly",
      every: 1,
      unit: "day",
      time: timeValue,
      weekday: parsedWeekday === 7 ? 0 : parsedWeekday
    };
  }
  return null;
}

function buildCronExpressionVisual(options = {}) {
  const mode = String(options.mode || "every").trim().toLowerCase();
  const unit = String(options.unit || "minute").trim().toLowerCase();
  const everyValueRaw = Number.parseInt(options.every, 10);
  const everyValue = Number.isFinite(everyValueRaw) && everyValueRaw > 0 ? everyValueRaw : 1;
  const timeValue = String(options.time || "09:00").trim();
  const timeMatch = /^(\d{1,2}):(\d{1,2})$/.exec(timeValue);
  const hour = timeMatch ? Math.min(Math.max(Number.parseInt(timeMatch[1], 10), 0), 23) : 9;
  const minute = timeMatch ? Math.min(Math.max(Number.parseInt(timeMatch[2], 10), 0), 59) : 0;
  const weekdayRaw = Number.parseInt(options.weekday, 10);
  const weekday = Number.isFinite(weekdayRaw) && weekdayRaw >= 0 && weekdayRaw <= 6 ? weekdayRaw : 1;

  if (mode === "daily") {
    return `${minute} ${hour} * * *`;
  }
  if (mode === "weekly") {
    return `${minute} ${hour} * * ${weekday}`;
  }
  if (unit === "hour") {
    return `0 */${everyValue} * * *`;
  }
  if (unit === "day") {
    return `0 0 */${everyValue} * *`;
  }
  return `*/${everyValue} * * * *`;
}

function renderSessionRouteCards(items) {
  if (!items.length) {
    return `<p class="route-empty">${t("route.sessions.empty")}</p>`;
  }
  return items.map((item) => {
    const sessionID = typeof item?.session_id === "string" ? item.session_id : "";
    const channelType = typeof item?.channel_type === "string" ? item.channel_type : "";
    const channelID = typeof item?.channel_id === "string" ? item.channel_id : "";
    const lastMessageID = typeof item?.last_message_id === "string" ? item.last_message_id : "";
    const updatedAt = typeof item?.updated_at === "string" && item.updated_at.trim()
      ? item.updated_at
      : item?.last_message_at;
    const createdAt = typeof item?.created_at === "string" && item.created_at.trim()
      ? item.created_at
      : item?.started_at;
    const messageCount = Number(item?.message_count || 0);
    const triggerType = typeof item?.trigger_type === "string" ? item.trigger_type : "";
    const jobID = typeof item?.job_id === "string" ? item.job_id : "";
    const jobName = typeof item?.job_name === "string" ? item.job_name : "";
    const firedAt = typeof item?.fired_at === "string" ? item.fired_at : "";
    const title = sessionID || t("route.sessions.title");
    const tags = [formatTriggerType(triggerType), formatChannelType(channelType)];
    if (jobName) {
      tags.push(jobName);
    }
    const detailBody = `<details class="session-route-detail">
      <summary>${t("route.sessions.open_detail")}</summary>
      <div class="route-meta">
        ${routeFieldRow("field.channel_id", channelID, { copyable: true, mono: true })}
        ${routeFieldRow("field.created", formatDateTime(createdAt))}
        ${routeFieldRow("field.messages", messageCount)}
        ${routeFieldRow("field.trigger_type", formatTriggerType(triggerType))}
        ${routeFieldRow("field.job_id", jobID, { copyable: true, mono: true })}
        ${routeFieldRow("field.job_name", jobName)}
        ${routeFieldRow("field.fired_at", formatDateTime(firedAt))}
      </div>
    </details>`;
    return routeCardTemplate(
      title,
      "session",
      [
        routeFieldRow("field.id", sessionID, { copyable: true, mono: true }),
        routeFieldRow("field.channel_type", formatChannelType(channelType)),
        routeFieldRow("field.last_message_id", lastMessageID, { copyable: true, mono: true }),
        routeFieldRow("field.updated", formatDateTime(updatedAt))
      ],
      true,
      detailBody,
      {
        className: "session-route-card",
        footer: renderRouteTagSection("field.tags", tags)
      }
    );
  }).join("");
}

async function loadSessionsView(container) {
  const filters = normalizeSessionRouteFilters(state.sessionRouteFilters || {});
  state.sessionRouteFilters = filters;
  const data = await fetchJSON(sessionListQuery(filters, 1, 50));
  const items = Array.isArray(data.items) ? data.items : [];
  container.innerHTML = `<section class="session-history-view">
    <form class="task-filter-form page-filter-form page-filter-grid-4 session-filter-form" data-session-filter-form>
      <label>
        <span>${t("route.sessions.filter.trigger_type")}</span>
        <select name="trigger_type">
          <option value="">-</option>
          <option value="user">${t("trigger.user")}</option>
          <option value="cron">${t("trigger.cron")}</option>
          <option value="system">${t("trigger.system")}</option>
        </select>
      </label>
      <label>
        <span>${t("route.sessions.filter.channel_type")}</span>
        <select name="channel_type">
          <option value="">-</option>
          <option value="cli">${t("channel.cli")}</option>
          <option value="web">${t("channel.web")}</option>
          <option value="scheduler">${t("channel.scheduler")}</option>
        </select>
      </label>
      <label>
        <span>${t("route.sessions.filter.channel_id")}</span>
        <input type="text" name="channel_id" placeholder="web-default">
      </label>
      <label>
        <span>${t("route.sessions.filter.message_id")}</span>
        <input type="text" name="message_id" placeholder="msg-123">
      </label>
      <label>
        <span>${t("route.sessions.filter.job_id")}</span>
        <input type="text" name="job_id" placeholder="job-daily-report">
      </label>
      <div class="task-filter-actions session-filter-actions">
        <button type="submit">${t("route.sessions.filter.apply")}</button>
        <button type="button" data-session-filter-reset>${t("route.sessions.filter.reset")}</button>
      </div>
    </form>
    <div class="task-summary-list">${renderSessionRouteCards(items)}</div>
  </section>`;

  const form = container.querySelector("[data-session-filter-form]");
  if (!form) {
    return;
  }
  form.trigger_type.value = filters.triggerType;
  form.channel_type.value = filters.channelType;
  form.channel_id.value = filters.channelID;
  form.message_id.value = filters.messageID;
  form.job_id.value = filters.jobID;

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    state.sessionRouteFilters = normalizeSessionRouteFilters({
      triggerType: form.trigger_type.value,
      channelType: form.channel_type.value,
      channelID: form.channel_id.value,
      messageID: form.message_id.value,
      jobID: form.job_id.value
    });
    void renderRoute("sessions");
  });

  const resetButton = form.querySelector("[data-session-filter-reset]");
  if (resetButton) {
    resetButton.addEventListener("click", () => {
      state.sessionRouteFilters = normalizeSessionRouteFilters({});
      void renderRoute("sessions");
    });
  }

}

function renderCronJobCards(items) {
  if (!items.length) {
    return `<p class="route-empty">${t("route.cron.empty")}</p>`;
  }
  return items.map((item) => {
    const jobID = typeof item?.id === "string" ? item.id : "";
    const jobName = typeof item?.name === "string" && item.name.trim() ? item.name : jobID;
    const expression = typeof item?.cron_expression === "string" ? item.cron_expression : "";
    const scheduleMode = typeof item?.schedule_mode === "string" ? item.schedule_mode : "";
    const timezone = typeof item?.timezone === "string" ? item.timezone : "";
    const taskInput = typeof item?.task_config?.input === "string" ? item.task_config.input : item?.content;
    const retryLimit = Number(item?.task_config?.retry_limit || 0);
    const actionButtons = `<div class="route-card-actions">
      <button type="button" data-cron-edit="${escapeHTML(jobID)}">${t("route.cron.action.edit")}</button>
      <button type="button" data-cron-runs-btn="${escapeHTML(jobID)}">${t("route.cron.action.runs")}</button>
      <button type="button" data-cron-delete="${escapeHTML(jobID)}">${t("route.cron.action.delete")}</button>
    </div>`;
    const runsBody = `<div class="cron-run-list" data-cron-runs="${escapeHTML(jobID)}" hidden></div>`;

    return routeCardTemplate(
      jobName,
      "cron",
      [
        routeFieldRow("field.id", jobID, { copyable: true, mono: true }),
        routeFieldRow("field.schedule_mode", normalizeText(scheduleMode)),
        routeFieldRow("field.cron_expression", expression, { multiline: true, mono: true }),
        routeFieldRow("field.timezone", timezone),
        routeFieldRow("field.input", taskInput, { multiline: true, preview: true, clampLines: 4 }),
        routeFieldRow("field.retry_limit", retryLimit)
      ],
      item.enabled,
      runsBody,
      {
        footer: `${renderRouteTagSection("field.tags", [normalizeText(scheduleMode), normalizeText(timezone)])}${actionButtons}`,
        footerClassName: "route-card-footer-spread"
      }
    );
  }).join("");
}

function renderCronRunsList(items, jobID) {
  if (!items.length) {
    return `<p class="route-empty">${t("route.cron.runs.empty")}</p>`;
  }
  return items.map((item) => {
    const runID = typeof item?.run_id === "string" ? item.run_id : "";
    const sessionID = typeof item?.session_id === "string" ? item.session_id : "";
    const status = typeof item?.status === "string" ? item.status : "";
    const firedAt = typeof item?.fired_at === "string" ? item.fired_at : "";
    return `<article class="route-card cron-run-item">
      <div class="route-meta cron-run-meta">
        ${routeFieldRow("field.id", runID, { copyable: true, mono: true })}
        ${routeFieldRow("field.session", sessionID, { copyable: true, mono: true })}
        ${routeFieldRow("field.fired_at", formatDateTime(firedAt))}
        ${routeFieldRow("field.status", formatTaskStatus(status))}
      </div>
      <button type="button" data-cron-open-sessions="${escapeHTML(jobID)}">${t("route.cron.runs.open_sessions")}</button>
    </article>`;
  }).join("");
}

async function loadCronJobsView(container) {
  const data = await fetchJSON("/api/control/cron/jobs");
  const items = Array.isArray(data.items) ? data.items : [];
  const defaultTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  container.innerHTML = `<section class="cron-view">
    <form class="task-filter-form page-filter-form page-filter-grid-4 cron-form" data-cron-form>
      <h4 class="cron-form-title">${t("route.cron.form.title")}</h4>
      <label><span>${t("route.cron.form.job_id")}</span><input type="text" name="job_id" required></label>
      <label><span>${t("route.cron.form.name")}</span><input type="text" name="name" required></label>
      <label><span>${t("route.cron.form.timezone")}</span><input type="text" name="timezone" value="${escapeHTML(defaultTimezone)}"></label>
      <label><span>${t("route.cron.form.retry")}</span><input type="number" min="0" name="retry_limit" value="0"></label>
      <label class="cron-form-wide"><span>${t("route.cron.form.input")}</span><textarea name="input" rows="3" required></textarea></label>
      <label><span>${t("route.cron.form.mode")}</span>
        <select name="schedule_mode">
          <option value="every">${t("route.cron.mode.every")}</option>
          <option value="daily">${t("route.cron.mode.daily")}</option>
          <option value="weekly">${t("route.cron.mode.weekly")}</option>
        </select>
      </label>
      <label data-cron-every-wrap><span>${t("route.cron.form.every")}</span><input type="number" min="1" name="every" value="1"></label>
      <label data-cron-unit-wrap><span>${t("route.cron.form.unit")}</span>
        <select name="unit">
          <option value="minute">${t("route.cron.unit.minute")}</option>
          <option value="hour">${t("route.cron.unit.hour")}</option>
          <option value="day">${t("route.cron.unit.day")}</option>
        </select>
      </label>
      <label data-cron-time-wrap hidden><span>${t("route.cron.form.time")}</span><input type="time" name="fixed_time" value="09:00"></label>
      <label data-cron-weekday-wrap hidden><span>${t("route.cron.form.weekday")}</span>
        <select name="weekday">
          <option value="0">${t("route.cron.weekday.0")}</option>
          <option value="1">${t("route.cron.weekday.1")}</option>
          <option value="2">${t("route.cron.weekday.2")}</option>
          <option value="3">${t("route.cron.weekday.3")}</option>
          <option value="4">${t("route.cron.weekday.4")}</option>
          <option value="5">${t("route.cron.weekday.5")}</option>
          <option value="6">${t("route.cron.weekday.6")}</option>
        </select>
      </label>
      <label class="cron-form-wide"><span>${t("route.cron.form.expression")}</span><input type="text" name="cron_expression" required></label>
      <label class="cron-enabled-wrap"><span>${t("route.cron.form.enabled")}</span><input type="checkbox" name="enabled" checked></label>
      <p class="cron-form-note" data-cron-note></p>
      <div class="task-filter-actions cron-form-actions">
        <button type="submit">${t("route.cron.form.submit")}</button>
        <button type="button" data-cron-reset>${t("route.cron.form.reset")}</button>
      </div>
    </form>
    <section class="cron-list-wrap">
      <h4 class="cron-form-title">${t("route.cron.list.title")}</h4>
      <div class="cron-job-grid">${renderCronJobCards(items)}</div>
    </section>
  </section>`;

  const form = container.querySelector("[data-cron-form]");
  if (!form) {
    return;
  }
  const cronComposer = createReusableComposer();
  const note = form.querySelector("[data-cron-note]");
  const everyWrap = form.querySelector("[data-cron-every-wrap]");
  const unitWrap = form.querySelector("[data-cron-unit-wrap]");
  const timeWrap = form.querySelector("[data-cron-time-wrap]");
  const weekdayWrap = form.querySelector("[data-cron-weekday-wrap]");
  const modeInput = form.schedule_mode;
  const expressionInput = form.cron_expression;
  const promptInput = form.input;

  const syncModeVisibility = () => {
    const mode = String(modeInput.value || "every").trim().toLowerCase();
    const everyMode = mode === "every";
    const weeklyMode = mode === "weekly";
    if (everyWrap) {
      everyWrap.hidden = !everyMode;
    }
    if (unitWrap) {
      unitWrap.hidden = !everyMode;
    }
    if (timeWrap) {
      timeWrap.hidden = everyMode;
    }
    if (weekdayWrap) {
      weekdayWrap.hidden = !weeklyMode;
    }
  };

  const syncExpressionFromVisual = () => {
    expressionInput.value = buildCronExpressionVisual({
      mode: modeInput.value,
      every: form.every.value,
      unit: form.unit.value,
      time: form.fixed_time.value,
      weekday: form.weekday.value
    });
    if (note) {
      note.textContent = "";
    }
  };

  const syncVisualFromExpression = () => {
    const parsed = parseCronExpressionVisual(expressionInput.value);
    if (!parsed) {
      if (note) {
        note.textContent = t("route.cron.expression_invalid");
      }
      return;
    }
    modeInput.value = parsed.mode;
    form.every.value = parsed.every;
    form.unit.value = parsed.unit;
    form.fixed_time.value = parsed.time;
    form.weekday.value = parsed.weekday;
    syncModeVisibility();
    if (note) {
      note.textContent = "";
    }
  };

  const setFormFromJob = (job) => {
    const jobID = typeof job?.id === "string" ? job.id : "";
    const jobName = typeof job?.name === "string" ? job.name : "";
    const timezone = typeof job?.timezone === "string" && job.timezone.trim() ? job.timezone : defaultTimezone;
    const expression = typeof job?.cron_expression === "string" ? job.cron_expression : "";
    const mode = typeof job?.schedule_mode === "string" ? job.schedule_mode : "every";
    const taskInput = typeof job?.task_config?.input === "string" ? job.task_config.input : (job?.content || "");
    const retryLimit = Number(job?.task_config?.retry_limit || 0);
    form.job_id.value = jobID;
    form.name.value = jobName;
    form.timezone.value = timezone;
    form.input.value = taskInput;
    form.retry_limit.value = retryLimit;
    form.enabled.checked = Boolean(job?.enabled);
    modeInput.value = mode || "every";
    expressionInput.value = expression;
    syncVisualFromExpression();
    if (!parseCronExpressionVisual(expression)) {
      syncExpressionFromVisual();
    }
  };

  const resetForm = () => {
    form.reset();
    form.enabled.checked = true;
    form.timezone.value = defaultTimezone;
    form.fixed_time.value = "09:00";
    modeInput.value = "every";
    form.every.value = "1";
    form.unit.value = "minute";
    form.weekday.value = "1";
    syncModeVisibility();
    syncExpressionFromVisual();
  };

  syncModeVisibility();
  syncExpressionFromVisual();

  modeInput.addEventListener("change", () => {
    syncModeVisibility();
    syncExpressionFromVisual();
  });
  form.every.addEventListener("input", syncExpressionFromVisual);
  form.unit.addEventListener("change", syncExpressionFromVisual);
  form.fixed_time.addEventListener("change", syncExpressionFromVisual);
  form.weekday.addEventListener("change", syncExpressionFromVisual);
  expressionInput.addEventListener("change", syncVisualFromExpression);

  cronComposer.bind(promptInput, form, {
    stableName: "cron-prompt",
    submitOnEnter: false,
    draftStorage: "session",
    draftKey: "cron.prompt"
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const jobID = String(form.job_id.value || "").trim();
    const cronExpression = String(form.cron_expression.value || "").trim();
    if (!jobID || !cronExpression) {
      return;
    }
    const retryLimit = Number.parseInt(form.retry_limit.value, 10);
    const safeRetryLimit = Number.isFinite(retryLimit) && retryLimit >= 0 ? retryLimit : 0;
    const payload = {
      name: String(form.name.value || "").trim(),
      enabled: Boolean(form.enabled.checked),
      timezone: String(form.timezone.value || "").trim() || defaultTimezone,
      schedule_mode: String(modeInput.value || "every").trim().toLowerCase(),
      cron_expression: cronExpression,
      task_config: {
        input: String(form.input.value || "").trim(),
        retry_limit: safeRetryLimit
      }
    };

    try {
      const response = await fetch(`/api/control/cron/jobs/${encodeURIComponent(jobID)}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify(payload)
      });
      if (!response.ok) {
        let reason = `HTTP ${response.status}`;
        try {
          const body = await response.json();
          if (body && typeof body.error === "string" && body.error.trim()) {
            reason = body.error;
          }
        } catch {}
        throw new Error(reason);
      }
      await loadCronJobsView(container);
    } catch (error) {
      if (note) {
        const reason = error instanceof Error ? error.message : "unknown_error";
        note.textContent = t("route.cron.save_failed", { error: reason });
      }
    }
  });

  const resetButton = form.querySelector("[data-cron-reset]");
  if (resetButton) {
    resetButton.addEventListener("click", resetForm);
  }

  const findJobByID = (jobID) => items.find((item) => String(item?.id || "") === jobID);
  const findRunsContainer = (jobID) => {
    const nodes = container.querySelectorAll("[data-cron-runs]");
    for (const node of nodes) {
      if (node.dataset.cronRuns === jobID) {
        return node;
      }
    }
    return null;
  };

  const bindOpenSessionsAction = (scope = container) => {
    const buttons = scope.querySelectorAll("[data-cron-open-sessions]");
    for (const button of buttons) {
      button.addEventListener("click", () => {
        const jobID = String(button.dataset.cronOpenSessions || "").trim();
        state.sessionRouteFilters = normalizeSessionRouteFilters({
          triggerType: "cron",
          jobID
        });
        navigateToRoute("sessions");
      });
    }
  };

  const editButtons = container.querySelectorAll("[data-cron-edit]");
  for (const button of editButtons) {
    button.addEventListener("click", () => {
      const jobID = String(button.dataset.cronEdit || "").trim();
      const job = findJobByID(jobID);
      if (!job) {
        return;
      }
      setFormFromJob(job);
      window.requestAnimationFrame(() => {
        form.job_id.focus();
      });
    });
  }

  const deleteButtons = container.querySelectorAll("[data-cron-delete]");
  for (const button of deleteButtons) {
    button.addEventListener("click", async () => {
      const jobID = String(button.dataset.cronDelete || "").trim();
      if (!jobID) {
        return;
      }
      try {
        const response = await fetch(`/api/control/cron/jobs/${encodeURIComponent(jobID)}`, {
          method: "DELETE"
        });
        if (!response.ok) {
          let reason = `HTTP ${response.status}`;
          try {
            const body = await response.json();
            if (body && typeof body.error === "string" && body.error.trim()) {
              reason = body.error;
            }
          } catch {}
          throw new Error(reason);
        }
        await loadCronJobsView(container);
      } catch (error) {
        if (note) {
          const reason = error instanceof Error ? error.message : "unknown_error";
          note.textContent = t("route.cron.delete_failed", { error: reason });
        }
      }
    });
  }

  const runsButtons = container.querySelectorAll("[data-cron-runs-btn]");
  for (const button of runsButtons) {
    button.addEventListener("click", async () => {
      const jobID = String(button.dataset.cronRunsBtn || "").trim();
      const runsContainer = findRunsContainer(jobID);
      if (!runsContainer) {
        return;
      }
      if (!runsContainer.hidden) {
        runsContainer.hidden = true;
        return;
      }
      if (runsContainer.dataset.loaded === "true") {
        runsContainer.hidden = false;
        return;
      }
      try {
        const data = await fetchJSON(`/api/control/cron/jobs/${encodeURIComponent(jobID)}/runs?page=1&page_size=20`);
        const runItems = Array.isArray(data.items) ? data.items : [];
        runsContainer.innerHTML = renderCronRunsList(runItems, jobID);
        runsContainer.dataset.loaded = "true";
        runsContainer.hidden = false;
        bindOpenSessionsAction(runsContainer);
      } catch (error) {
        const reason = error instanceof Error ? error.message : "unknown_error";
        runsContainer.innerHTML = `<p class="route-error">${t("route.cron.runs_failed", { error: reason })}</p>`;
        runsContainer.hidden = false;
      }
    });
  }
}

function controlTaskListQuery(filters = {}, page = 1, pageSize = 20) {
  const params = [];
  params.push(`page=${Math.max(page, 1)}`);
  params.push(`page_size=${Math.max(pageSize, 1)}`);
  if (filters.sessionID) {
    params.push(`session_id=${escapeQueryValue(filters.sessionID)}`);
  }
  if (filters.status) {
    params.push(`status=${escapeQueryValue(filters.status)}`);
  }
  if (filters.triggerType) {
    params.push(`trigger_type=${escapeQueryValue(filters.triggerType)}`);
  }
  if (filters.channelType) {
    params.push(`channel_type=${escapeQueryValue(filters.channelType)}`);
  }
  if (filters.channelID) {
    params.push(`channel_id=${escapeQueryValue(filters.channelID)}`);
  }
  if (filters.messageID) {
    params.push(`message_id=${escapeQueryValue(filters.messageID)}`);
  }
  if (filters.sourceMessageID) {
    params.push(`source_message_id=${escapeQueryValue(filters.sourceMessageID)}`);
  }
  if (filters.startAt && filters.endAt) {
    params.push(`time_range=${escapeQueryValue(`${filters.startAt},${filters.endAt}`)}`);
  } else {
    if (filters.startAt) {
      params.push(`start_at=${escapeQueryValue(filters.startAt)}`);
    }
    if (filters.endAt) {
      params.push(`end_at=${escapeQueryValue(filters.endAt)}`);
    }
  }
  return `/api/control/tasks?${params.join("&")}`;
}

function renderControlTaskList(payload, activeTaskID = "") {
  const items = Array.isArray(payload?.items) ? payload.items : [];
  if (!items.length) {
    return `<div class="task-empty-state">
      <div class="task-empty-icon" aria-hidden="true">${renderTaskEmptyIcon()}</div>
      <p class="task-empty-title">${t("route.tasks.empty")}</p>
      <p class="task-empty-hint">${t("route.tasks.empty_hint")}</p>
    </div>`;
  }
  return items.map((item) => {
    const taskID = typeof item?.task_id === "string" ? item.task_id : "-";
    const active = taskID && taskID === activeTaskID;
    const sessionID = typeof item?.session_id === "string" ? item.session_id : "-";
    const status = typeof item?.status === "string" ? item.status : "";
    const triggerType = typeof item?.trigger_type === "string" ? item.trigger_type : "";
    const channelType = typeof item?.channel_type === "string" ? item.channel_type : "";
    const sourceMessageID = typeof item?.source_message_id === "string" ? item.source_message_id : "";
    const updatedAt = typeof item?.updated_at === "string" ? item.updated_at : "";
    const lastHeartbeatAt = typeof item?.last_heartbeat_at === "string" ? item.last_heartbeat_at : "";
    const timeoutAt = typeof item?.timeout_at === "string" ? item.timeout_at : "";
    const jobID = typeof item?.job_id === "string" ? item.job_id : "";
    const statusClassName = taskStatusClassName(status);
    const heartbeatRow = formatTaskHeartbeatSummary(lastHeartbeatAt, timeoutAt) !== "-"
      ? renderTaskSummaryMetaRow("field.last_heartbeat_at", formatTaskHeartbeatSummary(lastHeartbeatAt, timeoutAt))
      : "";
    const summaryTags = [
      formatTriggerType(triggerType),
      formatChannelType(channelType)
    ];
    if (triggerType === "cron" && normalizeText(jobID) !== "-") {
      summaryTags.push(`${t("field.job_id")}: ${normalizeText(jobID)}`);
    }
    return `<article class="route-card task-summary-card ${active ? "active" : ""}" data-control-task-id="${escapeHTML(taskID)}" ${active ? 'aria-current="true"' : ""}>
      <header class="task-summary-head">
        <div class="task-summary-id-wrap">
          <h5 class="task-summary-id" title="${escapeHTML(taskID)}">${escapeHTML(taskID)}</h5>
          <button class="task-summary-copy" type="button" data-control-task-copy-id="${escapeHTML(taskID)}" title="${t("route.tasks.copy_task_id")}" aria-label="${t("route.tasks.copy_task_id")}">${renderCopyIcon()}</button>
        </div>
        <span class="task-summary-status ${statusClassName}">${escapeHTML(formatTaskStatus(status))}</span>
      </header>
      <div class="task-summary-meta">
        ${renderTaskSummaryMetaRow("field.session", sessionID, { mono: true })}
        ${renderTaskSummaryMetaRow("field.source_message", sourceMessageID, { mono: true })}
        ${renderTaskSummaryMetaRow("field.updated", formatDateTime(updatedAt))}
        ${heartbeatRow}
      </div>
      <footer class="route-card-footer control-task-summary-footer">
        ${renderRouteTagSection("field.tags", summaryTags, "control-task-summary-tags")}
        <button class="task-summary-open" type="button" data-control-task-open="${escapeHTML(taskID)}"><span class="task-summary-open-icon" aria-hidden="true">${renderPanelRightOpenIcon()}</span><span>${t("route.tasks.open_detail")}</span></button>
      </footer>
    </article>`;
  }).join("");
}

function renderControlTaskPagination(payload) {
  const pagination = payload?.pagination || {};
  const hasNext = Boolean(pagination?.has_next);
  const page = Number(pagination?.page || 1);
  const total = Number(pagination?.total || 0);
  return `<div class="task-summary-pagination">
    <p><span>${t("field.messages")}</span><strong>${escapeHTML(total)}</strong></p>
    <p><span>${t("route.tasks.page.label")}</span><strong>${escapeHTML(page)}</strong></p>
    <button class="task-summary-next" type="button" data-control-task-page-next ${hasNext ? "" : "disabled"}><span>${t("route.tasks.page.next")}</span><span class="task-summary-next-icon" aria-hidden="true">${renderChevronRightIcon()}</span></button>
  </div>`;
}

function renderControlTaskDetail(view, displayTaskID = "") {
  const task = view?.task || {};
  const source = view?.source || {};
  const actions = view?.actions || {};
  const link = view?.link || {};
  const taskID = typeof task?.id === "string" ? task.id : "-";
  const pinnedTaskID = normalizeText(displayTaskID);
  const shownTaskID = pinnedTaskID && pinnedTaskID !== "-" ? pinnedTaskID : taskID;
  const status = typeof task?.status === "string" ? task.status : "";
  const statusValue = String(status || "").trim().toLowerCase();
  const statusClassName = taskStatusClassName(status);
  const phase = typeof task?.phase === "string" && task.phase.trim() ? task.phase : status;
  const triggerType = typeof source?.trigger_type === "string" ? source.trigger_type : "";
  const channelType = typeof source?.channel_type === "string" ? source.channel_type : "";
  const channelID = typeof source?.channel_id === "string" ? source.channel_id : "";
  const correlationID = typeof source?.correlation_id === "string" ? source.correlation_id : "";
  const jobID = typeof source?.job_id === "string" ? source.job_id : "";
  const jobName = typeof source?.job_name === "string" ? source.job_name : "";
  const firedAt = typeof source?.fired_at === "string" ? source.fired_at : "";
  const messageID = typeof task?.message_id === "string" ? task.message_id : "";
  const requestMessageID = typeof link?.request_message_id === "string" ? link.request_message_id : "";
  const resultMessageID = typeof link?.result_message_id === "string" ? link.result_message_id : "";
  const terminalSessionID = typeof link?.terminal_session_id === "string" ? link.terminal_session_id : "";
  const queuePosition = Number(task?.queue_position || 0);
  const queueWaitMS = Number(task?.queue_wait_ms || 0);
  const lastHeartbeatAt = normalizeText(task?.last_heartbeat_at);
  const timeoutAt = normalizeText(task?.timeout_at);
  const resultOutput = typeof task?.result?.output === "string" ? task.result.output : "";
  const retryEnabled = Boolean(actions?.retry?.enabled);
  const cancelEnabled = Boolean(actions?.cancel?.enabled);
  const showCancelAction = cancelEnabled || !["success", "done", "failed", "error", "canceled"].includes(statusValue);
  const retryReason = typeof actions?.retry?.reason === "string" ? actions.retry.reason : "";
  const cancelReason = typeof actions?.cancel?.reason === "string" ? actions.cancel.reason : "";
  const progressRaw = Number(task?.progress);
  const progressValue = Number.isFinite(progressRaw) ? Math.min(100, Math.max(0, progressRaw)) : 0;
  const taskIDShort = shorten(shownTaskID, 24);
  const detailTags = [formatTriggerType(triggerType), formatChannelType(channelType)];
  if (terminalSessionID) {
    detailTags.push(terminalSessionID);
  }
  const cronRows = triggerType === "cron"
    ? `<p><span>${t("field.job_id")}</span><strong>${escapeHTML(normalizeText(jobID))}</strong></p>
      <p><span>${t("field.job_name")}</span><strong>${escapeHTML(normalizeText(jobName))}</strong></p>
      <p><span>${t("field.fired_at")}</span><strong>${escapeHTML(formatDateTime(firedAt))}</strong></p>`
    : "";
  const errorText = normalizeText(task?.error_message || task?.error_code || "-");
  const runtimeRows = `<p><span>${t("field.queue_position")}</span><strong>${escapeHTML(queuePosition > 0 ? queuePosition : "-")}</strong></p>
      <p><span>${t("field.queue_wait_ms")}</span><strong>${escapeHTML(formatDurationMS(queueWaitMS))}</strong></p>
      <p><span>${t("field.accepted_at")}</span><strong>${escapeHTML(formatDateTime(task?.accepted_at))}</strong></p>
      <p><span>${t("field.started_at")}</span><strong>${escapeHTML(formatDateTime(task?.started_at))}</strong></p>
      <p><span>${t("field.created")}</span><strong>${escapeHTML(formatDateTime(task?.created_at))}</strong></p>
      <p><span>${t("field.updated")}</span><strong>${escapeHTML(formatDateTime(task?.updated_at))}</strong></p>
      <p><span>${t("field.last_heartbeat_at")}</span><strong>${escapeHTML(formatDateTime(lastHeartbeatAt))}</strong></p>
      <p><span>${t("field.timeout_at")}</span><strong>${escapeHTML(formatDateTime(timeoutAt))}</strong></p>
      <p><span>${t("field.finished")}</span><strong>${escapeHTML(formatDateTime(task?.finished_at))}</strong></p>
      <p><span>${t("field.trigger_type")}</span><strong>${escapeHTML(formatTriggerType(triggerType))}</strong></p>
      <p><span>${t("field.channel_type")}</span><strong>${escapeHTML(formatChannelType(channelType))}</strong></p>
      ${cronRows}`;
  const identifierRows = `<p><span>${t("field.channel_id")}</span><strong>${escapeHTML(normalizeText(channelID))}</strong></p>
      <p><span>${t("field.correlation_id")}</span><strong>${escapeHTML(normalizeText(correlationID))}</strong></p>
      <p><span>${t("field.source_message")}</span><strong>${escapeHTML(normalizeText(task?.source_message_id))}</strong></p>
      <p><span>${t("field.message_id")}</span><strong>${escapeHTML(normalizeText(messageID))}</strong></p>
      <p><span>${t("field.request_message_id")}</span><strong>${escapeHTML(normalizeText(requestMessageID))}</strong></p>
      <p><span>${t("field.result_message_id")}</span><strong>${escapeHTML(normalizeText(resultMessageID))}</strong></p>
      <p><span>Error</span><strong>${escapeHTML(errorText)}</strong></p>
      <p><span>Detail API</span><strong>${escapeHTML(normalizeText(link?.task_detail_path))}</strong></p>`;
  return `<section class="task-detail-card" data-control-task-detail-id="${escapeHTML(taskID)}" data-control-task-session-id="${escapeHTML(normalizeText(task?.session_id))}" data-control-task-terminal-session-id="${escapeHTML(normalizeText(terminalSessionID))}">
    <header class="task-detail-head">
      <div class="task-detail-id-wrap">
        <h5 title="${escapeHTML(shownTaskID)}">${escapeHTML(`${t("field.id")}: ${taskIDShort}`)}</h5>
        <button class="task-summary-copy" type="button" data-control-task-copy-id="${escapeHTML(shownTaskID)}" title="${t("route.tasks.copy_task_id")}" aria-label="${t("route.tasks.copy_task_id")}">${renderCopyIcon()}</button>
      </div>
      <span class="task-summary-status ${statusClassName}">${escapeHTML(formatTaskStatus(status))}</span>
    </header>
    <div class="task-detail-meta">
      <div class="task-detail-meta-core">
        <p><span>${t("field.session")}</span><strong>${escapeHTML(normalizeText(task?.session_id))}</strong></p>
        <p><span>${t("field.task_type")}</span><strong>${escapeHTML(normalizeText(task?.task_type))}</strong></p>
        <p><span>${t("field.phase")}</span><strong>${escapeHTML(normalizeText(phase || "-"))}</strong></p>
        <p><span>${t("field.retry_count")}</span><strong>${escapeHTML(normalizeText(task?.retry_count))}</strong></p>
      </div>
      <div class="task-detail-progress">
        <p><span>${t("route.tasks.progress")}</span><strong>${escapeHTML(`${progressValue}%`)}</strong></p>
        <div class="task-detail-progress-track"><span class="task-detail-progress-fill" style="width:${escapeHTML(progressValue)}%"></span></div>
      </div>
      <details class="task-detail-meta-fold">
        <summary>${t("route.tasks.drawer.runtime")}</summary>
        <div class="task-detail-meta-fold-body">${runtimeRows}</div>
      </details>
      <details class="task-detail-meta-fold">
        <summary>${t("route.tasks.drawer.identifiers")}</summary>
        <div class="task-detail-meta-fold-body">${identifierRows}</div>
      </details>
    </div>
    ${renderRouteTagSection("field.tags", detailTags, "task-detail-tag-section")}
    <div class="task-detail-actions">
      <button type="button" data-control-task-action="retry" ${retryEnabled ? "" : "disabled"} title="${escapeHTML(retryEnabled ? t("route.tasks.actions.retry_tip") : normalizeText(retryReason))}">${t("route.tasks.actions.retry")}</button>
      ${showCancelAction ? `<button type="button" data-control-task-action="cancel" ${cancelEnabled ? "" : "disabled"} title="${escapeHTML(cancelEnabled ? "" : normalizeText(cancelReason))}">${t("route.tasks.actions.cancel")}</button>` : ""}
      <button type="button" data-control-task-log-reconnect>${t("route.tasks.logs.reconnect")}</button>
      <button type="button" data-control-task-log-replay title="${escapeHTML(t("route.tasks.actions.replay_tip"))}">${t("route.tasks.logs.replay")}</button>
    </div>
    ${renderRouteSection(t("route.tasks.terminal.title"), `
      <p class="control-task-log-state" data-control-task-log-status>${t("route.tasks.logs.empty")}</p>
      <div class="control-task-log-stream" data-control-task-log-stream>${t("route.tasks.logs.empty")}</div>
      <div class="control-task-quick-actions">
        <span>${t("route.tasks.drawer.quick_actions")}</span>
        <button type="button" data-control-task-quick-input="${escapeHTML(t("route.tasks.drawer.quick_timeline"))}">${t("route.tasks.drawer.quick_timeline")}</button>
        <button type="button" data-control-task-quick-input="${escapeHTML(t("route.tasks.drawer.quick_interview"))}">${t("route.tasks.drawer.quick_interview")}</button>
      </div>
      <form class="control-task-terminal-input" data-control-task-terminal-input-form>
        <input type="text" data-control-task-terminal-input maxlength="6000" placeholder="${escapeHTML(t("route.tasks.terminal.input_placeholder"))}">
        <button type="submit" data-control-task-terminal-submit>${t("route.tasks.terminal.send")}</button>
      </form>
      <p class="control-task-terminal-hint">${escapeHTML(t("route.tasks.terminal.hint"))}</p>
      <p class="control-task-terminal-note">${escapeHTML(t("route.tasks.terminal.followup_note"))}</p>
    `, { className: "task-detail-section" })}
    ${renderRouteSection(t("route.tasks.result.title"), `
      <div class="control-task-result-output">${renderControlTaskResultOutput(resultOutput)}</div>
    `, { className: "task-detail-section" })}
  </section>`;
}

function bindControlTaskView(container, initialPayload) {
  const view = container.querySelector("[data-control-task-view]");
  if (!view) {
    return;
  }
  const listNode = view.querySelector("[data-control-task-list]");
  const paginationNode = view.querySelector("[data-control-task-pagination]");
  const form = view.querySelector("[data-control-task-filter-form]");
  const layoutNode = view.querySelector("[data-control-task-layout]");
  const advancedFiltersNode = view.querySelector("[data-control-task-advanced]");
  const advancedToggleButton = view.querySelector("[data-control-task-advanced-toggle]");
  const drawer = view.querySelector("[data-control-task-drawer]");
  const drawerBody = view.querySelector("[data-control-task-drawer-body]");
  const localState = {
    filters: {
      sessionID: "",
      status: "",
      triggerType: "",
      channelType: "",
      channelID: "",
      messageID: "",
      sourceMessageID: "",
      startAt: "",
      endAt: ""
    },
    page: 1,
    pageSize: 20,
    activeTaskID: "",
    terminalAnchorTaskID: "",
    logCursor: 0,
    logDone: false,
    logItems: [],
    logSeqSet: new Set(),
    logStream: null,
    logStickToBottom: true,
    logStreamNode: null,
    logTouchStartY: null,
    logPaintFrame: 0,
    terminalSubmitting: false,
    advancedOpen: false
  };
  const controlTaskTerminalComposer = createReusableComposer();

  const syncAdvancedToggle = () => {
    if (advancedFiltersNode) {
      advancedFiltersNode.hidden = !localState.advancedOpen;
    }
    if (advancedToggleButton) {
      advancedToggleButton.setAttribute("aria-expanded", localState.advancedOpen ? "true" : "false");
      advancedToggleButton.innerHTML = renderAdvancedToggleLabel(localState.advancedOpen);
    }
  };

  const setFilterApplyingState = () => {
    listNode.innerHTML = renderControlTaskSkeleton(6);
    paginationNode.innerHTML = `<div class="task-summary-pagination"><p><span>${t("route.tasks.filter.applying")}</span></p></div>`;
  };

  const syncActiveTaskCards = () => {
    const cards = view.querySelectorAll("[data-control-task-id]");
    cards.forEach((card) => {
      const taskID = normalizeText(card.getAttribute("data-control-task-id") || "");
      const active = Boolean(localState.activeTaskID) && taskID === localState.activeTaskID;
      card.classList.toggle("active", active);
      if (active) {
        card.setAttribute("aria-current", "true");
      } else {
        card.removeAttribute("aria-current");
      }
    });
  };

  syncAdvancedToggle();

  if (advancedToggleButton) {
    advancedToggleButton.addEventListener("click", () => {
      localState.advancedOpen = !localState.advancedOpen;
      syncAdvancedToggle();
    });
  }

  const stopLogStream = () => {
    if (!localState.logStream) {
      return;
    }
    localState.logStream.close();
    localState.logStream = null;
  };

  const closeDrawer = () => {
    stopLogStream();
    if (localState.logPaintFrame) {
      window.cancelAnimationFrame(localState.logPaintFrame);
      localState.logPaintFrame = 0;
    }
    controlTaskTerminalComposer.unbind();
    localState.activeTaskID = "";
    localState.terminalAnchorTaskID = "";
    localState.logStreamNode = null;
    localState.logTouchStartY = null;
    localState.logStickToBottom = true;
    if (!drawer) {
      return;
    }
    view.classList.remove("is-detail-open");
    drawer.classList.remove("open");
    drawer.hidden = true;
    syncActiveTaskCards();
    if (drawerBody) {
      drawerBody.innerHTML = t("route.tasks.drawer.empty");
    }
  };

  const openDrawer = () => {
    if (!drawer) {
      return;
    }
    drawer.hidden = false;
    view.classList.add("is-detail-open");
    requestAnimationFrame(() => {
      drawer.classList.add("open");
      if (window.matchMedia("(max-width: 1100px)").matches) {
        const target = layoutNode || drawer;
        target.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    });
  };

  const setLogStatus = (message) => {
    const statusNode = drawerBody.querySelector("[data-control-task-log-status]");
    if (!statusNode) {
      return;
    }
    statusNode.textContent = normalizeText(message || t("route.tasks.logs.empty"));
  };

  const isNearLogBottom = (node, threshold = 24) => {
    if (!node) {
      return true;
    }
    const scrollTop = Math.max(0, Number(node.scrollTop || 0));
    const scrollHeight = Math.max(0, Number(node.scrollHeight || 0));
    const clientHeight = Math.max(0, Number(node.clientHeight || 0));
    return scrollTop + clientHeight >= scrollHeight - threshold;
  };

  const scrollLogToBottom = (node) => {
    if (!node) {
      return;
    }
    const apply = () => {
      node.scrollTop = node.scrollHeight;
    };
    apply();
    requestAnimationFrame(apply);
  };

  const bindLogStreamNode = (streamNode) => {
    if (!streamNode || localState.logStreamNode === streamNode) {
      return;
    }
    localState.logStreamNode = streamNode;
    localState.logStickToBottom = true;
    localState.logTouchStartY = null;

    streamNode.addEventListener("scroll", () => {
      localState.logStickToBottom = isNearLogBottom(streamNode);
    }, { passive: true });

    streamNode.addEventListener("touchstart", (event) => {
      const touch = event.touches && event.touches[0];
      localState.logTouchStartY = touch ? Number(touch.clientY) : null;
    }, { passive: true });

    streamNode.addEventListener("touchmove", (event) => {
      const touch = event.touches && event.touches[0];
      if (!touch || !Number.isFinite(localState.logTouchStartY)) {
        return;
      }
      const currentY = Number(touch.clientY);
      const deltaY = currentY - localState.logTouchStartY;
      const scrollTop = Math.max(0, Number(streamNode.scrollTop || 0));
      const maxScrollTop = Math.max(0, Number(streamNode.scrollHeight || 0) - Number(streamNode.clientHeight || 0));
      const atTop = scrollTop <= 0;
      const atBottom = scrollTop >= maxScrollTop - 1;
      if ((atTop && deltaY > 0) || (atBottom && deltaY < 0)) {
        event.preventDefault();
      }
    }, { passive: false });
  };

  const paintLogs = () => {
    if (localState.logPaintFrame) {
      window.cancelAnimationFrame(localState.logPaintFrame);
      localState.logPaintFrame = 0;
    }
    const streamNode = drawerBody.querySelector("[data-control-task-log-stream]");
    if (!streamNode) {
      return;
    }
    bindLogStreamNode(streamNode);
    const prevScrollTop = Math.max(0, Number(streamNode.scrollTop || 0));
    const prevScrollHeight = Math.max(0, Number(streamNode.scrollHeight || 0));
    const prevClientHeight = Math.max(0, Number(streamNode.clientHeight || 0));
    const stickToBottom = localState.logStickToBottom || (prevScrollTop + prevClientHeight >= prevScrollHeight - 20);

    streamNode.innerHTML = renderControlTaskLogStream(localState.logItems);

    if (stickToBottom) {
      localState.logStickToBottom = true;
      scrollLogToBottom(streamNode);
      return;
    }
    const maxScrollTop = Math.max(0, Number(streamNode.scrollHeight || 0) - Number(streamNode.clientHeight || 0));
    streamNode.scrollTop = Math.min(prevScrollTop, maxScrollTop);
    localState.logStickToBottom = isNearLogBottom(streamNode);
  };

  const schedulePaintLogs = () => {
    if (localState.logPaintFrame) {
      return;
    }
    localState.logPaintFrame = window.requestAnimationFrame(() => {
      localState.logPaintFrame = 0;
      paintLogs();
    });
  };

  const resetLogs = () => {
    localState.logCursor = 0;
    localState.logDone = false;
    localState.logItems = [];
    localState.logSeqSet = new Set();
    localState.logStickToBottom = true;
    localState.logTouchStartY = null;
    setLogStatus(t("route.tasks.logs.empty"));
    schedulePaintLogs();
  };

  const appendLogs = (items) => {
    if (!Array.isArray(items) || !items.length) {
      return;
    }
    items.forEach((item) => {
      const seq = Number(item?.seq || 0);
      const seqKey = seq > 0 ? `seq:${seq}` : `${item?.timestamp || item?.created_at || ""}:${item?.message || ""}`;
      if (localState.logSeqSet.has(seqKey)) {
        return;
      }
      localState.logSeqSet.add(seqKey);
      localState.logItems.push(item);
    });
    localState.logItems.sort((left, right) => Number(left?.seq || 0) - Number(right?.seq || 0));
  };

  const fetchLogPage = async (taskID, cursor, limit = 200) => {
    return fetchJSON(`/api/control/tasks/${encodeURIComponent(taskID)}/logs?cursor=${Math.max(cursor, 0)}&limit=${Math.max(limit, 1)}`);
  };

  const loadLogBackfill = async (taskID, cursor = 0) => {
    let nextCursor = Math.max(cursor, 0);
    let hasMore = true;
    let guard = 0;
    while (hasMore && guard < 40) {
      guard += 1;
      const page = await fetchLogPage(taskID, nextCursor, 200);
      appendLogs(Array.isArray(page?.items) ? page.items : []);
      nextCursor = Number(page?.next_cursor || nextCursor);
      hasMore = Boolean(page?.has_more);
      if (!hasMore) {
        break;
      }
    }
    localState.logCursor = nextCursor;
    schedulePaintLogs();
  };

  const startLogStream = (taskID, cursor = 0) => {
    stopLogStream();
    const safeTaskID = normalizeText(taskID);
    if (!safeTaskID || safeTaskID === "-") {
      return;
    }
    localState.logDone = false;
    const streamURL = `/api/control/tasks/${encodeURIComponent(safeTaskID)}/logs/stream?cursor=${Math.max(cursor, 0)}`;
    const stream = new EventSource(streamURL);
    localState.logStream = stream;
    setLogStatus(t("route.tasks.logs.streaming"));

    stream.addEventListener("start", (event) => {
      const payload = parseJSONPayload(event.data);
      if (!payload) {
        return;
      }
      const cursorValue = Number(payload?.cursor || localState.logCursor);
      if (Number.isFinite(cursorValue) && cursorValue >= 0) {
        localState.logCursor = cursorValue;
      }
      setLogStatus(t("route.tasks.logs.streaming"));
    });

    stream.addEventListener("log", (event) => {
      const payload = parseJSONPayload(event.data);
      if (!payload) {
        return;
      }
      appendLogs([payload?.log || {}]);
      const nextCursor = Number(payload?.next_cursor || localState.logCursor + 1);
      if (Number.isFinite(nextCursor) && nextCursor >= 0) {
        localState.logCursor = nextCursor;
      }
      schedulePaintLogs();
    });

    stream.addEventListener("done", (event) => {
      const payload = parseJSONPayload(event.data);
      if (payload) {
        const nextCursor = Number(payload?.next_cursor || localState.logCursor);
        if (Number.isFinite(nextCursor) && nextCursor >= 0) {
          localState.logCursor = nextCursor;
        }
      }
      localState.logDone = true;
      setLogStatus(t("route.tasks.logs.done"));
      stopLogStream();
    });

    stream.addEventListener("error", () => {
      if (localState.logDone) {
        return;
      }
      setLogStatus(t("route.tasks.logs.disconnected"));
      stopLogStream();
    });
  };

  const paint = (payload) => {
    listNode.innerHTML = renderControlTaskList(payload, localState.activeTaskID);
    paginationNode.innerHTML = renderControlTaskPagination(payload);
    syncActiveTaskCards();
  };

  const loadList = async () => {
    setFilterApplyingState();
    const payload = await fetchJSON(controlTaskListQuery(localState.filters, localState.page, localState.pageSize));
    paint(payload);
  };

  const setTerminalInputState = (submitting) => {
    controlTaskTerminalComposer.setDisabled(submitting);
    const inputNode = drawerBody.querySelector("[data-control-task-terminal-input]");
    const submitNode = drawerBody.querySelector("[data-control-task-terminal-submit]");
    if (inputNode) {
      inputNode.disabled = submitting;
    }
    if (submitNode) {
      submitNode.disabled = submitting;
      submitNode.textContent = submitting ? t("route.tasks.terminal.sending") : t("route.tasks.terminal.send");
    }
  };

  const bindControlTaskTerminalComposer = () => {
    const formNode = drawerBody.querySelector("[data-control-task-terminal-input-form]");
    const inputNode = drawerBody.querySelector("[data-control-task-terminal-input]");
    controlTaskTerminalComposer.bind(inputNode, formNode, {
      stableName: "control-task-terminal",
      submitOnEnter: true,
      submitStrategy: "form",
      draftStorage: "session",
      draftKey: () => `control-task-terminal:${normalizeText(localState.terminalAnchorTaskID || localState.activeTaskID || "default")}`,
      clearDraftOnSubmit: true,
      submitNode: drawerBody.querySelector("[data-control-task-terminal-submit]"),
      disabled: localState.terminalSubmitting,
      onSubmit: async (currentInputNode) => {
        const value = String(currentInputNode.value || "");
        controlTaskTerminalComposer.clearDraft();
        await submitTerminalInput(value);
      }
    });
  };

  const submitTerminalInput = async (rawInput) => {
    const taskID = normalizeText(localState.activeTaskID);
    if (!taskID || taskID === "-") {
      return;
    }

    const content = normalizeText(rawInput);
    if (!content || content === "-") {
      return;
    }

    localState.terminalSubmitting = true;
    setTerminalInputState(true);
    try {
      const anchorTaskID = normalizeText(localState.terminalAnchorTaskID || localState.activeTaskID || taskID);
      const response = await fetch(`/api/control/tasks/${encodeURIComponent(taskID)}/terminal/input`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          input: content,
          reuse_task: true,
          anchor_task_id: anchorTaskID
        })
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        const message = typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`;
        window.alert(message);
        return;
      }
      const nextTaskID = normalizeText(payload?.task_id || "");
      const nextAnchorTaskID = normalizeText(payload?.anchor_task_id || anchorTaskID || nextTaskID);
      if (nextTaskID && nextTaskID !== "-") {
        await loadDetail(nextTaskID, {
          preserveLogs: false,
          anchorTaskID: nextAnchorTaskID,
          displayTaskID: nextAnchorTaskID
        });
      }
    } finally {
      localState.terminalSubmitting = false;
      setTerminalInputState(false);
    }
  };

  const loadDetail = async (taskID, options = {}) => {
    if (!taskID) {
      return;
    }
    localState.activeTaskID = taskID;
    const explicitAnchorTaskID = normalizeText(options.anchorTaskID || "");
    if (explicitAnchorTaskID && explicitAnchorTaskID !== "-") {
      localState.terminalAnchorTaskID = explicitAnchorTaskID;
    } else {
      localState.terminalAnchorTaskID = taskID;
    }
    const displayTaskID = normalizeText(options.displayTaskID || localState.terminalAnchorTaskID || taskID);
    const payload = await fetchJSON(`/api/control/tasks/${encodeURIComponent(taskID)}`);
    drawerBody.innerHTML = renderControlTaskDetail(payload, displayTaskID);
    syncActiveTaskCards();
    localState.logStreamNode = null;
    localState.logTouchStartY = null;
    setTerminalInputState(localState.terminalSubmitting);
    bindControlTaskTerminalComposer();
    if (!options.preserveLogs) {
      resetLogs();
      await loadLogBackfill(taskID, 0);
    } else {
      schedulePaintLogs();
    }
    startLogStream(taskID, localState.logCursor);
    openDrawer();
  };

  const invokeControlTaskAction = async (action) => {
    const taskID = localState.activeTaskID;
    if (!taskID) {
      return;
    }
    const response = await fetch(`/api/control/tasks/${encodeURIComponent(taskID)}/${action}`, { method: "POST" });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok && !payload?.view) {
      const message = typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`;
      throw new Error(message);
    }
    const detailPayload = payload?.view || payload;
    const displayTaskID = normalizeText(localState.terminalAnchorTaskID || taskID);
    drawerBody.innerHTML = renderControlTaskDetail(detailPayload, displayTaskID);
    localState.logStreamNode = null;
    localState.logTouchStartY = null;
    bindControlTaskTerminalComposer();
    resetLogs();
    await loadLogBackfill(taskID, 0);
    startLogStream(taskID, localState.logCursor);
    await loadList();
  };

  if (form) {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const formData = new FormData(form);
      localState.filters.sessionID = String(formData.get("session_id") || "").trim();
      localState.filters.status = String(formData.get("status") || "").trim();
      localState.filters.triggerType = String(formData.get("trigger_type") || "").trim();
      localState.filters.channelType = String(formData.get("channel_type") || "").trim();
      localState.filters.channelID = String(formData.get("channel_id") || "").trim();
      localState.filters.messageID = String(formData.get("message_id") || "").trim();
      localState.filters.sourceMessageID = String(formData.get("source_message_id") || "").trim();
      localState.filters.startAt = parseDateTimeFilter(formData.get("start_at"));
      localState.filters.endAt = parseDateTimeFilter(formData.get("end_at"));
      localState.page = 1;
      closeDrawer();
      await loadList();
    });

    const resetButton = form.querySelector("[data-control-task-filter-reset]");
    if (resetButton) {
      resetButton.addEventListener("click", async () => {
        form.reset();
        localState.filters = {
          sessionID: "",
          status: "",
          triggerType: "",
          channelType: "",
          channelID: "",
          messageID: "",
          sourceMessageID: "",
          startAt: "",
          endAt: ""
        };
        localState.page = 1;
        localState.advancedOpen = false;
        syncAdvancedToggle();
        closeDrawer();
        await loadList();
      });
    }
  }

  view.addEventListener("click", async (event) => {
    const target = event.target.closest("button");
    if (!target) {
      return;
    }
    if (target.hasAttribute("data-control-task-copy-id")) {
      const taskID = target.getAttribute("data-control-task-copy-id") || "";
      if (taskID) {
        try {
          if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
            await navigator.clipboard.writeText(taskID);
          } else {
            const fallback = document.createElement("textarea");
            fallback.value = taskID;
            fallback.setAttribute("readonly", "readonly");
            fallback.style.position = "absolute";
            fallback.style.left = "-9999px";
            document.body.appendChild(fallback);
            fallback.select();
            document.execCommand("copy");
            fallback.remove();
          }
          target.classList.add("copied");
          window.setTimeout(() => target.classList.remove("copied"), 900);
        } catch (error) {
          console.warn("copy task id failed", error);
        }
      }
      return;
    }
    if (target.hasAttribute("data-control-task-open")) {
      const openTaskID = normalizeText(target.getAttribute("data-control-task-open") || "");
      await loadDetail(openTaskID, { anchorTaskID: openTaskID, displayTaskID: openTaskID });
      return;
    }
    if (target.hasAttribute("data-control-task-quick-input")) {
      const prompt = normalizeText(target.getAttribute("data-control-task-quick-input"));
      if (prompt && prompt !== "-") {
        await submitTerminalInput(prompt);
      }
      return;
    }
    if (target.hasAttribute("data-control-task-page-next")) {
      if (target.disabled) {
        return;
      }
      localState.page += 1;
      closeDrawer();
      await loadList();
      return;
    }
    if (target.hasAttribute("data-control-task-close")) {
      closeDrawer();
      return;
    }
    if (target.hasAttribute("data-control-task-action")) {
      const action = normalizeText(target.getAttribute("data-control-task-action")).toLowerCase();
      if (!action || action === "-") {
        return;
      }
      try {
        await invokeControlTaskAction(action);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error || "unknown error");
        alert(t("load_failed", { error: message }));
      }
      return;
    }
    if (target.hasAttribute("data-control-task-log-reconnect")) {
      if (!localState.activeTaskID) {
        return;
      }
      startLogStream(localState.activeTaskID, localState.logCursor);
      return;
    }
    if (target.hasAttribute("data-control-task-log-replay")) {
      if (!localState.activeTaskID) {
        return;
      }
      stopLogStream();
      resetLogs();
      await loadLogBackfill(localState.activeTaskID, 0);
      startLogStream(localState.activeTaskID, localState.logCursor);
    }
  });

  paint(initialPayload || { items: [], pagination: { page: 1, total: 0, has_next: false } });
}

async function loadControlTasksView(container) {
  const payload = await fetchJSON(controlTaskListQuery({}, 1, 20));
  container.innerHTML = `<section class="control-task-view" data-control-task-view>
    <form class="task-filter-form page-filter-form control-task-filter-form" data-control-task-filter-form>
      <div class="task-filter-primary-row control-task-filter-primary">
        <label><span>${t("route.tasks.filter.session")}</span><input type="text" name="session_id" placeholder="session-123"></label>
        <label><span>${t("route.tasks.filter.status")}</span>
          <select name="status">
            <option value="">-</option>
            <option value="queued">${t("status.queued")}</option>
            <option value="running">${t("status.running")}</option>
            <option value="success">${t("status.success")}</option>
            <option value="failed">${t("status.failed")}</option>
            <option value="canceled">${t("status.canceled")}</option>
          </select>
        </label>
        <label><span>${t("route.tasks.filter.trigger_type")}</span>
          <select name="trigger_type">
            <option value="">-</option>
            <option value="user">${t("trigger.user")}</option>
            <option value="cron">${t("trigger.cron")}</option>
            <option value="system">${t("trigger.system")}</option>
          </select>
        </label>
        <label><span>${t("route.tasks.filter.channel_type")}</span>
          <select name="channel_type">
            <option value="">-</option>
            <option value="cli">${t("channel.cli")}</option>
            <option value="web">${t("channel.web")}</option>
            <option value="scheduler">${t("channel.scheduler")}</option>
          </select>
        </label>
      </div>
      <div class="task-filter-primary-actions control-task-filter-toolbar">
        <button class="task-filter-advanced-toggle" type="button" data-control-task-advanced-toggle aria-expanded="false">${renderAdvancedToggleLabel(false)}</button>
        <div class="task-filter-actions">
          <button class="task-filter-apply" type="submit">${t("route.tasks.filter.apply")}</button>
          <button class="task-filter-reset" type="button" data-control-task-filter-reset>${t("route.tasks.filter.reset")}</button>
        </div>
      </div>
      <div class="task-filter-advanced control-task-filter-advanced-panel" data-control-task-advanced hidden>
        <label><span>${t("route.tasks.filter.channel_id")}</span><input type="text" name="channel_id" placeholder="web-default"></label>
        <label><span>${t("route.tasks.filter.message_id")}</span><input type="text" name="message_id" placeholder="msg-123"></label>
        <label><span>${t("route.tasks.filter.source_message_id")}</span><input type="text" name="source_message_id" placeholder="msg-source-123"></label>
        <label><span>${t("route.tasks.filter.start_at")}</span><input type="datetime-local" name="start_at"></label>
        <label><span>${t("route.tasks.filter.end_at")}</span><input type="datetime-local" name="end_at"></label>
      </div>
    </form>
    <section class="control-task-layout" data-control-task-layout>
      <div class="control-task-main">
        <div class="task-summary-list" data-control-task-list></div>
        <div class="task-summary-pagination-wrap" data-control-task-pagination></div>
      </div>
      <section class="control-task-drawer" data-control-task-drawer hidden>
        <aside class="control-task-drawer-panel">
          <header class="control-task-drawer-head">
            <h4>${t("route.tasks.drawer.title")}</h4>
            <button class="task-summary-next" type="button" data-control-task-close>${t("route.tasks.drawer.close")}</button>
          </header>
          <div class="control-task-drawer-body" data-control-task-drawer-body>${t("route.tasks.drawer.empty")}</div>
        </aside>
      </section>
    </section>
  </section>`;
  bindControlTaskView(container, payload);
}

function normalizeTerminalStoredEntry(item, fallbackAt) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const role = String(item.role || "").trim().toLowerCase();
  const normalizedRole = role === "user" ? "input" : role === "terminal" ? "output" : role;
  if (!["input", "output", "system"].includes(normalizedRole)) {
    return null;
  }
  const text = typeof item.text === "string" ? item.text : "";
  if (!text) {
    return null;
  }
  const at = Number.isFinite(item.at) ? item.at : fallbackAt;
  const kind = typeof item.kind === "string" ? item.kind.trim().toLowerCase() : "";
  const stream = typeof item.stream === "string" ? item.stream.trim().toLowerCase() : kind;
  const cursor = Number.isFinite(item.cursor) ? Number(item.cursor) : -1;
  return {
    id: typeof item.id === "string" && item.id.trim() ? item.id.trim() : makeID(),
    role: normalizedRole,
    text,
    at,
    kind,
    stream,
    cursor
  };
}

function createTerminalTurnSnapshot(id) {
  const now = Date.now();
  return {
    id: String(id || "").trim() || `turn-${makeID()}`,
    prompt: "",
    status: "running",
    started_at: now,
    finished_at: 0,
    duration_ms: 0,
    final_output: "",
    steps: []
  };
}

function createTerminalStepSnapshot(id) {
  const now = Date.now();
  return {
    id: String(id || "").trim() || `step-${makeID()}`,
    type: "message",
    title: "",
    status: "running",
    started_at: now,
    finished_at: 0,
    duration_ms: 0,
    preview: "",
    has_detail: false
  };
}

function normalizeTerminalStoredStep(item, fallbackAt) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const step = createTerminalStepSnapshot(item.id);
  step.type = String(item.type || step.type || "message").trim().toLowerCase() || "message";
  step.title = String(item.title || "").trim() || step.title || t("route.terminal.process.empty");
  step.status = String(item.status || step.status || "completed").trim().toLowerCase() || "completed";
  step.started_at = parseTerminalTime(item.started_at, fallbackAt);
  step.finished_at = parseTerminalTime(item.finished_at, 0);
  step.duration_ms = Number.isFinite(Number(item.duration_ms)) ? Math.max(Number(item.duration_ms), 0) : 0;
  step.preview = String(item.preview || "").trim();
  step.has_detail = Boolean(item.has_detail);
  return step;
}

function normalizeTerminalStoredTurn(item, fallbackAt) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const turn = createTerminalTurnSnapshot(item.id);
  turn.prompt = String(item.prompt || "").trim();
  turn.status = String(item.status || turn.status || "completed").trim().toLowerCase() || "completed";
  turn.started_at = parseTerminalTime(item.started_at, fallbackAt);
  turn.finished_at = parseTerminalTime(item.finished_at, 0);
  turn.duration_ms = Number.isFinite(Number(item.duration_ms)) ? Math.max(Number(item.duration_ms), 0) : 0;
  turn.final_output = typeof item.final_output === "string" ? item.final_output : "";
  const stepsRaw = Array.isArray(item.steps) ? item.steps : [];
  turn.steps = stepsRaw.map((step) => normalizeTerminalStoredStep(step, turn.started_at || fallbackAt)).filter(Boolean);
  return turn;
}

function normalizeTerminalStoredStepDetail(item, fallbackAt) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const summary = normalizeTerminalStoredStep(item.step || item, fallbackAt);
  if (!summary) {
    return null;
  }
  const blocksRaw = Array.isArray(item.blocks) ? item.blocks : [];
  return {
    turn_id: String(item.turn_id || "").trim(),
    step: summary,
    searchable: Boolean(item.searchable),
    blocks: blocksRaw.map((block) => ({
      type: String(block?.type || "text").trim().toLowerCase() || "text",
      title: String(block?.title || "").trim(),
      content: typeof block?.content === "string" ? block.content : "",
      language: String(block?.language || "").trim(),
      file: String(block?.file || "").trim(),
      start_line: Number.isFinite(Number(block?.start_line)) ? Number(block.start_line) : 0,
      status: String(block?.status || "").trim().toLowerCase(),
      exit_code: parseTerminalExitCode(block?.exit_code)
    }))
  };
}

function createTerminalSessionSnapshot(id) {
  const now = Date.now();
  const safeID = String(id || "").trim() || `terminal-${makeID()}`;
  return {
    id: safeID,
    title: t("route.terminal.new"),
    created_at: now,
    last_output_at: 0,
    updated_at: now,
    terminal_session_id: safeID,
    status: "ready",
    shell: "",
    working_dir: "",
    exit_code: null,
    error_message: "",
    entry_cursor: 0,
    disconnected_notice: false,
    entries: [],
    turns: [],
    process_collapsed: {},
    output_collapsed: {},
    expanded_steps: {},
    step_details: {},
    step_errors: {},
    step_loading: {},
    step_search: {},
    pending_create: false,
    meta_expanded: false,
    chat_scroll_top: 0,
    chat_bottom_offset: 0,
    chat_has_unread_output: false,
    chat_last_seen_output_at: 0,
    chat_stick_to_bottom: true
  };
}

function parseTerminalTime(value, fallbackValue) {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim()) {
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return fallbackValue;
}

function parseTerminalExitCode(value) {
  if (value === null || value === undefined || value === "") {
    return null;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function captureTerminalSessionRuntimeSignature(session) {
  if (!session) {
    return "";
  }
  const turns = Array.isArray(session.turns) ? session.turns : [];
  return [
    normalizeText(session.id),
    normalizeText(session.terminal_session_id),
    normalizeText(session.title),
    normalizeText(session.status),
    String(Number(session.created_at || 0)),
    String(Number(session.updated_at || 0)),
    String(Number(session.last_output_at || 0)),
    String(Number(session.entry_cursor || 0)),
    normalizeText(session.shell),
    normalizeText(session.working_dir),
    normalizeText(session.error_message),
    String(parseTerminalExitCode(session.exit_code)),
    String(Boolean(session.disconnected_notice)),
    turns.map((turn) => {
      const steps = Array.isArray(turn?.steps) ? turn.steps : [];
      return [
        normalizeText(turn?.id),
        normalizeText(turn?.status),
        String(Number(turn?.started_at || 0)),
        String(Number(turn?.finished_at || 0)),
        String(Number(turn?.duration_ms || 0)),
        String(turn?.final_output || ""),
        steps.map((step) => [
          normalizeText(step?.id),
          normalizeText(step?.status),
          String(Number(step?.started_at || 0)),
          String(Number(step?.finished_at || 0)),
          String(Number(step?.duration_ms || 0)),
          String(step?.preview || "")
        ].join("~")).join("^")
      ].join("::");
    }).join("||")
  ].join("|");
}

function captureTerminalSessionListSignature(sessions) {
  if (!Array.isArray(sessions) || !sessions.length) {
    return "";
  }
  return sessions.map((session) => captureTerminalSessionRuntimeSignature(session)).join("||");
}

function applyTerminalSessionSnapshot(session, snapshot) {
  if (!session || !snapshot || typeof snapshot !== "object") {
    return session;
  }
  const now = Date.now();
  const previousLastOutputAt = Number(session.last_output_at || 0);
  const sessionID = String(snapshot.id || session.id || snapshot.terminal_session_id || "").trim();
  const runtimeSessionID = String(snapshot.terminal_session_id || session.terminal_session_id || sessionID).trim();
  if (sessionID) {
    session.id = sessionID;
  }
  session.terminal_session_id = runtimeSessionID || sessionID;
  session.title = String(snapshot.title || "").trim() || session.title || t("route.terminal.new");
  session.created_at = parseTerminalTime(snapshot.created_at, session.created_at || now);
  session.last_output_at = parseTerminalTime(snapshot.last_output_at, session.last_output_at || 0);
  session.updated_at = parseTerminalTime(snapshot.updated_at, session.updated_at || session.created_at || now);
  session.status = normalizeTerminalSessionStatus(snapshot.status || session.status || "interrupted");
  session.shell = String(snapshot.shell || "").trim();
  session.working_dir = String(snapshot.working_dir || "").trim();
  session.error_message = String(snapshot.error_message || "").trim();
  session.exit_code = parseTerminalExitCode(snapshot.exit_code);
  const nextLastOutputAt = Number(session.last_output_at || 0);
  if (session.chat_stick_to_bottom === false) {
    const knownLastSeenOutputAt = Number(session.chat_last_seen_output_at || 0);
    if (nextLastOutputAt > Math.max(previousLastOutputAt, knownLastSeenOutputAt)) {
      session.chat_has_unread_output = true;
    }
  } else {
    session.chat_last_seen_output_at = nextLastOutputAt;
    session.chat_has_unread_output = false;
  }
  if (Array.isArray(snapshot.turns)) {
    mergeTerminalTurnSummaries(session, snapshot.turns);
  }
  return session;
}

function hasTerminalTurnFinalOutput(turn) {
  return typeof turn?.final_output === "string" && turn.final_output.trim().length > 0;
}

function shouldCollapseTerminalProcess(turn) {
  if (hasTerminalTurnFinalOutput(turn)) {
    return true;
  }
  return ["completed", "failed", "interrupted"].includes(String(turn?.status || "").trim().toLowerCase());
}

function resolveTerminalProcessCollapsed(session, turn) {
  if (!session || !turn) {
    return false;
  }
  const turnID = String(turn.id || "").trim();
  if (!turnID) {
    return false;
  }
  if (session.process_collapsed && Object.prototype.hasOwnProperty.call(session.process_collapsed, turnID)) {
    return Boolean(session.process_collapsed[turnID]);
  }
  return shouldCollapseTerminalProcess(turn);
}

function pruneTerminalStateMap(map, allowedKeys) {
  const next = {};
  if (!map || typeof map !== "object") {
    return next;
  }
  allowedKeys.forEach((key) => {
    if (Object.prototype.hasOwnProperty.call(map, key)) {
      next[key] = map[key];
    }
  });
  return next;
}

function mergeTerminalTurnSummaries(session, turnsRaw) {
  if (!session) {
    return;
  }
  const items = Array.isArray(turnsRaw) ? turnsRaw : [];
  const previousTurns = Array.isArray(session.turns) ? session.turns : [];
  const previousTurnsByID = new Map(previousTurns.map((turn) => [String(turn?.id || "").trim(), turn]));
  const previousProcess = session.process_collapsed && typeof session.process_collapsed === "object" ? session.process_collapsed : {};
  const previousOutputCollapsed = session.output_collapsed && typeof session.output_collapsed === "object" ? session.output_collapsed : {};
  const previousExpanded = session.expanded_steps && typeof session.expanded_steps === "object" ? session.expanded_steps : {};
  const previousDetails = session.step_details && typeof session.step_details === "object" ? session.step_details : {};
  const previousErrors = session.step_errors && typeof session.step_errors === "object" ? session.step_errors : {};
  const previousLoading = session.step_loading && typeof session.step_loading === "object" ? session.step_loading : {};
  const previousSearch = session.step_search && typeof session.step_search === "object" ? session.step_search : {};
  const turns = [];
  const allowedTurnIDs = [];
  const allowedStepIDs = [];

  items.forEach((item) => {
    const turn = normalizeTerminalStoredTurn(item, session.created_at || Date.now());
    if (!turn) {
      return;
    }
    turns.push(turn);
    allowedTurnIDs.push(turn.id);
    turn.steps.forEach((step) => {
      allowedStepIDs.push(step.id);
    });
  });

  session.turns = turns;
  session.process_collapsed = pruneTerminalStateMap(previousProcess, allowedTurnIDs);
  session.output_collapsed = pruneTerminalStateMap(previousOutputCollapsed, allowedTurnIDs);
  turns.forEach((turn) => {
    const previousTurn = previousTurnsByID.get(turn.id) || null;
    const gainedFinalOutput = hasTerminalTurnFinalOutput(turn) && !hasTerminalTurnFinalOutput(previousTurn);
    if (gainedFinalOutput) {
      session.process_collapsed[turn.id] = true;
      return;
    }
    if (!Object.prototype.hasOwnProperty.call(session.process_collapsed, turn.id)) {
      session.process_collapsed[turn.id] = shouldCollapseTerminalProcess(turn);
    }
  });
  session.expanded_steps = pruneTerminalStateMap(previousExpanded, allowedStepIDs);
  session.step_errors = pruneTerminalStateMap(previousErrors, allowedStepIDs);
  session.step_loading = pruneTerminalStateMap(previousLoading, allowedStepIDs);
  session.step_search = pruneTerminalStateMap(previousSearch, allowedStepIDs);
  session.step_details = pruneTerminalStateMap(previousDetails, allowedStepIDs);
}

function normalizeTerminalStoredSession(item) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const baseID = String(item.id || item.terminal_session_id || "").trim();
  if (!baseID) {
    return null;
  }
  const session = createTerminalSessionSnapshot(baseID);
  applyTerminalSessionSnapshot(session, item);
  session.entry_cursor = Number.isFinite(Number(item.entry_cursor)) && Number(item.entry_cursor) >= 0 ? Number(item.entry_cursor) : 0;
  session.disconnected_notice = Boolean(item.disconnected_notice);
  session.process_collapsed = item.process_collapsed && typeof item.process_collapsed === "object" ? { ...item.process_collapsed } : {};
  session.output_collapsed = item.output_collapsed && typeof item.output_collapsed === "object" ? { ...item.output_collapsed } : {};
  session.expanded_steps = item.expanded_steps && typeof item.expanded_steps === "object" ? { ...item.expanded_steps } : {};
  session.step_errors = item.step_errors && typeof item.step_errors === "object" ? { ...item.step_errors } : {};
  session.step_loading = item.step_loading && typeof item.step_loading === "object" ? { ...item.step_loading } : {};
  session.step_search = item.step_search && typeof item.step_search === "object" ? { ...item.step_search } : {};
  session.pending_create = false;
  session.meta_expanded = Boolean(item.meta_expanded);
  session.chat_scroll_top = Number.isFinite(Number(item.chat_scroll_top)) ? Math.max(Number(item.chat_scroll_top), 0) : 0;
  session.chat_bottom_offset = Number.isFinite(Number(item.chat_bottom_offset)) ? Math.max(Number(item.chat_bottom_offset), 0) : 0;
  session.chat_has_unread_output = Boolean(item.chat_has_unread_output);
  session.chat_last_seen_output_at = Number.isFinite(Number(item.chat_last_seen_output_at)) ? Math.max(Number(item.chat_last_seen_output_at), 0) : 0;
  session.chat_stick_to_bottom = typeof item.chat_stick_to_bottom === "boolean" ? item.chat_stick_to_bottom : true;
  const stepDetails = {};
  if (item.step_details && typeof item.step_details === "object") {
    Object.entries(item.step_details).forEach(([stepID, detail]) => {
      const parsed = normalizeTerminalStoredStepDetail(detail, session.created_at || Date.now());
      if (parsed) {
        stepDetails[String(stepID || "").trim()] = parsed;
      }
    });
  }
  session.step_details = stepDetails;
  const entriesRaw = Array.isArray(item.entries) ? item.entries : [];
  const entries = [];
  for (const row of entriesRaw) {
    const parsed = normalizeTerminalStoredEntry(row, session.created_at || Date.now());
    if (parsed) {
      entries.push(parsed);
    }
  }
  session.entries = entries;
  const turnsRaw = Array.isArray(item.turns) ? item.turns : [];
  mergeTerminalTurnSummaries(session, turnsRaw);
  return session;
}

function loadTerminalSessionsFromStorage() {
  const storage = getSessionStorage();
  if (!storage) {
    return [];
  }
  const raw = storage.getItem(TERMINAL_STORAGE_KEY);
  if (!raw) {
    return [];
  }
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return [];
  }
  if (!Array.isArray(parsed)) {
    return [];
  }
  const sessions = [];
  for (const item of parsed) {
    const normalized = normalizeTerminalStoredSession(item);
    if (normalized) {
      sessions.push(normalized);
    }
  }
  sessions.sort(compareTerminalSessions);
  return sessions;
}

function persistTerminalSessionsToStorage(sessions) {
  const storage = getSessionStorage();
  if (!storage) {
    return;
  }
  try {
    storage.setItem(TERMINAL_STORAGE_KEY, JSON.stringify((Array.isArray(sessions) ? sessions : []).filter((item) => !isPendingTerminalSession(item))));
  } catch {
  }
}

function getTerminalClientID() {
  const fallback = `terminal-client-${makeID()}`;
  const durableStorage = getSessionStorage();
  const transientStorage = getBrowserSessionStorage();
  const readStorageValue = (storage) => {
    if (!storage) {
      return "";
    }
    try {
      return String(storage.getItem(TERMINAL_CLIENT_STORAGE_KEY) || "").trim();
    } catch {
      return "";
    }
  };
  const writeStorageValue = (storage, value) => {
    if (!storage || !value) {
      return;
    }
    try {
      storage.setItem(TERMINAL_CLIENT_STORAGE_KEY, value);
    } catch {
    }
  };

  const durableValue = readStorageValue(durableStorage);
  if (durableValue) {
    writeStorageValue(transientStorage, durableValue);
    return durableValue;
  }

  const transientValue = readStorageValue(transientStorage);
  if (transientValue) {
    writeStorageValue(durableStorage, transientValue);
    return transientValue;
  }

  writeStorageValue(durableStorage, fallback);
  writeStorageValue(transientStorage, fallback);
  return fallback;
}

function renderTerminalStatus(status) {
  const normalized = normalizeTerminalSessionStatus(status);
  if (!normalized) {
    return "-";
  }
  return formatTaskStatus(normalized);
}

function normalizeTerminalSessionStatus(status) {
  const normalized = String(status || "").trim().toLowerCase();
  if (!normalized || normalized === "running") {
    return "ready";
  }
  if (normalized === "starting") {
    return "busy";
  }
  if (["ready", "busy", "exited", "failed", "interrupted"].includes(normalized)) {
    return normalized;
  }
  return normalized;
}

function isTerminalSessionLiveStatus(status) {
  const normalized = normalizeTerminalSessionStatus(status);
  return ["ready", "busy"].includes(normalized);
}

function canTerminalSessionAcceptInput(status) {
  return normalizeTerminalSessionStatus(status) !== "busy";
}

function isTerminalSessionBusyStatus(status) {
  return normalizeTerminalSessionStatus(status) === "busy";
}

function isTerminalTurnLiveStatus(status) {
  const normalized = String(status || "").trim().toLowerCase();
  return ["running", "starting"].includes(normalized);
}

function hasRecoverableTerminalThread(session) {
  if (!session) {
    return false;
  }
  const sessionID = normalizeText(session.id);
  const threadID = normalizeText(session.terminal_session_id);
  return Boolean(threadID && sessionID && threadID !== sessionID);
}

function getTerminalSessionLastOutputAt(session) {
  const snapshotLastOutputAt = Number(session?.last_output_at || 0);
  let lastOutputAt = Number.isFinite(snapshotLastOutputAt) && snapshotLastOutputAt > 0 ? snapshotLastOutputAt : 0;
  const entries = Array.isArray(session?.entries) ? session.entries : [];
  for (let index = entries.length - 1; index >= 0; index -= 1) {
    const entry = entries[index];
    if (String(entry?.role || "").trim().toLowerCase() !== "output") {
      continue;
    }
    const at = Number(entry?.at || 0);
    if (Number.isFinite(at) && at > 0) {
      lastOutputAt = Math.max(lastOutputAt, at);
      break;
    }
  }
  return lastOutputAt;
}

function getTerminalSessionSortAt(session) {
  const lastOutputAt = getTerminalSessionLastOutputAt(session);
  if (lastOutputAt > 0) {
    return lastOutputAt;
  }
  const createdAt = Number(session?.created_at || 0);
  if (Number.isFinite(createdAt) && createdAt > 0) {
    return createdAt;
  }
  const updatedAt = Number(session?.updated_at || 0);
  return Number.isFinite(updatedAt) && updatedAt > 0 ? updatedAt : 0;
}

function isPendingTerminalSession(session) {
  return Boolean(session?.pending_create);
}

function compareTerminalSessions(left, right) {
  if (isPendingTerminalSession(left) !== isPendingTerminalSession(right)) {
    return isPendingTerminalSession(left) ? -1 : 1;
  }
  const sortDiff = getTerminalSessionSortAt(right) - getTerminalSessionSortAt(left);
  if (sortDiff !== 0) {
    return sortDiff;
  }
  const createdDiff = Number(right?.created_at || 0) - Number(left?.created_at || 0);
  if (createdDiff !== 0) {
    return createdDiff;
  }
  return String(right?.id || "").localeCompare(String(left?.id || ""));
}

function normalizeTerminalLine(text, max = 240) {
  const safe = String(text || "").trim();
  if (!safe) {
    return "";
  }
  if (safe.length <= max) {
    return safe;
  }
  return `${safe.slice(0, max - 1)}...`;
}

function classifyTerminalLogKind(entry) {
  const role = String(entry?.role || "").trim().toLowerCase();
  const stream = String(entry?.stream || entry?.kind || "").trim().toLowerCase();
  if (role === "input" || stream === "input") {
    return "command";
  }
  if (role === "system" || stream === "system") {
    return "tag";
  }
  if (stream === "stderr") {
    return "action";
  }
  return "output";
}

function renderTerminalSessionCards(sessions, activeSessionID, options = {}) {
  if (!sessions.length) {
    return `<p class="route-empty-panel terminal-session-empty">${escapeHTML(t("route.terminal.empty"))}</p>`;
  }
  return sessions.map((session) => {
    const title = normalizeText(session.title);
    const sessionID = normalizeText(session.id);
    const active = session.id === activeSessionID;
    const statusClassName = taskStatusClassName(session.status);
    const listTimestamp = getTerminalSessionSortAt(session);
    const listTimeLabel = listTimestamp > 0 ? formatDateTime(new Date(listTimestamp).toISOString()) : "-";
    const lastOutputMeta = getTerminalSessionLastOutputAt(session) > 0
      ? t("route.terminal.last_output", { time: listTimeLabel })
      : t("route.terminal.no_output");
    const deleting = Boolean(options?.deleting) && sessionID === normalizeText(options?.deletingSessionID);
    const deleteLabel = deleting ? t("route.terminal.deleting") : t("route.terminal.delete");
    return `<div class="route-card terminal-session-card ${active ? "active" : ""}" data-terminal-session-card="${escapeHTML(sessionID)}" data-terminal-session-status="${escapeHTML(normalizeTerminalSessionStatus(session.status) || "unknown")}">
      <button class="route-card-button terminal-session-select ${active ? "active" : ""}" type="button" data-terminal-session-select="${escapeHTML(sessionID)}" ${active ? 'aria-current="true"' : ""}>
        <span class="terminal-session-head">
          <span class="route-card-title-copy">
            <span class="terminal-session-title">${escapeHTML(title)}</span>
            <span class="terminal-session-meta">${escapeHTML(lastOutputMeta)}</span>
          </span>
          <span class="task-summary-status ${statusClassName}">${escapeHTML(renderTerminalStatus(session.status))}</span>
        </span>
      </button>
      <button class="terminal-session-list-delete" type="button" data-terminal-delete-session="${escapeHTML(sessionID)}" aria-label="${escapeHTML(deleteLabel)}" ${Boolean(options?.deleting) ? "disabled" : ""}>${escapeHTML(deleteLabel)}</button>
    </div>`;
  }).join("");
}

function shouldCollapseTerminalOutput(text) {
  const content = String(text || "");
  if (!content.trim()) {
    return false;
  }
  const lines = content.replace(/\r\n/g, "\n").replace(/\r/g, "\n").split("\n");
  return content.length > 640 || lines.length > 12;
}

function resolveTerminalOutputCollapsed(session, turn) {
  if (!session || !turn) {
    return false;
  }
  const turnID = normalizeText(turn.id);
  if (!turnID || turnID === "-") {
    return false;
  }
  if (session.output_collapsed && Object.prototype.hasOwnProperty.call(session.output_collapsed, turnID)) {
    return Boolean(session.output_collapsed[turnID]);
  }
  return isMobileViewport() && shouldCollapseTerminalOutput(turn.final_output);
}

function renderTerminalWorkspaceMetaPanel(session) {
  return `<section class="terminal-meta-panel" data-terminal-meta-panel>
    ${routeFieldRow("route.terminal.session", session?.terminal_session_id || "-", { copyable: true, mono: true, clampLines: 2 })}
    ${routeFieldRow("route.terminal.shell", session?.shell || "-", { copyable: true, mono: true, clampLines: 2 })}
    ${routeFieldRow("route.terminal.path", session?.working_dir || "-", { copyable: true, multiline: true, mono: true, clampLines: 3 })}
    ${routeFieldRow("route.terminal.status", renderTerminalStatus(session?.status || "-"))}
  </section>`;
}

function renderTerminalLogRows(entries) {
  const safeEntries = Array.isArray(entries) ? entries : [];
  if (!safeEntries.length) {
    return `<div class="terminal-log-empty">${escapeHTML(t("route.terminal.logs.empty"))}</div>`;
  }
  return safeEntries.map((entry) => {
    const kind = classifyTerminalLogKind(entry);
    const rowClass = `terminal-log-row kind-${escapeHTML(kind)}`;
    const text = String(entry?.text || "");
    const prefix = kind === "command" ? ">" : "";
    if (!text) {
      return "";
    }
    return `<div class="${rowClass}">
      <div class="terminal-log-main">
        ${prefix ? `<span class="terminal-log-prefix">${escapeHTML(prefix)}</span>` : ""}
        <span class="terminal-log-text">${escapeHTML(text)}</span>
      </div>
      <span class="terminal-log-time">${escapeHTML(timeLabel(entry?.at))}</span>
    </div>`;
  }).join("");
}

function formatTerminalDuration(durationMS, startedAt = 0, finishedAt = 0) {
  let total = Number(durationMS || 0);
  if ((!Number.isFinite(total) || total <= 0) && Number.isFinite(Number(startedAt)) && Number(startedAt) > 0) {
    const end = Number.isFinite(Number(finishedAt)) && Number(finishedAt) > 0 ? Number(finishedAt) : Date.now();
    total = Math.max(0, end - Number(startedAt));
  }
  if (!Number.isFinite(total) || total <= 0) {
    return "<1s";
  }
  const seconds = Math.max(1, Math.round(total / 1000));
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainSeconds = seconds % 60;
  if (minutes < 60) {
    return remainSeconds > 0 ? `${minutes}m ${remainSeconds}s` : `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const remainMinutes = minutes % 60;
  return remainMinutes > 0 ? `${hours}h ${remainMinutes}m` : `${hours}h`;
}

function escapeHTMLWithHighlight(text, query = "") {
  const source = String(text || "");
  const needle = String(query || "").trim();
  if (!needle) {
    return escapeHTML(source);
  }
  const lowerSource = source.toLowerCase();
  const lowerNeedle = needle.toLowerCase();
  let cursor = 0;
  let html = "";
  while (cursor < source.length) {
    const index = lowerSource.indexOf(lowerNeedle, cursor);
    if (index < 0) {
      html += escapeHTML(source.slice(cursor));
      break;
    }
    html += escapeHTML(source.slice(cursor, index));
    html += `<mark>${escapeHTML(source.slice(index, index + needle.length))}</mark>`;
    cursor = index + needle.length;
  }
  return html;
}

function renderTerminalNumberedBlock(content, searchQuery = "") {
  const text = String(content || "");
  if (!text) {
    return `<div class="terminal-rich-empty">-</div>`;
  }
  const lines = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n").split("\n");
  return `<div class="terminal-numbered-block">${lines.map((line, index) => `<div class="terminal-numbered-line"><span class="terminal-numbered-index">${String(index + 1)}</span><span class="terminal-numbered-text">${escapeHTMLWithHighlight(line, searchQuery)}</span></div>`).join("")}</div>`;
}

function renderTerminalDiffBlock(content, searchQuery = "") {
  const text = String(content || "");
  const lines = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n").split("\n");
  return `<div class="terminal-diff-block">${lines.map((line, index) => {
    const marker = line.startsWith("+") ? "add" : line.startsWith("-") ? "remove" : "context";
    return `<div class="terminal-diff-line kind-${marker}"><span class="terminal-numbered-index">${String(index + 1)}</span><span class="terminal-numbered-text">${escapeHTMLWithHighlight(line, searchQuery)}</span></div>`;
  }).join("")}</div>`;
}

function renderTerminalRichBlock(stepID, block, searchQuery = "") {
  const type = String(block?.type || "text").trim().toLowerCase();
  const title = String(block?.title || "").trim();
  const status = String(block?.status || "").trim().toLowerCase();
  const exitCode = parseTerminalExitCode(block?.exit_code);
  const fileLabel = String(block?.file || "").trim();
  const lineLabel = Number.isFinite(Number(block?.start_line)) && Number(block.start_line) > 0
    ? `:${String(Number(block.start_line))}`
    : "";
  const blockHeader = title || status || exitCode !== null
    ? `<div class="terminal-rich-head">
        <div class="terminal-rich-copy">
          ${title ? `<strong>${escapeHTML(title)}</strong>` : ""}
          ${fileLabel ? `<span>${escapeHTML(`${fileLabel}${lineLabel}`)}</span>` : ""}
        </div>
        <div class="terminal-rich-meta">
          ${status ? `<span class="terminal-step-status status-${escapeHTML(status)}">${escapeHTML(renderTerminalStatus(status))}</span>` : ""}
          ${exitCode !== null ? `<span class="terminal-rich-exit">exit ${escapeHTML(String(exitCode))}</span>` : ""}
        </div>
      </div>`
    : "";
  let body = "";
  if (type === "terminal") {
    body = `<pre class="terminal-rich-pre">${escapeHTML(String(block?.content || ""))}</pre>`;
  } else if (type === "diff") {
    body = renderTerminalDiffBlock(block?.content || "", searchQuery);
  } else {
    body = renderTerminalNumberedBlock(block?.content || "", searchQuery);
  }
  return `<section class="route-surface-dark terminal-rich-block type-${escapeHTML(type)}">${blockHeader}${body}</section>`;
}

function renderTerminalStepDetail(session, turn, step) {
  const stepID = normalizeText(step?.id);
  const detail = session?.step_details?.[stepID] || null;
  const loading = Boolean(session?.step_loading?.[stepID]);
  const error = String(session?.step_errors?.[stepID] || "").trim();
  const searchQuery = String(session?.step_search?.[stepID] || "");
  if (loading) {
    return `<div class="terminal-step-detail-state">${escapeHTML(t("route.terminal.step.loading"))}</div>`;
  }
  if (error) {
    return `<div class="terminal-step-detail-state is-error">${escapeHTML(t("route.terminal.step.error", { error }))}</div>`;
  }
  if (!detail) {
    return "";
  }
  const blocks = Array.isArray(detail.blocks) ? detail.blocks : [];
  return `<div class="terminal-step-detail">
    ${detail.searchable ? `<div class="terminal-step-search"><input type="search" data-terminal-step-search="${escapeHTML(stepID)}" value="${escapeHTML(searchQuery)}" placeholder="${escapeHTML(t("route.terminal.step.search"))}"></div>` : ""}
    ${blocks.length ? blocks.map((block) => renderTerminalRichBlock(stepID, block, searchQuery)).join("") : `<div class="terminal-step-detail-state">${escapeHTML(t("route.terminal.process.empty"))}</div>`}
  </div>`;
}

function renderTerminalStepItems(session, turn) {
  const steps = Array.isArray(turn?.steps) ? turn.steps : [];
  if (!steps.length) {
    const waiting = String(turn?.status || "").trim().toLowerCase() === "running";
    return `<div class="terminal-process-empty">${escapeHTML(waiting ? t("route.terminal.process.loading") : t("route.terminal.process.empty"))}</div>`;
  }
  return steps.map((step) => {
    const stepID = normalizeText(step.id);
    const expanded = Boolean(session?.expanded_steps?.[stepID]);
    const duration = formatTerminalDuration(step.duration_ms, step.started_at, step.finished_at);
    const preview = String(step.preview || "").trim();
    const summary = preview || String(step.title || "-").trim() || "-";
    return `<article class="terminal-step-item" data-terminal-step-item="${escapeHTML(stepID)}">
      <button class="terminal-step-toggle" type="button" data-terminal-step-toggle="${escapeHTML(stepID)}" data-terminal-turn-id="${escapeHTML(normalizeText(turn.id))}" aria-expanded="${expanded ? "true" : "false"}">
        <span class="terminal-step-toggle-icon">${expanded ? "v" : ">"}</span>
        <span class="terminal-step-summary">
          <span class="terminal-step-title">${escapeHTML(summary)}</span>
        </span>
        <span class="terminal-step-meta">
          <span class="terminal-step-duration">${escapeHTML(duration)}</span>
          <span class="terminal-step-status status-${escapeHTML(String(step.status || "completed"))}">${escapeHTML(renderTerminalStatus(step.status))}</span>
        </span>
      </button>
      <div class="terminal-step-body" ${expanded ? "" : "hidden"}>
        ${renderTerminalStepDetail(session, turn, step)}
      </div>
    </article>`;
  }).join("");
}

function renderTerminalTurnProcess(session, turn) {
  const turnID = normalizeText(turn?.id);
  const collapsed = resolveTerminalProcessCollapsed(session, turn);
  const duration = formatTerminalDuration(turn?.duration_ms, turn?.started_at, turn?.finished_at);
  const stepCount = Array.isArray(turn?.steps) ? turn.steps.length : 0;
  return `<section class="terminal-process-shell ${collapsed ? "is-collapsed" : ""}" data-terminal-process-shell="${escapeHTML(turnID)}">
    <button class="terminal-process-toggle" type="button" data-terminal-process-toggle="${escapeHTML(turnID)}" aria-expanded="${collapsed ? "false" : "true"}">
      <span class="terminal-step-toggle-icon">${collapsed ? ">" : "v"}</span>
      <span class="terminal-process-copy">
        <span class="terminal-process-title">${escapeHTML(t("route.terminal.process.label"))}</span>
        <span class="terminal-process-summary">${escapeHTML(t("route.terminal.process.steps", { count: String(stepCount) }))}</span>
      </span>
      <span class="terminal-process-meta">${escapeHTML(duration)}</span>
    </button>
    <div class="terminal-process-body" ${collapsed ? "hidden" : ""}>
      ${renderTerminalStepItems(session, turn)}
    </div>
  </section>`;
}

function renderTerminalFinalOutput(session, turn) {
  const content = typeof turn?.final_output === "string" ? turn.final_output : "";
  if (!content.trim()) {
    return "";
  }
  const turnID = normalizeText(turn?.id);
  return `<div class="msg assistant terminal-final-output terminal-turn-output" data-terminal-final-output="${escapeHTML(turnID)}">
    <div class="msg-bubble">
      <div class="terminal-final-text">
        <div class="assistant-message-toolbar terminal-final-toolbar">
          ${renderAssistantCopyButton(content, "terminal-final-copy")}
        </div>
        <div class="terminal-final-rendered">${renderMarkdownToHTML(content)}</div>
      </div>
    </div>
  </div>`;
}

function renderTerminalTurns(session) {
  const turns = Array.isArray(session?.turns) ? session.turns : [];
  if (!turns.length) {
    return renderTerminalLogRows(session?.entries || []);
  }
  return turns.map((turn) => {
    const promptText = String(turn?.prompt || "").trim();
    const hasProcess = Array.isArray(turn?.steps) && turn.steps.length > 0;
    const bodyHTML = [
      hasProcess || String(turn?.status || "").trim().toLowerCase() === "running" ? renderTerminalTurnProcess(session, turn) : "",
      renderTerminalFinalOutput(session, turn)
    ].filter(Boolean).join("");
    return `<article class="terminal-turn-card" data-terminal-turn="${escapeHTML(normalizeText(turn.id))}">
      ${promptText ? `<div class="terminal-log-row kind-command terminal-turn-prompt"><div class="terminal-log-main"><span class="terminal-log-text">${escapeHTML(promptText)}</span></div><span class="terminal-log-time">${escapeHTML(timeLabel(turn?.started_at))}</span></div>` : ""}
      ${bodyHTML ? `<div class="terminal-turn-surface">${bodyHTML}</div>` : ""}
    </article>`;
  }).join("");
}

function renderTerminalWorkspace(session, sending, closing = false, deleting = false, options = {}) {
  if (!session) {
    return `<section class="route-empty-panel terminal-workspace-empty">
      <p>${escapeHTML(t("route.terminal.pick"))}</p>
    </section>`;
  }
  const logRef = normalizeText(session.terminal_session_id);
  const isLive = isTerminalSessionLiveStatus(session.status);
  const canInput = canTerminalSessionAcceptInput(session.status);
  const normalizedStatus = normalizeTerminalSessionStatus(session.status);
  const placeholder = canInput
    ? t("route.terminal.input")
    : (isTerminalSessionBusyStatus(session.status) ? t("route.terminal.busy") : t("route.terminal.closed"));
  const showRuntimeNote = !sending && !Boolean(session?.pending_create);
  const note = showRuntimeNote
    ? (normalizedStatus === "interrupted"
      ? t("route.terminal.interrupted")
      : ((normalizedStatus === "exited" || normalizedStatus === "failed") ? t("route.terminal.closed") : ""))
    : "";
  const detail = showRuntimeNote
    ? (session.error_message || (parseTerminalExitCode(session.exit_code) !== null ? `exit code ${String(parseTerminalExitCode(session.exit_code))}` : ""))
    : "";
  const closeDisabled = sending || closing || deleting || !isTerminalSessionLiveStatus(session.status);
  const deleteDisabled = sending || closing || deleting;
  const showJumpBottom = Boolean(session?.chat_has_unread_output) || Number(session?.chat_bottom_offset || 0) > TERMINAL_JUMP_BOTTOM_SHOW_THRESHOLD;
  const showJumpTop = Number(session?.chat_scroll_top || 0) > TERMINAL_JUMP_TOP_SHOW_THRESHOLD;
  const previousTurnID = normalizeText(session?.chat_previous_turn_id);
  const nextTurnID = normalizeText(session?.chat_next_turn_id);
  const metaExpanded = Boolean(session?.meta_expanded);
  const sessionCount = Number.isFinite(Number(options?.sessionCount)) ? Math.max(Number(options.sessionCount), 0) : 0;
  const sessionToggleLabel = options?.sessionSheetOpen ? t("route.terminal.hide_sessions") : t("route.terminal.sessions");
  const headerSubcopy = getTerminalSessionLastOutputAt(session) > 0
    ? t("route.terminal.last_output", { time: formatDateTime(new Date(getTerminalSessionLastOutputAt(session)).toISOString()) })
    : t("route.terminal.no_output");
  return `<section class="terminal-workspace-body" data-terminal-workspace data-terminal-session-id="${escapeHTML(session.id)}" data-terminal-workspace-status="${escapeHTML(normalizeTerminalSessionStatus(session.status) || "unknown")}" data-terminal-workspace-live="${isLive ? "true" : "false"}">
    <header class="terminal-workspace-head">
      <div class="terminal-mobile-actions">
        <button class="terminal-inline-button" type="button" data-terminal-session-pane-toggle aria-expanded="${options?.sessionSheetOpen ? "true" : "false"}">${escapeHTML(sessionToggleLabel)}</button>
        <button class="terminal-inline-button is-primary" type="button" data-terminal-create>${escapeHTML(t("route.terminal.new_short"))}</button>
      </div>
      <div class="terminal-workspace-row">
        <div class="terminal-workspace-copy">
          <p class="terminal-workspace-eyebrow">${escapeHTML(t("route.terminal.logs.heading", { session: shorten(logRef === "-" ? "n/a" : logRef, 24) }))}</p>
          <div class="terminal-workspace-title-row">
            <h4>${escapeHTML(normalizeText(session.title))}</h4>
            <span class="terminal-runtime-state" data-terminal-runtime-state="${escapeHTML(normalizedStatus || "unknown")}">
              <span class="terminal-runtime-state-dot" aria-hidden="true"></span>
              <span class="terminal-runtime-state-text">${escapeHTML(renderTerminalStatus(session.status))}</span>
            </span>
          </div>
          <p class="terminal-workspace-subcopy">${escapeHTML(headerSubcopy)}</p>
        </div>
        <div class="terminal-workspace-actions">
          <button class="terminal-inline-button is-quiet" type="button" data-terminal-meta-toggle aria-expanded="${metaExpanded ? "true" : "false"}">${escapeHTML(metaExpanded ? t("route.terminal.details_hide") : t("route.terminal.details_show"))}</button>
          <button class="terminal-session-close" type="button" data-terminal-close ${closeDisabled ? "disabled" : ""}>${escapeHTML(closing ? t("route.terminal.closing") : t("route.terminal.close"))}</button>
          <button class="terminal-session-delete" type="button" data-terminal-delete ${deleteDisabled ? "disabled" : ""}>${escapeHTML(deleting ? t("route.terminal.deleting") : t("route.terminal.delete"))}</button>
        </div>
      </div>
      ${metaExpanded ? renderTerminalWorkspaceMetaPanel(session) : ""}
    </header>
    <section class="route-surface-dark terminal-console-panel" data-terminal-console-panel>
      <div class="terminal-chat-screen" data-terminal-chat-screen data-terminal-chat-status="${escapeHTML(normalizeTerminalSessionStatus(session.status) || "unknown")}">
        <div class="terminal-log-tree">
          ${renderTerminalTurns(session)}
        </div>
      </div>
      <div class="terminal-jump-cluster" aria-label="${escapeHTML(t("route.terminal.navigation"))}">
        <button class="terminal-jump-control terminal-jump-top ${showJumpTop ? "is-visible" : ""}" type="button" data-terminal-jump-top aria-label="${escapeHTML(t("route.terminal.jump_top"))}" title="${escapeHTML(t("route.terminal.jump_top"))}">
          <span class="terminal-jump-control-icon" aria-hidden="true">&uarr;&uarr;</span>
        </button>
        <button class="terminal-jump-control terminal-jump-prev ${previousTurnID ? "is-visible" : ""}" type="button" data-terminal-jump-prev data-terminal-jump-target="${escapeHTML(previousTurnID)}" aria-label="${escapeHTML(t("route.terminal.jump_prev"))}" title="${escapeHTML(t("route.terminal.jump_prev"))}">
          <span class="terminal-jump-control-icon" aria-hidden="true">&uarr;</span>
        </button>
        <button class="terminal-jump-control terminal-jump-next ${nextTurnID ? "is-visible" : ""}" type="button" data-terminal-jump-next data-terminal-jump-target="${escapeHTML(nextTurnID)}" aria-label="${escapeHTML(t("route.terminal.jump_next"))}" title="${escapeHTML(t("route.terminal.jump_next"))}">
          <span class="terminal-jump-control-icon" aria-hidden="true">&darr;</span>
        </button>
        <button class="terminal-jump-control terminal-jump-bottom ${showJumpBottom ? "is-visible" : ""} ${session?.chat_has_unread_output ? "has-unread" : ""}" type="button" data-terminal-jump-bottom aria-label="${escapeHTML(t("route.terminal.jump_bottom"))}" title="${escapeHTML(t("route.terminal.jump_bottom"))}">
          <span class="terminal-jump-control-icon" aria-hidden="true">&darr;&darr;</span>
        </button>
      </div>
    </section>
    <section class="terminal-composer-shell">
      <form class="terminal-chat-form" data-terminal-input-form data-composer-form="terminal-runtime">
        ${note || detail ? `<div class="terminal-composer-note" data-terminal-runtime-note data-terminal-runtime-status="${escapeHTML(normalizeTerminalSessionStatus(session.status) || "unknown")}">${escapeHTML([note, detail].filter(Boolean).join(" | "))}</div>` : ""}
        <textarea data-terminal-input data-composer-input="terminal-runtime" maxlength="6000" rows="1" placeholder="${escapeHTML(placeholder)}" ${(sending || !canInput) ? "disabled" : ""}></textarea>
        <button type="submit" data-terminal-submit data-composer-submit="terminal-runtime" aria-label="${escapeHTML(sending ? t("route.terminal.sending") : t("route.terminal.send"))}" ${(sending || !canInput) ? "disabled" : ""}>
          <span class="terminal-chat-form-button-icon" aria-hidden="true">&uarr;</span>
          <span class="sr-only">${escapeHTML(sending ? t("route.terminal.sending") : t("route.terminal.send"))}</span>
        </button>
      </form>
      ${sessionCount > 0 ? `<div class="terminal-composer-meta">${escapeHTML(t("route.terminal.session_count", { count: String(sessionCount) }))}</div>` : ""}
    </section>
  </section>`;
}

function syncNodeText(node, value) {
  if (!node) {
    return;
  }
  const nextValue = String(value || "");
  if (node.textContent !== nextValue) {
    node.textContent = nextValue;
  }
}

function syncNodeHTML(node, html) {
  if (!node) {
    return;
  }
  const nextHTML = String(html || "");
  if (node.innerHTML !== nextHTML) {
    node.innerHTML = nextHTML;
  }
}

function syncNodeAttribute(node, name, value) {
  if (!node || !name) {
    return;
  }
  if (value === null || value === undefined) {
    node.removeAttribute(name);
    return;
  }
  const nextValue = String(value);
  if (node.getAttribute(name) !== nextValue) {
    node.setAttribute(name, nextValue);
  }
}

function syncRenderedBlock(parent, selector, html, anchorNode = null, position = "beforeend") {
  if (!parent || !selector) {
    return null;
  }
  const nextHTML = String(html || "").trim();
  const existing = parent.querySelector(selector);
  if (!nextHTML) {
    if (existing) {
      existing.remove();
    }
    return null;
  }
  if (existing) {
    if (existing.outerHTML !== nextHTML) {
      existing.outerHTML = nextHTML;
    }
    return parent.querySelector(selector);
  }
  if (anchorNode && typeof anchorNode.insertAdjacentHTML === "function") {
    anchorNode.insertAdjacentHTML(position, nextHTML);
    return parent.querySelector(selector);
  }
  parent.insertAdjacentHTML("beforeend", nextHTML);
  return parent.querySelector(selector);
}

function patchTerminalSessionPane(container, sessions, activeSessionID, mobileSessionListOpen, options = {}) {
  const paneNode = container.querySelector("[data-terminal-session-pane]");
  if (!paneNode) {
    return null;
  }
  paneNode.classList.toggle("is-open", Boolean(mobileSessionListOpen));
  syncNodeAttribute(paneNode.querySelector(".terminal-session-pane-backdrop"), "aria-label", t("route.terminal.hide_sessions"));
  syncNodeText(paneNode.querySelector(".terminal-session-pane-copy strong"), t("route.terminal.sessions"));
  syncNodeText(paneNode.querySelector(".terminal-session-pane-copy span"), t("route.terminal.session_count", { count: String(sessions.length) }));
  paneNode.querySelectorAll("[data-terminal-create]").forEach((node) => {
    syncNodeText(node, t("route.terminal.new_short"));
  });
  paneNode.querySelectorAll("[data-terminal-session-pane-close]").forEach((node) => {
    syncNodeText(node, t("route.terminal.hide_sessions"));
  });
  const sessionListNode = paneNode.querySelector("[data-terminal-session-list]");
  syncNodeHTML(sessionListNode, renderTerminalSessionCards(sessions, activeSessionID, options));
  return sessionListNode;
}

function patchTerminalWorkspaceNode(container, session, sending, closing = false, deleting = false, options = {}) {
  const workspaceNode = container.querySelector("[data-terminal-workspace]");
  if (!workspaceNode || !session) {
    return null;
  }
  const currentSessionID = normalizeText(workspaceNode.getAttribute("data-terminal-session-id"));
  const targetSessionID = normalizeText(session.id);
  if (!targetSessionID || currentSessionID !== targetSessionID) {
    return null;
  }

  const logRef = normalizeText(session.terminal_session_id);
  const isLive = isTerminalSessionLiveStatus(session.status);
  const canInput = canTerminalSessionAcceptInput(session.status);
  const normalizedStatus = normalizeTerminalSessionStatus(session.status);
  const placeholder = canInput
    ? t("route.terminal.input")
    : (isTerminalSessionBusyStatus(session.status) ? t("route.terminal.busy") : t("route.terminal.closed"));
  const showRuntimeNote = !sending && !Boolean(session?.pending_create);
  const note = showRuntimeNote
    ? (normalizedStatus === "interrupted"
      ? t("route.terminal.interrupted")
      : ((normalizedStatus === "exited" || normalizedStatus === "failed") ? t("route.terminal.closed") : ""))
    : "";
  const detail = showRuntimeNote
    ? (session.error_message || (parseTerminalExitCode(session.exit_code) !== null ? `exit code ${String(parseTerminalExitCode(session.exit_code))}` : ""))
    : "";
  const closeDisabled = sending || closing || deleting || !isTerminalSessionLiveStatus(session.status);
  const deleteDisabled = sending || closing || deleting;
  const showJumpBottom = Boolean(session?.chat_has_unread_output) || Number(session?.chat_bottom_offset || 0) > TERMINAL_JUMP_BOTTOM_SHOW_THRESHOLD;
  const metaExpanded = Boolean(session?.meta_expanded);
  const sessionCount = Number.isFinite(Number(options?.sessionCount)) ? Math.max(Number(options.sessionCount), 0) : 0;
  const sessionToggleLabel = options?.sessionSheetOpen ? t("route.terminal.hide_sessions") : t("route.terminal.sessions");
  const headerSubcopy = getTerminalSessionLastOutputAt(session) > 0
    ? t("route.terminal.last_output", { time: formatDateTime(new Date(getTerminalSessionLastOutputAt(session)).toISOString()) })
    : t("route.terminal.no_output");

  syncNodeAttribute(workspaceNode, "data-terminal-session-id", session.id);
  syncNodeAttribute(workspaceNode, "data-terminal-workspace-status", normalizeTerminalSessionStatus(session.status) || "unknown");
  syncNodeAttribute(workspaceNode, "data-terminal-workspace-live", isLive ? "true" : "false");

  const sessionPaneToggle = workspaceNode.querySelector("[data-terminal-session-pane-toggle]");
  syncNodeText(sessionPaneToggle, sessionToggleLabel);
  syncNodeAttribute(sessionPaneToggle, "aria-expanded", options?.sessionSheetOpen ? "true" : "false");
  workspaceNode.querySelectorAll(".terminal-mobile-actions [data-terminal-create]").forEach((node) => {
    syncNodeText(node, t("route.terminal.new_short"));
  });

  syncNodeText(workspaceNode.querySelector(".terminal-workspace-eyebrow"), t("route.terminal.logs.heading", { session: shorten(logRef === "-" ? "n/a" : logRef, 24) }));
  syncNodeText(workspaceNode.querySelector(".terminal-workspace-copy h4"), normalizeText(session.title));
  syncNodeText(workspaceNode.querySelector(".terminal-workspace-subcopy"), headerSubcopy);
  const runtimeStateNode = workspaceNode.querySelector("[data-terminal-runtime-state]");
  syncNodeAttribute(runtimeStateNode, "data-terminal-runtime-state", normalizedStatus || "unknown");
  syncNodeText(runtimeStateNode ? runtimeStateNode.querySelector(".terminal-runtime-state-text") : null, renderTerminalStatus(session.status));

  const metaToggleNode = workspaceNode.querySelector("[data-terminal-meta-toggle]");
  syncNodeText(metaToggleNode, metaExpanded ? t("route.terminal.details_hide") : t("route.terminal.details_show"));
  syncNodeAttribute(metaToggleNode, "aria-expanded", metaExpanded ? "true" : "false");

  const closeButtonNode = workspaceNode.querySelector("[data-terminal-close]");
  if (closeButtonNode) {
    closeButtonNode.disabled = closeDisabled;
  }
  syncNodeText(closeButtonNode, closing ? t("route.terminal.closing") : t("route.terminal.close"));
  const deleteButtonNode = workspaceNode.querySelector("[data-terminal-delete]");
  if (deleteButtonNode) {
    deleteButtonNode.disabled = deleteDisabled;
  }
  syncNodeText(deleteButtonNode, deleting ? t("route.terminal.deleting") : t("route.terminal.delete"));

  const workspaceHead = workspaceNode.querySelector(".terminal-workspace-head");
  if (workspaceHead) {
    syncRenderedBlock(
      workspaceHead,
      "[data-terminal-meta-panel]",
      metaExpanded ? renderTerminalWorkspaceMetaPanel(session) : ""
    );
  }

  const consolePanelNode = workspaceNode.querySelector("[data-terminal-console-panel]");
  const chatNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-chat-screen]") : null;
  if (chatNode) {
    syncNodeAttribute(chatNode, "data-terminal-chat-status", normalizeTerminalSessionStatus(session.status) || "unknown");
    syncNodeHTML(chatNode.querySelector(".terminal-log-tree"), renderTerminalTurns(session));
  }
  const jumpTopNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-jump-top]") : null;
  if (jumpTopNode) {
    jumpTopNode.classList.toggle("is-visible", Number(session?.chat_scroll_top || 0) > TERMINAL_JUMP_TOP_SHOW_THRESHOLD);
    syncNodeAttribute(jumpTopNode, "aria-label", t("route.terminal.jump_top"));
    syncNodeAttribute(jumpTopNode, "title", t("route.terminal.jump_top"));
  }
  const jumpPrevNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-jump-prev]") : null;
  if (jumpPrevNode) {
    const previousTurnID = normalizeText(session?.chat_previous_turn_id);
    jumpPrevNode.classList.toggle("is-visible", Boolean(previousTurnID));
    syncNodeAttribute(jumpPrevNode, "data-terminal-jump-target", previousTurnID);
    syncNodeAttribute(jumpPrevNode, "aria-label", t("route.terminal.jump_prev"));
    syncNodeAttribute(jumpPrevNode, "title", t("route.terminal.jump_prev"));
  }
  const jumpNextNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-jump-next]") : null;
  if (jumpNextNode) {
    const nextTurnID = normalizeText(session?.chat_next_turn_id);
    jumpNextNode.classList.toggle("is-visible", Boolean(nextTurnID));
    syncNodeAttribute(jumpNextNode, "data-terminal-jump-target", nextTurnID);
    syncNodeAttribute(jumpNextNode, "aria-label", t("route.terminal.jump_next"));
    syncNodeAttribute(jumpNextNode, "title", t("route.terminal.jump_next"));
  }
  const jumpBottomNode = consolePanelNode ? consolePanelNode.querySelector("[data-terminal-jump-bottom]") : null;
  if (jumpBottomNode) {
    jumpBottomNode.classList.toggle("is-visible", showJumpBottom);
    jumpBottomNode.classList.toggle("has-unread", Boolean(session?.chat_has_unread_output));
    syncNodeAttribute(jumpBottomNode, "aria-label", t("route.terminal.jump_bottom"));
    syncNodeAttribute(jumpBottomNode, "title", t("route.terminal.jump_bottom"));
  }

  const composerShellNode = workspaceNode.querySelector(".terminal-composer-shell");
  const formNode = composerShellNode ? composerShellNode.querySelector("[data-terminal-input-form]") : null;
  const inputNode = formNode ? formNode.querySelector("[data-terminal-input]") : null;
  if (composerShellNode) {
    syncRenderedBlock(
      formNode,
      "[data-terminal-runtime-note]",
      note || detail
        ? `<div class="terminal-composer-note" data-terminal-runtime-note data-terminal-runtime-status="${escapeHTML(normalizeTerminalSessionStatus(session.status) || "unknown")}">${escapeHTML([note, detail].filter(Boolean).join(" | "))}</div>`
        : "",
      inputNode,
      "beforebegin"
    );
    syncRenderedBlock(
      composerShellNode,
      ".terminal-composer-meta",
      sessionCount > 0 ? `<div class="terminal-composer-meta">${escapeHTML(t("route.terminal.session_count", { count: String(sessionCount) }))}</div>` : "",
      formNode,
      "afterend"
    );
  }
  if (inputNode) {
    inputNode.placeholder = placeholder;
    inputNode.disabled = sending || !canInput;
  }
  const submitNode = formNode ? formNode.querySelector("[data-terminal-submit]") : null;
  if (submitNode) {
    submitNode.disabled = sending || !canInput;
    syncNodeAttribute(submitNode, "aria-label", sending ? t("route.terminal.sending") : t("route.terminal.send"));
    syncNodeText(submitNode.querySelector(".sr-only"), sending ? t("route.terminal.sending") : t("route.terminal.send"));
  }

  return {
    workspaceNode,
    chatNode,
    inputNode
  };
}

async function loadTerminalView(container) {
  const localState = {
    clientID: getTerminalClientID(),
    sessions: loadTerminalSessionsFromStorage(),
    activeSessionID: "",
    sending: false,
    closing: false,
    deleting: false,
    deletingSessionID: "",
    polling: false,
    timer: 0,
    drafts: {},
    focusedInputSessionID: "",
    focusedInputSelectionStart: -1,
    focusedInputSelectionEnd: -1,
    composingInputSessionID: "",
    sessionListScrollTop: 0,
    revealActiveSessionCard: false,
    mobileSessionListOpen: false,
    mobileSessionListAutoOpened: false,
    pendingPaint: false,
    pendingScrollToBottom: false,
    deferredPaintTimer: 0,
    scrollCaptureFrame: 0,
    pendingScrollCapture: null,
    persistTimer: 0,
    lastChatScrollAt: 0,
    lastComposerInputAt: 0,
    nextListSyncAt: 0,
    creating: false,
    composerBinding: null,
    turnNavigationCache: null
  };
  const terminalComposer = createReusableComposer();
  container.__alter0TerminalVisible = () => {
    localState.nextListSyncAt = 0;
    void startPolling();
  };
  localState.activeSessionID = localState.sessions[0] ? localState.sessions[0].id : "";
  localState.mobileSessionListOpen = localState.sessions.length === 0;
  localState.mobileSessionListAutoOpened = localState.mobileSessionListOpen;

  const resolveTerminalSessionSheetOpen = () => {
    return isTerminalSessionSheetViewport() && Boolean(localState.mobileSessionListOpen);
  };

  const getActiveSession = () => {
    return localState.sessions.find((item) => item.id === localState.activeSessionID) || null;
  };

  const computeTerminalListPollInterval = () => {
    if (!isDocumentVisible()) {
      return TERMINAL_SESSION_LIST_POLL_HIDDEN_INTERVAL_MS;
    }
    return isMobileViewport() ? Math.max(TERMINAL_SESSION_LIST_POLL_INTERVAL_MS * 2, TERMINAL_SESSION_LIST_POLL_INTERVAL_MS) : TERMINAL_SESSION_LIST_POLL_INTERVAL_MS;
  };

  const computeTerminalPollDelay = (session) => {
    if (!session) {
      return TERMINAL_POLL_INTERVAL_IDLE_MS;
    }
    if (!isDocumentVisible()) {
      return TERMINAL_POLL_INTERVAL_HIDDEN_MS;
    }
    if (isMobileViewport()) {
      if (isTerminalInputFocused(session.id) || isTerminalInputComposing(session.id)) {
        return TERMINAL_POLL_INTERVAL_IDLE_MS;
      }
      const recentlyScrolled = Date.now() - Number(localState.lastChatScrollAt || 0) < 1500;
      if (recentlyScrolled) {
        return TERMINAL_POLL_INTERVAL_IDLE_MS;
      }
    }
    return TERMINAL_POLL_INTERVAL_ACTIVE_MS;
  };

  const discardTerminalSession = (sessionID) => {
    const key = normalizeText(sessionID);
    if (!key) {
      return false;
    }
    const index = localState.sessions.findIndex((item) => normalizeText(item.id) === key);
    if (index < 0) {
      return false;
    }
    localState.sessions.splice(index, 1);
    delete localState.drafts[key];
    clearTerminalInputComposition(key);
    if (normalizeText(localState.focusedInputSessionID) === key) {
      clearTerminalInputFocus();
    }
    return true;
  };

  const sortTerminalSessions = () => {
    localState.sessions.sort(compareTerminalSessions);
  };

  const markTerminalSessionActivity = (session, at = Date.now()) => {
    if (!session) {
      return;
    }
    session.updated_at = Math.max(Number(session.updated_at || 0), Number(at || 0));
    sortTerminalSessions();
  };

  const persist = (options = {}) => {
    if (localState.persistTimer) {
      window.clearTimeout(localState.persistTimer);
      localState.persistTimer = 0;
    }
    if (options && options.immediate) {
      persistTerminalSessionsToStorage(localState.sessions);
      return;
    }
    const elapsed = Date.now() - Number(localState.lastChatScrollAt || 0);
    const delay = isMobileViewport()
      ? (elapsed >= TERMINAL_STORAGE_PERSIST_IDLE_DELAY_MS
        ? TERMINAL_STORAGE_PERSIST_ACTIVE_DELAY_MS
        : Math.max(TERMINAL_STORAGE_PERSIST_ACTIVE_DELAY_MS, TERMINAL_STORAGE_PERSIST_IDLE_DELAY_MS - elapsed))
      : (elapsed >= 240 ? 80 : Math.max(80, 240 - elapsed));
    localState.persistTimer = window.setTimeout(() => {
      localState.persistTimer = 0;
      persistTerminalSessionsToStorage(localState.sessions);
    }, delay);
  };

  const readTerminalDraft = (sessionID) => {
    const key = normalizeText(sessionID);
    if (!key || !Object.prototype.hasOwnProperty.call(localState.drafts, key)) {
      return "";
    }
    return String(localState.drafts[key] || "");
  };

  const writeTerminalDraft = (sessionID, value) => {
    const key = normalizeText(sessionID);
    if (!key) {
      return;
    }
    const normalized = String(value || "");
    if (normalized) {
      localState.drafts[key] = normalized;
      return;
    }
    delete localState.drafts[key];
  };

  const rememberTerminalInputFocus = (sessionID, inputNode = null) => {
    localState.focusedInputSessionID = normalizeText(sessionID);
    if (inputNode && Number.isFinite(inputNode.selectionStart) && Number.isFinite(inputNode.selectionEnd)) {
      localState.focusedInputSelectionStart = Number(inputNode.selectionStart);
      localState.focusedInputSelectionEnd = Number(inputNode.selectionEnd);
      return;
    }
    localState.focusedInputSelectionStart = -1;
    localState.focusedInputSelectionEnd = -1;
  };

  const clearTerminalInputFocus = () => {
    localState.focusedInputSessionID = "";
    localState.focusedInputSelectionStart = -1;
    localState.focusedInputSelectionEnd = -1;
  };

  const stopDeferredTerminalPaintFlush = () => {
    if (localState.deferredPaintTimer) {
      window.clearTimeout(localState.deferredPaintTimer);
      localState.deferredPaintTimer = 0;
    }
  };

  const markTerminalComposerActivity = () => {
    localState.lastComposerInputAt = Date.now();
  };

  const isTerminalInputFocused = (sessionID) => {
    const key = normalizeText(sessionID);
    if (key === "" || key !== normalizeText(localState.focusedInputSessionID)) {
      return false;
    }
    const inputNode = container.querySelector("[data-terminal-input]");
    return Boolean(inputNode && document.activeElement === inputNode);
  };

  const isTerminalInputComposing = (sessionID) => {
    const key = normalizeText(sessionID);
    return key !== "" && key === normalizeText(localState.composingInputSessionID);
  };

  const resizeTerminalComposerInput = (inputNode = null) => {
    const node = inputNode || container.querySelector("[data-terminal-input]");
    if (!(node instanceof HTMLTextAreaElement)) {
      return;
    }
    const style = window.getComputedStyle(node);
    const lineHeight = Math.max(parseFloat(style.lineHeight || "24"), 20);
    const maxHeight = lineHeight * 5;
    node.style.height = "auto";
    const nextHeight = Math.min(Math.max(node.scrollHeight, lineHeight + 16), maxHeight);
    node.style.height = `${Math.ceil(nextHeight)}px`;
    node.style.overflowY = node.scrollHeight > maxHeight + 1 ? "auto" : "hidden";
  };

  const invalidateTerminalTurnNavigationCache = () => {
    localState.turnNavigationCache = null;
  };

  const resolveTerminalNodeOffsetWithin = (node, boundary) => {
    if (!(node instanceof HTMLElement) || !(boundary instanceof HTMLElement)) {
      return 0;
    }
    let total = 0;
    let current = node;
    while (current && current !== boundary) {
      total += Math.max(Number(current.offsetTop || 0), 0);
      const next = current.offsetParent instanceof HTMLElement ? current.offsetParent : current.parentElement;
      if (!(next instanceof HTMLElement)) {
        break;
      }
      current = next;
    }
    if (current === boundary) {
      return Math.max(total, 0);
    }
    return measureTerminalNodeScrollOffset(node, boundary);
  };

  const resolveTerminalTurnNavigationEntries = (sessionID, chatNode) => {
    const key = normalizeText(sessionID);
    if (!key || !(chatNode instanceof HTMLElement)) {
      return [];
    }
    const cache = localState.turnNavigationCache;
    const scrollHeight = Math.max(Number(chatNode.scrollHeight || 0), 0);
    const clientWidth = Math.max(Number(chatNode.clientWidth || 0), 0);
    if (
      cache &&
      cache.sessionID === key &&
      cache.chatNode === chatNode &&
      cache.scrollHeight === scrollHeight &&
      cache.clientWidth === clientWidth
    ) {
      return Array.isArray(cache.entries) ? cache.entries : [];
    }
    const entries = [...chatNode.querySelectorAll("[data-terminal-turn]")].map((turnNode) => {
      if (!(turnNode instanceof HTMLElement)) {
        return null;
      }
      const id = normalizeText(turnNode.getAttribute("data-terminal-turn"));
      if (!id || id === "-") {
        return null;
      }
      const promptNode = turnNode.querySelector(".terminal-turn-prompt");
      const anchorNode = promptNode instanceof HTMLElement ? promptNode : turnNode;
      return {
        id,
        anchor: resolveTerminalNodeOffsetWithin(anchorNode, chatNode),
        start: resolveTerminalNodeOffsetWithin(turnNode, chatNode)
      };
    }).filter(Boolean);
    localState.turnNavigationCache = {
      sessionID: key,
      chatNode,
      scrollHeight,
      clientWidth,
      entries
    };
    return entries;
  };

  const measureTerminalNodeScrollOffset = (node, scrollNode) => {
    if (!(node instanceof HTMLElement) || !(scrollNode instanceof HTMLElement)) {
      return 0;
    }
    const nodeRect = node.getBoundingClientRect();
    const scrollRect = scrollNode.getBoundingClientRect();
    return Math.max(scrollNode.scrollTop + nodeRect.top - scrollRect.top, 0);
  };

  const resolveTerminalTurnNavigation = (sessionID, chatNode) => {
    if (!(chatNode instanceof HTMLElement)) {
      return {
        previousTurnID: "",
        nextTurnID: ""
      };
    }
    const viewportAnchor = Math.max(Number(chatNode.scrollTop || 0) + 24, 0);
    const turnEntries = resolveTerminalTurnNavigationEntries(sessionID, chatNode);
    if (!turnEntries.length) {
      return {
        previousTurnID: "",
        nextTurnID: ""
      };
    }
    let currentIndex = 0;
    for (let index = 0; index < turnEntries.length; index += 1) {
      const entry = turnEntries[index];
      const nextEntry = turnEntries[index + 1] || null;
      if (viewportAnchor < entry.anchor) {
        currentIndex = Math.max(index - 1, 0);
        break;
      }
      currentIndex = index;
      if (!nextEntry || viewportAnchor < nextEntry.anchor) {
        break;
      }
    }
    return {
      previousTurnID: currentIndex > 0 ? turnEntries[currentIndex - 1].id : "",
      nextTurnID: currentIndex < turnEntries.length - 1 ? turnEntries[currentIndex + 1].id : ""
    };
  };

  const shouldDeferTerminalPaint = (sessionID) => {
    const key = normalizeText(sessionID);
    if (!key) {
      return false;
    }
    if (isTerminalInputComposing(key)) {
      return true;
    }
    if (!isTerminalInputFocused(key)) {
      return false;
    }
    if (isMobileViewport()) {
      return true;
    }
    return Date.now() - Number(localState.lastComposerInputAt || 0) < TERMINAL_INPUT_PAINT_IDLE_MS;
  };

  const scheduleDeferredTerminalPaintFlush = (sessionID = "") => {
    stopDeferredTerminalPaintFlush();
    if (!localState.pendingPaint) {
      return;
    }
    const key = normalizeText(sessionID);
    if (!key) {
      return;
    }
    if (isTerminalInputComposing(key)) {
      return;
    }
    if (isMobileViewport() && isTerminalInputFocused(key)) {
      return;
    }
    const elapsed = Date.now() - Number(localState.lastComposerInputAt || 0);
    const delay = Math.max(TERMINAL_INPUT_PAINT_IDLE_MS - elapsed, 16);
    localState.deferredPaintTimer = window.setTimeout(() => {
      localState.deferredPaintTimer = 0;
      flushDeferredTerminalPaint();
    }, delay);
  };

  const rememberTerminalInputComposition = (sessionID) => {
    localState.composingInputSessionID = normalizeText(sessionID);
  };

  const clearTerminalInputComposition = (sessionID = "") => {
    const key = normalizeText(sessionID);
    if (key && key !== normalizeText(localState.composingInputSessionID)) {
      return;
    }
    localState.composingInputSessionID = "";
  };

  const captureTerminalChatScroll = (sessionID, chatNode = null, options = {}) => {
    const key = normalizeText(sessionID);
    if (!key) {
      return;
    }
    const session = localState.sessions.find((item) => normalizeText(item.id) === key) || null;
    if (!session) {
      return;
    }
    const node = chatNode || container.querySelector("[data-terminal-chat-screen]");
    if (!node || node.hidden) {
      return;
    }
    if (options.trackActivity !== false) {
      localState.lastChatScrollAt = Date.now();
    }
    session.chat_scroll_top = Math.max(Number(node.scrollTop || 0), 0);
    const remaining = Math.max(Number(node.scrollHeight || 0) - Number(node.scrollTop || 0) - Number(node.clientHeight || 0), 0);
    session.chat_bottom_offset = remaining;
    session.chat_stick_to_bottom = remaining <= TERMINAL_SCROLL_STICKY_THRESHOLD;
    const turnNavigation = resolveTerminalTurnNavigation(key, node);
    session.chat_previous_turn_id = turnNavigation.previousTurnID;
    session.chat_next_turn_id = turnNavigation.nextTurnID;
    if (session.chat_stick_to_bottom) {
      session.chat_last_seen_output_at = Number(session.last_output_at || 0);
      session.chat_has_unread_output = false;
    }
    const activeSession = getActiveSession();
    if (activeSession && normalizeText(activeSession.id) === key) {
      const jumpTopButton = container.querySelector("[data-terminal-jump-top]");
      const jumpPrevButton = container.querySelector("[data-terminal-jump-prev]");
      const jumpNextButton = container.querySelector("[data-terminal-jump-next]");
      const jumpButton = container.querySelector("[data-terminal-jump-bottom]");
      if (jumpTopButton) {
        jumpTopButton.classList.toggle("is-visible", Number(session.chat_scroll_top || 0) > TERMINAL_JUMP_TOP_SHOW_THRESHOLD);
      }
      if (jumpPrevButton) {
        jumpPrevButton.classList.toggle("is-visible", Boolean(turnNavigation.previousTurnID));
        syncNodeAttribute(jumpPrevButton, "data-terminal-jump-target", turnNavigation.previousTurnID);
      }
      if (jumpNextButton) {
        jumpNextButton.classList.toggle("is-visible", Boolean(turnNavigation.nextTurnID));
        syncNodeAttribute(jumpNextButton, "data-terminal-jump-target", turnNavigation.nextTurnID);
      }
      if (jumpButton) {
        const shouldShow = session.chat_has_unread_output || remaining > TERMINAL_JUMP_BOTTOM_SHOW_THRESHOLD;
        jumpButton.classList.toggle("is-visible", shouldShow);
        jumpButton.classList.toggle("has-unread", session.chat_has_unread_output);
      }
    }
  };

  const scrollTerminalChatToBottom = () => {
    const activeSession = getActiveSession();
    const chatNode = container.querySelector("[data-terminal-chat-screen]");
    if (chatNode && !chatNode.hidden && (!activeSession || activeSession.chat_stick_to_bottom !== false)) {
      chatNode.scrollTop = chatNode.scrollHeight;
      if (activeSession) {
        activeSession.chat_scroll_top = Math.max(Number(chatNode.scrollTop || 0), 0);
        activeSession.chat_bottom_offset = 0;
        activeSession.chat_last_seen_output_at = Number(activeSession.last_output_at || 0);
        activeSession.chat_has_unread_output = false;
        activeSession.chat_stick_to_bottom = true;
      }
    }
  };

  const scrollTerminalChatToTop = () => {
    const activeSession = getActiveSession();
    const chatNode = container.querySelector("[data-terminal-chat-screen]");
    if (!(chatNode instanceof HTMLElement) || chatNode.hidden) {
      return;
    }
    chatNode.scrollTop = 0;
    if (activeSession) {
      activeSession.chat_has_unread_output = false;
      captureTerminalChatScroll(activeSession.id, chatNode, { trackActivity: false });
    }
  };

  const scrollTerminalChatToTurn = (turnID) => {
    const key = normalizeText(turnID);
    if (!key) {
      return;
    }
    const activeSession = getActiveSession();
    const chatNode = container.querySelector("[data-terminal-chat-screen]");
    if (!(chatNode instanceof HTMLElement) || chatNode.hidden) {
      return;
    }
    const escapedTurnID = window.CSS && typeof window.CSS.escape === "function"
      ? window.CSS.escape(key)
      : key.replace(/["\\]/g, "\\$&");
    const turnNode = chatNode.querySelector(`[data-terminal-turn="${escapedTurnID}"]`);
    if (!(turnNode instanceof HTMLElement)) {
      return;
    }
    chatNode.scrollTop = Math.max(measureTerminalNodeScrollOffset(turnNode, chatNode) - 12, 0);
    if (activeSession) {
      captureTerminalChatScroll(activeSession.id, chatNode, { trackActivity: false });
    }
  };

  const canScrollNode = (node, deltaY) => {
    if (!node) {
      return false;
    }
    const scrollTop = Math.max(Number(node.scrollTop || 0), 0);
    const maxScrollTop = Math.max(Number(node.scrollHeight || 0) - Number(node.clientHeight || 0), 0);
    if (maxScrollTop <= 0) {
      return false;
    }
    if (deltaY < 0) {
      return scrollTop > 0;
    }
    if (deltaY > 0) {
      return scrollTop < maxScrollTop;
    }
    return false;
  };

  const findScrollableAncestorWithin = (target, boundary) => {
    let current = target instanceof Element ? target : null;
    while (current && current !== boundary) {
      const style = window.getComputedStyle(current);
      const overflowY = String(style.overflowY || "").toLowerCase();
      const scrollable = ["auto", "scroll", "overlay"].includes(overflowY) && current.scrollHeight > current.clientHeight + 1;
      if (scrollable) {
        return current;
      }
      current = current.parentElement;
    }
    return null;
  };

  const routeTerminalWheelToChat = (event, chatNode, boundary) => {
    if (!chatNode || !(boundary instanceof Element)) {
      return;
    }
    if (!(event.target instanceof Element)) {
      return;
    }
    const target = event.target;
    if (target.closest("textarea, input, select, option")) {
      return;
    }
    const nearestScrollable = findScrollableAncestorWithin(target, boundary);
    if (nearestScrollable && nearestScrollable !== chatNode && canScrollNode(nearestScrollable, event.deltaY)) {
      return;
    }
    if (!canScrollNode(chatNode, event.deltaY)) {
      return;
    }
    chatNode.scrollTop += event.deltaY;
    event.preventDefault();
  };

  const requestRevealActiveSessionCard = () => {
    localState.revealActiveSessionCard = true;
  };

  const appendEntry = (session, role, text, options = {}) => {
    const content = String(text || "");
    if (!session || !content) {
      return;
    }
    session.entries.push({
      id: makeID(),
      role,
      text: content,
      at: Number.isFinite(Number(options.at)) ? Number(options.at) : Date.now(),
      kind: String(options.kind || "").trim().toLowerCase(),
      stream: String(options.stream || options.kind || "").trim().toLowerCase(),
      cursor: Number.isFinite(Number(options.cursor)) ? Number(options.cursor) : -1
    });
    if (session.entries.length > 1200) {
      session.entries = session.entries.slice(session.entries.length - 1200);
    }
    if (String(role || "").trim().toLowerCase() === "output") {
      const outputAt = Number.isFinite(Number(options.at)) ? Number(options.at) : Date.now();
      session.last_output_at = Math.max(Number(session.last_output_at || 0), outputAt);
    }
    markTerminalSessionActivity(session, Number.isFinite(Number(options.at)) ? Number(options.at) : Date.now());
  };

  const fetchTerminalStepDetail = async (session, turnID, stepID) => {
    if (!session || !turnID || !stepID) {
      return null;
    }
    const key = normalizeText(stepID);
    session.step_loading[key] = true;
    delete session.step_errors[key];
    requestTerminalPaint();
    try {
      const payload = await requestTerminalJSON(`/api/terminal/sessions/${encodeURIComponent(session.id)}/turns/${encodeURIComponent(turnID)}/steps/${encodeURIComponent(stepID)}`);
      const detail = normalizeTerminalStoredStepDetail(payload?.step || {}, Date.now());
      if (detail) {
        session.step_details[key] = detail;
      }
      return detail;
    } catch (error) {
      const message = error instanceof Error ? error.message : "unknown_error";
      session.step_errors[key] = message;
      throw error;
    } finally {
      session.step_loading[key] = false;
      requestTerminalPaint();
    }
  };

  const requestTerminalJSON = async (path, options = {}) => {
    const headers = new Headers(options.headers || {});
    headers.set("X-Alter0-Terminal-Client", localState.clientID);
    if (options.body && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }
    const response = await fetch(path, {
      method: options.method || "GET",
      body: options.body,
      headers
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      const error = new Error(typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`);
      error.code = typeof payload?.error_code === "string" ? payload.error_code : "";
      error.status = response.status;
      error.payload = payload;
      throw error;
    }
    return payload;
  };

  const upsertSession = (snapshot) => {
    const sessionID = String(snapshot?.id || snapshot?.terminal_session_id || "").trim();
    if (!sessionID) {
      return null;
    }
    let session = localState.sessions.find((item) => item.id === sessionID) || null;
    if (!session) {
      session = createTerminalSessionSnapshot(sessionID);
      localState.sessions.unshift(session);
    }
    applyTerminalSessionSnapshot(session, snapshot);
    sortTerminalSessions();
    return session;
  };

  const upsertSessionWithChange = (snapshot) => {
    const sessionID = String(snapshot?.id || snapshot?.terminal_session_id || "").trim();
    if (!sessionID) {
      return { session: null, changed: false };
    }
    let session = localState.sessions.find((item) => item.id === sessionID) || null;
    const isNew = !session;
    if (!session) {
      session = createTerminalSessionSnapshot(sessionID);
      localState.sessions.unshift(session);
    }
    const before = captureTerminalSessionRuntimeSignature(session);
    applyTerminalSessionSnapshot(session, snapshot);
    sortTerminalSessions();
    return {
      session,
      changed: isNew || before !== captureTerminalSessionRuntimeSignature(session)
    };
  };

  const removeTerminalSession = (sessionID) => {
    const key = normalizeText(sessionID);
    if (!key) {
      return false;
    }
    const index = localState.sessions.findIndex((item) => normalizeText(item.id) === key);
    if (index < 0) {
      return false;
    }
    const removedActive = normalizeText(localState.activeSessionID) === key;
    discardTerminalSession(key);
    const activeStillExists = localState.sessions.some((item) => normalizeText(item.id) === normalizeText(localState.activeSessionID));
    if (removedActive || !activeStillExists) {
      const nextIndex = Math.min(index, Math.max(localState.sessions.length - 1, 0));
      localState.activeSessionID = localState.sessions[nextIndex] ? localState.sessions[nextIndex].id : "";
      if (localState.activeSessionID) {
        localState.mobileSessionListOpen = false;
        localState.mobileSessionListAutoOpened = false;
        requestRevealActiveSessionCard();
      }
    }
    if (!localState.sessions.length) {
      localState.activeSessionID = "";
      localState.mobileSessionListOpen = true;
      localState.mobileSessionListAutoOpened = true;
    }
    return true;
  };

  const markSessionInterrupted = (session, message) => {
    if (!session) {
      return;
    }
    session.status = "interrupted";
    session.error_message = String(message || t("route.terminal.interrupted"));
    if (!session.disconnected_notice) {
      appendEntry(session, "system", t("route.terminal.interrupted"), {
        kind: "tag",
        stream: "system"
      });
      session.disconnected_notice = true;
    }
    sortTerminalSessions();
  };

  const interruptRecoveringTurnState = (session) => {
    if (!session || !Array.isArray(session.turns) || !session.turns.length) {
      return;
    }
    const now = Date.now();
    let changed = false;
    session.turns.forEach((turn) => {
      if (!turn) {
        return;
      }
      if (isTerminalTurnLiveStatus(turn.status)) {
        turn.status = "interrupted";
        turn.finished_at = Number(turn.finished_at || 0) > 0 ? Number(turn.finished_at) : now;
        changed = true;
      }
      const steps = Array.isArray(turn.steps) ? turn.steps : [];
      steps.forEach((step) => {
        if (!step || !isTerminalTurnLiveStatus(step.status)) {
          return;
        }
        step.status = "interrupted";
        step.finished_at = Number(step.finished_at || 0) > 0 ? Number(step.finished_at) : now;
        changed = true;
      });
    });
    if (changed) {
      session.updated_at = Math.max(Number(session.updated_at || 0), now);
    }
  };

  const serializeTerminalRecoverRequest = (session) => {
    const createdAt = Number(session?.created_at || 0);
    const lastOutputAt = Number(session?.last_output_at || 0);
    const updatedAt = Number(session?.updated_at || 0);
    const payload = {
      id: String(session?.id || "").trim(),
      terminal_session_id: String(session?.terminal_session_id || "").trim(),
      title: String(session?.title || "").trim()
    };
    if (createdAt > 0) {
      payload.created_at = new Date(createdAt).toISOString();
    }
    if (lastOutputAt > 0) {
      payload.last_output_at = new Date(lastOutputAt).toISOString();
    }
    if (updatedAt > 0) {
      payload.updated_at = new Date(updatedAt).toISOString();
    }
    return payload;
  };

  const recoverTerminalSession = async (session) => {
    if (!session) {
      throw new Error("terminal session missing");
    }
    const payload = await requestTerminalJSON("/api/terminal/sessions/recover", {
      method: "POST",
      body: JSON.stringify(serializeTerminalRecoverRequest(session))
    });
    interruptRecoveringTurnState(session);
    applyTerminalSessionSnapshot(session, payload?.session || {});
    session.disconnected_notice = false;
    return session;
  };

  const recoverStoredSessions = async () => {
    const recoverable = localState.sessions.filter((session) => hasRecoverableTerminalThread(session));
    if (!recoverable.length) {
      return;
    }
    await Promise.allSettled(recoverable.map(async (session) => {
      await recoverTerminalSession(session);
    }));
    sortTerminalSessions();
    persist();
  };

  const createNewTerminalSession = async () => {
    if (localState.creating) {
      return getActiveSession();
    }
    localState.creating = true;
    const previousActiveSessionID = localState.activeSessionID;
    const pendingSession = createTerminalSessionSnapshot(`terminal-pending-${makeID()}`);
    pendingSession.pending_create = true;
    pendingSession.status = "busy";
    localState.sessions.unshift(pendingSession);
    localState.activeSessionID = pendingSession.id;
    localState.mobileSessionListOpen = false;
    localState.mobileSessionListAutoOpened = false;
    requestRevealActiveSessionCard();
    requestTerminalPaint();
    const payload = await requestTerminalJSON("/api/terminal/sessions", {
      method: "POST",
      body: JSON.stringify({})
    }).catch((error) => {
      discardTerminalSession(pendingSession.id);
      localState.activeSessionID = localState.sessions.some((item) => normalizeText(item.id) === normalizeText(previousActiveSessionID))
        ? previousActiveSessionID
        : (localState.sessions[0] ? localState.sessions[0].id : "");
      if (!localState.activeSessionID) {
        localState.mobileSessionListOpen = true;
        localState.mobileSessionListAutoOpened = true;
      }
      localState.creating = false;
      requestTerminalPaint();
      throw error;
    });
    discardTerminalSession(pendingSession.id);
    const session = upsertSession(payload?.session || {});
    localState.creating = false;
    if (!session) {
      localState.activeSessionID = localState.sessions[0] ? localState.sessions[0].id : "";
      requestTerminalPaint();
      throw new Error("terminal session missing");
    }
    localState.activeSessionID = session.id;
    localState.mobileSessionListOpen = false;
    localState.mobileSessionListAutoOpened = false;
    requestRevealActiveSessionCard();
    persist();
    return session;
  };

  const mergeRuntimeSessions = (items) => {
    const before = captureTerminalSessionListSignature(localState.sessions);
    const activeRuntimeIDs = new Set();
    (Array.isArray(items) ? items : []).forEach((item) => {
      const { session } = upsertSessionWithChange(item);
      if (session) {
        activeRuntimeIDs.add(session.id);
      }
    });
    localState.sessions.forEach((session) => {
      if (isPendingTerminalSession(session)) {
        return;
      }
      if (!activeRuntimeIDs.has(session.id) && isTerminalSessionLiveStatus(session.status)) {
        markSessionInterrupted(session, t("route.terminal.interrupted"));
      }
    });
    if (!localState.activeSessionID && localState.sessions[0]) {
      localState.activeSessionID = localState.sessions[0].id;
    }
    if (!isTerminalSessionSheetViewport()) {
      localState.mobileSessionListOpen = false;
      localState.mobileSessionListAutoOpened = false;
    }
    if (localState.mobileSessionListAutoOpened && localState.sessions.length > 0 && localState.activeSessionID) {
      localState.mobileSessionListOpen = false;
      localState.mobileSessionListAutoOpened = false;
    }
    const changed = before !== captureTerminalSessionListSignature(localState.sessions);
    if (changed) {
      persist();
    }
    return changed;
  };

  const syncSessionList = async (options = {}) => {
    const force = Boolean(options.force);
    if (!force && Date.now() < Number(localState.nextListSyncAt || 0)) {
      return false;
    }
    const payload = await requestTerminalJSON("/api/terminal/sessions");
    localState.nextListSyncAt = Date.now() + computeTerminalListPollInterval();
    return mergeRuntimeSessions(Array.isArray(payload?.items) ? payload.items : []);
  };

  const stopPolling = () => {
    if (localState.timer) {
      window.clearTimeout(localState.timer);
      localState.timer = 0;
    }
  };

  const paint = () => {
    flushScheduledTerminalChatScrollCapture();
    invalidateTerminalTurnNavigationCache();
    const active = getActiveSession();
    const previousWorkspace = container.querySelector("[data-terminal-workspace]");
    const previousSessionList = container.querySelector("[data-terminal-session-list]");
    const previousChatNode = container.querySelector("[data-terminal-chat-screen]");
    const previousSessionID = normalizeText(previousWorkspace ? previousWorkspace.getAttribute("data-terminal-session-id") : "");
    const previousInput = container.querySelector("[data-terminal-input]");
    const previousValue = previousInput ? String(previousInput.value || "") : "";
    localState.sessionListScrollTop = previousSessionList ? previousSessionList.scrollTop : localState.sessionListScrollTop;
    if (previousSessionID && previousChatNode) {
      captureTerminalChatScroll(previousSessionID, previousChatNode, { trackActivity: false });
    }
    if (previousSessionID && previousInput && document.activeElement === previousInput) {
      rememberTerminalInputFocus(previousSessionID, previousInput);
    }
    const shouldRestoreFocus = active && normalizeText(active.id) === normalizeText(localState.focusedInputSessionID);
    const selectionStart = shouldRestoreFocus ? localState.focusedInputSelectionStart : -1;
    const selectionEnd = shouldRestoreFocus ? localState.focusedInputSelectionEnd : -1;
    if (previousSessionID && previousInput) {
      writeTerminalDraft(previousSessionID, previousInput.value);
    }
    const canPatchActiveWorkspace = Boolean(
      active &&
      previousWorkspace &&
      previousSessionID === normalizeText(active.id) &&
      container.querySelector("[data-terminal-view]")
    );
    const sessionSheetOpen = resolveTerminalSessionSheetOpen();
    if (canPatchActiveWorkspace) {
      patchTerminalSessionPane(container, localState.sessions, localState.activeSessionID, sessionSheetOpen, {
        deleting: localState.deleting,
        deletingSessionID: localState.deletingSessionID
      });
      const patched = patchTerminalWorkspaceNode(container, active, localState.sending, localState.closing, localState.deleting, {
        sessionCount: localState.sessions.length,
        sessionSheetOpen
      });
      if (!patched) {
        container.innerHTML = `<section class="terminal-view" data-terminal-view>
          <aside class="terminal-session-pane ${sessionSheetOpen ? "is-open" : ""}" data-terminal-session-pane>
            <button class="terminal-session-pane-backdrop" type="button" data-terminal-session-pane-close aria-label="${escapeHTML(t("route.terminal.hide_sessions"))}"></button>
            <div class="route-surface terminal-session-pane-shell">
              <div class="terminal-session-pane-head">
                <div class="terminal-session-pane-copy">
                  <strong>${escapeHTML(t("route.terminal.sessions"))}</strong>
                  <span>${escapeHTML(t("route.terminal.session_count", { count: String(localState.sessions.length) }))}</span>
                </div>
                <div class="terminal-session-pane-actions">
                  <button class="terminal-session-pane-action is-primary" type="button" data-terminal-create>${escapeHTML(t("route.terminal.new_short"))}</button>
                  <button class="terminal-session-pane-action terminal-session-pane-close" type="button" data-terminal-session-pane-close>${escapeHTML(t("route.terminal.hide_sessions"))}</button>
                </div>
              </div>
            <div class="terminal-session-list" data-terminal-session-list>${renderTerminalSessionCards(localState.sessions, localState.activeSessionID, {
              deleting: localState.deleting,
              deletingSessionID: localState.deletingSessionID
            })}</div>
            </div>
          </aside>
          <section class="terminal-workspace">
            ${renderTerminalWorkspace(active, localState.sending, localState.closing, localState.deleting, {
              sessionCount: localState.sessions.length,
              sessionSheetOpen
            })}
          </section>
        </section>`;
      }
    } else {
      container.innerHTML = `<section class="terminal-view" data-terminal-view>
        <aside class="terminal-session-pane ${sessionSheetOpen ? "is-open" : ""}" data-terminal-session-pane>
          <button class="terminal-session-pane-backdrop" type="button" data-terminal-session-pane-close aria-label="${escapeHTML(t("route.terminal.hide_sessions"))}"></button>
          <div class="route-surface terminal-session-pane-shell">
            <div class="terminal-session-pane-head">
              <div class="terminal-session-pane-copy">
                <strong>${escapeHTML(t("route.terminal.sessions"))}</strong>
                <span>${escapeHTML(t("route.terminal.session_count", { count: String(localState.sessions.length) }))}</span>
              </div>
              <div class="terminal-session-pane-actions">
                <button class="terminal-session-pane-action is-primary" type="button" data-terminal-create>${escapeHTML(t("route.terminal.new_short"))}</button>
                <button class="terminal-session-pane-action terminal-session-pane-close" type="button" data-terminal-session-pane-close>${escapeHTML(t("route.terminal.hide_sessions"))}</button>
              </div>
            </div>
          <div class="terminal-session-list" data-terminal-session-list>${renderTerminalSessionCards(localState.sessions, localState.activeSessionID, {
            deleting: localState.deleting,
            deletingSessionID: localState.deletingSessionID
          })}</div>
          </div>
        </aside>
        <section class="terminal-workspace">
          ${renderTerminalWorkspace(active, localState.sending, localState.closing, localState.deleting, {
            sessionCount: localState.sessions.length,
            sessionSheetOpen
          })}
        </section>
      </section>`;
    }
    const terminalViewNode = container.querySelector("[data-terminal-view]");
    if (terminalViewNode instanceof HTMLElement && Number(terminalViewNode.scrollTop || 0) !== 0) {
      terminalViewNode.scrollTop = 0;
    }
    const sessionListNode = container.querySelector("[data-terminal-session-list]");
    if (sessionListNode) {
      sessionListNode.scrollTop = Math.max(Number(localState.sessionListScrollTop || 0), 0);
      if (localState.revealActiveSessionCard) {
        const activeCard = sessionListNode.querySelector(".terminal-session-card.active");
        if (activeCard) {
          activeCard.scrollIntoView({ block: "nearest" });
        }
        localState.sessionListScrollTop = sessionListNode.scrollTop;
      }
    }
    localState.revealActiveSessionCard = false;
    const chatNode = container.querySelector("[data-terminal-chat-screen]");
    const workspaceNode = container.querySelector("[data-terminal-workspace]");
    if (active && chatNode) {
      if (active.chat_stick_to_bottom === false) {
        chatNode.scrollTop = Math.max(Number(active.chat_scroll_top || 0), 0);
      } else {
        chatNode.scrollTop = chatNode.scrollHeight;
      }
      chatNode.onscroll = () => {
        scheduleTerminalChatScrollCapture(active.id, chatNode);
      };
      chatNode.onwheel = (event) => {
        routeTerminalWheelToChat(event, chatNode, chatNode);
      };
      captureTerminalChatScroll(active.id, chatNode, { trackActivity: false });
    }
    if (active && chatNode && workspaceNode) {
      workspaceNode.onwheel = (event) => {
        if (event.target instanceof Element && event.target.closest("[data-terminal-chat-screen]")) {
          return;
        }
        routeTerminalWheelToChat(event, chatNode, workspaceNode);
      };
    }
    const inputNode = container.querySelector("[data-terminal-input]");
    if (active && inputNode) {
      const currentInputValue = String(inputNode.value || "");
      const draft = readTerminalDraft(active.id) || (previousSessionID === normalizeText(active.id) ? previousValue : "");
      if (draft && (!canPatchActiveWorkspace || !currentInputValue)) {
        inputNode.value = draft;
        writeTerminalDraft(active.id, draft);
      }
    }
    bindTerminalComposer(active);
    resizeTerminalComposerInput(inputNode);
    if (shouldRestoreFocus && active && inputNode && !inputNode.disabled) {
      window.requestAnimationFrame(() => {
        const nextInput = container.querySelector("[data-terminal-input]");
        if (!nextInput || nextInput.disabled) {
          return;
        }
        nextInput.focus({ preventScroll: true });
        if (selectionStart >= 0 && selectionEnd >= 0) {
          const maxLength = nextInput.value.length;
          nextInput.setSelectionRange(Math.min(selectionStart, maxLength), Math.min(selectionEnd, maxLength));
        }
      });
    }
  };

  const requestTerminalPaint = (options = {}) => {
    const active = getActiveSession();
    const scrollToBottom = Boolean(options.scrollToBottom);
    if (active && shouldDeferTerminalPaint(active.id)) {
      localState.pendingPaint = true;
      localState.pendingScrollToBottom = localState.pendingScrollToBottom || scrollToBottom;
      scheduleDeferredTerminalPaintFlush(active.id);
      return false;
    }
    stopDeferredTerminalPaintFlush();
    paint();
    if (scrollToBottom && (!active || active.chat_stick_to_bottom !== false)) {
      scrollTerminalChatToBottom();
    }
    return true;
  };

  const flushDeferredTerminalPaint = () => {
    if (!localState.pendingPaint) {
      return;
    }
    const active = getActiveSession();
    if (active && shouldDeferTerminalPaint(active.id)) {
      scheduleDeferredTerminalPaintFlush(active.id);
      return;
    }
    const scrollToBottom = localState.pendingScrollToBottom;
    localState.pendingPaint = false;
    localState.pendingScrollToBottom = false;
    stopDeferredTerminalPaintFlush();
    paint();
    const nextActive = getActiveSession();
    if (scrollToBottom && (!nextActive || nextActive.chat_stick_to_bottom !== false)) {
      scrollTerminalChatToBottom();
    }
  };

  const flushScheduledTerminalChatScrollCapture = () => {
    if (localState.scrollCaptureFrame) {
      window.cancelAnimationFrame(localState.scrollCaptureFrame);
      localState.scrollCaptureFrame = 0;
    }
    const payload = localState.pendingScrollCapture;
    localState.pendingScrollCapture = null;
    if (!payload) {
      return;
    }
    captureTerminalChatScroll(payload.sessionID, payload.chatNode, {
      trackActivity: payload.trackActivity
    });
  };

  const scheduleTerminalChatScrollCapture = (sessionID, chatNode = null, options = {}) => {
    const key = normalizeText(sessionID);
    if (!key) {
      return;
    }
    const node = chatNode || container.querySelector("[data-terminal-chat-screen]");
    if (!(node instanceof HTMLElement) || node.hidden) {
      return;
    }
    const trackActivity = options.trackActivity !== false;
    if (!localState.pendingScrollCapture || normalizeText(localState.pendingScrollCapture.sessionID) !== key) {
      localState.pendingScrollCapture = {
        sessionID: key,
        chatNode: node,
        trackActivity
      };
    } else {
      localState.pendingScrollCapture.chatNode = node;
      localState.pendingScrollCapture.trackActivity = localState.pendingScrollCapture.trackActivity || trackActivity;
    }
    if (localState.scrollCaptureFrame) {
      return;
    }
    localState.scrollCaptureFrame = window.requestAnimationFrame(() => {
      localState.scrollCaptureFrame = 0;
      const payload = localState.pendingScrollCapture;
      localState.pendingScrollCapture = null;
      if (!payload) {
        return;
      }
      captureTerminalChatScroll(payload.sessionID, payload.chatNode, {
        trackActivity: payload.trackActivity
      });
    });
  };

  const bindTerminalComposer = (session) => {
    const formNode = container.querySelector("[data-terminal-input-form]");
    const inputNode = container.querySelector("[data-terminal-input]");
    const submitNode = container.querySelector("[data-terminal-submit]");
    const sessionID = normalizeText(session?.id || "");
    const disabled = localState.sending || !session || !canTerminalSessionAcceptInput(session.status);
    const currentBinding = localState.composerBinding;
    if (!inputNode || !formNode) {
      terminalComposer.unbind();
      localState.composerBinding = null;
      return;
    }
    if (
      currentBinding &&
      currentBinding.inputNode === inputNode &&
      currentBinding.formNode === formNode &&
      currentBinding.submitNode === submitNode &&
      currentBinding.sessionID === sessionID &&
      currentBinding.disabled === disabled
    ) {
      resizeTerminalComposerInput(inputNode);
      return;
    }
    terminalComposer.bind(inputNode, formNode, {
      stableName: "terminal-runtime",
      submitOnEnter: true,
      allowShiftEnter: true,
      submitStrategy: "keydown",
      draftStorage: "local",
      draftKey: () => `terminal:${normalizeText(session?.id || "default")}`,
      clearDraftOnSubmit: true,
      submitNode,
      disabled,
      onDraftRestore: (_inputNode, restoredDraft) => {
        if (!session) {
          return;
        }
        resizeTerminalComposerInput(_inputNode);
        writeTerminalDraft(session.id, restoredDraft);
      },
      onInput: (currentInputNode) => {
        if (!session) {
          return;
        }
        markTerminalComposerActivity();
        resizeTerminalComposerInput(currentInputNode);
        rememberTerminalInputFocus(session.id, currentInputNode);
        writeTerminalDraft(session.id, currentInputNode.value);
        scheduleDeferredTerminalPaintFlush(session.id);
      },
      onFocus: (currentInputNode) => {
        if (!session) {
          return;
        }
        markTerminalComposerActivity();
        rememberTerminalInputFocus(session.id, currentInputNode);
      },
      onBlur: () => {
        clearTerminalInputComposition();
        clearTerminalInputFocus();
        stopDeferredTerminalPaintFlush();
        flushDeferredTerminalPaint();
      },
      onCompositionStart: (currentInputNode) => {
        if (!session) {
          return;
        }
        markTerminalComposerActivity();
        rememberTerminalInputFocus(session.id, currentInputNode);
        rememberTerminalInputComposition(session.id);
        writeTerminalDraft(session.id, currentInputNode.value);
      },
      onCompositionEnd: (currentInputNode) => {
        markTerminalComposerActivity();
        if (!session) {
          clearTerminalInputComposition();
          flushDeferredTerminalPaint();
          return;
        }
        rememberTerminalInputFocus(session.id, currentInputNode);
        writeTerminalDraft(session.id, currentInputNode.value);
        clearTerminalInputComposition(session.id);
        flushDeferredTerminalPaint();
      },
      onSubmit: async (currentInputNode) => {
        const value = String(currentInputNode.value || "");
        terminalComposer.clearDraft();
        resizeTerminalComposerInput(currentInputNode);
        if (session) {
          writeTerminalDraft(session.id, "");
        }
        clearTerminalInputFocus();
        clearTerminalInputComposition(session ? session.id : "");
        await sendTerminalInput(value);
      }
    });
    localState.composerBinding = {
      inputNode,
      formNode,
      submitNode,
      sessionID,
      disabled
    };
    resizeTerminalComposerInput(inputNode);
  };

  const refreshSessionState = async (session) => {
    if (!session) {
      return false;
    }
    const before = captureTerminalSessionRuntimeSignature(session);
    const payload = await requestTerminalJSON(`/api/terminal/sessions/${encodeURIComponent(session.id)}`);
    applyTerminalSessionSnapshot(session, payload?.session || {});
    return before !== captureTerminalSessionRuntimeSignature(session);
  };

  const scheduleNextPoll = () => {
    stopPolling();
    const session = getActiveSession();
    if (!session || !isTerminalSessionLiveStatus(session.status)) {
      return;
    }
    localState.timer = window.setTimeout(() => {
      localState.timer = 0;
      void pollActiveSession();
    }, computeTerminalPollDelay(session));
  };

  const pollActiveSession = async () => {
    if (localState.polling) {
      return;
    }
    if (state.currentRoute !== "terminal" || !document.body.contains(container)) {
      stopPolling();
      return;
    }
    const session = getActiveSession();
    if (!session) {
      stopPolling();
      return;
    }
    localState.polling = true;
    let shouldScheduleNextPoll = true;
    try {
      const listChanged = await syncSessionList();
      const detailChanged = await refreshSessionState(session);
      const changed = listChanged || detailChanged;
      if (detailChanged) {
        persist();
      }
      if (changed) {
        requestTerminalPaint({ scrollToBottom: true });
      }
      if (!isTerminalSessionLiveStatus(session.status)) {
        stopPolling();
      }
    } catch (error) {
      shouldScheduleNextPoll = false;
      if (Number(error?.status) === 404) {
        markSessionInterrupted(session, t("route.terminal.interrupted"));
      } else {
        const message = error instanceof Error ? error.message : "unknown_error";
        appendEntry(session, "system", t("route.terminal.logs_failed", { error: message }), {
          kind: "tag",
          stream: "system"
        });
      }
      touchSession(session);
      persist();
      requestTerminalPaint();
      stopPolling();
    } finally {
      localState.polling = false;
      if (shouldScheduleNextPoll && !localState.timer) {
        scheduleNextPoll();
      }
    }
  };

  const closeTerminalSession = async (session) => {
    if (!session || localState.closing || localState.deleting || !isTerminalSessionLiveStatus(session.status)) {
      return;
    }
    localState.closing = true;
    paint();
    try {
      const payload = await requestTerminalJSON(`/api/terminal/sessions/${encodeURIComponent(session.id)}/close`, {
        method: "POST"
      });
      applyTerminalSessionSnapshot(session, payload?.session || {});
      stopPolling();
      sortTerminalSessions();
      persist();
      paint();
    } catch (error) {
      if (Number(error?.status) === 404) {
        markSessionInterrupted(session, t("route.terminal.interrupted"));
      }
      const message = error instanceof Error ? error.message : "unknown_error";
      appendEntry(session, "system", t("route.terminal.close_failed", { error: message }), {
        kind: "tag",
        stream: "system"
      });
      sortTerminalSessions();
      persist();
      paint();
    } finally {
      localState.closing = false;
      paint();
    }
  };

  const deleteTerminalSession = async (session) => {
    if (!session || localState.deleting || localState.closing || localState.sending) {
      return;
    }
    if (!window.confirm(t("route.terminal.delete_confirm"))) {
      return;
    }
    const sessionID = normalizeText(session.id);
    const deletingActive = sessionID === normalizeText(localState.activeSessionID);
    localState.deleting = true;
    localState.deletingSessionID = sessionID;
    if (deletingActive) {
      localState.mobileSessionListOpen = false;
      localState.mobileSessionListAutoOpened = false;
    }
    paint();
    try {
      await requestTerminalJSON(`/api/terminal/sessions/${encodeURIComponent(sessionID)}`, {
        method: "DELETE"
      });
      stopPolling();
      removeTerminalSession(sessionID);
      persist();
      paint();
      if (deletingActive) {
        const nextActive = getActiveSession();
        if (nextActive) {
          await startPolling();
        }
      }
    } catch (error) {
      if (Number(error?.status) === 404) {
        stopPolling();
        removeTerminalSession(sessionID);
        persist();
        paint();
        if (deletingActive) {
          const nextActive = getActiveSession();
          if (nextActive) {
            await startPolling();
          }
        }
        return;
      }
      const message = error instanceof Error ? error.message : "unknown_error";
      appendEntry(session, "system", t("route.terminal.delete_failed", { error: message }), {
        kind: "tag",
        stream: "system"
      });
      sortTerminalSessions();
      persist();
      paint();
    } finally {
      localState.deleting = false;
      localState.deletingSessionID = "";
      paint();
    }
  };

  const startPolling = async () => {
    stopPolling();
    const session = getActiveSession();
    if (!session || !isTerminalSessionLiveStatus(session.status)) {
      return;
    }
    localState.nextListSyncAt = 0;
    await pollActiveSession();
  };

  const sendTerminalInput = async (content) => {
    const text = String(content || "").trim();
    if (!text || localState.sending) {
      return;
    }
    let session = getActiveSession();
    if (!session) {
      try {
        session = await createNewTerminalSession();
      } catch (error) {
        const message = error instanceof Error ? error.message : "unknown_error";
        window.alert(t("route.terminal.send_failed", { error: message }));
        return;
      }
    }
    localState.sending = true;
    requestTerminalPaint();
    const requestInput = async () => {
      return requestTerminalJSON(`/api/terminal/sessions/${encodeURIComponent(session.id)}/input`, {
        method: "POST",
        body: JSON.stringify({
          input: normalizeTerminalLine(text, 6000)
        })
      });
    };
    try {
      let payload;
      try {
        payload = await requestInput();
      } catch (error) {
        if (!session || Number(error?.status) !== 404) {
          throw error;
        }
        await recoverTerminalSession(session);
        payload = await requestInput();
      }
      applyTerminalSessionSnapshot(session, payload?.session || {});
      sortTerminalSessions();
      persist();
      requestTerminalPaint();
      await startPolling();
    } catch (error) {
      if (Number(error?.status) === 404) {
        markSessionInterrupted(session, t("route.terminal.interrupted"));
      }
      const message = error instanceof Error ? error.message : "unknown_error";
      appendEntry(session, "system", t("route.terminal.send_failed", { error: message }), {
        kind: "tag",
        stream: "system"
      });
      sortTerminalSessions();
      persist();
      requestTerminalPaint();
    } finally {
      localState.sending = false;
      requestTerminalPaint();
      const inputNode = container.querySelector("[data-terminal-input]");
      if (inputNode) {
        inputNode.focus({ preventScroll: true });
      }
      scrollTerminalChatToBottom();
    }
  };

  paint();
  try {
    await recoverStoredSessions();
    await syncSessionList();
  } catch {
  }
  paint();
  const initialSession = getActiveSession();
  if (initialSession) {
    await startPolling();
  }

  container.onclick = (event) => {
    const target = event.target instanceof Element ? event.target.closest("button") : null;
    if (!target) {
      return;
    }
    if (target.hasAttribute("data-terminal-create")) {
      void (async () => {
        try {
          await createNewTerminalSession();
          paint();
          await startPolling();
        } catch (error) {
          const message = error instanceof Error ? error.message : "unknown_error";
          window.alert(t("route.terminal.send_failed", { error: message }));
        }
      })();
      return;
    }
    if (target.hasAttribute("data-terminal-session-pane-toggle")) {
      if (!isTerminalSessionSheetViewport()) {
        localState.mobileSessionListOpen = false;
        localState.mobileSessionListAutoOpened = false;
        requestTerminalPaint();
        return;
      }
      localState.mobileSessionListOpen = !localState.mobileSessionListOpen;
      localState.mobileSessionListAutoOpened = false;
      requestTerminalPaint();
      return;
    }
    if (target.hasAttribute("data-terminal-session-pane-close")) {
      localState.mobileSessionListOpen = false;
      localState.mobileSessionListAutoOpened = false;
      requestTerminalPaint();
      return;
    }
    if (target.hasAttribute("data-terminal-meta-toggle")) {
      const active = getActiveSession();
      if (!active) {
        return;
      }
      active.meta_expanded = !Boolean(active.meta_expanded);
      persist({ immediate: true });
      requestTerminalPaint();
      return;
    }
    if (target.hasAttribute("data-terminal-process-toggle")) {
      const active = getActiveSession();
      if (!active) {
        return;
      }
      const turnID = normalizeText(target.getAttribute("data-terminal-process-toggle"));
      if (turnID !== "-") {
        active.process_collapsed[turnID] = !resolveTerminalProcessCollapsed(active, active.turns.find((turn) => normalizeText(turn?.id) === turnID));
        persist({ immediate: true });
        requestTerminalPaint();
      }
      return;
    }
    if (target.hasAttribute("data-terminal-jump-bottom")) {
      const active = getActiveSession();
      if (!active) {
        return;
      }
      active.chat_stick_to_bottom = true;
      scrollTerminalChatToBottom();
      persist();
      requestTerminalPaint({ scrollToBottom: true });
      return;
    }
    if (target.hasAttribute("data-terminal-jump-top")) {
      const active = getActiveSession();
      if (!active) {
        return;
      }
      active.chat_stick_to_bottom = false;
      active.chat_has_unread_output = false;
      scrollTerminalChatToTop();
      persist();
      return;
    }
    if (target.hasAttribute("data-terminal-jump-prev") || target.hasAttribute("data-terminal-jump-next")) {
      const active = getActiveSession();
      if (!active) {
        return;
      }
      active.chat_stick_to_bottom = false;
      scrollTerminalChatToTurn(target.getAttribute("data-terminal-jump-target"));
      persist();
      return;
    }
    if (target.hasAttribute("data-terminal-output-toggle")) {
      const active = getActiveSession();
      if (!active) {
        return;
      }
      const turnID = normalizeText(target.getAttribute("data-terminal-output-toggle"));
      if (turnID === "-") {
        return;
      }
      active.output_collapsed[turnID] = !resolveTerminalOutputCollapsed(active, active.turns.find((turn) => normalizeText(turn?.id) === turnID));
      persist({ immediate: true });
      requestTerminalPaint();
      return;
    }
    if (target.hasAttribute("data-terminal-step-toggle")) {
      const active = getActiveSession();
      if (!active) {
        return;
      }
      const stepID = normalizeText(target.getAttribute("data-terminal-step-toggle"));
      const turnID = normalizeText(target.getAttribute("data-terminal-turn-id"));
      if (stepID === "-" || turnID === "-") {
        return;
      }
      const currentlyExpanded = Boolean(active.expanded_steps[stepID]);
      active.expanded_steps[stepID] = !currentlyExpanded;
      persist({ immediate: true });
      requestTerminalPaint();
      if (!currentlyExpanded && !active.step_details[stepID] && !active.step_loading[stepID]) {
        void fetchTerminalStepDetail(active, turnID, stepID).then(() => {
          persist();
        }).catch(() => {
          persist();
        });
      }
      return;
    }
    if (target.hasAttribute("data-terminal-close")) {
      const active = getActiveSession();
      if (active) {
        void closeTerminalSession(active);
      }
      return;
    }
    if (target.hasAttribute("data-terminal-delete")) {
      const active = getActiveSession();
      if (active) {
        void deleteTerminalSession(active);
      }
      return;
    }
    if (target.hasAttribute("data-terminal-delete-session")) {
      const sessionID = normalizeText(target.getAttribute("data-terminal-delete-session"));
      if (sessionID === "-") {
        return;
      }
      const session = localState.sessions.find((item) => normalizeText(item.id) === sessionID) || null;
      if (session) {
        void deleteTerminalSession(session);
      }
      return;
    }
    if (target.hasAttribute("data-terminal-session-select")) {
      const sessionID = normalizeText(target.getAttribute("data-terminal-session-select"));
      if (sessionID !== "-") {
        localState.activeSessionID = sessionID;
        localState.mobileSessionListOpen = false;
        localState.mobileSessionListAutoOpened = false;
        paint();
        const active = getActiveSession();
        if (active) {
          void startPolling();
          return;
        }
        stopPolling();
      }
    }
  };

  const handleTerminalStepSearch = (event) => {
    const target = event.target;
    if (!(target instanceof HTMLInputElement)) {
      return;
    }
    if (!target.hasAttribute("data-terminal-step-search")) {
      return;
    }
    const active = getActiveSession();
    if (!active) {
      return;
    }
    const stepID = normalizeText(target.getAttribute("data-terminal-step-search"));
    if (stepID === "-") {
      return;
    }
    active.step_search[stepID] = String(target.value || "");
    persist();
    requestTerminalPaint();
  };
  container.oninput = handleTerminalStepSearch;
  container.onchange = handleTerminalStepSearch;

}

function formatDateTime(value) {
  const text = typeof value === "string" ? value.trim() : "";
  if (!text) {
    return "-";
  }
  const parsed = new Date(text);
  if (Number.isNaN(parsed.getTime())) {
    return text;
  }
  return parsed.toLocaleString(state.lang === "zh" ? "zh-CN" : "en-US", {
    hour12: false
  });
}

function formatRelativeDateTime(value) {
  const text = typeof value === "string" ? value.trim() : "";
  if (!text) {
    return "-";
  }
  const parsed = new Date(text);
  if (Number.isNaN(parsed.getTime())) {
    return text;
  }
  return formatSince(parsed.getTime());
}

function formatTaskHeartbeatSummary(lastHeartbeatAt, timeoutAt) {
  const heartbeatText = formatRelativeDateTime(lastHeartbeatAt);
  const timeoutText = formatDateTime(timeoutAt);
  if (heartbeatText !== "-" && timeoutText !== "-") {
    return `${heartbeatText} / ${timeoutText}`;
  }
  if (heartbeatText !== "-") {
    return heartbeatText;
  }
  if (timeoutText !== "-") {
    return timeoutText;
  }
  return "-";
}

function formatTaskStatus(value) {
  const status = String(value || "").trim().toLowerCase();
  if (!status) {
    return "-";
  }
  const key = `status.${status}`;
  const translated = t(key);
  return translated === key ? status : translated;
}

function taskStatusClassName(value) {
  const status = String(value || "").trim().toLowerCase();
  if (["success", "done", "ready"].includes(status)) {
    return "status-success";
  }
  if (["queued", "running", "pending", "in_progress", "busy"].includes(status)) {
    return "status-pending";
  }
  if (["failed", "error", "canceled"].includes(status)) {
    return "status-failed";
  }
  return "status-neutral";
}

function renderCopyIcon() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="12" height="12" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>`;
}

function renderPanelRightOpenIcon() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"></rect><path d="M15 3v18"></path><path d="m10 9 3 3-3 3"></path></svg>`;
}

function renderSettings2Icon() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20 7h-9"></path><path d="M14 17H4"></path><circle cx="17" cy="17" r="3"></circle><circle cx="7" cy="7" r="3"></circle></svg>`;
}

function renderChevronRightIcon() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"></path></svg>`;
}

function renderTaskEmptyIcon() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M9 6h10"></path><path d="M9 12h10"></path><path d="M9 18h7"></path><path d="M4 6h.01"></path><path d="M4 12h.01"></path><path d="M4 18h.01"></path></svg>`;
}

function renderAdvancedToggleLabel(expanded) {
  const label = expanded ? t("route.tasks.filter.advanced_hide") : t("route.tasks.filter.advanced_show");
  return `<span class="task-filter-advanced-icon" aria-hidden="true">${renderSettings2Icon()}</span><span>${escapeHTML(label)}</span>`;
}

function renderControlTaskSkeleton(count = 6) {
  return Array.from({ length: Math.max(count, 1) }).map(() => `<article class="task-summary-card task-skeleton-card" aria-hidden="true">
    <div class="task-skeleton-line task-skeleton-line-title"></div>
    <div class="task-skeleton-line task-skeleton-line-meta"></div>
    <div class="task-skeleton-line task-skeleton-line-meta"></div>
    <div class="task-skeleton-line task-skeleton-line-meta"></div>
    <div class="task-skeleton-line task-skeleton-line-button"></div>
  </article>`).join("");
}

function formatTriggerType(value) {
  const triggerType = String(value || "").trim().toLowerCase();
  if (!triggerType) {
    return "-";
  }
  const key = `trigger.${triggerType}`;
  const translated = t(key);
  return translated === key ? triggerType : translated;
}

function formatChannelType(value) {
  const channelType = String(value || "").trim().toLowerCase();
  if (!channelType) {
    return "-";
  }
  const key = `channel.${channelType}`;
  const translated = t(key);
  return translated === key ? channelType : translated;
}

function formatDurationMS(value) {
  const duration = Number(value || 0);
  if (!Number.isFinite(duration) || duration <= 0) {
    return "-";
  }
  if (duration < 1000) {
    return `${Math.round(duration)}ms`;
  }
  const seconds = duration / 1000;
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainSeconds = Math.round(seconds % 60);
  return `${minutes}m ${remainSeconds}s`;
}

function parseJSONPayload(raw) {
  if (typeof raw !== "string" || !raw.trim()) {
    return null;
  }
  try {
    return JSON.parse(raw);
  } catch (error) {
    return null;
  }
}

function renderControlTaskLogLine(item) {
  const stage = normalizeText(item?.stage || "runtime").toLowerCase();
  const message = normalizeText(item?.message || "");
  if (!message) {
    return "";
  }
  if (stage === "terminal") {
    return message;
  }
  return `[${stage}] ${message}`;
}

function renderControlTaskLogStream(logs) {
  if (!Array.isArray(logs) || !logs.length) {
    return `<div class="control-task-terminal-screen is-empty">${escapeHTML(t("route.tasks.logs.empty"))}</div>`;
  }
  const lines = logs
    .map((item) => {
      const line = renderControlTaskLogLine(item);
      const match = line.match(/^\[([a-z_]+)\]\s*(.*)$/i);
      if (!match) {
        return `<div class="control-task-log-line"><span class="control-task-log-message">${escapeHTML(line)}</span></div>`;
      }
      const rawTag = String(match[1] || "").toLowerCase();
      const message = String(match[2] || "");
      const tagClass = rawTag === "success"
        ? "is-success"
        : (rawTag === "running" ? "is-running" : (rawTag === "accept" || rawTag === "accepted" ? "is-accept" : "is-default"));
      return `<div class="control-task-log-line"><span class="control-task-log-tag ${tagClass}">[${escapeHTML(rawTag)}]</span><span class="control-task-log-message">${escapeHTML(message)}</span></div>`;
    })
    .filter(Boolean);
  if (!lines.length) {
    return `<div class="control-task-terminal-screen is-empty">${escapeHTML(t("route.tasks.logs.empty"))}</div>`;
  }
  return `<div class="control-task-terminal-screen">${lines.join("")}</div>`;
}

function renderControlTaskResultOutput(content) {
  const text = typeof content === "string" ? content.replace(/\r\n/g, "\n").trim() : "";
  if (!text) {
    return `<p>${escapeHTML("-")}</p>`;
  }
  const lines = text.split("\n").map((line) => line.trim());
  const blocks = [];
  let listItems = [];
  const flushList = () => {
    if (!listItems.length) {
      return;
    }
    blocks.push(`<ul>${listItems.join("")}</ul>`);
    listItems = [];
  };
  const renderInline = (line) => {
    const parts = String(line || "").split("**");
    return parts.map((part, index) => index % 2 === 1 ? `<strong>${escapeHTML(part)}</strong>` : escapeHTML(part)).join("");
  };
  for (const line of lines) {
    if (!line) {
      flushList();
      continue;
    }
    const bullet = line.match(/^-+\s+(.+)$/);
    if (bullet) {
      listItems.push(`<li>${renderInline(bullet[1])}</li>`);
      continue;
    }
    flushList();
    blocks.push(`<p>${renderInline(line)}</p>`);
  }
  flushList();
  return blocks.join("");
}

function escapeQueryValue(value) {
  return encodeURIComponent(String(value ?? "").trim());
}

function taskHistoryQuery(filters = {}, page = 1, pageSize = 10) {
  const params = [];
  params.push(`page=${Math.max(page, 1)}`);
  params.push(`page_size=${Math.max(pageSize, 1)}`);
  if (filters.status) {
    params.push(`status=${escapeQueryValue(filters.status)}`);
  }
  if (filters.taskType) {
    params.push(`task_type=${escapeQueryValue(filters.taskType.toLowerCase())}`);
  }
  if (filters.startAt) {
    params.push(`start_at=${escapeQueryValue(filters.startAt)}`);
  }
  if (filters.endAt) {
    params.push(`end_at=${escapeQueryValue(filters.endAt)}`);
  }
  return `/api/memory/tasks?${params.join("&")}`;
}

function taskSummaryAnchorID(taskID) {
  return `task-summary-${String(taskID || "").replace(/[^a-zA-Z0-9_-]/g, "-")}`;
}

function summarizeMemoryContent(content) {
  const text = typeof content === "string" ? content.trim() : "";
  if (!text) {
    return "-";
  }
  const rows = text
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  if (!rows.length) {
    return "-";
  }
  return shorten(rows[0], 72);
}

function splitMarkdownSections(content) {
  const text = typeof content === "string" ? content.replace(/\r\n/g, "\n") : "";
  if (!text.trim()) {
    return [];
  }
  const rows = text.split("\n");
  const sections = [];
  let current = null;
  const pushCurrent = () => {
    if (!current) {
      return;
    }
    const value = (current.content || "").trim();
    if (!current.title && !value) {
      return;
    }
    sections.push({
      title: current.title || t("route.memory.spec.section.default"),
      content: current.content || ""
    });
  };

  for (const row of rows) {
    const heading = row.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      pushCurrent();
      current = {
        title: heading[2].trim(),
        content: ""
      };
      continue;
    }

    if (!current) {
      current = {
        title: t("route.memory.spec.section.default"),
        content: ""
      };
    }
    current.content += `${row}\n`;
  }
  pushCurrent();
  return sections;
}

function renderMemorySpecificationBody(content) {
  const text = typeof content === "string" ? content.trim() : "";
  if (!text) {
    return "";
  }

  const sections = splitMarkdownSections(content);
  if (!sections.length) {
    return `<pre class="memory-content">${escapeHTML(content)}</pre>`;
  }

  return `<div class="memory-spec-sections">
    ${sections.map((section) => `<section class="memory-spec-section">
      <h5 class="memory-spec-title">${escapeHTML(section.title)}</h5>
      <pre class="memory-content">${escapeHTML(section.content.trim())}</pre>
    </section>`).join("")}
  </div>`;
}

function renderMemoryDocumentCard(title, type, payload, emptyKey) {
  const path = typeof payload?.path === "string" ? payload.path : "";
  const updatedAt = typeof payload?.updated_at === "string" ? payload.updated_at : "";
  const content = typeof payload?.content === "string" ? payload.content : "";
  const error = typeof payload?.error === "string" ? payload.error : "";
  const fields = [
    routeFieldRow("field.path", path || "-", { copyable: true, multiline: true, mono: true }),
    routeFieldRow("field.updated", formatDateTime(updatedAt)),
    routeFieldRow("field.read_only", t("route.memory.read_only"))
  ];

  let body = `<p class="route-empty">${t(emptyKey)}</p>`;
  if (error) {
    body = `<p class="route-error">${t("load_failed", { error })}</p>`;
  } else if (payload?.exists && content.trim()) {
    body = `<pre class="memory-content">${escapeHTML(content)}</pre>`;
  }

  return routeCardTemplate(title, type, fields, true, body);
}

function renderMemorySpecificationCard(payload) {
  const path = typeof payload?.path === "string" ? payload.path : "";
  const updatedAt = typeof payload?.updated_at === "string" ? payload.updated_at : "";
  const content = typeof payload?.content === "string" ? payload.content : "";
  const error = typeof payload?.error === "string" ? payload.error : "";
  const fields = [
    routeFieldRow("field.path", path || "-", { copyable: true, multiline: true, mono: true }),
    routeFieldRow("field.updated", formatDateTime(updatedAt)),
    routeFieldRow("field.read_only", t("route.memory.read_only"))
  ];

  let body = `<p class="route-empty">${t("route.memory.empty.specification")}</p>`;
  if (error) {
    body = `<p class="route-error">${t("load_failed", { error })}</p>`;
  } else if (payload?.exists && content.trim()) {
    body = renderMemorySpecificationBody(content);
  }

  return routeCardTemplate(
    t("route.memory.tab.specification"),
    "memory",
    fields,
    true,
    body
  );
}

function renderDailyMemoryCards(payload) {
  if (typeof payload?.error === "string" && payload.error.trim()) {
    return `<p class="route-error">${t("load_failed", { error: payload.error })}</p>`;
  }
  const items = Array.isArray(payload?.items) ? payload.items : [];
  if (!items.length) {
    return `<p class="route-empty">${t("route.memory.empty.daily")}</p>`;
  }
  return items.map((item) => {
    const date = typeof item?.date === "string" ? item.date : "-";
    const path = typeof item?.path === "string" ? item.path : "";
    const updatedAt = typeof item?.updated_at === "string" ? item.updated_at : "";
    const content = typeof item?.content === "string" ? item.content : "";
    const error = typeof item?.error === "string" ? item.error : "";
    const fields = [
      routeFieldRow("field.date", date),
      routeFieldRow("field.path", path || "-", { copyable: true, multiline: true, mono: true }),
      routeFieldRow("field.updated", formatDateTime(updatedAt)),
      routeFieldRow("field.read_only", t("route.memory.read_only"))
    ];
    let body = `<p class="route-empty">${t("route.memory.empty.daily")}</p>`;
    if (error) {
      body = `<p class="route-error">${t("load_failed", { error })}</p>`;
    } else if (content.trim()) {
      body = `<p class="memory-summary"><span>${t("route.memory.daily.summary")}</span><strong>${escapeHTML(summarizeMemoryContent(content))}</strong></p>
<pre class="memory-content">${escapeHTML(content)}</pre>`;
    }
    return routeCardTemplate(date, "memory", fields, true, body);
  }).join("");
}

function renderTaskSummaryMetaRow(labelKey, value, options = {}) {
  const safeValue = normalizeText(value);
  const classNames = ["task-summary-value"];
  if (options?.mono) {
    classNames.push("is-mono");
  }
  return `<p><span>${t(labelKey)}</span><strong class="${classNames.join(" ")}" title="${escapeHTML(safeValue)}">${escapeHTML(safeValue)}</strong></p>`;
}

function renderTaskSummaryPreview(labelKey, value, options = {}) {
  const safeValue = normalizeText(value);
  const lineClamp = Math.max(2, Number(options?.lineClamp || (options?.log ? 4 : 3)));
  const classNames = ["task-summary-preview"];
  if (options?.log) {
    classNames.push("is-log");
  }
  return `<div class="task-summary-meta-block">
    <span>${t(labelKey)}</span>
    <div class="${classNames.join(" ")}" style="--line-clamp:${lineClamp}" title="${escapeHTML(safeValue)}">${escapeHTML(safeValue)}</div>
  </div>`;
}

function renderTaskSummaryCards(payload, activeTaskID = "") {
  const items = Array.isArray(payload?.items) ? payload.items : [];
  if (!items.length) {
    return `<p class="route-empty">${t("route.memory.tasks.empty")}</p>`;
  }
  return items.map((item) => {
    const taskID = typeof item?.task_id === "string" ? item.task_id : "-";
    const taskType = typeof item?.task_type === "string" ? item.task_type : "-";
    const goal = typeof item?.goal === "string" ? item.goal : "-";
    const result = typeof item?.result === "string" ? item.result : "-";
    const status = typeof item?.status === "string" ? item.status : "";
    const finishedAt = typeof item?.finished_at === "string" ? item.finished_at : "";
    const updatedAt = typeof item?.updated_at === "string" ? item.updated_at : "";
    const lastHeartbeatAt = typeof item?.last_heartbeat_at === "string" ? item.last_heartbeat_at : "";
    const timeoutAt = typeof item?.timeout_at === "string" ? item.timeout_at : "";
    const tags = Array.isArray(item?.tags) ? item.tags : [];
    const anchorID = taskSummaryAnchorID(taskID);
    const active = taskID && taskID === activeTaskID;
    const statusClassName = taskStatusClassName(status);
    const heartbeatRow = formatTaskHeartbeatSummary(lastHeartbeatAt, timeoutAt) !== "-"
      ? renderTaskSummaryMetaRow("field.last_heartbeat_at", formatTaskHeartbeatSummary(lastHeartbeatAt, timeoutAt))
      : "";
    return `<article class="route-card task-summary-card ${active ? "active" : ""}" id="${escapeHTML(anchorID)}" data-task-summary-id="${escapeHTML(taskID)}" ${active ? 'aria-current="true"' : ""}>
      <header class="task-summary-head">
        <div class="task-summary-id-wrap">
          <h5 class="task-summary-id" title="${escapeHTML(taskID)}">${escapeHTML(taskID)}</h5>
        </div>
        <span class="task-summary-status ${statusClassName}">${escapeHTML(formatTaskStatus(status))}</span>
      </header>
      <div class="task-summary-meta">
        ${renderTaskSummaryMetaRow("field.task_type", taskType)}
        ${renderTaskSummaryPreview("field.goal", goal, { lineClamp: 3 })}
        ${renderTaskSummaryPreview("field.result", result, { lineClamp: 4, log: true })}
        ${renderTaskSummaryMetaRow("field.updated", formatDateTime(updatedAt))}
        ${heartbeatRow}
        ${renderTaskSummaryMetaRow("field.finished", formatDateTime(finishedAt))}
      </div>
      <footer class="route-card-footer">
        ${renderRouteTagSection("field.tags", tags)}
        <button class="task-summary-open" type="button" data-task-open="${escapeHTML(taskID)}"><span class="task-summary-open-icon" aria-hidden="true">${renderPanelRightOpenIcon()}</span><span>${t("route.memory.tasks.open_detail")}</span></button>
      </footer>
    </article>`;
  }).join("");
}

function renderTaskSummaryPagination(payload) {
  const pagination = payload?.pagination || {};
  const hasNext = Boolean(pagination?.has_next);
  const page = Number(pagination?.page || 1);
  const total = Number(pagination?.total || 0);
  return `<div class="task-summary-pagination">
    <p><span>${t("field.messages")}</span><strong>${escapeHTML(total)}</strong></p>
    <p><span>${t("route.memory.tasks.page.label")}</span><strong>${escapeHTML(page)}</strong></p>
    <button class="task-summary-next" type="button" data-task-page-next ${hasNext ? "" : "disabled"}>${t("route.memory.tasks.page.next")}</button>
  </div>`;
}

function renderTaskDetail(meta, refs) {
  const taskID = typeof meta?.task_id === "string" ? meta.task_id : "-";
  const status = typeof meta?.status === "string" ? meta.status : "";
  const taskType = typeof meta?.task_type === "string" ? meta.task_type : "-";
  const statusClassName = taskStatusClassName(status);
  const refsList = Array.isArray(refs) ? refs : [];
  const refsBody = refsList.length
    ? `<ul class="task-detail-refs">
      ${refsList.map((item) => `<li><strong>${escapeHTML(normalizeText(item.tier))}</strong><span>${escapeHTML(normalizeText(item.date))}</span><code>${escapeHTML(normalizeText(item.path))}</code></li>`).join("")}
    </ul>`
    : `<p class="route-empty">-</p>`;
  return `<section class="task-detail-card" data-task-detail-id="${escapeHTML(taskID)}">
    <header class="task-detail-head">
      <div class="task-detail-id-wrap">
        <h5 class="task-summary-id" title="${escapeHTML(taskID)}">${escapeHTML(taskID)}</h5>
      </div>
      <span class="task-summary-status ${statusClassName}">${escapeHTML(formatTaskStatus(status))}</span>
    </header>
    <div class="task-detail-meta route-meta">
      ${routeFieldRow("field.task_type", taskType)}
      ${routeFieldRow("field.session", meta?.session_id, { copyable: true, mono: true })}
      ${routeFieldRow("field.source_message", meta?.source_message_id, { copyable: true, mono: true })}
      ${routeFieldRow("field.progress", meta?.progress)}
      ${routeFieldRow("field.retry_count", meta?.retry_count)}
      ${routeFieldRow("field.created", formatDateTime(meta?.created_at))}
      ${routeFieldRow("field.updated", formatDateTime(meta?.updated_at))}
      ${routeFieldRow("field.last_heartbeat_at", formatDateTime(meta?.last_heartbeat_at))}
      ${routeFieldRow("field.timeout_at", formatDateTime(meta?.timeout_at))}
      ${routeFieldRow("field.finished_at", formatDateTime(meta?.finished_at))}
    </div>
    ${renderRouteSection("Summary Refs", `
      ${refsBody}
    `, { className: "task-detail-section" })}
    <div class="task-detail-actions">
      <button type="button" data-task-load-logs>${t("route.memory.tasks.logs.load")}</button>
      <button type="button" data-task-load-artifacts>${t("route.memory.tasks.artifacts.load")}</button>
      <button type="button" data-task-rebuild>${t("route.memory.tasks.rebuild")}</button>
      <button type="button" data-task-back>${t("route.memory.tasks.back")}</button>
    </div>
    ${renderRouteSection("Logs", `
      <div class="task-detail-logs" data-task-logs>${t("route.memory.tasks.logs.empty")}</div>
    `, { className: "task-detail-section" })}
    ${renderRouteSection("Artifacts", `
      <div class="task-detail-artifacts" data-task-artifacts>${t("route.memory.tasks.artifacts.empty")}</div>
    `, { className: "task-detail-section" })}
  </section>`;
}

function renderTaskLogs(payload) {
  if (payload?.error_code) {
    return `<p class="route-error">${escapeHTML(t("route.memory.tasks.logs_hint"))}</p>`;
  }
  const items = Array.isArray(payload?.items) ? payload.items : [];
  if (!items.length) {
    return `<p class="route-empty">${t("route.memory.tasks.logs.empty")}</p>`;
  }
  const list = `<ul class="task-detail-log-list">
    ${items.map((item) => `<li class="task-detail-list-item">
      <div class="task-detail-list-head">
        <strong>#${escapeHTML(normalizeText(item.seq))}</strong>
        <span>${escapeHTML(normalizeText(item.stage))}</span>
        <span>${escapeHTML(normalizeText(formatTaskStatus(item.level)))}</span>
        <span>${escapeHTML(formatDateTime(item.created_at || item.timestamp))}</span>
      </div>
      <pre class="task-detail-list-content">${escapeHTML(normalizeText(item.message))}</pre>
    </li>`).join("")}
  </ul>`;
  if (!payload?.has_more) {
    return list;
  }
  return `${list}<button type="button" class="task-summary-next" data-task-load-more-logs>${t("route.memory.tasks.logs.more")}</button>`;
}

function renderTaskArtifacts(payload) {
  if (typeof payload?.error === "string" && payload.error.trim()) {
    return `<p class="route-error">${t("load_failed", { error: payload.error })}</p>`;
  }
  const items = Array.isArray(payload?.items) ? payload.items : [];
  if (!items.length) {
    return `<p class="route-empty">${t("route.memory.tasks.artifacts.empty")}</p>`;
  }
  return `<ul class="task-detail-artifact-list">
    ${items.map((item) => `<li class="task-detail-list-item">
      <div class="task-detail-list-head">
        <strong>${escapeHTML(normalizeText(item.artifact_type || item.name))}</strong>
        <span>${escapeHTML(formatDateTime(item.created_at))}</span>
      </div>
      <p class="task-detail-list-content">${escapeHTML(normalizeText(item.summary || item.content_type))}</p>
      <p class="task-artifact-actions">
        ${item.download_url
    ? `<button type="button" data-task-artifact-download="${escapeHTML(item.download_url)}" data-task-artifact-name="${escapeHTML(normalizeText(item.name || item.artifact_id || "artifact.bin"))}">${t("route.memory.tasks.artifacts.download")}</button>`
    : ""}
        ${item.preview_url
    ? `<button type="button" data-task-artifact-preview="${escapeHTML(item.preview_url)}">${t("route.memory.tasks.artifacts.preview")}</button>`
    : ""}
      </p>
    </li>`).join("")}
  </ul>`;
}

function parseDateTimeFilter(value) {
  const text = String(value || "").trim();
  if (!text) {
    return "";
  }
  const parsed = new Date(text);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  return parsed.toISOString();
}

function bindTaskHistoryView(container, initialPayload) {
  const view = container.querySelector("[data-task-history-view]");
  if (!view) {
    return;
  }
  const listNode = view.querySelector("[data-task-summary-list]");
  const paginationNode = view.querySelector("[data-task-summary-pagination]");
  const detailNode = view.querySelector("[data-task-detail]");
  const detailWrap = view.querySelector(".task-detail-wrap");
  const form = view.querySelector("[data-task-filter-form]");
  const state = {
    filters: { status: "", taskType: "", startAt: "", endAt: "" },
    page: 1,
    pageSize: 10,
    activeTaskID: "",
    nextLogCursor: 0
  };

  const syncDetailState = (open) => {
    view.classList.toggle("is-detail-open", open);
    if (detailWrap) {
      detailWrap.hidden = !open;
    }
    if (!open && detailNode) {
      detailNode.innerHTML = t("route.memory.tasks.detail.empty");
    }
  };

  const syncActiveTaskCards = () => {
    const cards = view.querySelectorAll("[data-task-summary-id]");
    cards.forEach((card) => {
      const taskID = normalizeText(card.getAttribute("data-task-summary-id") || "");
      const active = Boolean(state.activeTaskID) && taskID === state.activeTaskID;
      card.classList.toggle("active", active);
      if (active) {
        card.setAttribute("aria-current", "true");
      } else {
        card.removeAttribute("aria-current");
      }
    });
  };

  const paintList = (payload) => {
    listNode.innerHTML = renderTaskSummaryCards(payload, state.activeTaskID);
    paginationNode.innerHTML = renderTaskSummaryPagination(payload);
    syncActiveTaskCards();
  };

  const loadList = async () => {
    const payload = await fetchJSON(taskHistoryQuery(state.filters, state.page, state.pageSize));
    paintList(payload);
  };

  const loadDetail = async (taskID) => {
    state.activeTaskID = taskID;
    state.nextLogCursor = 0;
    const payload = await fetchJSON(`/api/memory/tasks/${encodeURIComponent(taskID)}`);
    detailNode.innerHTML = renderTaskDetail(payload?.meta, payload?.summary_refs);
    syncDetailState(true);
    syncActiveTaskCards();
  };

  const loadLogs = async (append = false) => {
    if (!state.activeTaskID) {
      return;
    }
    const cursor = append ? state.nextLogCursor : 0;
    const payload = await fetchJSON(`/api/memory/tasks/${encodeURIComponent(state.activeTaskID)}/logs?cursor=${cursor}&limit=20`);
    const target = detailNode.querySelector("[data-task-logs]");
    if (!target) {
      return;
    }
    if (!append) {
      target.innerHTML = renderTaskLogs(payload);
    } else {
      const current = target.querySelector(".task-detail-log-list");
      const incoming = Array.isArray(payload?.items) ? payload.items : [];
      if (current && incoming.length) {
        const fragment = incoming.map((item) => `<li>
          <p><strong>#${escapeHTML(normalizeText(item.seq))}</strong><span>${escapeHTML(normalizeText(item.stage))}</span><span>${escapeHTML(normalizeText(formatTaskStatus(item.level)))}</span><span>${escapeHTML(formatDateTime(item.created_at || item.timestamp))}</span></p>
          <pre>${escapeHTML(normalizeText(item.message))}</pre>
        </li>`).join("");
        current.insertAdjacentHTML("beforeend", fragment);
      } else {
        target.innerHTML = renderTaskLogs(payload);
      }
      const moreButton = target.querySelector("[data-task-load-more-logs]");
      if (moreButton && !payload?.has_more) {
        moreButton.remove();
      }
    }
    state.nextLogCursor = Number(payload?.next_cursor || 0);
  };

  const loadArtifacts = async () => {
    if (!state.activeTaskID) {
      return;
    }
    const payload = await fetchJSON(`/api/memory/tasks/${encodeURIComponent(state.activeTaskID)}/artifacts`);
    const target = detailNode.querySelector("[data-task-artifacts]");
    if (!target) {
      return;
    }
    target.innerHTML = renderTaskArtifacts(payload);
  };

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const formData = new FormData(form);
    state.filters.status = String(formData.get("status") || "").trim();
    state.filters.taskType = String(formData.get("task_type") || "").trim();
    state.filters.startAt = parseDateTimeFilter(formData.get("start_at"));
    state.filters.endAt = parseDateTimeFilter(formData.get("end_at"));
    state.page = 1;
    await loadList();
  });

  const resetButton = form.querySelector("[data-task-filter-reset]");
  if (resetButton) {
    resetButton.addEventListener("click", async () => {
      form.reset();
      state.filters = { status: "", taskType: "", startAt: "", endAt: "" };
      state.page = 1;
      await loadList();
    });
  }

  view.addEventListener("click", async (event) => {
    const target = event.target.closest("button");
    if (!target) {
      return;
    }
    if (target.hasAttribute("data-task-open")) {
      await loadDetail(target.getAttribute("data-task-open") || "");
      return;
    }
    if (target.hasAttribute("data-task-page-next")) {
      if (target.disabled) {
        return;
      }
      state.page += 1;
      await loadList();
      return;
    }
    if (target.hasAttribute("data-task-load-logs")) {
      await loadLogs(false);
      return;
    }
    if (target.hasAttribute("data-task-load-more-logs")) {
      await loadLogs(true);
      return;
    }
    if (target.hasAttribute("data-task-load-artifacts")) {
      await loadArtifacts();
      return;
    }
    if (target.hasAttribute("data-task-artifact-download")) {
      const downloadURL = target.getAttribute("data-task-artifact-download") || "";
      const artifactName = target.getAttribute("data-task-artifact-name") || "artifact.bin";
      if (!downloadURL) {
        return;
      }
      try {
        await downloadTaskArtifact(downloadURL, artifactName);
      } catch (err) {
        const message = err instanceof Error ? err.message : "unknown_error";
        alert(t("route.memory.tasks.artifacts.download_fail", { error: message }));
      }
      return;
    }
    if (target.hasAttribute("data-task-artifact-preview")) {
      const previewURL = target.getAttribute("data-task-artifact-preview") || "";
      if (!previewURL) {
        return;
      }
      window.open(previewURL, "_blank", "noopener,noreferrer");
      return;
    }
    if (target.hasAttribute("data-task-rebuild")) {
      if (!state.activeTaskID) {
        return;
      }
      try {
        await fetch(`/api/memory/tasks/${encodeURIComponent(state.activeTaskID)}/rebuild-summary`, { method: "POST" });
        await loadDetail(state.activeTaskID);
      } catch (err) {
        const message = err instanceof Error ? err.message : "unknown_error";
        alert(t("route.memory.tasks.rebuild_fail", { error: message }));
      }
      return;
    }
    if (target.hasAttribute("data-task-back")) {
      const previousTaskID = state.activeTaskID;
      state.activeTaskID = "";
      state.nextLogCursor = 0;
      syncDetailState(false);
      syncActiveTaskCards();
      const anchor = document.getElementById(taskSummaryAnchorID(previousTaskID));
      if (anchor) {
        anchor.scrollIntoView({ behavior: "smooth", block: "center" });
      }
      return;
    }
  });

  syncDetailState(false);
  paintList(initialPayload || { items: [], pagination: { page: 1, total: 0, has_next: false } });
}

function bindMemoryTabSwitch(container) {
  const tabs = container.querySelectorAll("[data-memory-tab]");
  const panels = container.querySelectorAll("[data-memory-panel]");
  const activate = (tabName) => {
    tabs.forEach((tab) => {
      const active = tab.dataset.memoryTab === tabName;
      tab.classList.toggle("active", active);
      tab.setAttribute("aria-selected", active ? "true" : "false");
    });
    panels.forEach((panel) => {
      const active = panel.dataset.memoryPanel === tabName;
      panel.classList.toggle("active", active);
      panel.hidden = !active;
    });
  };
  tabs.forEach((tab) => {
    tab.addEventListener("click", () => activate(tab.dataset.memoryTab || "long_term"));
  });
  const hasTasks = Array.from(tabs).some((tab) => tab.dataset.memoryTab === "tasks");
  activate(hasTasks ? "tasks" : "long_term");
}

async function loadMemoryView(container) {
  const [payload, taskPayload] = await Promise.all([
    fetchJSON("/api/agent/memory"),
    fetchJSON(taskHistoryQuery({}, 1, 10))
  ]);
  const tabs = [
    { id: "tasks", label: t("route.memory.tab.tasks") },
    { id: "long_term", label: t("route.memory.tab.long_term") },
    { id: "daily", label: t("route.memory.tab.daily") },
    { id: "mandatory", label: t("route.memory.tab.mandatory") },
    { id: "specification", label: t("route.memory.tab.specification") }
  ];
  const dailySourceDir = typeof payload?.daily?.directory === "string"
    ? payload.daily.directory
    : "-";

  container.innerHTML = `<section class="memory-view">
    <div class="memory-tabs" role="tablist" aria-label="${t("route.memory.title")}">
      ${tabs.map((tab) => `<button class="memory-tab" type="button" role="tab" data-memory-tab="${tab.id}" aria-selected="false">${escapeHTML(tab.label)}</button>`).join("")}
    </div>
    <section class="memory-panel memory-panel-tasks" data-memory-panel="tasks" hidden>
      <section class="task-history-view" data-task-history-view>
        <form class="task-filter-form page-filter-form page-filter-grid-4" data-task-filter-form>
          <label><span>${t("route.memory.tasks.filter.status")}</span>
            <select name="status">
              <option value="">-</option>
              <option value="queued">${t("status.queued")}</option>
              <option value="running">${t("status.running")}</option>
              <option value="success">${t("status.success")}</option>
              <option value="failed">${t("status.failed")}</option>
              <option value="canceled">${t("status.canceled")}</option>
            </select>
          </label>
          <label><span>${t("route.memory.tasks.filter.task_type")}</span><input type="text" name="task_type" placeholder="release"></label>
          <label><span>${t("route.memory.tasks.filter.start_at")}</span><input type="datetime-local" name="start_at"></label>
          <label><span>${t("route.memory.tasks.filter.end_at")}</span><input type="datetime-local" name="end_at"></label>
          <div class="task-filter-actions">
            <button type="submit">${t("route.memory.tasks.filter.apply")}</button>
            <button type="button" data-task-filter-reset>${t("route.memory.tasks.filter.reset")}</button>
          </div>
        </form>
        <div class="task-summary-list" data-task-summary-list></div>
        <div class="task-summary-pagination-wrap" data-task-summary-pagination></div>
        <section class="task-detail-wrap">
          <h4>${t("route.memory.tasks.detail.title")}</h4>
          <div class="task-detail-body" data-task-detail>${t("route.memory.tasks.detail.empty")}</div>
        </section>
      </section>
    </section>
    <section class="memory-panel" data-memory-panel="long_term" hidden>
      ${renderMemoryDocumentCard(
        t("route.memory.tab.long_term"),
        "memory",
        payload?.long_term,
        "route.memory.empty.long_term"
      )}
    </section>
    <section class="memory-panel" data-memory-panel="daily" hidden>
      ${routeCardTemplate(
        t("route.memory.tab.daily"),
        "memory",
        [routeFieldRow("route.memory.daily.source", dailySourceDir, { copyable: true, multiline: true, mono: true })],
        true
      )}
      <div class="memory-daily-list">${renderDailyMemoryCards(payload?.daily)}</div>
    </section>
    <section class="memory-panel" data-memory-panel="mandatory" hidden>
      ${renderMemoryDocumentCard(
        t("route.memory.tab.mandatory"),
        "memory",
        payload?.mandatory,
        "route.memory.empty.mandatory"
      )}
    </section>
    <section class="memory-panel memory-panel-spec" data-memory-panel="specification" hidden>
      ${renderMemorySpecificationCard(payload?.specification)}
    </section>
  </section>`;
  bindMemoryTabSwitch(container);
  bindTaskHistoryView(container, taskPayload);
}

function formatEnvironmentApplyMode(mode) {
  const normalized = String(mode || "").trim().toLowerCase();
  if (normalized === "immediate") {
    return t("route.envs.apply.immediate");
  }
  return t("route.envs.apply.restart");
}

function formatEnvironmentSource(source) {
  const normalized = String(source || "").trim().toLowerCase();
  if (normalized === "persisted") {
    return t("route.envs.source.persisted");
  }
  if (normalized === "runtime") {
    return t("route.envs.source.runtime");
  }
  return t("route.envs.source.default");
}

function formatEnvironmentValueType(type) {
  const normalized = String(type || "").trim().toLowerCase();
  if (normalized === "integer") {
    return t("route.envs.type.integer");
  }
  if (normalized === "duration") {
    return t("route.envs.type.duration");
  }
  if (normalized === "string") {
    return t("route.envs.type.string");
  }
  if (normalized === "enum") {
    return t("route.envs.type.enum");
  }
  return t("route.envs.type.unknown");
}

function renderEnvironmentValidation(definition) {
  const validation = definition?.validation || {};
  const parts = [];
  if (validation?.required) {
    parts.push("required");
  }
  if (typeof validation?.min === "string" && validation.min.trim()) {
    parts.push(`min=${validation.min.trim()}`);
  }
  if (typeof validation?.max === "string" && validation.max.trim()) {
    parts.push(`max=${validation.max.trim()}`);
  }
  const allowed = Array.isArray(validation?.allowed) ? validation.allowed.filter((item) => String(item || "").trim()) : [];
  if (allowed.length) {
    parts.push(`allowed=${allowed.join("|")}`);
  }
  if (!parts.length) {
    return t("route.envs.validation.none");
  }
  return parts.join(", ");
}

function renderEnvironmentInput(item) {
  const definition = item?.definition || {};
  const key = normalizeText(definition?.key || "");
  const type = normalizeText(definition?.type || "string").toLowerCase();
  const value = normalizeText(item?.value || "");
  const masked = Boolean(item?.masked);
  const validation = definition?.validation || {};
  const requiredAttr = validation?.required ? "required" : "";
  const disabledAttr = masked ? "disabled" : "";
  const originalValue = escapeHTML(value);
  const baseAttrs = `name="${escapeHTML(key)}" data-env-input data-env-key="${escapeHTML(key)}" data-original="${originalValue}" ${requiredAttr} ${disabledAttr}`;

  if (type === "integer") {
    const minAttr = typeof validation?.min === "string" && validation.min.trim()
      ? `min="${escapeHTML(validation.min.trim())}"`
      : "";
    const maxAttr = typeof validation?.max === "string" && validation.max.trim()
      ? `max="${escapeHTML(validation.max.trim())}"`
      : "";
    return `<input type="number" ${baseAttrs} value="${originalValue}" ${minAttr} ${maxAttr}>`;
  }

  if (type === "enum") {
    const allowed = Array.isArray(validation?.allowed) ? validation.allowed.filter((itemValue) => String(itemValue || "").trim()) : [];
    if (!allowed.includes(value)) {
      allowed.unshift(value);
    }
    return `<select ${baseAttrs}>
      ${allowed.map((option) => `<option value="${escapeHTML(option)}" ${option === value ? "selected" : ""}>${escapeHTML(option)}</option>`).join("")}
    </select>`;
  }

  const placeholderAttr = masked ? `placeholder="${escapeHTML(t("route.envs.hidden"))}"` : "";
  return `<input type="${masked ? "password" : "text"}" ${baseAttrs} value="${masked ? "" : originalValue}" ${placeholderAttr}>`;
}

function renderEnvironmentItem(item) {
  const definition = item?.definition || {};
  const key = normalizeText(definition?.key || "-");
  const name = normalizeText(definition?.name || key);
  const description = normalizeText(definition?.description || "-");
  const currentValue = normalizeText(item?.value || "");
  const effectiveValue = normalizeText(item?.effective_value || "");
  const defaultValue = normalizeText(definition?.default_value || "");
  const pendingRestart = Boolean(item?.pending_restart);
  const hotReload = Boolean(definition?.hot_reload);
  const valueType = formatEnvironmentValueType(definition?.type);
  const applyMode = formatEnvironmentApplyMode(definition?.apply_mode);
  const source = formatEnvironmentSource(item?.value_source);
  const validation = renderEnvironmentValidation(definition);
  const inputControl = renderEnvironmentInput(item);
  const pendingBadge = pendingRestart
    ? `<span class="environment-pending">${t("route.envs.pending_restart")}</span>`
    : "";
  const titleMetaHTML = `<span class="environment-item-key">
    <code title="${escapeHTML(key)}">${escapeHTML(key)}</code>
    <button class="route-field-copy" type="button" data-copy-value="${escapeHTML(key)}" title="${escapeHTML(t("route.copy_value"))}" aria-label="${escapeHTML(t("route.copy_value"))}">${renderCopyIcon()}</button>
  </span>`;
  const detailFields = [
      routeFieldRow("field.description", description, { multiline: true, preview: true, clampLines: 8 }),
      routeFieldRow("route.envs.value_type", valueType),
      routeFieldRow("route.envs.default_value", defaultValue || "-", { multiline: true, mono: true, copyable: defaultValue && defaultValue !== "-" }),
      routeFieldRow("route.envs.effective_value", effectiveValue || "-", { multiline: true, mono: true, copyable: effectiveValue && effectiveValue !== "-" }),
      routeFieldRow("route.envs.apply_mode", applyMode),
      routeFieldRow("route.envs.source", source),
      routeFieldRow("route.envs.validation", validation, { multiline: true, mono: true }),
      routeFieldRow("route.envs.hot_reload", hotReload ? t("status.enabled") : t("status.disabled"))
    ];
  const body = `<section class="environment-summary">
      <div class="environment-description">
        <span>${t("field.description")}</span>
        <p class="environment-description-text" title="${escapeHTML(description)}">${escapeHTML(description)}</p>
      </div>
      <label class="environment-input-row">
      <span>${t("route.envs.current_value")}</span>
      ${inputControl}
    </label>
      <details class="environment-details">
        <summary>${t("route.envs.details")}</summary>
        <div class="environment-details-body">
          ${detailFields.join("")}
        </div>
      </details>
    </section>`;
  const footer = `${renderRouteTagSection("field.tags", [applyMode, source, hotReload ? t("status.enabled") : t("status.disabled")])}${pendingRestart ? `<p class="environment-item-notice">${escapeHTML(t("route.envs.restart_notice", { keys: key }))}</p>` : ""}`;
  return routeCardTemplate(
    name,
    "env",
    [],
    true,
    body,
    {
      className: "environment-item",
      titleMetaHTML,
      badgeHTML: pendingBadge,
      footer,
      bodyClassName: "environment-card-body",
      footerClassName: "route-card-footer-spread"
    }
  );
}

function renderEnvironmentModules(items) {
  const safeItems = Array.isArray(items) ? items : [];
  if (!safeItems.length) {
    return `<p class="route-empty">${t("route.envs.loading")}</p>`;
  }
  const modules = [];
  const moduleMap = new Map();
  safeItems.forEach((item) => {
    const moduleName = normalizeText(item?.definition?.module || "General");
    if (!moduleMap.has(moduleName)) {
      moduleMap.set(moduleName, []);
      modules.push(moduleName);
    }
    moduleMap.get(moduleName).push(item);
  });
  return modules.map((moduleName) => {
    const moduleItems = moduleMap.get(moduleName) || [];
    return `<section class="environment-module">
      <h4>${escapeHTML(moduleName)}</h4>
      <div class="environment-module-grid">
        ${moduleItems.map((item) => renderEnvironmentItem(item)).join("")}
      </div>
    </section>`;
  }).join("");
}

function renderEnvironmentAudits(items) {
  const safeItems = Array.isArray(items) ? items : [];
  if (!safeItems.length) {
    return `<p class="route-empty">${t("route.envs.audit.empty")}</p>`;
  }
  return `<div class="environment-audit-list">
    ${safeItems.map((item) => {
      const changes = Array.isArray(item?.changes) ? item.changes : [];
      const changesBody = changes.length
        ? `<ul>${changes.map((change) => `<li>${escapeHTML(t("route.envs.audit.change", {
          key: normalizeText(change?.key || "-"),
          old: normalizeText(change?.old_value || "-"),
          new: normalizeText(change?.new_value || "-"),
          mode: formatEnvironmentApplyMode(change?.apply_mode)
        }))}</li>`).join("")}</ul>`
        : `<p>-</p>`;
      return routeCardTemplate(
        normalizeText(item?.operator || t("route.envs.audit.title")),
        "env",
        [
          routeFieldRow("route.envs.audit.operator", normalizeText(item?.operator || "-")),
          routeFieldRow("route.envs.audit.at", formatDateTime(item?.occurred_at)),
          routeFieldRow("route.envs.audit.requires_restart", item?.requires_restart ? t("status.enabled") : t("status.disabled"))
        ],
        true,
        changesBody,
        {
          className: "environment-audit-item",
          footer: renderRouteTagSection("field.tags", [item?.requires_restart ? t("status.enabled") : t("status.disabled")])
        }
      );
    }).join("")}
  </div>`;
}

function formatCommitHash(value) {
  const text = normalizeText(value || "");
  if (!text) {
    return "-";
  }
  if (text.length <= 12) {
    return text;
  }
  return `${text.slice(0, 12)}...`;
}

function renderEnvironmentRuntimeMeta(runtimeInfo) {
  const commitHash = normalizeText(runtimeInfo?.commit_hash || "");
  const commitDisplay = formatCommitHash(commitHash);
  return `<div class="environment-runtime-meta" data-environment-runtime-meta>
    <div class="environment-runtime-meta-item">
      <span>${escapeHTML(t("route.envs.runtime.last_restart_at"))}</span>
      <strong>${escapeHTML(formatDateTime(runtimeInfo?.started_at))}</strong>
    </div>
    <div class="environment-runtime-meta-item">
      <span>${escapeHTML(t("route.envs.runtime.commit_hash"))}</span>
      <strong><code title="${escapeHTML(commitHash || "-")}">${escapeHTML(commitDisplay)}</code></strong>
    </div>
  </div>`;
}

async function loadEnvironmentsView(container) {
  const localState = {
    revealSensitive: false,
    restarting: false,
    configItems: [],
    audits: [],
    runtimeInfo: null
  };

  const fetchEnvironments = async () => {
    const query = localState.revealSensitive ? "?reveal_sensitive=true" : "";
    const [configPayload, auditPayload, runtimePayload] = await Promise.all([
      fetchJSON(`/api/control/environments${query}`),
      fetchJSON(`/api/control/environments/audits${query}`),
      fetchJSON("/api/control/runtime").catch(() => null)
    ]);
    return {
      configItems: Array.isArray(configPayload?.items) ? configPayload.items : [],
      audits: Array.isArray(auditPayload?.items) ? auditPayload.items : [],
      runtimeInfo: runtimePayload
    };
  };

  const paint = (configItems, audits, runtimeInfo, statusMessage = "") => {
    const revealButtonLabel = localState.revealSensitive ? t("route.envs.hide_sensitive") : t("route.envs.show_sensitive");
    const restartButtonLabel = localState.restarting ? t("route.envs.restarting") : t("route.envs.restart_service");
    container.innerHTML = `<section class="environment-view" data-environment-view>
      <form class="environment-form" data-environment-form>
        <div class="environment-toolbar route-card">
          <div class="environment-toolbar-main">
            <p class="environment-status" data-environment-status>${escapeHTML(statusMessage)}</p>
            ${renderEnvironmentRuntimeMeta(runtimeInfo)}
          </div>
          <div class="task-filter-actions">
            <button type="button" data-environment-reveal>${escapeHTML(revealButtonLabel)}</button>
            <button type="button" data-environment-refresh>${t("route.envs.refresh")}</button>
            <button type="button" data-environment-restart ${localState.restarting ? "disabled" : ""}>${escapeHTML(restartButtonLabel)}</button>
            <button type="submit">${t("route.envs.save")}</button>
          </div>
        </div>
        <div class="environment-modules" data-environment-modules>${renderEnvironmentModules(configItems)}</div>
      </form>
      <section class="environment-audits" data-environment-audits>
        <h4>${t("route.envs.audit.title")}</h4>
        ${renderEnvironmentAudits(audits)}
      </section>
    </section>`;
  };

  const reload = async (statusMessage = "") => {
    const payload = await fetchEnvironments();
    localState.configItems = payload.configItems;
    localState.audits = payload.audits;
    localState.runtimeInfo = payload.runtimeInfo;
    paint(payload.configItems, payload.audits, payload.runtimeInfo, statusMessage);
    bindView();
  };

  const waitForRuntimeReady = async () => {
    const deadline = Date.now() + 45000;
    while (Date.now() < deadline) {
      await new Promise((resolve) => window.setTimeout(resolve, 1000));
      try {
        const response = await fetch("/readyz", {
          method: "GET",
          cache: "no-store",
          headers: { "Cache-Control": "no-store" }
        });
        if (response.ok) {
          localState.restarting = false;
          persistRuntimeRestartNotice();
          window.location.reload();
          return;
        }
      } catch {
      }
    }
    localState.restarting = false;
    paint(localState.configItems, localState.audits, localState.runtimeInfo, t("route.envs.restart_wait_timeout"));
    bindView();
  };

  const requestRuntimeRestart = async () => {
    if (localState.restarting) {
      return;
    }
    const decision = await showGlobalConfirmModal({
      title: t("route.envs.restart_service"),
      message: t("route.envs.restart_confirm"),
      description: t("route.envs.restart_confirm_desc"),
      confirmLabel: t("route.envs.restart_service"),
      checkboxLabel: t("route.envs.restart_sync_master"),
      checkboxHint: t("route.envs.restart_sync_master_hint"),
      checkboxChecked: true
    });
    if (!decision.confirmed) {
      return;
    }
    const shouldSyncRemoteMaster = Boolean(decision.checked);
    localState.restarting = true;
    paint(localState.configItems, localState.audits, localState.runtimeInfo, t(shouldSyncRemoteMaster ? "route.envs.restarting_sync" : "route.envs.restarting"));
    bindView();
    try {
      const response = await fetch("/api/control/runtime/restart", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          "sync_remote_master": shouldSyncRemoteMaster
        })
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        const message = typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`;
        throw new Error(message);
      }
      void waitForRuntimeReady();
    } catch (err) {
      localState.restarting = false;
      const message = err instanceof Error ? err.message : "unknown_error";
      await reload(t("route.envs.restart_failed", { error: message }));
    }
  };

  const submitChanges = async (form) => {
    const controls = form.querySelectorAll("[data-env-input]");
    const changes = {};
    controls.forEach((control) => {
      if (control.disabled) {
        return;
      }
      const key = normalizeText(control.getAttribute("data-env-key") || control.name || "");
      if (!key) {
        return;
      }
      const original = normalizeText(control.getAttribute("data-original") || "");
      const value = normalizeText(control.value || "");
      if (value !== original) {
        changes[key] = value;
      }
    });

    if (!Object.keys(changes).length) {
      await reload(t("route.envs.no_changes"));
      return;
    }

    try {
      const response = await fetch("/api/control/environments", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ operator: "web-ui", values: changes })
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        const message = typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`;
        throw new Error(message);
      }
      const restartKeys = Array.isArray(payload?.restart_keys) ? payload.restart_keys : [];
      let message = t("route.envs.saved");
      if (payload?.needs_restart && restartKeys.length) {
        message = t("route.envs.restart_notice", { keys: restartKeys.join(", ") });
      }
      await reload(message);
    } catch (err) {
      const message = err instanceof Error ? err.message : "unknown_error";
      await reload(t("route.envs.save_failed", { error: message }));
    }
  };

  const bindView = () => {
    const form = container.querySelector("[data-environment-form]");
    const refreshButton = container.querySelector("[data-environment-refresh]");
    const revealButton = container.querySelector("[data-environment-reveal]");
    const restartButton = container.querySelector("[data-environment-restart]");
    if (!form || !refreshButton || !revealButton || !restartButton) {
      return;
    }
    refreshButton.addEventListener("click", async () => {
      await reload("");
    });
    revealButton.addEventListener("click", async () => {
      localState.revealSensitive = !localState.revealSensitive;
      await reload("");
    });
    restartButton.addEventListener("click", async () => {
      await requestRuntimeRestart();
    });
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      await submitChanges(form);
    });
  };

  const initialPayload = await fetchEnvironments();
  localState.configItems = initialPayload.configItems;
  localState.audits = initialPayload.audits;
  localState.runtimeInfo = initialPayload.runtimeInfo;
  paint(initialPayload.configItems, initialPayload.audits, initialPayload.runtimeInfo, "");
  bindView();
}

async function loadModelsView(container) {
  let providers = [];
  const modalHost = (() => {
    const existing = document.querySelector("[data-route-modal-root='models']");
    if (existing instanceof HTMLElement) {
      existing.remove();
    }
    const node = document.createElement("div");
    node.setAttribute("data-route-modal-root", "models");
    document.body.appendChild(node);
    return node;
  })();
  const localState = {
    showModal: false,
    editingProvider: null,
    statusMessage: ""
  };
  const DEFAULT_PROVIDER_TYPE = "openai-compatible";
  const OPENAI_PROVIDER_TEMPLATE = {
    id: "openai",
    name: "OpenAI",
    provider_type: "openai-compatible",
    api_type: "openai-responses",
    base_url: "https://api.openai.com/v1",
    models: [
      { id: "gpt-4o", name: "GPT-4o", is_enabled: true, supports_tools: true, supports_vision: true, supports_streaming: true },
      { id: "gpt-4o-mini", name: "GPT-4o Mini", is_enabled: true, supports_tools: true, supports_vision: true, supports_streaming: true }
    ]
  };
  const OPENROUTER_PROVIDER_TEMPLATE = {
    id: "openrouter",
    name: "OpenRouter",
    provider_type: "openrouter",
    api_type: "openai-completions",
    base_url: "https://openrouter.ai/api/v1",
    openrouter: {
      site_url: "",
      app_name: "",
      fallback_models: [],
      provider_order: [],
      allow_fallbacks: true,
      require_parameters: false
    },
    models: [
      { id: "openai/gpt-5.4", name: "OpenAI / GPT-5.4", is_enabled: true, supports_tools: true, supports_vision: false, supports_streaming: true },
      { id: "openai/gpt-5.3-codex", name: "OpenAI / GPT-5.3 Codex", is_enabled: true, supports_tools: true, supports_vision: false, supports_streaming: true },
      { id: "anthropic/claude-3.7-sonnet", name: "Anthropic / Claude 3.7 Sonnet", is_enabled: true, supports_tools: true, supports_vision: true, supports_streaming: true }
    ]
  };
  const PROVIDER_TEMPLATES = {
    "openai-compatible": OPENAI_PROVIDER_TEMPLATE,
    openrouter: OPENROUTER_PROVIDER_TEMPLATE
  };

  const requestJSON = async (url, options) => {
    const response = await fetch(url, options);
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      const message = typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`;
      throw new Error(message);
    }
    return payload;
  };

  const sanitizeModel = (model) => {
    const id = normalizeText(model?.id || "");
    if (!id) {
      return null;
    }
    return {
      id,
      name: normalizeText(model?.name || id) || id,
      is_enabled: Boolean(model?.is_enabled),
      supports_tools: Boolean(model?.supports_tools),
      supports_vision: Boolean(model?.supports_vision),
      supports_streaming: Boolean(model?.supports_streaming),
      max_tokens: Number.isFinite(Number(model?.max_tokens)) ? Number(model.max_tokens) : 0
    };
  };

  const normalizeListField = (value) => {
    const raw = Array.isArray(value) ? value.join("\n") : String(value || "");
    const items = raw
      .split(/[\n,]/)
      .map((item) => normalizeText(item || ""))
      .filter((item) => item !== "-")
      .filter(Boolean);
    return Array.from(new Set(items));
  };

  const normalizeOptionalProviderField = (value) => {
    const normalized = normalizeText(value || "");
    return normalized === "-" ? "" : normalized;
  };

  const normalizeProviderAPIKeyField = (value) => normalizeOptionalProviderField(value);

  const sanitizeOpenRouterConfig = (config, providerType) => {
    if (providerType !== "openrouter" && !config) {
      return null;
    }
    return {
      site_url: normalizeOptionalProviderField(config?.site_url || ""),
      app_name: normalizeOptionalProviderField(config?.app_name || ""),
      fallback_models: normalizeListField(config?.fallback_models || []),
      provider_order: normalizeListField(config?.provider_order || []),
      allow_fallbacks: config?.allow_fallbacks === undefined ? true : Boolean(config?.allow_fallbacks),
      require_parameters: Boolean(config?.require_parameters)
    };
  };

  const sanitizeProvider = (provider) => {
    const models = Array.isArray(provider?.models)
      ? provider.models.map(sanitizeModel).filter(Boolean)
      : [];
    const providerType = normalizeText(provider?.provider_type || "") || DEFAULT_PROVIDER_TYPE;
    return {
      id: normalizeText(provider?.id || ""),
      name: normalizeText(provider?.name || ""),
      provider_type: providerType,
      api_type: normalizeText(provider?.api_type || "") || "openai-responses",
      base_url: normalizeText(provider?.base_url || ""),
      api_key: normalizeProviderAPIKeyField(provider?.api_key || ""),
      openrouter: sanitizeOpenRouterConfig(provider?.openrouter, providerType),
      default_model: normalizeText(provider?.default_model || ""),
      models,
      is_enabled: Boolean(provider?.is_enabled),
      is_default: Boolean(provider?.is_default)
    };
  };

  const cloneProvider = (provider) => sanitizeProvider(JSON.parse(JSON.stringify(provider || {})));

  const createEmptyModel = (overrides = {}) => ({
    id: "",
    name: "",
    is_enabled: true,
    supports_tools: true,
    supports_vision: false,
    supports_streaming: true,
    max_tokens: 0,
    ...overrides
  });

  const createProviderDraft = () => createProviderDraftFromType(DEFAULT_PROVIDER_TYPE);

  const createProviderDraftFromType = (providerType) => {
    const template = PROVIDER_TEMPLATES[providerType] || PROVIDER_TEMPLATES[DEFAULT_PROVIDER_TYPE];
    return {
      id: "",
      name: template.name,
      provider_type: template.provider_type,
      api_type: template.api_type,
      base_url: template.base_url,
      api_key: "",
      openrouter: sanitizeOpenRouterConfig(template.openrouter, template.provider_type),
      default_model: template.models[0]?.id || "",
      models: template.models.map((model) => createEmptyModel(model)),
      is_enabled: true,
      is_default: false
    };
  };

  const providerTypeLabel = (providerType) => providerType === "openrouter" ? "OpenRouter" : "OpenAI Compatible";

  const getEnabledModels = (provider) => {
    const models = Array.isArray(provider?.models) ? provider.models : [];
    return models.filter((model) => model && model.is_enabled);
  };

  const renderModelsEmptyIcon = () => `<svg viewBox="0 0 64 64" aria-hidden="true" focusable="false">
    <rect x="10" y="12" width="44" height="30" rx="8" fill="#eaf2ff" stroke="#93c5fd" stroke-width="2"></rect>
    <path d="M18 48h28" stroke="#1d4ed8" stroke-width="3" stroke-linecap="round"></path>
    <path d="M24 22h16M24 30h10" stroke="#2563eb" stroke-width="3" stroke-linecap="round"></path>
    <circle cx="46" cy="46" r="10" fill="#2563eb"></circle>
    <path d="M46 41v10M41 46h10" stroke="#fff" stroke-width="3" stroke-linecap="round"></path>
  </svg>`;

  const findProvider = (providerID) => providers.find((provider) => provider.id === providerID) || null;
  const normalizeProviderNameKey = (value) => normalizeText(value || "").toLowerCase();
  const hasDuplicateProviderName = (name, currentProviderID = "") => {
    const normalizedName = normalizeProviderNameKey(name);
    if (!normalizedName) {
      return false;
    }
    return providers.some((provider) => provider.id !== currentProviderID && normalizeProviderNameKey(provider.name) === normalizedName);
  };

  const fetchProviders = async () => {
    const payload = await fetchJSON("/api/control/llm/providers");
    const items = Array.isArray(payload?.items) ? payload.items : [];
    return items.map(sanitizeProvider);
  };

  const showModal = (provider) => {
    localState.showModal = true;
    localState.editingProvider = provider ? cloneProvider(provider) : createProviderDraft();
    paintModal();
    bindModalEvents();
  };

  const hideModal = () => {
    localState.showModal = false;
    localState.editingProvider = null;
    paintModal();
  };

  const activateModal = () => {
    const modalBackdrop = modalHost.querySelector("[data-modal]");
    if (!(modalBackdrop instanceof HTMLElement)) {
      return;
    }
    activateModalBackdrop(modalBackdrop);
  };

  const renderEditableModelRow = (model) => {
    const safeModel = sanitizeModel(model) || createEmptyModel();
    return `<div class="provider-model-row" data-model-row>
      <label>
        <span>Model ID</span>
        <input type="text" name="model_id" value="${escapeHTML(safeModel.id)}" placeholder="gpt-4o" autocomplete="off">
      </label>
      <label>
        <span>Model Name</span>
        <input type="text" name="model_name" value="${escapeHTML(safeModel.name === safeModel.id ? "" : safeModel.name)}" placeholder="GPT-4o" autocomplete="off">
      </label>
      <label class="provider-model-checkbox">
        <input type="checkbox" name="model_enabled" ${safeModel.is_enabled ? "checked" : ""}>
        <span>启用</span>
      </label>
      <button type="button" class="provider-model-remove" data-remove-model>删除</button>
    </div>`;
  };

  const renderProviderModels = (provider) => {
    const models = Array.isArray(provider?.models) ? provider.models : [];
    if (!models.length) {
      return '<p class="provider-model-empty">暂无模型</p>';
    }
    return `<ul class="provider-model-list">
      ${models.map((model) => {
        const defaultClass = model.id === provider.default_model ? " provider-model-item-default" : "";
        const disabledClass = model.is_enabled ? "" : " provider-model-item-disabled";
        const selectableClass = provider.is_enabled && model.is_enabled ? " provider-model-item-selectable" : "";
        const content = `<span class="provider-model-name">${escapeHTML(model.name || model.id)}</span>
          <span class="provider-model-id">${escapeHTML(model.id)}</span>
          <span class="provider-model-badge">${model.id === provider.default_model ? "默认模型" : (model.is_enabled ? "点击设为默认" : "已禁用")}</span>`;
        if (provider.is_enabled && model.is_enabled) {
          return `<li>
            <button type="button" class="provider-model-trigger provider-model-item${defaultClass}${disabledClass}${selectableClass}" data-action="set-default-model" data-id="${escapeHTML(provider.id)}" data-model-id="${escapeHTML(model.id)}">
              ${content}
            </button>
          </li>`;
        }
        return `<li class="provider-model-item${defaultClass}${disabledClass}">
          ${content}
        </li>`;
      }).join("")}
    </ul>`;
  };

  const renderProviderCard = (provider) => {
    const statusClass = provider.is_enabled ? "provider-enabled" : "provider-disabled";
    const defaultBadge = provider.is_default ? '<span class="badge-default">默认 Provider</span>' : "";
    const enabledModelCount = getEnabledModels(provider).length;
    const openrouter = provider.provider_type === "openrouter" ? sanitizeOpenRouterConfig(provider.openrouter, provider.provider_type) : null;
    const note = provider.is_default
      ? (localState.statusMessage || "当前 Provider 正在提供默认模型")
      : (provider.is_enabled ? "直接点击模型项即可切换默认模型" : "启用后可设置为默认 Provider 和默认模型");
    return `<article class="provider-card ${statusClass}">
      <div class="provider-header">
        <div>
          <h3>${escapeHTML(provider.name || "未命名 Provider")} ${defaultBadge}</h3>
          <p class="provider-card-note">${escapeHTML(note)}</p>
        </div>
        <span class="provider-status">${provider.is_enabled ? "已启用" : "已禁用"}</span>
      </div>
      <div class="provider-info">
        <p><strong>Provider Type:</strong> ${escapeHTML(providerTypeLabel(provider.provider_type))}</p>
        <p><strong>Base URL:</strong> ${escapeHTML(provider.base_url || "-")}</p>
        <p><strong>API Type:</strong> ${escapeHTML(provider.api_type || "-")}</p>
        <p><strong>API Key:</strong> ${escapeHTML(provider.api_key || "-")}</p>
        <p><strong>默认模型:</strong> ${escapeHTML(provider.default_model || "-")}</p>
        <p><strong>模型数量:</strong> ${escapeHTML(String(provider.models.length))} / 可用 ${escapeHTML(String(enabledModelCount))}</p>
        ${openrouter ? `<p><strong>App 标识:</strong> ${escapeHTML(openrouter.app_name || "-")} / ${escapeHTML(openrouter.site_url || "-")}</p>` : ""}
        ${openrouter ? `<p><strong>回退模型:</strong> ${escapeHTML(openrouter.fallback_models.join(", ") || "-")}</p>` : ""}
        ${openrouter ? `<p><strong>Provider 路由:</strong> ${escapeHTML(openrouter.provider_order.join(", ") || "-")}</p>` : ""}
      </div>
      <section class="provider-models">
        <div class="provider-models-head">
          <h4>Models</h4>
        </div>
        ${renderProviderModels(provider)}
      </section>
      <div class="provider-actions">
        <button type="button" data-action="edit" data-id="${escapeHTML(provider.id)}">编辑</button>
        <button type="button" data-action="${provider.is_enabled ? "disable" : "enable"}" data-id="${escapeHTML(provider.id)}">${provider.is_enabled ? "禁用" : "启用"}</button>
        <button type="button" data-action="delete" data-id="${escapeHTML(provider.id)}">删除</button>
      </div>
    </article>`;
  };

  const renderProviderModal = () => {
    if (!localState.showModal) {
      return "";
    }
    const form = localState.editingProvider || createProviderDraft();
    const isEditing = Boolean(localState.editingProvider && localState.editingProvider.id);
    const defaultModels = getEnabledModels(form);
    const defaultModel = normalizeText(form.default_model || "") || (defaultModels[0] ? defaultModels[0].id : "");
    const isOpenRouter = form.provider_type === "openrouter";
    const openrouter = sanitizeOpenRouterConfig(form.openrouter, form.provider_type) || sanitizeOpenRouterConfig(null, "openrouter");
    const modelOptions = defaultModels.length
      ? defaultModels.map((model) => `<option value="${escapeHTML(model.id)}" ${model.id === defaultModel ? "selected" : ""}>${escapeHTML(model.name || model.id)}</option>`).join("")
      : '<option value="">请先添加启用的模型</option>';
    return `<div class="modal-backdrop" data-modal data-modal-state="enter">
      <div class="modal-dialog modal-dialog-wide">
        <div class="modal-header">
          <h3>${isEditing ? "编辑 Provider" : "新增 Provider"}</h3>
          <button type="button" data-close>&times;</button>
        </div>
        <form data-provider-form autocomplete="off">
          <div class="modal-body">
          <div class="provider-form-grid">
            <label>
              <span>Provider 名称</span>
              <input type="text" name="name" value="${escapeHTML(form.name || "")}" placeholder="OpenAI" required autocomplete="off">
            </label>
            <label>
              <span>Provider Type</span>
              <select name="provider_type" data-provider-type>
                <option value="openai-compatible" ${form.provider_type === "openai-compatible" ? "selected" : ""}>OpenAI Compatible</option>
                <option value="openrouter" ${form.provider_type === "openrouter" ? "selected" : ""}>OpenRouter</option>
              </select>
            </label>
            <label class="provider-form-full">
              <span>API Type</span>
              <select name="api_type">
                <option value="openai-responses" ${form.api_type === "openai-responses" ? "selected" : ""}>openai-responses</option>
                <option value="openai-completions" ${form.api_type === "openai-completions" ? "selected" : ""}>openai-completions</option>
              </select>
            </label>
            <label class="provider-form-full">
              <span>Base URL</span>
              <input type="text" name="base_url" value="${escapeHTML(form.base_url || "")}" placeholder="${escapeHTML(OPENAI_PROVIDER_TEMPLATE.base_url)}" required autocomplete="off">
            </label>
            <label class="provider-form-full">
              <span>API Key</span>
              <input type="password" name="api_key" value="" placeholder="${escapeHTML(isEditing ? "留空则保持现有 API Key" : (isOpenRouter ? "sk-or-v1-..." : "sk-..."))}" ${isEditing ? "" : "required"} autocomplete="new-password">
            </label>
          </div>
          <section class="provider-model-editor" data-openrouter-fields ${isOpenRouter ? "" : "hidden"}>
            <div class="provider-model-editor-head">
              <div>
                <h4>OpenRouter 配置</h4>
                <p>站点标识会通过官方头部写入请求；回退模型与 Provider 路由会附加到请求体，便于统一走 OpenRouter 调度。</p>
              </div>
            </div>
            <div class="provider-form-grid">
              <label class="provider-form-full">
                <span>Site URL</span>
                <input type="text" name="openrouter_site_url" value="${escapeHTML(openrouter.site_url || "")}" placeholder="https://your-app.example" autocomplete="off">
              </label>
              <label class="provider-form-full">
                <span>App Name</span>
                <input type="text" name="openrouter_app_name" value="${escapeHTML(openrouter.app_name || "")}" placeholder="Alter0" autocomplete="off">
              </label>
              <label class="provider-form-full">
                <span>Fallback Models</span>
                <textarea name="openrouter_fallback_models" rows="3" placeholder="openai/gpt-5.4&#10;anthropic/claude-3.7-sonnet">${escapeHTML(openrouter.fallback_models.join("\n"))}</textarea>
              </label>
              <label class="provider-form-full">
                <span>Provider Order</span>
                <input type="text" name="openrouter_provider_order" value="${escapeHTML(openrouter.provider_order.join(", "))}" placeholder="openai, anthropic" autocomplete="off">
              </label>
              <label class="provider-checkbox">
                <input type="checkbox" name="openrouter_allow_fallbacks" ${openrouter.allow_fallbacks ? "checked" : ""}>
                <span>允许 Provider Fallback</span>
              </label>
              <label class="provider-checkbox">
                <input type="checkbox" name="openrouter_require_parameters" ${openrouter.require_parameters ? "checked" : ""}>
                <span>仅选择支持全部参数的 Provider</span>
              </label>
            </div>
          </section>
          <section class="provider-model-editor">
            <div class="provider-model-editor-head">
              <div>
                <h4>模型列表</h4>
                <p>${isOpenRouter ? "维护 OpenRouter 可选模型列表；默认模型会作为主模型发送，回退模型可在上方单独配置。" : "为当前 Provider 维护可选模型。默认提供 OpenAI 兼容参数，也可改成其他兼容服务。"}
                </p>
              </div>
              <button type="button" data-add-model>+ 添加模型</button>
            </div>
            <div class="provider-model-rows" data-model-rows>
              ${form.models.map((model) => renderEditableModelRow(model)).join("")}
            </div>
          </section>
          <div class="provider-form-grid">
            <label>
              <span>默认模型</span>
              <select name="default_model" data-default-model ${defaultModels.length ? "" : "disabled"}>
                ${modelOptions}
              </select>
            </label>
            <label class="provider-checkbox">
              <input type="checkbox" name="is_enabled" ${form.is_enabled ? "checked" : ""}>
              <span>启用 Provider</span>
            </label>
          </div>
          </div>
          <div class="modal-footer">
            <button type="button" data-close>取消</button>
            <button type="submit">保存</button>
          </div>
        </form>
      </div>
    </div>`;
  };

  const renderEmptyProvidersState = () => `<div class="providers-empty-state">
    <div class="models-empty-illustration">${renderModelsEmptyIcon()}</div>
    <div class="providers-empty-copy">
      <h3>还没有配置任何 Provider</h3>
      <p>您还没有配置任何 LLM Provider。添加第一个 OpenAI Compatible 或 OpenRouter Provider 后，即可开启模型对话并设置默认模型。</p>
    </div>
    <button type="button" class="btn-primary" data-action="add">+ 新增 Provider</button>
  </div>`;

  const paint = () => {
    const cards = providers.map(renderProviderCard).join("");
    const providerAction = providers.length
      ? '<button type="button" class="btn-primary providers-panel-add" data-action="add">+ 新增 Provider</button>'
      : "";
    const providersNote = "统一在这里维护 OpenAI Compatible / OpenRouter Provider、API Key 与模型列表，不再通过环境变量配置默认 Provider 或默认模型。";
    container.innerHTML = `<section class="route-view models-view">
      <p class="models-view-intro">${escapeHTML(providersNote)}</p>
      <section class="route-card providers-panel">
        <div class="providers-panel-header">
          <div class="providers-panel-heading">
            <div class="providers-panel-title-row">
              <h3>Provider 列表</h3>
              <button type="button" class="providers-panel-info" title="${escapeHTML(providersNote)}" aria-label="${escapeHTML(providersNote)}">i</button>
            </div>
            <p>已配置 ${escapeHTML(String(providers.length))} 个 Provider</p>
          </div>
          ${providerAction}
        </div>
        ${cards ? `<div class="providers-list">${cards}</div>` : renderEmptyProvidersState()}
      </section>
    </section>`;
    paintModal();
  };

  const paintModal = () => {
    modalHost.innerHTML = renderProviderModal();
  };

  const readModelsFromForm = (form) => {
    const rows = Array.from(form.querySelectorAll("[data-model-row]"));
    const models = [];
    const seenIDs = new Set();
    rows.forEach((row) => {
      const idInput = row.querySelector('input[name="model_id"]');
      const nameInput = row.querySelector('input[name="model_name"]');
      const enabledInput = row.querySelector('input[name="model_enabled"]');
      const id = normalizeText(idInput ? idInput.value : "");
      const name = normalizeText(nameInput ? nameInput.value : "") || id;
      const isEnabled = Boolean(enabledInput && enabledInput.checked);
      if (!id) {
        return;
      }
      if (seenIDs.has(id)) {
        throw new Error(`模型 ID 重复: ${id}`);
      }
      seenIDs.add(id);
      models.push({
        id,
        name,
        is_enabled: isEnabled,
        supports_tools: true,
        supports_vision: false,
        supports_streaming: true
      });
    });
    if (!models.length) {
      throw new Error("请至少配置一个模型");
    }
    if (!models.some((model) => model.is_enabled)) {
      throw new Error("请至少启用一个模型");
    }
    return models;
  };

  const readOpenRouterConfigFromForm = (form, providerType) => {
    if (providerType !== "openrouter") {
      return null;
    }
    return {
      site_url: normalizeText(form.querySelector('[name="openrouter_site_url"]')?.value || ""),
      app_name: normalizeText(form.querySelector('[name="openrouter_app_name"]')?.value || ""),
      fallback_models: normalizeListField(form.querySelector('[name="openrouter_fallback_models"]')?.value || ""),
      provider_order: normalizeListField(form.querySelector('[name="openrouter_provider_order"]')?.value || ""),
      allow_fallbacks: Boolean(form.querySelector('[name="openrouter_allow_fallbacks"]')?.checked),
      require_parameters: Boolean(form.querySelector('[name="openrouter_require_parameters"]')?.checked)
    };
  };

  const syncProviderTypeFields = (form) => {
    const providerType = normalizeText(form.querySelector('[name="provider_type"]')?.value || "") || DEFAULT_PROVIDER_TYPE;
    const openrouterFields = form.querySelector("[data-openrouter-fields]");
    if (openrouterFields instanceof HTMLElement) {
      openrouterFields.hidden = providerType !== "openrouter";
    }
  };

  const applyProviderTemplateToForm = (form, providerType) => {
    const template = createProviderDraftFromType(providerType);
    const nameInput = form.querySelector('[name="name"]');
    const apiTypeSelect = form.querySelector('[name="api_type"]');
    const baseURLInput = form.querySelector('[name="base_url"]');
    const defaultModelSelect = form.querySelector('[name="default_model"]');
    const siteURLInput = form.querySelector('[name="openrouter_site_url"]');
    const appNameInput = form.querySelector('[name="openrouter_app_name"]');
    const fallbackModelsInput = form.querySelector('[name="openrouter_fallback_models"]');
    const providerOrderInput = form.querySelector('[name="openrouter_provider_order"]');
    const allowFallbacksInput = form.querySelector('[name="openrouter_allow_fallbacks"]');
    const requireParametersInput = form.querySelector('[name="openrouter_require_parameters"]');
    const rows = form.querySelector("[data-model-rows]");

    if (nameInput instanceof HTMLInputElement) {
      nameInput.value = template.name;
    }
    if (apiTypeSelect instanceof HTMLSelectElement) {
      apiTypeSelect.value = template.api_type;
    }
    if (baseURLInput instanceof HTMLInputElement) {
      baseURLInput.value = template.base_url;
    }
    if (rows instanceof HTMLElement) {
      rows.innerHTML = template.models.map((model) => renderEditableModelRow(model)).join("");
    }
    if (defaultModelSelect instanceof HTMLSelectElement) {
      defaultModelSelect.value = template.default_model;
    }
    if (siteURLInput instanceof HTMLInputElement) {
      siteURLInput.value = template.openrouter?.site_url || "";
    }
    if (appNameInput instanceof HTMLInputElement) {
      appNameInput.value = template.openrouter?.app_name || "";
    }
    if (fallbackModelsInput instanceof HTMLTextAreaElement) {
      fallbackModelsInput.value = (template.openrouter?.fallback_models || []).join("\n");
    }
    if (providerOrderInput instanceof HTMLInputElement) {
      providerOrderInput.value = (template.openrouter?.provider_order || []).join(", ");
    }
    if (allowFallbacksInput instanceof HTMLInputElement) {
      allowFallbacksInput.checked = Boolean(template.openrouter?.allow_fallbacks);
    }
    if (requireParametersInput instanceof HTMLInputElement) {
      requireParametersInput.checked = Boolean(template.openrouter?.require_parameters);
    }

    syncProviderTypeFields(form);
    syncModalDefaultModelSelect(form);
  };

  const syncModalDefaultModelSelect = (form) => {
    const select = form.querySelector("[data-default-model]");
    if (!select) {
      return;
    }
    const previousValue = normalizeText(select.value || "");
    let models = [];
    try {
      models = readModelsFromForm(form).filter((model) => model.is_enabled);
    } catch (_err) {
      models = [];
    }
    if (!models.length) {
      select.innerHTML = '<option value="">请先添加启用的模型</option>';
      select.value = "";
      select.disabled = true;
      return;
    }
    select.disabled = false;
    select.innerHTML = models.map((model) => `<option value="${escapeHTML(model.id)}">${escapeHTML(model.name || model.id)}</option>`).join("");
    const nextValue = models.some((model) => model.id === previousValue) ? previousValue : models[0].id;
    select.value = nextValue;
  };

  const refresh = async (statusMessage) => {
    providers = await fetchProviders();
    localState.statusMessage = normalizeText(statusMessage || "");
    paint();
    bindEvents();
  };

  const applyProviderDefaultModel = async (providerID, modelID) => {
    const provider = findProvider(providerID);
    if (!provider || !normalizeText(modelID || "")) {
      return;
    }
    const providerURL = "/api/control/llm/providers/" + encodeURIComponent(provider.id);
    await requestJSON(providerURL, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: provider.name,
        provider_type: provider.provider_type,
        base_url: provider.base_url,
        api_type: provider.api_type,
        api_key: "",
        openrouter: provider.openrouter,
        default_model: modelID,
        models: provider.models,
        is_enabled: provider.is_enabled
      })
    });
    if (!provider.is_default) {
      await requestJSON(providerURL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "set-default" })
      });
    }
  };

  const bindEvents = () => {
    container.querySelectorAll('[data-action="add"]').forEach((addBtn) => {
      addBtn.addEventListener("click", () => {
        showModal(null);
      });
    });

    container.querySelectorAll("[data-action]:not([data-action='add'])").forEach((button) => {
      button.addEventListener("click", async () => {
        const action = normalizeText(button.getAttribute("data-action") || "");
        const id = normalizeText(button.getAttribute("data-id") || "");
        const providerURL = "/api/control/llm/providers/" + encodeURIComponent(id);
        try {
          if (action === "edit") {
            const provider = findProvider(id);
            showModal(provider);
            return;
          }
          if (action === "delete") {
            if (!confirm("确定删除该 Provider 吗？")) {
              return;
            }
            await requestJSON(providerURL, { method: "DELETE" });
            await refresh("Provider 已删除");
            return;
          }
          if (action === "set-default-model") {
            const modelID = normalizeText(button.getAttribute("data-model-id") || "");
            await applyProviderDefaultModel(id, modelID);
            await refresh("默认模型已更新");
            return;
          }
          if (action === "enable" || action === "disable") {
            await requestJSON(providerURL, {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ action: action === "enable" ? "enable" : "disable" })
            });
            await refresh(action === "enable" ? "Provider 已启用" : "Provider 已禁用");
          }
        } catch (err) {
          alert("操作失败: " + (err instanceof Error ? err.message : "unknown_error"));
        }
      });
    });
    bindModalEvents();
  };

  const bindModalEvents = () => {
    modalHost.querySelectorAll("[data-close]").forEach((element) => {
      element.addEventListener("click", () => {
        hideModal();
      });
    });

    const modalBackdrop = modalHost.querySelector("[data-modal]");
    if (modalBackdrop) {
      modalBackdrop.addEventListener("click", (event) => {
        if (event.target && event.target.hasAttribute("data-modal")) {
          hideModal();
        }
      });
    }
    activateModal();

    const providerForm = modalHost.querySelector("[data-provider-form]");
    if (providerForm) {
      const isEditing = Boolean(localState.editingProvider && localState.editingProvider.id);
      syncModalDefaultModelSelect(providerForm);
      syncProviderTypeFields(providerForm);
      providerForm.addEventListener("click", (event) => {
        const target = event.target;
        if (!(target instanceof HTMLElement)) {
          return;
        }
        if (target.hasAttribute("data-add-model")) {
          const rows = providerForm.querySelector("[data-model-rows]");
          if (rows) {
            rows.insertAdjacentHTML("beforeend", renderEditableModelRow(createEmptyModel({
              id: "",
              name: "",
              supports_tools: true,
              supports_vision: true,
              supports_streaming: true
            })));
            syncModalDefaultModelSelect(providerForm);
          }
          return;
        }
        if (target.hasAttribute("data-remove-model")) {
          const row = target.closest("[data-model-row]");
          if (row) {
            row.remove();
            syncModalDefaultModelSelect(providerForm);
          }
        }
      });
      providerForm.addEventListener("input", () => {
        syncProviderTypeFields(providerForm);
        syncModalDefaultModelSelect(providerForm);
      });
      providerForm.addEventListener("change", (event) => {
        const target = event.target;
        if (!isEditing && target instanceof HTMLSelectElement && target.name === "provider_type") {
          applyProviderTemplateToForm(providerForm, normalizeText(target.value || "") || DEFAULT_PROVIDER_TYPE);
          return;
        }
        syncProviderTypeFields(providerForm);
        syncModalDefaultModelSelect(providerForm);
      });
      providerForm.addEventListener("submit", async (event) => {
        event.preventDefault();
        try {
          const formData = new FormData(providerForm);
          const originalProviderID = isEditing ? localState.editingProvider.id : "";
          const providerName = normalizeText(formData.get("name") || "");
          const providerType = normalizeText(formData.get("provider_type") || "") || DEFAULT_PROVIDER_TYPE;
          if (hasDuplicateProviderName(providerName, originalProviderID)) {
            throw new Error("Provider 名称已存在，请使用其他名称");
          }
          const models = readModelsFromForm(providerForm);
          const enabledModels = models.filter((model) => model.is_enabled);
          const rawDefaultModel = normalizeText(formData.get("default_model") || "");
          const defaultModel = enabledModels.some((model) => model.id === rawDefaultModel)
            ? rawDefaultModel
            : enabledModels[0].id;
          const body = {
            name: providerName,
            provider_type: providerType,
            api_type: normalizeText(formData.get("api_type") || "") || "openai-responses",
            base_url: normalizeText(formData.get("base_url") || ""),
            api_key: normalizeProviderAPIKeyField(formData.get("api_key") || ""),
            openrouter: readOpenRouterConfigFromForm(providerForm, providerType),
            default_model: defaultModel,
            models,
            is_enabled: formData.get("is_enabled") === "on"
          };
          const url = isEditing
            ? "/api/control/llm/providers/" + encodeURIComponent(originalProviderID)
            : "/api/control/llm/providers";
          await requestJSON(url, {
            method: isEditing ? "PUT" : "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body)
          });
          hideModal();
          await refresh(isEditing ? "Provider 已更新" : "Provider 已创建");
        } catch (err) {
          alert("保存失败: " + (err instanceof Error ? err.message : "unknown_error"));
        }
      });
    }
  };

  await refresh("");
}

async function loadPlaceholderView(container) {
  container.innerHTML = routeCardTemplate(
    t("route.connected"),
    "web",
    [routeFieldRow("field.description", t("route.connected_desc"))],
    true
  );
}

async function renderRoute(route) {
  const safe = ROUTES[route] ? route : DEFAULT_ROUTE;
  state.currentRoute = safe;
  document.querySelectorAll("[data-route-modal-root]").forEach((node) => node.remove());
  activeMenuRoute(safe);
  collapseMobileSidebar();
  if (!isMobileViewport()) {
    appShell.classList.remove("nav-open");
    appShell.classList.remove("panel-open");
    syncOverlayState();
  }

  const config = ROUTES[safe];
  const routeKey = config.key || "chat"; // fallback
  
  // Update titles using translation keys
  // Assuming keys follow pattern route.{key}.title
  const titleKey = `route.${routeKey}.title`;
  const subtitleKey = `route.${routeKey}.subtitle`;
  
  if (config.mode === "chat") {
    setMainContentMode("chat");
    if (safe === "agent-runtime") {
      reconcileAgentRuntimeTarget();
    }
    const activeSession = ensureActiveConversationSession(safe);
    if (syncSessionTargetForRoute(activeSession, safe)) {
      persistSessions();
    }
    syncRouteAction("");
    syncMainChatComposerDraft(activeConversationSessionID(), { preserveCurrent: false });
    renderSessions();
    renderMessages();
    syncWelcomeCopy();
    renderWelcomeTargetPicker();
    renderChatRuntimePanel();
    if (config.conversation === "agent") {
      void refreshChatAgentCatalog();
    }
    void refreshChatProviderCatalog();
    void refreshChatCapabilityCatalog();
    syncHeader();
    return;
  }

  setMainContentMode("page");
  closeTransientPanels();
  chatPane.dataset.route = safe;
  routeView.dataset.route = safe;
  routeBody.dataset.route = safe;
  routeView.classList.toggle("terminal-route", safe === "terminal");
  routeBody.classList.toggle("terminal-route-body", safe === "terminal");
  routeTitle.textContent = t(titleKey);
  routeSubtitle.textContent = t(subtitleKey);
  syncRouteAction(safe);
  routeBody.innerHTML = `<p class="route-loading">${t("loading")}</p>`;
  syncHeader();

  const token = ++state.pageRenderToken;
  try {
    await config.loader(routeBody);
  } catch (err) {
    if (token !== state.pageRenderToken) {
      return;
    }
    const message = err instanceof Error ? err.message : "unknown_error";
    routeBody.innerHTML = `<p class="route-error">${t("load_failed", { error: message })}</p>`;
  }
}

function bindEvents() {
  bindNavTooltipEvents();

  messageArea.addEventListener("click", (event) => {
    const copyTarget = event.target.closest("[data-copy-value]");
    if (copyTarget) {
      const value = copyTarget.getAttribute("data-copy-value") || "";
      if (!value) {
        return;
      }
      event.preventDefault();
      void (async () => {
        try {
          const copied = await copyTextValue(value);
          if (!copied) {
            return;
          }
          copyTarget.classList.add("copied");
          window.setTimeout(() => copyTarget.classList.remove("copied"), 900);
        } catch (error) {
          console.warn("copy value failed", error);
        }
      })();
      return;
    }
    const target = event.target.closest("[data-agent-process-toggle]");
    if (!target) {
      return;
    }
    event.preventDefault();
    toggleAgentProcessMessage(target.getAttribute("data-agent-process-toggle"));
  });

  routeBody.addEventListener("click", async (event) => {
    const target = event.target.closest("[data-copy-value]");
    if (!target) {
      return;
    }
    const value = target.getAttribute("data-copy-value") || "";
    if (!value) {
      return;
    }
    try {
      const copied = await copyTextValue(value);
      if (!copied) {
        return;
      }
      target.classList.add("copied");
      window.setTimeout(() => target.classList.remove("copied"), 900);
    } catch (error) {
      console.warn("copy value failed", error);
    }
  });

  mainChatComposer.bind(input, chatForm, {
    stableName: "chat-main",
    submitOnEnter: true,
    submitStrategy: "keydown",
    allowShiftEnter: true,
    draftStorage: "session",
    draftKey: () => getMainChatDraftKey(),
    counterNode: charCount,
    maxLength: MAX_CHARS,
    submitNode: sendButton,
    clearDraftOnSubmit: true,
    onInput: () => {
      updateCharCount();
    },
    onSubmit: async (composerInput) => {
      await sendMessage(composerInput.value);
    },
    onFocus: () => {
      scheduleViewportInsetSync({ alignFocusedInput: true });
      if (isMobileViewport()) {
        requestAnimationFrame(() => {
          input.scrollIntoView({ block: "nearest", behavior: "smooth" });
        });
        window.setTimeout(() => scheduleViewportInsetSync({ alignFocusedInput: true }), 120);
      }
    },
    onBlur: () => {
      window.setTimeout(() => scheduleViewportInsetSync(), 120);
    }
  });

  newChatButton.addEventListener("click", () => {
    if (isAgentConversationRoute()) {
      startNewAgentSession();
      return;
    }
    startNewChatSession();
  });
  if (mobileNewChatButton) {
    mobileNewChatButton.addEventListener("click", () => {
      if ((ROUTES[state.currentRoute] || ROUTES.chat).key === "terminal") {
        const terminalCreateButton = routeBody.querySelector("[data-terminal-create]");
        if (terminalCreateButton instanceof HTMLElement) {
          terminalCreateButton.click();
        }
        return;
      }
      if (isAgentConversationRoute()) {
        startNewAgentSession();
        return;
      }
      startNewChatSession();
    });
  }
  if (sessionHistoryToggle) {
    sessionHistoryToggle.addEventListener("click", () => {
      setSessionHistoryCollapsed(!state.sessionHistoryCollapsed);
      persistSessionHistoryCollapsedState();
    });
  }

  for (const node of menuRouteItems) {
    node.addEventListener("click", () => {
      const route = node.dataset.route || DEFAULT_ROUTE;
      collapseMobileSidebar();
      navigateToRoute(route);
    });
  }

  if (routeActionButton) {
    routeActionButton.addEventListener("click", () => {
      const targetRoute = routeActionButton.dataset.route;
      if (!targetRoute) {
        return;
      }
      navigateToRoute(targetRoute);
    });
  }

  navToggle.addEventListener("click", (event) => {
    event.stopPropagation();
    const open = !appShell.classList.contains("nav-open");
    closeTransientPanels();
    if (open) {
      appShell.classList.add("nav-open");
      syncOverlayState();
    }
  });

  sessionToggle.addEventListener("click", (event) => {
    event.stopPropagation();
    if ((ROUTES[state.currentRoute] || ROUTES.chat).key === "terminal") {
      const terminalToggleButton = routeBody.querySelector("[data-terminal-session-pane-toggle]");
      if (terminalToggleButton instanceof HTMLElement) {
        terminalToggleButton.click();
      }
      return;
    }
    if ((ROUTES[state.currentRoute] || ROUTES.chat).mode !== "chat") {
      return;
    }
    const open = !appShell.classList.contains("panel-open");
    closeTransientPanels();
    if (open) {
      appShell.classList.add("panel-open");
      syncOverlayState();
    }
  });

  togglePaneButton.addEventListener("click", () => {
    closeTransientPanels();
  });

  navCollapseButton.addEventListener("click", () => {
    if (isMobileViewport()) {
      closeTransientPanels();
      return;
    }
    setSidebarCollapsed(!state.navCollapsed);
  });

  mobileBackdrop.addEventListener("click", () => {
    closeTransientPanels();
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeTransientPanels();
    }
  });

  document.addEventListener("visibilitychange", () => {
    ensureChatTaskPolling();
    if (isDocumentVisible()) {
      scheduleViewportInsetSync({ alignFocusedInput: Boolean(activeViewportInput()) });
      const terminalVisibleHandler = routeBody && typeof routeBody.__alter0TerminalVisible === "function"
        ? routeBody.__alter0TerminalVisible
        : null;
      if (state.currentRoute === "terminal" && terminalVisibleHandler) {
        terminalVisibleHandler();
      }
    }
  });

  if (chatRuntimePanel) {
    chatRuntimePanel.addEventListener("click", (event) => {
      event.stopPropagation();
    });
  }

  document.addEventListener("click", (event) => {
    if (!state.chatRuntime.openPopover || !chatRuntimePanel) {
      return;
    }
    if (chatRuntimePanel.contains(event.target)) {
      return;
    }
    closeChatRuntimePopover();
  });

  chatPane.addEventListener("click", (event) => {
    const hasOverlay = appShell.classList.contains("panel-open") || appShell.classList.contains("nav-open");
    if (!hasOverlay) {
      return;
    }
    if (sessionPane && sessionPane.contains(event.target)) {
      return;
    }
    if (primaryNav && primaryNav.contains(event.target)) {
      return;
    }
    closeTransientPanels();
  });

  const quickPrompts = document.querySelectorAll(".prompt[data-prompt]");
  for (const node of quickPrompts) {
    node.addEventListener("click", async () => {
      const prompt = node.getAttribute("data-prompt");
      if (!prompt) {
        return;
      }
      input.value = prompt;
      updateCharCount();
      await sendMessage(prompt);
    });
  }

  window.addEventListener("hashchange", () => {
    const nextRoute = parseHashRoute();
    if (state.suppressHashRouteConfirm === nextRoute) {
      state.suppressHashRouteConfirm = "";
      void renderRoute(nextRoute);
      return;
    }
    state.suppressHashRouteConfirm = "";
    if (nextRoute !== state.currentRoute && !confirmComposerNavigation()) {
      const fallbackHash = `#${state.currentRoute || DEFAULT_ROUTE}`;
      if (window.location.hash !== fallbackHash) {
        window.location.hash = fallbackHash;
      }
      return;
    }
    void renderRoute(nextRoute);
  });

  window.addEventListener("beforeunload", (event) => {
    if (!hasBlockingComposerDraft()) {
      return;
    }
    event.preventDefault();
    event.returnValue = t("composer.unsaved_confirm");
  });

  window.addEventListener("resize", () => {
    if (isMobileViewport() && state.navCollapsed) {
      setSidebarCollapsed(false);
    }
    if (!isMobileViewport()) {
      closeTransientPanels();
    }
    if (navTooltipTarget) {
      if (!shouldShowNavTooltipFor(navTooltipTarget)) {
        hideNavTooltip(true);
      } else {
        positionNavTooltip(navTooltipTarget);
      }
    }
    scheduleViewportInsetSync();
    if (chatRuntimePanel && (state.currentRoute === "chat" || state.currentRoute === "agent-runtime")) {
      renderChatRuntimePanel();
    }
  });

  window.addEventListener("orientationchange", () => {
    state.mobileViewport.baselineHeight = 0;
    state.mobileViewport.width = 0;
    state.mobileViewport.height = 0;
    state.mobileViewport.keyboardOffset = 0;
    scheduleViewportInsetSync();
  });

  if (window.visualViewport) {
    window.visualViewport.addEventListener("resize", () => {
      scheduleViewportInsetSync({ alignFocusedInput: Boolean(activeViewportInput()) });
    });
    window.visualViewport.addEventListener("scroll", () => {
      if (!isMobileViewport() || !activeViewportInput()) {
        return;
      }
      scheduleViewportInsetSync();
    });
  }

  bindSwipeClose(primaryNav, "nav-open");
  bindSwipeClose(sessionPane, "panel-open");
}

function init() {
  setComposerConfirmState("idle");
  setSidebarCollapsed(false);
  setSessionHistoryCollapsed(loadSessionHistoryCollapsedState());
  bootstrapSessions();
  renderSessions();
  renderMessages();
  syncHeader();
  syncWelcomeCopy();
  renderChatRuntimePanel();
  renderWelcomeTargetPicker();
  bindEvents();
  ensureChatTaskPolling();
  void refreshChatProviderCatalog();
  void refreshChatCapabilityCatalog();
  updateCharCount();
  updateKeyboardInset();
  syncComposerGuardState();
  void renderRoute(parseHashRoute());
  document.body.setAttribute("data-app-ready", "true");
  showPendingRuntimeRestartNotice();
  input.focus();
}

init();
