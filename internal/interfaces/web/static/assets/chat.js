const appShell = document.getElementById("appShell");
const sessionList = document.getElementById("sessionList");
const sessionEmpty = document.getElementById("sessionEmpty");
const sessionLoadError = document.getElementById("sessionLoadError");
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
const I18N = {
  en: {
    // Navigation
    "nav.chat": "Chat",
    "nav.channels": "Channels",
    "nav.sessions": "Sessions",
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
    "field.type": "Type",
    "field.description": "Description",
    "field.name": "Name",
    "field.scope": "Scope",
    "field.version": "Version",
    "field.interval": "Interval",
    "field.session": "Session",
    "field.id": "ID",
    "field.messages": "Messages",
    "field.created": "Created",
    "field.path": "Path",
    "field.updated": "Updated",
    "field.date": "Date",
    "field.read_only": "Mode",
    
    // Routes
    "route.chat.title": "Chat",
    "route.chat.subtitle": "Ready to start a new conversation",
    "route.channels.title": "Channels",
    "route.channels.subtitle": "Manage connection channels",
    "route.channels.empty": "No Channels available.",
    "route.sessions.title": "Sessions",
    "route.sessions.subtitle": "View current session list",
    "route.sessions.empty": "No local sessions available.",
    "route.cron.title": "Cron Jobs",
    "route.cron.subtitle": "View scheduled jobs",
    "route.cron.empty": "No Cron Jobs available.",
    "route.memory.title": "Memory",
    "route.memory.subtitle": "Read-only memory view for long-term, daily, SOUL.md and specification",
    "route.memory.tab.long_term": "Long-Term",
    "route.memory.tab.daily": "Daily",
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
    "route.connected": "Page Connected",
    "route.connected_desc": "This page route is active. Content can be expanded by module.",
    "loading": "Loading...",
    "load_failed": "Load failed: {error}"
  },
  zh: {
    // Navigation
    "nav.chat": "对话",
    "nav.channels": "通道",
    "nav.sessions": "会话列表",
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
    "field.type": "类型",
    "field.description": "描述",
    "field.name": "名称",
    "field.scope": "范围",
    "field.version": "版本",
    "field.interval": "间隔",
    "field.session": "会话",
    "field.id": "ID",
    "field.messages": "消息数",
    "field.created": "创建时间",
    "field.path": "路径",
    "field.updated": "更新时间",
    "field.date": "日期",
    "field.read_only": "模式",
    
    // Routes
    "route.chat.title": "对话",
    "route.chat.subtitle": "准备好开始新的对话",
    "route.channels.title": "通道",
    "route.channels.subtitle": "管理连接通道",
    "route.channels.empty": "暂无可用通道。",
    "route.sessions.title": "会话列表",
    "route.sessions.subtitle": "查看当前会话列表",
    "route.sessions.empty": "暂无本地会话。",
    "route.cron.title": "定时任务",
    "route.cron.subtitle": "查看计划任务",
    "route.cron.empty": "暂无定时任务。",
    "route.memory.title": "记忆",
    "route.memory.subtitle": "统一只读查看长期记忆、天级记忆、SOUL.md 与说明文档",
    "route.memory.tab.long_term": "长期记忆",
    "route.memory.tab.daily": "天级记忆",
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
    loader: loadPlaceholderView
  }
};

const state = {
  activeSessionID: "",
  currentRoute: DEFAULT_ROUTE,
  sessions: [],
  sessionLoadError: "",
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
  const text = typeof value === "string" ? value.trim() : "";
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

function routeFieldRow(labelKey, value) {
  return `<p><span>${t(labelKey)}</span><strong>${escapeHTML(normalizeText(value))}</strong></p>`;
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
      <div class="route-card-icon" aria-hidden="true">${routeTypeIcon(type)}</div>
      <h4>${escapeHTML(normalizeText(title))}</h4>
    </div>
    ${routeStatusBadge(enabled)}
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
        routeFieldRow("field.type", item.type),
        routeFieldRow("field.description", item.description)
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
        routeFieldRow("field.type", item.type),
        routeFieldRow("field.name", item.name),
        routeFieldRow("field.scope", item.scope),
        routeFieldRow("field.version", item.version)
      ],
      item.enabled
    )
  );
}

async function loadCronJobsView(container) {
  const data = await fetchJSON("/api/control/cron/jobs");
  const items = Array.isArray(data.items) ? data.items : [];
  container.innerHTML = renderRouteCards(
    items,
    t("route.cron.empty"),
    (item) => routeCardTemplate(
      item.id,
      "cron",
      [
        routeFieldRow("field.interval", item.interval),
        routeFieldRow("field.session", item.session_id)
      ],
      item.enabled
    )
  );
}

async function loadSessionsView(container) {
  const items = state.sessions;
  container.innerHTML = renderRouteCards(
    items,
    t("route.sessions.empty"),
    (item) => routeCardTemplate(
      item.title,
      "session",
      [
        routeFieldRow("field.id", item.id),
        routeFieldRow("field.messages", item.messages.length),
        routeFieldRow("field.created", formatSince(item.createdAt))
      ],
      true
    )
  );
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
    routeFieldRow("field.path", path || "-"),
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
    routeFieldRow("field.path", path || "-"),
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
      routeFieldRow("field.path", path || "-"),
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
  activate("long_term");
}

async function loadMemoryView(container) {
  const payload = await fetchJSON("/api/agent/memory");
  const tabs = [
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
        [routeFieldRow("route.memory.daily.source", dailySourceDir)],
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
