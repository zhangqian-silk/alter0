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
const menuRouteItems = document.querySelectorAll(".menu-item[data-route]");
const navTooltipTargets = [...menuRouteItems, navCollapseButton];
const rootStyle = document.documentElement.style;

const MAX_CHARS = 10000;
const DEFAULT_ROUTE = "chat";
const SWIPE_CLOSE_THRESHOLD = 46;
const NAV_TOOLTIP_SHOW_DELAY = 90;
const NAV_TOOLTIP_HIDE_DELAY = 40;
const NAV_TOOLTIP_OFFSET = 12;
const STREAM_ENDPOINT = "/api/messages/stream";
const FALLBACK_ENDPOINT = "/api/messages";
const SESSION_STORAGE_KEY = "alter0.web.sessions.v1";
const SESSION_HISTORY_PANEL_STORAGE_KEY = "alter0.web.session-history-panel.v1";
const TERMINAL_STORAGE_KEY = "alter0.web.terminal.sessions.v1";
const I18N = {
  en: {
    // Navigation
    "nav.chat": "Chat",
    "nav.control": "Control",
    "nav.agent": "Agent",
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
    
    // Session Pane
    "session.header": "Work with alter0",
    "session.close": "Close",
    "session.new": "New Chat",
    "session.delete": "Delete",
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
    "session.empty_sub": "Empty session, waiting for your first message",
    "session.no_active": "No active session. Click New Chat to start.",
    "session.empty_body": "This session is empty. Type a message to start.",
    "status.in_progress": "In Progress",
    "status.failed": "Failed",
    "status.done": "Done",
    "status.enabled": "Enabled",
    "status.disabled": "Disabled",
    "status.queued": "Queued",
    "status.running": "Running",
    "status.success": "Success",
    "status.canceled": "Canceled",
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
    "route.chat.subtitle": "Ready to start a new conversation",
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
    "route.tasks.terminal.hint": "Supports follow-up interaction in current terminal session (max {max} concurrent sessions).",
    "route.tasks.terminal.followup_note": "Each Send may create a child task, but stays in the same shell session.",
    "route.tasks.terminal.limit_reached": "Terminal session limit reached ({max}). Please finish an active session first.",
    "route.tasks.actions.retry": "Retry",
    "route.tasks.actions.cancel": "Cancel",
    "route.tasks.result.title": "Result Output",
    "route.terminal.title": "Terminal",
    "route.terminal.subtitle": "Chat-style terminal proxy with continuous session chaining",
    "route.terminal.new": "New Terminal Session",
    "route.terminal.empty": "No terminal sessions yet. Create one and send your first command.",
    "route.terminal.pick": "Select a terminal session to start.",
    "route.terminal.input": "Type command or follow-up...",
    "route.terminal.send": "Send",
    "route.terminal.sending": "Sending...",
    "route.terminal.session": "Session",
    "route.terminal.anchor_task": "Anchor Task",
    "route.terminal.active_task": "Active Task",
    "route.terminal.status": "Status",
    "route.terminal.logs.heading": "Task Execution Logs ({task})",
    "route.terminal.logs.empty": "No terminal logs yet.",
    "route.terminal.followup.title": "AI Follow-up",
    "route.terminal.limit_reached": "Terminal session limit reached ({max}). Please finish an active session first.",
    "route.terminal.send_failed": "Send failed: {error}",
    "route.terminal.logs_failed": "Load terminal logs failed: {error}",
    "route.terminal.loading": "Loading terminal session...",
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
    "route.envs.refresh": "Reload",
    "route.envs.show_sensitive": "Reveal Sensitive",
    "route.envs.hide_sensitive": "Hide Sensitive",
    "route.envs.current_value": "Configured",
    "route.envs.default_value": "Default",
    "route.envs.effective_value": "Effective",
    "route.envs.apply_mode": "Apply Mode",
    "route.envs.source": "Source",
    "route.envs.validation": "Validation",
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
    "route.envs.validation.none": "No constraints",
    "route.envs.hidden": "Hidden value",
    "route.connected": "Page Connected",
    "route.connected_desc": "This page route is active. Content can be expanded by module.",
    "loading": "Loading...",
    "load_failed": "Load failed: {error}"
  },
  zh: {
    // Navigation
    "nav.chat": "对话",
    "nav.control": "控制台",
    "nav.agent": "智能体",
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
    
    // Session Pane
    "session.header": "与 alter0 协作",
    "session.close": "关闭",
    "session.new": "新对话",
    "session.delete": "删除",
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
    "session.empty_sub": "空会话，等待你的第一条消息",
    "session.no_active": "没有活动会话。点击“新对话”开始。",
    "session.empty_body": "当前会话为空。输入消息开始对话。",
    "status.in_progress": "进行中",
    "status.failed": "失败",
    "status.done": "完成",
    "status.enabled": "启用",
    "status.disabled": "停用",
    "status.queued": "排队中",
    "status.running": "运行中",
    "status.success": "成功",
    "status.canceled": "已取消",
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
    "route.chat.subtitle": "准备好开始新的对话",
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
    "route.tasks.terminal.hint": "支持在当前终端会话中继续交互（最多并发 {max} 个终端会话）。",
    "route.tasks.terminal.followup_note": "每次发送可能生成子任务，但仍在同一个 Shell 会话内连续处理。",
    "route.tasks.terminal.limit_reached": "终端会话已达上限（{max}），请先结束一个活跃会话。",
    "route.tasks.actions.retry": "重试",
    "route.tasks.actions.cancel": "取消",
    "route.tasks.result.title": "终态输出",
    "route.terminal.title": "终端",
    "route.terminal.subtitle": "以对话方式代理终端操作，并在同一终端会话内连续串联",
    "route.terminal.new": "新建终端会话",
    "route.terminal.empty": "暂无终端会话。创建后可直接发送第一条命令。",
    "route.terminal.pick": "选择一个终端会话开始交互。",
    "route.terminal.input": "输入命令或追问继续...",
    "route.terminal.send": "发送",
    "route.terminal.sending": "发送中...",
    "route.terminal.session": "会话",
    "route.terminal.anchor_task": "锚点任务",
    "route.terminal.active_task": "当前任务",
    "route.terminal.status": "状态",
    "route.terminal.logs.heading": "任务执行日志（{task}）",
    "route.terminal.logs.empty": "暂无终端日志。",
    "route.terminal.followup.title": "AI 交互建议",
    "route.terminal.limit_reached": "终端会话已达上限（{max}），请先结束一个活跃会话。",
    "route.terminal.send_failed": "发送失败：{error}",
    "route.terminal.logs_failed": "终端日志加载失败：{error}",
    "route.terminal.loading": "正在加载终端会话...",
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
    "route.envs.refresh": "重新加载",
    "route.envs.show_sensitive": "显示敏感项",
    "route.envs.hide_sensitive": "隐藏敏感项",
    "route.envs.current_value": "配置值",
    "route.envs.default_value": "默认值",
    "route.envs.effective_value": "生效值",
    "route.envs.apply_mode": "生效方式",
    "route.envs.source": "来源",
    "route.envs.validation": "校验规则",
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
    mode: "chat"
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
    loader: loadPlaceholderView
  },
  environments: {
    key: "envs",
    mode: "page",
    loader: loadEnvironmentsView
  }
};

const state = {
  activeSessionID: "",
  currentRoute: DEFAULT_ROUTE,
  sessions: [],
  sessionRouteFilters: {
    triggerType: "",
    channelType: "",
    channelID: "",
    messageID: "",
    jobID: ""
  },
  sessionLoadError: "",
  sessionHistoryCollapsed: false,
  pending: false,
  pageRenderToken: 0,
  navCollapsed: false,
  lang: "en" // default
};

let navTooltipNode = null;
let navTooltipTarget = null;
let navTooltipShowTimer = 0;
let navTooltipHideTimer = 0;

function t(key, params = {}) {
  const dict = I18N[state.lang] || I18N.en;
  let val = dict[key] || key;
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
  
  // Re-render current route if it's a page
  if (state.currentRoute !== "chat") {
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

function getSession(id = state.activeSessionID) {
  return state.sessions.find((item) => item.id === id);
}

function sortSessionsByCreatedAtDesc(items) {
  items.sort((left, right) => right.createdAt - left.createdAt);
}

function isBlankSession(item) {
  return Boolean(item) && Array.isArray(item.messages) && item.messages.length === 0;
}

function getLatestBlankSession() {
  const blankSessions = state.sessions.filter((item) => isBlankSession(item));
  if (!blankSessions.length) {
    return null;
  }
  sortSessionsByCreatedAtDesc(blankSessions);
  return blankSessions[0];
}

function enforceSingleBlankSession() {
  const latestBlank = getLatestBlankSession();
  if (!latestBlank) {
    return false;
  }
  const originalCount = state.sessions.length;
  state.sessions = state.sessions.filter((item) => !isBlankSession(item) || item.id === latestBlank.id);
  if (state.activeSessionID && !getSession(state.activeSessionID)) {
    state.activeSessionID = latestBlank.id;
  }
  return state.sessions.length !== originalCount;
}

function focusSession(sessionID) {
  if (!getSession(sessionID)) {
    return;
  }
  state.activeSessionID = sessionID;
  navigateToRoute("chat");
  renderSessions();
  renderMessages();
  syncHeader();
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
    error: Boolean(item.error),
    status,
    retryable: Boolean(item.retryable)
  };
}

function normalizeStoredSession(item) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const id = typeof item.id === "string" && item.id ? item.id : makeID();
  const title = typeof item.title === "string" && item.title.trim() ? item.title.trim() : "New Chat";
  const createdAt = Number.isFinite(item.createdAt) ? item.createdAt : Date.now();
  const rawMessages = Array.isArray(item.messages) ? item.messages : [];
  const messages = [];
  for (const raw of rawMessages) {
    const normalized = normalizeStoredMessage(raw, createdAt);
    if (normalized) {
      messages.push(normalized);
    }
  }
  return { id, title, createdAt, messages };
}

function loadSessionsFromStorage() {
  const storage = getSessionStorage();
  if (!storage) {
    return [];
  }
  const raw = storage.getItem(SESSION_STORAGE_KEY);
  if (!raw) {
    return [];
  }

  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error("local_storage_corrupted");
  }

  if (!Array.isArray(parsed)) {
    throw new Error("local_storage_invalid");
  }

  const sessions = [];
  for (const entry of parsed) {
    const normalized = normalizeStoredSession(entry);
    if (normalized) {
      sessions.push(normalized);
    }
  }
  sortSessionsByCreatedAtDesc(sessions);
  return sessions;
}

function persistSessions() {
  const storage = getSessionStorage();
  if (!storage) {
    return;
  }
  try {
    storage.setItem(SESSION_STORAGE_KEY, JSON.stringify(state.sessions));
    state.sessionLoadError = "";
  } catch {
    state.sessionLoadError = "session_save_failed";
  }
  syncSessionLoadHint();
}

function bootstrapSessions() {
  state.sessionLoadError = "";
  state.sessions = [];
  state.activeSessionID = "";

  try {
    const sessions = loadSessionsFromStorage();
    state.sessions = sessions;
    if (enforceSingleBlankSession()) {
      persistSessions();
    }
    if (state.sessions.length) {
      state.activeSessionID = state.sessions[0].id;
    }
  } catch (err) {
    const message = err instanceof Error ? err.message : "unknown_error";
    state.sessionLoadError = message;
  }
}

function createSession() {
  const latestBlank = getLatestBlankSession();
  if (latestBlank) {
    state.activeSessionID = latestBlank.id;
    renderSessions();
    renderMessages();
    syncHeader();
    persistSessions();
    return latestBlank;
  }

  const createdAt = Date.now();
  const item = {
    id: makeID(),
    title: t("session.new_title"),
    createdAt,
    messages: []
  };
  state.sessions.unshift(item);
  state.activeSessionID = item.id;
  renderSessions();
  renderMessages();
  syncHeader();
  syncWelcomeCopy();
  persistSessions();
  return item;
}

function removeSession(sessionID) {
  const nextSessions = state.sessions.filter((item) => item.id !== sessionID);
  if (nextSessions.length === state.sessions.length) {
    return;
  }

  state.sessions = nextSessions;
  if (state.activeSessionID === sessionID || !getSession(state.activeSessionID)) {
    const latestBlank = getLatestBlankSession();
    if (latestBlank) {
      state.activeSessionID = latestBlank.id;
    } else if (state.sessions.length) {
      state.activeSessionID = state.sessions[0].id;
    } else {
      state.activeSessionID = "";
    }
  }

  enforceSingleBlankSession();
  renderSessions();
  renderMessages();
  syncHeader();
  persistSessions();
}

function syncHeader() {
  const route = ROUTES[state.currentRoute] || ROUTES.chat;

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
  if (active.messages.length === 0) {
    sessionSubheading.textContent = t("session.empty_sub");
    return;
  }
  sessionSubheading.textContent = `${active.messages.length} messages`;
}

function syncWelcomeCopy() {
  const active = getSession();
  if (!active) {
    welcomeHeading.textContent = t("welcome.heading");
    welcomeDescription.textContent = t("session.no_active");
    return;
  }
  welcomeHeading.textContent = t("welcome.heading");
  welcomeDescription.textContent = t("welcome.desc");
}

function syncSessionLoadHint() {
  sessionLoadError.textContent = state.sessionLoadError;
  sessionLoadError.style.display = state.sessionLoadError ? "block" : "none";
}

function renderSessions() {
  sessionList.innerHTML = "";
  syncSessionLoadHint();
  if (!state.sessions.length) {
    sessionEmpty.textContent = t("session.empty");
    sessionEmpty.style.display = "block";
    return;
  }
  sessionEmpty.style.display = "none";

  for (const item of state.sessions) {
    const row = document.createElement("div");
    row.className = "session-card-row";

    const card = document.createElement("button");
    card.type = "button";
    card.className = "session-card";
    card.setAttribute("role", "option");
    card.setAttribute("aria-selected", item.id === state.activeSessionID ? "true" : "false");
    if (item.id === state.activeSessionID) {
      card.classList.add("active");
    }

    const title = document.createElement("p");
    title.className = "session-card-title";
    title.textContent = item.title;

    const meta = document.createElement("p");
    meta.className = "session-card-meta";
    meta.textContent = `${item.messages.length} messages · ${formatSince(item.createdAt)}`;

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
    deleteButton.addEventListener("click", (event) => {
      event.stopPropagation();
      removeSession(item.id);
    });

    row.appendChild(card);
    row.appendChild(deleteButton);
    sessionList.appendChild(row);
  }
}

function updateSessionTitle(session, fallbackText) {
  if (session.title !== t("session.new_title") && session.title !== "New Chat" && session.title !== "新对话") {
    return;
  }
  const text = fallbackText.trim();
  if (!text) {
    return;
  }
  session.title = shorten(text, 18);
  persistSessions();
}

function appendMessage(role, text, options = {}) {
  let session = getSession();
  if (!session) {
    session = getLatestBlankSession();
    if (session) {
      state.activeSessionID = session.id;
    } else {
      session = createSession();
    }
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
    error: Boolean(options.error),
    status: options.status || (options.error ? "error" : "done"),
    retryable: Boolean(options.retryable)
  };
  session.messages.push(message);
  enforceSingleBlankSession();
  renderSessions();
  renderMessages();
  syncHeader();
  persistSessions();
  return message;
}

function updateMessage(message, patch = {}) {
  if (!message) {
    return;
  }
  Object.assign(message, patch);
  renderMessages();
  syncHeader();
  persistSessions();
}

function assistantStatusLabel(status) {
  if (status === "streaming") {
    return t("status.in_progress");
  }
  if (status === "error") {
    return t("status.failed");
  }
  return t("status.done");
}

function renderMessages() {
  const active = getSession();
  const hasMessages = Boolean(active && active.messages.length);
  welcomeScreen.style.display = hasMessages ? "none" : "block";
  messageArea.style.display = hasMessages ? "block" : "none";
  chatPane.classList.toggle("empty-state", !hasMessages);

  if (!hasMessages) {
    syncWelcomeCopy();
    messageArea.innerHTML = "";
    return;
  }

  const list = document.createElement("div");
  list.className = "message-list";

  for (const msg of active.messages) {
    const container = document.createElement("article");
    container.className = `msg ${msg.role}`;
    if (msg.error) {
      container.classList.add("error");
    }
    if (msg.status === "streaming") {
      container.classList.add("streaming");
    }

    const bubble = document.createElement("div");
    bubble.className = "msg-bubble";
    bubble.textContent = msg.text;

    const meta = document.createElement("div");
    meta.className = "msg-meta";

    if (msg.route && msg.role === "assistant") {
      const pill = document.createElement("span");
      pill.className = "route-pill";
      pill.textContent = msg.route.toUpperCase();
      meta.appendChild(pill);
    }

    if (msg.role === "assistant") {
      const status = document.createElement("span");
      status.className = `status-pill ${msg.status || "done"}`;
      status.textContent = assistantStatusLabel(msg.status);
      meta.appendChild(status);
    }

    const time = document.createElement("span");
    time.textContent = timeLabel(msg.at);
    meta.appendChild(time);

    container.appendChild(bubble);
    container.appendChild(meta);
    list.appendChild(container);
  }

  messageArea.innerHTML = "";
  messageArea.appendChild(list);
  messageArea.scrollTop = messageArea.scrollHeight;
}

function setPending(flag) {
  state.pending = flag;
  sendButton.disabled = flag;
  input.disabled = flag;
}

function updateCharCount() {
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

async function sendMessageStream(payload, assistantMessage) {
  let sawEvent = false;
  let sawDone = false;
  let routeHint = "";
  let output = "";

  try {
    const response = await fetch(STREAM_ENDPOINT, {
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
            const finalOutput = typeof result.output === "string" ? result.output : output;
            updateMessage(assistantMessage, {
              text: finalOutput.trim() || t("msg.received_empty"),
              route,
              error: false,
              status: "done",
              retryable: false,
              at: Date.now()
            });
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

async function sendMessageFallback(payload, assistantMessage) {
  const response = await fetch(FALLBACK_ENDPOINT, {
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
  updateMessage(assistantMessage, {
    text: output,
    route: body?.result?.route || "",
    error: false,
    status: "done",
    retryable: false,
    at: Date.now()
  });
}

async function sendMessage(rawContent) {
  if (state.currentRoute !== "chat") {
    navigateToRoute("chat");
  }
  const content = rawContent.trim();
  if (!content || state.pending) {
    return;
  }

  appendMessage("user", content);
  input.value = "";
  updateCharCount();
  setPending(true);

  const active = getSession();
  const payload = {
    session_id: active ? active.id : "",
    channel_id: "web-default",
    content
  };
  const assistantMessage = appendMessage("assistant", t("msg.processing"), { status: "streaming" });

  try {
    const streamResult = await sendMessageStream(payload, assistantMessage);
    if (streamResult.ok) {
      return;
    }

    if (streamResult.canFallback) {
      await sendMessageFallback(payload, assistantMessage);
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
  syncOverlayState();
}

function collapseMobileSidebar() {
  if (!isMobileViewport()) {
    return;
  }
  closeTransientPanels();
}

function updateKeyboardInset() {
  if (!isMobileViewport() || !window.visualViewport) {
    rootStyle.setProperty("--keyboard-offset", "0px");
    return;
  }

  const viewport = window.visualViewport;
  const inset = Math.max(0, window.innerHeight - viewport.height - viewport.offsetTop);
  rootStyle.setProperty("--keyboard-offset", `${Math.round(inset)}px`);
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

function navigateToRoute(route) {
  const safe = ROUTES[route] ? route : DEFAULT_ROUTE;
  collapseMobileSidebar();
  const targetHash = `#${safe}`;
  if (window.location.hash !== targetHash) {
    window.location.hash = targetHash;
    return;
  }
  void renderRoute(safe);
}

function startNewChatSession() {
  const existingBlank = getLatestBlankSession();
  if (existingBlank) {
    focusSession(existingBlank.id);
  } else {
    createSession();
  }
  navigateToRoute("chat");
  closeTransientPanels();
  window.requestAnimationFrame(() => {
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
  const classNames = ["route-field-value"];
  if (multiline) {
    classNames.push("is-multiline");
  }
  if (mono) {
    classNames.push("is-mono");
  }
  const copyButton = copyable
    ? `<button class="route-field-copy" type="button" data-copy-value="${escapeHTML(safeValue)}" title="${escapeHTML(t("route.copy_value"))}" aria-label="${escapeHTML(t("route.copy_value"))}">${renderCopyIcon()}</button>`
    : "";
  return `<p class="route-field-row">
    <span>${t(labelKey)}</span>
    <span class="route-field-value-wrap">
      <strong class="${classNames.join(" ")}" title="${escapeHTML(safeValue)}">${escapeHTML(safeValue)}</strong>
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

function routeCardTemplate(title, type, fields = [], enabled = false, body = "") {
  return `<article class="route-card">
    <div class="route-card-head">
      <div class="route-card-title-wrap">
        <div class="route-card-icon" aria-hidden="true">${routeTypeIcon(type)}</div>
        <h4 title="${escapeHTML(normalizeText(title))}">${escapeHTML(normalizeText(title))}</h4>
      </div>
      ${routeStatusBadge(enabled)}
    </div>
    <div class="route-meta">
      ${fields.join("")}
    </div>
    ${body ? `<div class="memory-card-body">${body}</div>` : ""}
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
        routeFieldRow("field.description", item.description, { multiline: true })
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
    return `<article class="route-card session-route-card">
      <div class="route-card-head session-route-head">
        <div class="session-route-title-wrap">
          <div class="route-card-icon" aria-hidden="true">${routeTypeIcon("session")}</div>
          <h4 title="${escapeHTML(normalizeText(title))}">${escapeHTML(normalizeText(title))}</h4>
        </div>
        ${routeStatusBadge(true)}
      </div>
      <div class="route-meta session-route-meta">
        ${routeFieldRow("field.id", sessionID, { copyable: true, mono: true })}
        ${routeFieldRow("field.channel_type", formatChannelType(channelType))}
        ${routeFieldRow("field.last_message_id", lastMessageID, { copyable: true, mono: true })}
        ${routeFieldRow("field.updated", formatDateTime(updatedAt))}
      </div>
      <div class="memory-card-body">${detailBody}</div>
    </article>`;
  }).join("");
}

async function loadSessionsView(container) {
  const filters = normalizeSessionRouteFilters(state.sessionRouteFilters || {});
  state.sessionRouteFilters = filters;
  const data = await fetchJSON(sessionListQuery(filters, 1, 50));
  const items = Array.isArray(data.items) ? data.items : [];
  container.innerHTML = `<section class="session-history-view">
    <form class="task-filter-form page-filter-form page-filter-grid-3 session-filter-form" data-session-filter-form>
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
    const actionBody = `<div class="cron-job-actions">
      <button type="button" data-cron-edit="${escapeHTML(jobID)}">${t("route.cron.action.edit")}</button>
      <button type="button" data-cron-runs-btn="${escapeHTML(jobID)}">${t("route.cron.action.runs")}</button>
      <button type="button" data-cron-delete="${escapeHTML(jobID)}">${t("route.cron.action.delete")}</button>
    </div>
    <div class="cron-run-list" data-cron-runs="${escapeHTML(jobID)}" hidden></div>`;

    return routeCardTemplate(
      jobName,
      "cron",
      [
        routeFieldRow("field.id", jobID, { copyable: true, mono: true }),
        routeFieldRow("field.schedule_mode", normalizeText(scheduleMode)),
        routeFieldRow("field.cron_expression", expression, { multiline: true, mono: true }),
        routeFieldRow("field.timezone", timezone),
        routeFieldRow("field.input", taskInput, { multiline: true }),
        routeFieldRow("field.retry_limit", retryLimit)
      ],
      item.enabled,
      actionBody
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
    <form class="task-filter-form page-filter-form page-filter-grid-3 cron-form" data-cron-form>
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
  const note = form.querySelector("[data-cron-note]");
  const everyWrap = form.querySelector("[data-cron-every-wrap]");
  const unitWrap = form.querySelector("[data-cron-unit-wrap]");
  const timeWrap = form.querySelector("[data-cron-time-wrap]");
  const weekdayWrap = form.querySelector("[data-cron-weekday-wrap]");
  const modeInput = form.schedule_mode;
  const expressionInput = form.cron_expression;

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

function renderControlTaskList(payload) {
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
    const sessionID = typeof item?.session_id === "string" ? item.session_id : "-";
    const status = typeof item?.status === "string" ? item.status : "";
    const triggerType = typeof item?.trigger_type === "string" ? item.trigger_type : "";
    const channelType = typeof item?.channel_type === "string" ? item.channel_type : "";
    const sourceMessageID = typeof item?.source_message_id === "string" ? item.source_message_id : "";
    const updatedAt = typeof item?.updated_at === "string" ? item.updated_at : "";
    const jobID = typeof item?.job_id === "string" ? item.job_id : "";
    const statusClassName = taskStatusClassName(status);
    const cronRow = triggerType === "cron"
      ? `<p><span>${t("field.job_id")}</span><strong>${escapeHTML(normalizeText(jobID))}</strong></p>`
      : "";
    return `<article class="task-summary-card" data-control-task-id="${escapeHTML(taskID)}">
      <header class="task-summary-head">
        <div class="task-summary-id-wrap">
          <h5 class="task-summary-id" title="${escapeHTML(taskID)}">${escapeHTML(taskID)}</h5>
          <button class="task-summary-copy" type="button" data-control-task-copy-id="${escapeHTML(taskID)}" title="${t("route.tasks.copy_task_id")}" aria-label="${t("route.tasks.copy_task_id")}">${renderCopyIcon()}</button>
        </div>
        <span class="task-summary-status ${statusClassName}">${escapeHTML(formatTaskStatus(status))}</span>
      </header>
      <div class="task-summary-meta">
        <p><span>${t("field.session")}</span><strong>${escapeHTML(normalizeText(sessionID))}</strong></p>
        <p><span>${t("field.trigger_type")}</span><strong>${escapeHTML(formatTriggerType(triggerType))}</strong></p>
        <p><span>${t("field.channel_type")}</span><strong>${escapeHTML(formatChannelType(channelType))}</strong></p>
        <p><span>${t("field.source_message")}</span><strong>${escapeHTML(normalizeText(sourceMessageID))}</strong></p>
        <p><span>${t("field.updated")}</span><strong>${escapeHTML(formatDateTime(updatedAt))}</strong></p>
        ${cronRow}
      </div>
      <button class="task-summary-open" type="button" data-control-task-open="${escapeHTML(taskID)}"><span class="task-summary-open-icon" aria-hidden="true">${renderPanelRightOpenIcon()}</span><span>${t("route.tasks.open_detail")}</span></button>
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
  const terminalMaxSessionsRaw = Number(link?.terminal_max_sessions || 5);
  const terminalMaxSessions = Number.isFinite(terminalMaxSessionsRaw) && terminalMaxSessionsRaw > 0 ? terminalMaxSessionsRaw : 5;
  const queuePosition = Number(task?.queue_position || 0);
  const queueWaitMS = Number(task?.queue_wait_ms || 0);
  const resultOutput = typeof task?.result?.output === "string" ? task.result.output : "";
  const retryEnabled = Boolean(actions?.retry?.enabled);
  const cancelEnabled = Boolean(actions?.cancel?.enabled);
  const showCancelAction = cancelEnabled || !["success", "done", "failed", "error", "canceled"].includes(statusValue);
  const retryReason = typeof actions?.retry?.reason === "string" ? actions.retry.reason : "";
  const cancelReason = typeof actions?.cancel?.reason === "string" ? actions.cancel.reason : "";
  const progressRaw = Number(task?.progress);
  const progressValue = Number.isFinite(progressRaw) ? Math.min(100, Math.max(0, progressRaw)) : 0;
  const taskIDShort = shorten(shownTaskID, 24);
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
  return `<section class="task-detail-card" data-control-task-detail-id="${escapeHTML(taskID)}" data-control-task-session-id="${escapeHTML(normalizeText(task?.session_id))}" data-control-task-terminal-session-id="${escapeHTML(normalizeText(terminalSessionID))}" data-control-task-terminal-max-sessions="${escapeHTML(terminalMaxSessions)}">
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
    <div class="task-detail-actions">
      <button type="button" data-control-task-action="retry" ${retryEnabled ? "" : "disabled"} title="${escapeHTML(retryEnabled ? t("route.tasks.actions.retry_tip") : normalizeText(retryReason))}">${t("route.tasks.actions.retry")}</button>
      ${showCancelAction ? `<button type="button" data-control-task-action="cancel" ${cancelEnabled ? "" : "disabled"} title="${escapeHTML(cancelEnabled ? "" : normalizeText(cancelReason))}">${t("route.tasks.actions.cancel")}</button>` : ""}
      <button type="button" data-control-task-log-reconnect>${t("route.tasks.logs.reconnect")}</button>
      <button type="button" data-control-task-log-replay title="${escapeHTML(t("route.tasks.actions.replay_tip"))}">${t("route.tasks.logs.replay")}</button>
    </div>
    <section class="task-detail-section">
      <h6>${t("route.tasks.terminal.title")}</h6>
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
      <p class="control-task-terminal-hint">${escapeHTML(t("route.tasks.terminal.hint", { max: String(terminalMaxSessions) }))}</p>
      <p class="control-task-terminal-note">${escapeHTML(t("route.tasks.terminal.followup_note"))}</p>
    </section>
    <section class="task-detail-section">
      <h6>${t("route.tasks.result.title")}</h6>
      <div class="control-task-result-output">${renderControlTaskResultOutput(resultOutput)}</div>
    </section>
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
    terminalSubmitting: false,
    advancedOpen: false
  };

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
    localState.activeTaskID = "";
    localState.terminalAnchorTaskID = "";
    localState.logStreamNode = null;
    localState.logTouchStartY = null;
    localState.logStickToBottom = true;
    if (!drawer) {
      return;
    }
    drawer.classList.remove("open");
    drawer.hidden = true;
  };

  const openDrawer = () => {
    if (!drawer) {
      return;
    }
    drawer.hidden = false;
    requestAnimationFrame(() => {
      drawer.classList.add("open");
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

  const resetLogs = () => {
    localState.logCursor = 0;
    localState.logDone = false;
    localState.logItems = [];
    localState.logSeqSet = new Set();
    localState.logStickToBottom = true;
    localState.logTouchStartY = null;
    setLogStatus(t("route.tasks.logs.empty"));
    paintLogs();
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
    paintLogs();
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
      paintLogs();
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
    listNode.innerHTML = renderControlTaskList(payload);
    paginationNode.innerHTML = renderControlTaskPagination(payload);
  };

  const loadList = async () => {
    setFilterApplyingState();
    const payload = await fetchJSON(controlTaskListQuery(localState.filters, localState.page, localState.pageSize));
    paint(payload);
  };

  const setTerminalInputState = (submitting) => {
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

  const submitTerminalInput = async (rawInput) => {
    const taskID = normalizeText(localState.activeTaskID);
    if (!taskID || taskID === "-") {
      return;
    }
    const detailNode = drawerBody.querySelector("[data-control-task-detail-id]");
    const terminalMaxSessionsRaw = Number(detailNode?.dataset?.controlTaskTerminalMaxSessions || 5);
    const terminalMaxSessions = Number.isFinite(terminalMaxSessionsRaw) && terminalMaxSessionsRaw > 0 ? terminalMaxSessionsRaw : 5;

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
        if ((payload?.error_code || "") === "terminal_session_limit_reached") {
          window.alert(t("route.tasks.terminal.limit_reached", { max: String(terminalMaxSessions) }));
          return;
        }
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
    localState.logStreamNode = null;
    localState.logTouchStartY = null;
    setTerminalInputState(localState.terminalSubmitting);
    if (!options.preserveLogs) {
      resetLogs();
      await loadLogBackfill(taskID, 0);
    } else {
      paintLogs();
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
        await loadList();
      });
    }
  }

  view.addEventListener("submit", async (event) => {
    const terminalForm = event.target.closest("[data-control-task-terminal-input-form]");
    if (!terminalForm) {
      return;
    }
    event.preventDefault();
    const inputNode = terminalForm.querySelector("[data-control-task-terminal-input]");
    const value = inputNode ? String(inputNode.value || "") : "";
    if (inputNode) {
      inputNode.value = "";
    }
    await submitTerminalInput(value);
  });

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
    <form class="task-filter-form page-filter-form" data-control-task-filter-form>
      <div class="task-filter-primary-row">
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
      </div>
      <div class="task-filter-primary-actions">
        <button class="task-filter-advanced-toggle" type="button" data-control-task-advanced-toggle aria-expanded="false">${renderAdvancedToggleLabel(false)}</button>
        <div class="task-filter-actions">
          <button class="task-filter-apply" type="submit">${t("route.tasks.filter.apply")}</button>
          <button class="task-filter-reset" type="button" data-control-task-filter-reset>${t("route.tasks.filter.reset")}</button>
        </div>
      </div>
      <div class="task-filter-advanced" data-control-task-advanced hidden>
        <label><span>${t("route.tasks.filter.channel_type")}</span>
          <select name="channel_type">
            <option value="">-</option>
            <option value="cli">${t("channel.cli")}</option>
            <option value="web">${t("channel.web")}</option>
            <option value="scheduler">${t("channel.scheduler")}</option>
          </select>
        </label>
        <label><span>${t("route.tasks.filter.channel_id")}</span><input type="text" name="channel_id" placeholder="web-default"></label>
        <label><span>${t("route.tasks.filter.message_id")}</span><input type="text" name="message_id" placeholder="msg-123"></label>
        <label><span>${t("route.tasks.filter.source_message_id")}</span><input type="text" name="source_message_id" placeholder="msg-source-123"></label>
        <label><span>${t("route.tasks.filter.start_at")}</span><input type="datetime-local" name="start_at"></label>
        <label><span>${t("route.tasks.filter.end_at")}</span><input type="datetime-local" name="end_at"></label>
      </div>
    </form>
    <div class="task-summary-list" data-control-task-list></div>
    <div class="task-summary-pagination-wrap" data-control-task-pagination></div>
    <section class="control-task-drawer" data-control-task-drawer hidden>
      <button class="control-task-drawer-backdrop" type="button" aria-label="close" data-control-task-close></button>
      <aside class="control-task-drawer-panel">
        <header class="control-task-drawer-head">
          <h4>${t("route.tasks.drawer.title")}</h4>
          <button class="task-summary-next" type="button" data-control-task-close>${t("route.tasks.drawer.close")}</button>
        </header>
        <div class="control-task-drawer-body" data-control-task-drawer-body>${t("route.tasks.drawer.empty")}</div>
      </aside>
    </section>
  </section>`;
  bindControlTaskView(container, payload);
}

function normalizeTerminalStoredEntry(item, fallbackAt) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const role = String(item.role || "").trim().toLowerCase();
  if (!["user", "terminal", "system", "assistant"].includes(role)) {
    return null;
  }
  const text = typeof item.text === "string" ? item.text.trim() : "";
  if (!text) {
    return null;
  }
  const taskID = typeof item.task_id === "string" ? item.task_id.trim() : "";
  const at = Number.isFinite(item.at) ? item.at : fallbackAt;
  const kind = typeof item.kind === "string" ? item.kind.trim().toLowerCase() : "";
  const stage = typeof item.stage === "string" ? item.stage.trim().toLowerCase() : "";
  return {
    id: typeof item.id === "string" && item.id.trim() ? item.id.trim() : makeID(),
    role,
    text,
    task_id: taskID,
    at,
    kind,
    stage
  };
}

function normalizeTerminalStoredSession(item) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const now = Date.now();
  const id = typeof item.id === "string" && item.id.trim() ? item.id.trim() : makeID();
  const createdAt = Number.isFinite(item.created_at) ? item.created_at : now;
  const updatedAt = Number.isFinite(item.updated_at) ? item.updated_at : createdAt;
  const taskCursor = item.task_cursor && typeof item.task_cursor === "object" ? item.task_cursor : {};
  const safeCursor = {};
  for (const [taskID, cursor] of Object.entries(taskCursor)) {
    const safeTaskID = String(taskID || "").trim();
    const safeCursorValue = Number(cursor || 0);
    if (!safeTaskID || !Number.isFinite(safeCursorValue) || safeCursorValue < 0) {
      continue;
    }
    safeCursor[safeTaskID] = safeCursorValue;
  }
  const taskOrderRaw = Array.isArray(item.task_order) ? item.task_order : [];
  const taskOrder = taskOrderRaw.map((taskID) => String(taskID || "").trim()).filter(Boolean);
  const entriesRaw = Array.isArray(item.entries) ? item.entries : [];
  const entries = [];
  for (const row of entriesRaw) {
    const parsed = normalizeTerminalStoredEntry(row, createdAt);
    if (parsed) {
      entries.push(parsed);
    }
  }
  const runtimeSessionID = String(item.runtime_session_id || "").trim() || `terminal-runtime-${id}`;
  const terminalSessionID = String(item.terminal_session_id || "").trim() || `terminal-${id}`;
  const title = String(item.title || "").trim() || t("route.terminal.new");
  const status = String(item.status || "").trim().toLowerCase();
  return {
    id,
    title,
    created_at: createdAt,
    updated_at: updatedAt,
    runtime_session_id: runtimeSessionID,
    terminal_session_id: terminalSessionID,
    anchor_task_id: String(item.anchor_task_id || "").trim(),
    active_task_id: String(item.active_task_id || "").trim(),
    status: status || "idle",
    log_collapsed: typeof item.log_collapsed === "boolean" ? item.log_collapsed : null,
    task_order: taskOrder,
    task_cursor: safeCursor,
    entries
  };
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
  sessions.sort((left, right) => Number(right.updated_at || 0) - Number(left.updated_at || 0));
  return sessions;
}

function persistTerminalSessionsToStorage(sessions) {
  const storage = getSessionStorage();
  if (!storage) {
    return;
  }
  try {
    storage.setItem(TERMINAL_STORAGE_KEY, JSON.stringify(sessions));
  } catch {
  }
}

function renderTerminalStatus(status) {
  const normalized = String(status || "").trim().toLowerCase();
  if (!normalized || normalized === "idle") {
    return "-";
  }
  return formatTaskStatus(normalized);
}

function isTerminalTaskActiveStatus(status) {
  const normalized = String(status || "").trim().toLowerCase();
  return ["queued", "running", "in_progress", "processing"].includes(normalized);
}

function resolveTerminalLogCollapsed(session) {
  if (!session) {
    return false;
  }
  if (typeof session.log_collapsed === "boolean") {
    return session.log_collapsed;
  }
  return !isTerminalTaskActiveStatus(session.status);
}

function normalizeTerminalLine(text, max = 240) {
  const safe = String(text || "").replace(/\s+/g, " ").trim();
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
  const stage = String(entry?.stage || "").trim().toLowerCase();
  const text = String(entry?.text || "");
  if (role === "user") {
    return "command";
  }
  if (role === "system") {
    return "action";
  }
  if (stage && ["accept", "accepted", "running", "success", "failed", "error", "warn", "info"].includes(stage)) {
    return "tag";
  }
  if (/^\s*\[[a-z_]+\]/i.test(text)) {
    return "tag";
  }
  if (/^(accepted|已运行|executed|executing|run |running )/i.test(text)) {
    return "action";
  }
  return "output";
}

function isAssistantFollowupLog(stage, message) {
  const safeStage = String(stage || "").trim().toLowerCase();
  const safeMessage = String(message || "").trim();
  if (!safeMessage) {
    return false;
  }
  if (safeStage && !["terminal", "assistant", "chat", "nl"].includes(safeStage)) {
    return false;
  }
  if (/^\s*\[[a-z_]+\]\s+/i.test(safeMessage)) {
    return false;
  }
  if (/^(accepted|已运行|任务|task |running|success|failed|error|warning|stderr|stdout|\$ |>)/i.test(safeMessage.toLowerCase())) {
    return false;
  }
  if (/^(要不要|是否|我可以|你希望|Would you|Do you want|Shall I|Should I|I can|Let me)/i.test(safeMessage)) {
    return true;
  }
  if (/[?？]\s*$/.test(safeMessage) && safeMessage.length >= 8) {
    return true;
  }
  return false;
}

function buildTerminalLogLine(item) {
  const stage = String(item?.stage || "").trim().toLowerCase();
  const message = String(item?.message || "").trim();
  if (!message) {
    return "";
  }
  if (!stage || stage === "terminal" || stage === "stdout" || stage === "stderr") {
    return message;
  }
  return `[${stage}] ${message}`;
}

function renderTerminalSessionCards(sessions, activeSessionID) {
  if (!sessions.length) {
    return `<p class="terminal-session-empty">${escapeHTML(t("route.terminal.empty"))}</p>`;
  }
  return sessions.map((session) => {
    const title = normalizeText(session.title);
    const sessionID = normalizeText(session.terminal_session_id);
    const active = session.id === activeSessionID;
    return `<button class="terminal-session-card ${active ? "active" : ""}" type="button" data-terminal-session-select="${escapeHTML(session.id)}">
      <span class="terminal-session-title">${escapeHTML(title)}</span>
      <span class="terminal-session-meta">${escapeHTML(shorten(sessionID, 30))}</span>
      <span class="terminal-session-meta">${escapeHTML(renderTerminalStatus(session.status))} | ${escapeHTML(formatDateTime(new Date(Number(session.updated_at || 0)).toISOString()))}</span>
    </button>`;
  }).join("");
}

function splitTerminalEntries(entries) {
  const logs = [];
  const followups = [];
  (Array.isArray(entries) ? entries : []).forEach((entry) => {
    const role = String(entry?.role || "").trim().toLowerCase();
    if (role === "assistant") {
      followups.push(entry);
      return;
    }
    logs.push(entry);
  });
  return { logs, followups };
}

function renderTerminalLogRows(entries) {
  const safeEntries = Array.isArray(entries) ? entries : [];
  if (!safeEntries.length) {
    return `<div class="terminal-log-empty">${escapeHTML(t("route.terminal.logs.empty"))}</div>`;
  }
  return safeEntries.map((entry) => {
    const kind = classifyTerminalLogKind(entry);
    const rowClass = `terminal-log-row kind-${escapeHTML(kind)}`;
    const text = normalizeText(entry?.text || "");
    const prefix = kind === "command" ? "$" : "";
    const safeText = text === "-" ? "" : text;
    if (!safeText) {
      return "";
    }
    return `<div class="${rowClass}">
      <div class="terminal-log-main">
        ${prefix ? `<span class="terminal-log-prefix">${escapeHTML(prefix)}</span>` : ""}
        <span class="terminal-log-text">${escapeHTML(safeText)}</span>
      </div>
      <span class="terminal-log-time">${escapeHTML(timeLabel(entry?.at))}</span>
    </div>`;
  }).join("");
}

function renderTerminalFollowupRows(entries) {
  const safeEntries = Array.isArray(entries) ? entries : [];
  if (!safeEntries.length) {
    return "";
  }
  return `<section class="terminal-followup-list">
    <h6>${escapeHTML(t("route.terminal.followup.title"))}</h6>
    ${safeEntries.map((entry) => `<article class="terminal-followup-bubble">
      <div class="terminal-followup-text">${escapeHTML(normalizeText(entry?.text || ""))}</div>
      <div class="terminal-followup-meta">
        <span>${escapeHTML(timeLabel(entry?.at))}</span>
      </div>
    </article>`).join("")}
  </section>`;
}

function renderTerminalWorkspace(session, sending) {
  if (!session) {
    return `<section class="terminal-workspace-empty">
      <p>${escapeHTML(t("route.terminal.pick"))}</p>
    </section>`;
  }
  const collapsed = resolveTerminalLogCollapsed(session);
  const logToggleIcon = collapsed ? ">" : "v";
  const logTaskRef = normalizeText(session.active_task_id || session.anchor_task_id || session.terminal_session_id);
  const logTitle = t("route.terminal.logs.heading", { task: shorten(logTaskRef === "-" ? "n/a" : logTaskRef, 32) });
  const split = splitTerminalEntries(session.entries);
  return `<section class="terminal-workspace-body" data-terminal-workspace data-terminal-session-id="${escapeHTML(session.id)}">
    <header class="terminal-workspace-head">
      <div class="terminal-workspace-copy">
        <h4>${escapeHTML(logTitle)}</h4>
        <p>${escapeHTML(normalizeText(session.title))}</p>
      </div>
      <div class="terminal-workspace-badges">
        <span class="terminal-context-badge"><em>${t("route.terminal.session")}</em><strong>${escapeHTML(normalizeText(session.terminal_session_id))}</strong></span>
        <span class="terminal-context-badge"><em>${t("route.terminal.anchor_task")}</em><strong>${escapeHTML(normalizeText(session.anchor_task_id))}</strong></span>
        <span class="terminal-context-badge"><em>${t("route.terminal.active_task")}</em><strong>${escapeHTML(normalizeText(session.active_task_id))}</strong></span>
        <span class="terminal-context-badge is-status"><em>${t("route.terminal.status")}</em><strong>${escapeHTML(renderTerminalStatus(session.status))}</strong></span>
      </div>
    </header>
    <section class="terminal-console-panel">
      <button class="terminal-log-toggle" type="button" data-terminal-log-toggle aria-expanded="${collapsed ? "false" : "true"}">
        <span class="terminal-log-toggle-icon">${logToggleIcon}</span>
        <span class="terminal-log-toggle-text">${escapeHTML(logTitle)}</span>
      </button>
      <div class="terminal-chat-screen ${collapsed ? "is-collapsed" : ""}" data-terminal-chat-screen ${collapsed ? "hidden" : ""}>
        <div class="terminal-log-tree">
          ${renderTerminalLogRows(split.logs)}
        </div>
      </div>
      ${renderTerminalFollowupRows(split.followups)}
      <form class="terminal-chat-form" data-terminal-input-form>
        <input type="text" data-terminal-input maxlength="6000" placeholder="${escapeHTML(t("route.terminal.input"))}" ${sending ? "disabled" : ""}>
        <button type="submit" data-terminal-submit ${sending ? "disabled" : ""}>${escapeHTML(sending ? t("route.terminal.sending") : t("route.terminal.send"))}</button>
      </form>
    </section>
  </section>`;
}

async function loadTerminalView(container) {
  const localState = {
    sessions: loadTerminalSessionsFromStorage(),
    activeSessionID: "",
    sending: false,
    polling: false,
    timer: 0
  };
  localState.activeSessionID = localState.sessions[0] ? localState.sessions[0].id : "";

  const getActiveSession = () => {
    return localState.sessions.find((item) => item.id === localState.activeSessionID) || null;
  };

  const touchSession = (session) => {
    if (!session) {
      return;
    }
    session.updated_at = Date.now();
    localState.sessions.sort((left, right) => Number(right.updated_at || 0) - Number(left.updated_at || 0));
  };

  const persist = () => {
    persistTerminalSessionsToStorage(localState.sessions);
  };

  const appendEntry = (session, role, text, taskID = "", options = {}) => {
    const content = String(text || "").trim();
    if (!session || !content) {
      return;
    }
    session.entries.push({
      id: makeID(),
      role,
      text: content,
      task_id: String(taskID || "").trim(),
      at: Date.now(),
      kind: String(options.kind || "").trim().toLowerCase(),
      stage: String(options.stage || "").trim().toLowerCase()
    });
    if (session.entries.length > 1200) {
      session.entries = session.entries.slice(session.entries.length - 1200);
    }
    touchSession(session);
  };

  const ensureTaskTracked = (session, taskID) => {
    const safeTaskID = String(taskID || "").trim();
    if (!session || !safeTaskID) {
      return;
    }
    if (!Array.isArray(session.task_order)) {
      session.task_order = [];
    }
    if (!session.task_order.includes(safeTaskID)) {
      session.task_order.push(safeTaskID);
    }
    if (!session.task_cursor || typeof session.task_cursor !== "object") {
      session.task_cursor = {};
    }
    if (!Number.isFinite(Number(session.task_cursor[safeTaskID]))) {
      session.task_cursor[safeTaskID] = 0;
    }
  };

  const createNewTerminalSession = () => {
    const id = makeID();
    const createdAt = Date.now();
    const session = {
      id,
      title: t("route.terminal.new"),
      created_at: createdAt,
      updated_at: createdAt,
      runtime_session_id: `terminal-runtime-${id}`,
      terminal_session_id: `terminal-${id}`,
      anchor_task_id: "",
      active_task_id: "",
      status: "idle",
      log_collapsed: false,
      task_order: [],
      task_cursor: {},
      entries: []
    };
    localState.sessions.unshift(session);
    localState.activeSessionID = session.id;
    persist();
    return session;
  };

  const stopPolling = () => {
    if (localState.timer) {
      window.clearInterval(localState.timer);
      localState.timer = 0;
    }
  };

  const paint = () => {
    const active = getActiveSession();
    container.innerHTML = `<section class="terminal-view" data-terminal-view>
      <aside class="terminal-session-pane">
        <button class="terminal-session-create" type="button" data-terminal-create>${escapeHTML(t("route.terminal.new"))}</button>
        <div class="terminal-session-list">${renderTerminalSessionCards(localState.sessions, localState.activeSessionID)}</div>
      </aside>
      <section class="terminal-workspace">
        ${renderTerminalWorkspace(active, localState.sending)}
      </section>
    </section>`;
  };

  const appendLogItems = (session, taskID, logs) => {
    if (!session || !Array.isArray(logs) || !logs.length) {
      return;
    }
    logs.forEach((item) => {
      const stage = String(item?.stage || "").trim().toLowerCase();
      const line = buildTerminalLogLine(item);
      if (!line) {
        return;
      }
      if (isAssistantFollowupLog(stage, line)) {
        appendEntry(session, "assistant", line, taskID, { kind: "followup", stage });
        return;
      }
      appendEntry(session, "terminal", line, taskID, { kind: classifyTerminalLogKind({ role: "terminal", stage, text: line }), stage });
    });
  };

  const pollLogsForTask = async (session, taskID) => {
    if (!session || !taskID) {
      return;
    }
    ensureTaskTracked(session, taskID);
    let cursor = Number(session.task_cursor[taskID] || 0);
    let hasMore = true;
    let guard = 0;
    while (hasMore && guard < 20) {
      guard += 1;
      const page = await fetchJSON(`/api/control/tasks/${encodeURIComponent(taskID)}/logs?cursor=${Math.max(cursor, 0)}&limit=200`);
      const items = Array.isArray(page?.items) ? page.items : [];
      appendLogItems(session, taskID, items);
      const nextCursor = Number(page?.next_cursor || cursor);
      cursor = Number.isFinite(nextCursor) && nextCursor >= 0 ? nextCursor : cursor;
      hasMore = Boolean(page?.has_more);
    }
    session.task_cursor[taskID] = cursor;
  };

  const refreshActiveTaskStatus = async (session) => {
    const taskID = String(session?.active_task_id || "").trim();
    if (!session || !taskID) {
      return;
    }
    const payload = await fetchJSON(`/api/control/tasks/${encodeURIComponent(taskID)}`);
    session.status = normalizeText(payload?.task?.status || "idle").toLowerCase();
    if (typeof session.log_collapsed !== "boolean") {
      session.log_collapsed = !isTerminalTaskActiveStatus(session.status);
    }
    const linkTerminalSessionID = normalizeText(payload?.link?.terminal_session_id || "");
    if (linkTerminalSessionID !== "-") {
      session.terminal_session_id = linkTerminalSessionID;
    }
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
    const activeTaskID = String(session?.active_task_id || "").trim();
    if (!session || !activeTaskID) {
      stopPolling();
      return;
    }
    localState.polling = true;
    try {
      await pollLogsForTask(session, activeTaskID);
      await refreshActiveTaskStatus(session);
      touchSession(session);
      persist();
      paint();
      const chatNode = container.querySelector("[data-terminal-chat-screen]");
      if (chatNode) {
        chatNode.scrollTop = chatNode.scrollHeight;
      }
      if (["success", "failed", "canceled", "done"].includes(String(session.status || "").toLowerCase())) {
        stopPolling();
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "unknown_error";
      appendEntry(session, "system", t("route.terminal.logs_failed", { error: message }));
      session.status = "failed";
      touchSession(session);
      persist();
      paint();
      stopPolling();
    } finally {
      localState.polling = false;
    }
  };

  const startPolling = async () => {
    stopPolling();
    await pollActiveSession();
    const session = getActiveSession();
    const activeTaskID = String(session?.active_task_id || "").trim();
    if (!activeTaskID) {
      return;
    }
    localState.timer = window.setInterval(() => {
      void pollActiveSession();
    }, 1200);
  };

  const sendTerminalInput = async (content) => {
    const text = String(content || "").trim();
    if (!text || localState.sending) {
      return;
    }
    let session = getActiveSession();
    if (!session) {
      session = createNewTerminalSession();
    }

    session.log_collapsed = false;
    appendEntry(session, "user", normalizeTerminalLine(text), "", { kind: "command", stage: "command" });
    localState.sending = true;
    paint();
    try {
      const currentTaskID = String(session.active_task_id || "").trim();
      if (!currentTaskID) {
        const response = await fetch("/api/tasks", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            session_id: session.runtime_session_id,
            task_type: "terminal",
            async_hint: "force",
            input: text,
            metadata: {
              "alter0.task.terminal_session_id": session.terminal_session_id,
              "alter0.task.terminal_interactive": "true",
              codex_sandbox: "danger-full-access",
              codex_workspace_mode: "repo-root"
            }
          })
        });
        const payload = await response.json().catch(() => ({}));
        if (!response.ok) {
          const message = typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`;
          throw new Error(message);
        }
        const newTaskID = normalizeText(payload?.task_id || "");
        if (newTaskID === "-") {
          throw new Error("task_id missing");
        }
        session.anchor_task_id = newTaskID;
        session.active_task_id = newTaskID;
        session.status = normalizeText(payload?.status || "queued").toLowerCase();
        ensureTaskTracked(session, newTaskID);
        appendEntry(session, "system", `accepted ${newTaskID}`, newTaskID, { kind: "action", stage: "accept" });
      } else {
        const anchorTaskID = String(session.anchor_task_id || "").trim() || currentTaskID;
        const response = await fetch(`/api/control/tasks/${encodeURIComponent(currentTaskID)}/terminal/input`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            input: text,
            reuse_task: true,
            anchor_task_id: anchorTaskID
          })
        });
        const payload = await response.json().catch(() => ({}));
        if (!response.ok) {
          if ((payload?.error_code || "") === "terminal_session_limit_reached") {
            const maxSessions = Number(payload?.max_sessions || 5);
            window.alert(t("route.terminal.limit_reached", { max: String(maxSessions) }));
            return;
          }
          const message = typeof payload?.error === "string" ? payload.error : `HTTP ${response.status}`;
          throw new Error(message);
        }
        const newTaskID = normalizeText(payload?.task_id || "");
        const anchorTask = normalizeText(payload?.anchor_task_id || anchorTaskID || newTaskID);
        if (newTaskID === "-") {
          throw new Error("task_id missing");
        }
        session.anchor_task_id = anchorTask === "-" ? session.anchor_task_id : anchorTask;
        session.active_task_id = newTaskID;
        session.status = normalizeText(payload?.status || "queued").toLowerCase();
        const terminalSessionID = normalizeText(payload?.terminal_session_id || "");
        if (terminalSessionID !== "-") {
          session.terminal_session_id = terminalSessionID;
        }
        ensureTaskTracked(session, newTaskID);
        appendEntry(session, "system", `accepted ${newTaskID}`, newTaskID, { kind: "action", stage: "accept" });
      }
      touchSession(session);
      persist();
      paint();
      await startPolling();
    } catch (error) {
      const message = error instanceof Error ? error.message : "unknown_error";
      appendEntry(session, "system", t("route.terminal.send_failed", { error: message }), session.active_task_id, { kind: "tag", stage: "error" });
      touchSession(session);
      persist();
      paint();
    } finally {
      localState.sending = false;
      paint();
      const inputNode = container.querySelector("[data-terminal-input]");
      if (inputNode) {
        inputNode.focus();
      }
      const chatNode = container.querySelector("[data-terminal-chat-screen]");
      if (chatNode && !chatNode.hidden) {
        chatNode.scrollTop = chatNode.scrollHeight;
      }
    }
  };

  paint();
  const initialSession = getActiveSession();
  if (initialSession && String(initialSession.active_task_id || "").trim()) {
    await startPolling();
  }

  container.onclick = (event) => {
    const target = event.target.closest("button");
    if (!target) {
      return;
    }
    if (target.hasAttribute("data-terminal-create")) {
      createNewTerminalSession();
      paint();
      return;
    }
    if (target.hasAttribute("data-terminal-log-toggle")) {
      const active = getActiveSession();
      if (active) {
        active.log_collapsed = !resolveTerminalLogCollapsed(active);
        persist();
        paint();
      }
      return;
    }
    if (target.hasAttribute("data-terminal-session-select")) {
      const sessionID = normalizeText(target.getAttribute("data-terminal-session-select"));
      if (sessionID !== "-") {
        localState.activeSessionID = sessionID;
        paint();
        const active = getActiveSession();
        if (active && String(active.active_task_id || "").trim()) {
          void startPolling();
        } else {
          stopPolling();
        }
      }
    }
  };

  container.onsubmit = async (event) => {
    const form = event.target.closest("[data-terminal-input-form]");
    if (!form) {
      return;
    }
    event.preventDefault();
    const inputNode = form.querySelector("[data-terminal-input]");
    const value = inputNode ? String(inputNode.value || "") : "";
    if (inputNode) {
      inputNode.value = "";
    }
    await sendTerminalInput(value);
  };
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
  if (["success", "done"].includes(status)) {
    return "status-success";
  }
  if (["queued", "running", "pending", "in_progress"].includes(status)) {
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

function renderTaskSummaryCards(payload) {
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
    const tags = Array.isArray(item?.tags) ? item.tags : [];
    const anchorID = taskSummaryAnchorID(taskID);
    return `<article class="task-summary-card" id="${escapeHTML(anchorID)}" data-task-summary-id="${escapeHTML(taskID)}">
      <header class="task-summary-head">
        <h5>${escapeHTML(taskID)}</h5>
        <span class="task-summary-status">${escapeHTML(formatTaskStatus(status))}</span>
      </header>
      <div class="task-summary-meta">
        <p><span>${t("field.task_type")}</span><strong>${escapeHTML(normalizeText(taskType))}</strong></p>
        <p><span>${t("field.goal")}</span><strong>${escapeHTML(normalizeText(goal))}</strong></p>
        <p><span>${t("field.result")}</span><strong>${escapeHTML(normalizeText(result))}</strong></p>
        <p><span>${t("field.finished")}</span><strong>${escapeHTML(formatDateTime(finishedAt))}</strong></p>
        <p><span>${t("field.tags")}</span><strong>${escapeHTML(tags.length ? tags.join(", ") : "-")}</strong></p>
      </div>
      <button class="task-summary-open" type="button" data-task-open="${escapeHTML(taskID)}">${t("route.memory.tasks.open_detail")}</button>
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
  const refsList = Array.isArray(refs) ? refs : [];
  const refsBody = refsList.length
    ? `<ul class="task-detail-refs">
      ${refsList.map((item) => `<li><strong>${escapeHTML(normalizeText(item.tier))}</strong><span>${escapeHTML(normalizeText(item.date))}</span><code>${escapeHTML(normalizeText(item.path))}</code></li>`).join("")}
    </ul>`
    : `<p class="route-empty">-</p>`;
  return `<section class="task-detail-card" data-task-detail-id="${escapeHTML(taskID)}">
    <header class="task-detail-head">
      <h5>${escapeHTML(taskID)}</h5>
      <span class="task-summary-status">${escapeHTML(formatTaskStatus(status))}</span>
    </header>
    <div class="task-detail-meta route-meta">
      ${routeFieldRow("field.task_type", taskType)}
      ${routeFieldRow("field.session", meta?.session_id, { copyable: true, mono: true })}
      ${routeFieldRow("field.source_message", meta?.source_message_id, { copyable: true, mono: true })}
      ${routeFieldRow("field.progress", meta?.progress)}
      ${routeFieldRow("field.retry_count", meta?.retry_count)}
      ${routeFieldRow("field.created", formatDateTime(meta?.created_at))}
      ${routeFieldRow("field.finished_at", formatDateTime(meta?.finished_at))}
    </div>
    <section class="task-detail-section">
      <h6>Summary Refs</h6>
      ${refsBody}
    </section>
    <div class="task-detail-actions">
      <button type="button" data-task-load-logs>${t("route.memory.tasks.logs.load")}</button>
      <button type="button" data-task-load-artifacts>${t("route.memory.tasks.artifacts.load")}</button>
      <button type="button" data-task-rebuild>${t("route.memory.tasks.rebuild")}</button>
      <button type="button" data-task-back>${t("route.memory.tasks.back")}</button>
    </div>
    <section class="task-detail-section">
      <h6>Logs</h6>
      <div class="task-detail-logs" data-task-logs>${t("route.memory.tasks.logs.empty")}</div>
    </section>
    <section class="task-detail-section">
      <h6>Artifacts</h6>
      <div class="task-detail-artifacts" data-task-artifacts>${t("route.memory.tasks.artifacts.empty")}</div>
    </section>
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
  const form = view.querySelector("[data-task-filter-form]");
  const state = {
    filters: { status: "", taskType: "", startAt: "", endAt: "" },
    page: 1,
    pageSize: 10,
    activeTaskID: "",
    nextLogCursor: 0
  };

  const paintList = (payload) => {
    listNode.innerHTML = renderTaskSummaryCards(payload);
    paginationNode.innerHTML = renderTaskSummaryPagination(payload);
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
      const anchor = document.getElementById(taskSummaryAnchorID(state.activeTaskID));
      if (anchor) {
        anchor.scrollIntoView({ behavior: "smooth", block: "center" });
      }
      return;
    }
  });

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
        <form class="task-filter-form page-filter-form page-filter-grid-2" data-task-filter-form>
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
  const currentValue = normalizeText(item?.value || "");
  const effectiveValue = normalizeText(item?.effective_value || "");
  const defaultValue = normalizeText(definition?.default_value || "");
  const pendingRestart = Boolean(item?.pending_restart);
  const hotReload = Boolean(definition?.hot_reload);
  const applyMode = formatEnvironmentApplyMode(definition?.apply_mode);
  const source = formatEnvironmentSource(item?.value_source);
  const validation = renderEnvironmentValidation(definition);
  const inputControl = renderEnvironmentInput(item);
  const pendingBadge = pendingRestart
    ? `<span class="environment-pending">${t("route.envs.pending_restart")}</span>`
    : "";
  return `<article class="route-card environment-item" data-environment-item="${escapeHTML(key)}">
    <header class="route-card-head environment-item-head">
      <div class="route-card-title-wrap environment-item-title">
        <div class="route-card-icon" aria-hidden="true">${routeTypeIcon("env")}</div>
        <div class="environment-item-title-copy">
          <h4 title="${escapeHTML(name)}">${escapeHTML(name)}</h4>
          <span class="environment-item-key">
            <code title="${escapeHTML(key)}">${escapeHTML(key)}</code>
            <button class="route-field-copy" type="button" data-copy-value="${escapeHTML(key)}" title="${escapeHTML(t("route.copy_value"))}" aria-label="${escapeHTML(t("route.copy_value"))}">${renderCopyIcon()}</button>
          </span>
        </div>
      </div>
      ${pendingBadge}
    </header>
    <label class="environment-input-row">
      <span>${t("route.envs.current_value")}</span>
      ${inputControl}
    </label>
    <div class="route-meta environment-meta">
      ${routeFieldRow("route.envs.default_value", defaultValue || "-", { multiline: true, mono: true, copyable: defaultValue && defaultValue !== "-" })}
      ${routeFieldRow("route.envs.effective_value", effectiveValue || "-", { multiline: true, mono: true, copyable: effectiveValue && effectiveValue !== "-" })}
      ${routeFieldRow("route.envs.apply_mode", applyMode)}
      ${routeFieldRow("route.envs.source", source)}
      ${routeFieldRow("route.envs.validation", validation, { multiline: true, mono: true })}
      ${routeFieldRow("route.envs.hot_reload", hotReload ? t("status.enabled") : t("status.disabled"))}
    </div>
    ${pendingRestart ? `<p class="environment-item-notice">${escapeHTML(t("route.envs.restart_notice", { keys: key }))}</p>` : ""}
    <input type="hidden" data-env-current-value="${escapeHTML(key)}" value="${escapeHTML(currentValue)}">
  </article>`;
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
      return `<article class="route-card environment-audit-item">
        <div class="route-meta">
          ${routeFieldRow("route.envs.audit.operator", normalizeText(item?.operator || "-"))}
          ${routeFieldRow("route.envs.audit.at", formatDateTime(item?.occurred_at))}
          ${routeFieldRow("route.envs.audit.requires_restart", item?.requires_restart ? t("status.enabled") : t("status.disabled"))}
        </div>
        ${changesBody}
      </article>`;
    }).join("")}
  </div>`;
}

async function loadEnvironmentsView(container) {
  const localState = { revealSensitive: false };

  const fetchEnvironments = async () => {
    const query = localState.revealSensitive ? "?reveal_sensitive=true" : "";
    const [configPayload, auditPayload] = await Promise.all([
      fetchJSON(`/api/control/environments${query}`),
      fetchJSON(`/api/control/environments/audits${query}`)
    ]);
    return {
      configItems: Array.isArray(configPayload?.items) ? configPayload.items : [],
      audits: Array.isArray(auditPayload?.items) ? auditPayload.items : []
    };
  };

  const paint = (configItems, audits, statusMessage = "") => {
    const revealButtonLabel = localState.revealSensitive ? t("route.envs.hide_sensitive") : t("route.envs.show_sensitive");
    container.innerHTML = `<section class="environment-view" data-environment-view>
      <form class="environment-form" data-environment-form>
        <div class="environment-toolbar route-card">
          <p class="environment-status" data-environment-status>${escapeHTML(statusMessage)}</p>
          <div class="task-filter-actions">
            <button type="button" data-environment-reveal>${escapeHTML(revealButtonLabel)}</button>
            <button type="button" data-environment-refresh>${t("route.envs.refresh")}</button>
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
    paint(payload.configItems, payload.audits, statusMessage);
    bindView();
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
    if (!form || !refreshButton || !revealButton) {
      return;
    }
    refreshButton.addEventListener("click", async () => {
      await reload("");
    });
    revealButton.addEventListener("click", async () => {
      localState.revealSensitive = !localState.revealSensitive;
      await reload("");
    });
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      await submitChanges(form);
    });
  };

  const initialPayload = await fetchEnvironments();
  paint(initialPayload.configItems, initialPayload.audits, "");
  bindView();
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
    syncRouteAction("");
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

  chatForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    await sendMessage(input.value);
  });

  input.addEventListener("input", updateCharCount);
  input.addEventListener("keydown", async (event) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      await sendMessage(input.value);
    }
  });

  newChatButton.addEventListener("click", startNewChatSession);
  if (mobileNewChatButton) {
    mobileNewChatButton.addEventListener("click", startNewChatSession);
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
    if (state.currentRoute !== "chat") {
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
    void renderRoute(parseHashRoute());
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
    updateKeyboardInset();
  });

  if (window.visualViewport) {
    window.visualViewport.addEventListener("resize", updateKeyboardInset);
    window.visualViewport.addEventListener("scroll", updateKeyboardInset);
  }

  input.addEventListener("focus", () => {
    updateKeyboardInset();
    if (isMobileViewport()) {
      requestAnimationFrame(() => {
        input.scrollIntoView({ block: "nearest", behavior: "smooth" });
      });
    }
  });

  input.addEventListener("blur", () => {
    window.setTimeout(updateKeyboardInset, 80);
  });

  bindSwipeClose(primaryNav, "nav-open");
  bindSwipeClose(sessionPane, "panel-open");
}

function init() {
  setSidebarCollapsed(false);
  setSessionHistoryCollapsed(loadSessionHistoryCollapsedState());
  bootstrapSessions();
  renderSessions();
  renderMessages();
  syncHeader();
  bindEvents();
  updateCharCount();
  updateKeyboardInset();
  void renderRoute(parseHashRoute());
  input.focus();
}

init();
