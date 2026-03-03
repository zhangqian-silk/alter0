const appShell = document.getElementById("appShell");
const sessionList = document.getElementById("sessionList");
const sessionEmpty = document.getElementById("sessionEmpty");
const welcomeScreen = document.getElementById("welcomeScreen");
const messageArea = document.getElementById("messageArea");
const chatForm = document.getElementById("chatForm");
const input = document.getElementById("composerInput");
const sendButton = document.getElementById("sendButton");
const charCount = document.getElementById("charCount");
const newChatButton = document.getElementById("newChatButton");
const navToggle = document.getElementById("navToggle");
const sessionToggle = document.getElementById("sessionToggle");
const togglePaneButton = document.getElementById("togglePaneButton");
const navCloseButton = document.getElementById("navCloseButton");
const mobileBackdrop = document.getElementById("mobileBackdrop");
const sessionHeading = document.getElementById("sessionHeading");
const sessionSubheading = document.getElementById("sessionSubheading");
const sessionPane = document.querySelector(".session-pane");
const primaryNav = document.querySelector(".primary-nav");
const chatPane = document.querySelector(".chat-pane");
const chatView = document.getElementById("chatView");
const routeView = document.getElementById("routeView");
const routeTitle = document.getElementById("routeTitle");
const routeSubtitle = document.getElementById("routeSubtitle");
const routeBody = document.getElementById("routeBody");
const menuRouteItems = document.querySelectorAll(".menu-item[data-route]");

const MAX_CHARS = 10000;
const DEFAULT_ROUTE = "chat";

const ROUTES = {
  chat: {
    title: "Chat",
    subtitle: "准备好开始新的会话",
    mode: "chat"
  },
  channels: {
    title: "Channels",
    subtitle: "管理接入通道配置",
    mode: "page",
    loader: loadChannelsView
  },
  sessions: {
    title: "Sessions",
    subtitle: "查看当前会话列表",
    mode: "page",
    loader: loadSessionsView
  },
  "cron-jobs": {
    title: "Cron Jobs",
    subtitle: "查看调度任务配置",
    mode: "page",
    loader: loadCronJobsView
  },
  workspace: {
    title: "Workspace",
    subtitle: "工作区信息",
    mode: "page",
    loader: loadPlaceholderView
  },
  skills: {
    title: "Skills",
    subtitle: "技能配置列表",
    mode: "page",
    loader: loadSkillsView
  },
  mcp: {
    title: "MCP",
    subtitle: "Model Context Protocol 配置",
    mode: "page",
    loader: loadPlaceholderView
  },
  configuration: {
    title: "Configuration",
    subtitle: "运行时配置概览",
    mode: "page",
    loader: loadPlaceholderView
  },
  models: {
    title: "Models",
    subtitle: "模型能力入口",
    mode: "page",
    loader: loadPlaceholderView
  },
  environments: {
    title: "Environments",
    subtitle: "环境与部署设置",
    mode: "page",
    loader: loadPlaceholderView
  }
};

const state = {
  activeSessionID: "",
  currentRoute: DEFAULT_ROUTE,
  sessions: [],
  pending: false,
  pageRenderToken: 0
};

function makeID() {
  if (window.crypto && typeof window.crypto.randomUUID === "function") {
    return window.crypto.randomUUID();
  }
  return `session-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

function timeLabel(epochMillis = Date.now()) {
  return new Date(epochMillis).toLocaleTimeString("zh-CN", {
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

function formatSince(epochMillis) {
  const delta = Date.now() - epochMillis;
  const minutes = Math.floor(delta / 60000);
  if (minutes < 1) {
    return "刚刚";
  }
  if (minutes < 60) {
    return `${minutes} 分钟前`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours} 小时前`;
  }
  const days = Math.floor(hours / 24);
  return `${days} 天前`;
}

function createSession() {
  const createdAt = Date.now();
  const item = {
    id: makeID(),
    title: "新对话",
    createdAt,
    messages: []
  };
  state.sessions.unshift(item);
  state.activeSessionID = item.id;
  renderSessions();
  renderMessages();
  syncHeader();
  return item;
}

function syncHeader() {
  const route = ROUTES[state.currentRoute] || ROUTES.chat;
  if (route.mode !== "chat") {
    sessionHeading.textContent = route.title;
    sessionSubheading.textContent = route.subtitle;
    return;
  }

  const active = getSession();
  if (!active) {
    sessionHeading.textContent = route.title;
    sessionSubheading.textContent = route.subtitle;
    return;
  }
  sessionHeading.textContent = active.title;
  sessionSubheading.textContent = `${active.messages.length} messages`;
}

function renderSessions() {
  sessionList.innerHTML = "";
  if (!state.sessions.length) {
    sessionEmpty.style.display = "block";
    return;
  }
  sessionEmpty.style.display = "none";

  for (const item of state.sessions) {
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
      state.activeSessionID = item.id;
      navigateToRoute("chat");
      renderSessions();
      renderMessages();
      syncHeader();
      closeTransientPanels();
    });
    sessionList.appendChild(card);
  }
}

function updateSessionTitle(session, fallbackText) {
  if (session.title !== "新对话") {
    return;
  }
  const text = fallbackText.trim();
  if (!text) {
    return;
  }
  session.title = shorten(text, 18);
}

function appendMessage(role, text, options = {}) {
  let session = getSession();
  if (!session) {
    session = createSession();
  }
  if (role === "user") {
    updateSessionTitle(session, text);
  }
  session.messages.push({
    role,
    text,
    at: Date.now(),
    route: options.route || "",
    error: Boolean(options.error)
  });
  renderSessions();
  renderMessages();
  syncHeader();
}

function renderMessages() {
  const active = getSession();
  const hasMessages = Boolean(active && active.messages.length);
  welcomeScreen.style.display = hasMessages ? "none" : "block";
  messageArea.style.display = hasMessages ? "block" : "none";
  chatPane.classList.toggle("empty-state", !hasMessages);

  if (!hasMessages) {
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

function renderTyping(show) {
  if (!show) {
    const exists = document.getElementById("typingPlaceholder");
    if (exists) {
      exists.remove();
    }
    return;
  }
  const root = messageArea.querySelector(".message-list");
  if (!root) {
    return;
  }
  if (document.getElementById("typingPlaceholder")) {
    return;
  }

  const item = document.createElement("article");
  item.id = "typingPlaceholder";
  item.className = "msg assistant typing";

  const bubble = document.createElement("div");
  bubble.className = "msg-bubble";
  bubble.textContent = "处理中...";

  const meta = document.createElement("div");
  meta.className = "msg-meta";
  const time = document.createElement("span");
  time.textContent = timeLabel();
  meta.appendChild(time);

  item.appendChild(bubble);
  item.appendChild(meta);
  root.appendChild(item);
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
  renderTyping(true);

  const active = getSession();
  const payload = {
    session_id: active ? active.id : "",
    channel_id: "web-default",
    content
  };

  try {
    const response = await fetch("/api/messages", {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(payload)
    });

    let body = {};
    try {
      body = await response.json();
    } catch {
      body = {};
    }

    if (!response.ok) {
      const failure = body.error || body?.result?.error_code || `HTTP ${response.status}`;
      appendMessage("assistant", `请求失败：${failure}`, { error: true, route: body?.result?.route });
      return;
    }

    const output = (body?.result?.output || "").trim() || "已收到请求，但当前没有输出内容。";
    appendMessage("assistant", output, { route: body?.result?.route });
  } catch (err) {
    const message = err instanceof Error ? err.message : "未知网络异常";
    appendMessage("assistant", `网络异常：${message}`, { error: true });
  } finally {
    renderTyping(false);
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

function closeTransientPanels() {
  appShell.classList.remove("nav-open");
  appShell.classList.remove("panel-open");
}

function navigateToRoute(route) {
  const safe = ROUTES[route] ? route : DEFAULT_ROUTE;
  const targetHash = `#${safe}`;
  if (window.location.hash !== targetHash) {
    window.location.hash = targetHash;
    return;
  }
  void renderRoute(safe);
}

function renderRouteCards(items, emptyText, renderItem) {
  if (!items.length) {
    return `<p class="route-empty">${emptyText}</p>`;
  }
  return items.map((item) => renderItem(item)).join("");
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
    "暂无 Channel。",
    (item) => `<article class="route-card">
      <h4>${item.id}</h4>
      <p>Type: ${item.type || "-"}</p>
      <p>Enabled: ${item.enabled ? "true" : "false"}</p>
      <p>Description: ${item.description || "-"}</p>
    </article>`
  );
}

async function loadSkillsView(container) {
  const data = await fetchJSON("/api/control/skills");
  const items = Array.isArray(data.items) ? data.items : [];
  container.innerHTML = renderRouteCards(
    items,
    "暂无 Skill。",
    (item) => `<article class="route-card">
      <h4>${item.id}</h4>
      <p>Name: ${item.name || "-"}</p>
      <p>Version: ${item.version || "-"}</p>
      <p>Enabled: ${item.enabled ? "true" : "false"}</p>
    </article>`
  );
}

async function loadCronJobsView(container) {
  const data = await fetchJSON("/api/control/cron/jobs");
  const items = Array.isArray(data.items) ? data.items : [];
  container.innerHTML = renderRouteCards(
    items,
    "暂无 Cron Job。",
    (item) => `<article class="route-card">
      <h4>${item.id}</h4>
      <p>Interval: ${item.interval || "-"}</p>
      <p>Session: ${item.session_id || "-"}</p>
      <p>Enabled: ${item.enabled ? "true" : "false"}</p>
    </article>`
  );
}

async function loadSessionsView(container) {
  const items = state.sessions;
  container.innerHTML = renderRouteCards(
    items,
    "暂无本地会话。",
    (item) => `<article class="route-card">
      <h4>${item.title}</h4>
      <p>ID: ${item.id}</p>
      <p>Messages: ${item.messages.length}</p>
      <p>Created: ${formatSince(item.createdAt)}</p>
    </article>`
  );
}

async function loadPlaceholderView(container) {
  container.innerHTML = `<article class="route-card">
    <h4>页面已接入路由</h4>
    <p>该页面入口已可跳转，业务内容可按模块逐步扩展。</p>
  </article>`;
}

async function renderRoute(route) {
  const safe = ROUTES[route] ? route : DEFAULT_ROUTE;
  state.currentRoute = safe;
  activeMenuRoute(safe);
  if (isMobileViewport()) {
    closeTransientPanels();
  } else {
    appShell.classList.remove("nav-open");
  }

  const config = ROUTES[safe];
  if (config.mode === "chat") {
    appShell.classList.remove("route-mode");
    chatView.hidden = false;
    routeView.hidden = true;
    syncHeader();
    return;
  }

  appShell.classList.add("route-mode");
  closeTransientPanels();
  chatView.hidden = true;
  routeView.hidden = false;
  routeTitle.textContent = config.title;
  routeSubtitle.textContent = config.subtitle;
  routeBody.innerHTML = '<p class="route-loading">Loading...</p>';
  syncHeader();

  const token = ++state.pageRenderToken;
  try {
    await config.loader(routeBody);
  } catch (err) {
    if (token !== state.pageRenderToken) {
      return;
    }
    const message = err instanceof Error ? err.message : "unknown error";
    routeBody.innerHTML = `<p class="route-error">加载失败：${message}</p>`;
  }
}

function bindEvents() {
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

  newChatButton.addEventListener("click", () => {
    createSession();
    navigateToRoute("chat");
    input.focus();
    closeTransientPanels();
  });

  for (const node of menuRouteItems) {
    node.addEventListener("click", () => {
      const route = node.dataset.route || DEFAULT_ROUTE;
      navigateToRoute(route);
    });
  }

  navToggle.addEventListener("click", () => {
    const open = !appShell.classList.contains("nav-open");
    closeTransientPanels();
    if (open) {
      appShell.classList.add("nav-open");
    }
  });

  sessionToggle.addEventListener("click", () => {
    if (state.currentRoute !== "chat") {
      return;
    }
    const open = !appShell.classList.contains("panel-open");
    closeTransientPanels();
    if (open) {
      appShell.classList.add("panel-open");
    }
  });

  togglePaneButton.addEventListener("click", () => {
    appShell.classList.remove("panel-open");
  });

  navCloseButton.addEventListener("click", () => {
    appShell.classList.remove("nav-open");
  });

  mobileBackdrop.addEventListener("click", () => {
    closeTransientPanels();
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
    if (!isMobileViewport()) {
      closeTransientPanels();
    }
  });
}

function init() {
  createSession();
  bindEvents();
  updateCharCount();
  void renderRoute(parseHashRoute());
  input.focus();
}

init();
