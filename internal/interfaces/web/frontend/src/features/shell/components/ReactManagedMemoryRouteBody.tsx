import { useEffect, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTime } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { RouteCard, RouteFieldRow, RouteMarkdownContent } from "./RouteBodyPrimitives";

type MemoryDocument = {
  exists?: boolean;
  path?: string;
  updated_at?: string;
  content?: string;
  error?: string;
};

type DailyMemoryItem = MemoryDocument & {
  date?: string;
};

type MemoryPayload = {
  long_term?: MemoryDocument;
  daily?: {
    directory?: string;
    items?: DailyMemoryItem[];
    error?: string;
  };
  root_instructions?: MemoryDocument;
  mandatory?: MemoryDocument;
  specification?: MemoryDocument;
};

type MemoryTab = "long_term" | "daily" | "root_instructions" | "mandatory" | "specification";

type Copy = {
  loading: string;
  longTerm: string;
  daily: string;
  rootInstructions: string;
  mandatory: string;
  specification: string;
  readOnly: string;
  noLongTerm: string;
  noDaily: string;
  noRootInstructions: string;
  noMandatory: string;
  noSpecification: string;
  sourceDirectory: string;
  summary: string;
  path: string;
  updated: string;
  date: string;
  loadFailed: (message: string) => string;
};

const COPY: Record<LegacyShellLanguage, Copy> = {
  en: {
    loading: "Loading...",
    longTerm: "Long-Term",
    daily: "Daily",
    rootInstructions: "AGENTS.md",
    mandatory: "SOUL.md",
    specification: "Specification",
    readOnly: "Read-only",
    noLongTerm: "No long-term memory file available.",
    noDaily: "No daily memory files available.",
    noRootInstructions: "No AGENTS.md file available.",
    noMandatory: "No SOUL.md file available.",
    noSpecification: "No memory specification document available.",
    sourceDirectory: "Source Directory",
    summary: "Summary",
    path: "Path",
    updated: "Updated",
    date: "Date",
    loadFailed: (message) => `Load failed: ${message}`,
  },
  zh: {
    loading: "加载中...",
    longTerm: "长期记忆",
    daily: "天级记忆",
    rootInstructions: "AGENTS.md",
    mandatory: "SOUL.md",
    specification: "说明文档",
    readOnly: "只读",
    noLongTerm: "暂无长期记忆文件。",
    noDaily: "暂无天级记忆文件。",
    noRootInstructions: "暂无 AGENTS.md 文件。",
    noMandatory: "暂无 SOUL.md 文件。",
    noSpecification: "暂无记忆模块说明文档。",
    sourceDirectory: "来源目录",
    summary: "摘要",
    path: "路径",
    updated: "更新时间",
    date: "日期",
    loadFailed: (message) => `加载失败：${message}`,
  },
};

export function ReactManagedMemoryRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = COPY[language];
  const apiClient = createAPIClient();
  const [activeTab, setActiveTab] = useState<MemoryTab>("long_term");
  const [memoryPayload, setMemoryPayload] = useState<MemoryPayload | null>(null);
  const [requestState, setRequestState] = useState<{ status: "loading" | "ready" | "error"; error: string }>({
    status: "loading",
    error: "",
  });

  useEffect(() => {
    let disposed = false;

    async function loadInitial() {
      setRequestState({ status: "loading", error: "" });
      try {
        const memory = await apiClient.get<MemoryPayload>("/api/memory/context");
        if (disposed) {
          return;
        }
        setMemoryPayload(memory);
        setRequestState({ status: "ready", error: "" });
      } catch (error: unknown) {
        if (disposed) {
          return;
        }
        setRequestState({
          status: "error",
          error: error instanceof Error ? error.message : "unknown_error",
        });
      }
    }

    void loadInitial();
    return () => {
      disposed = true;
    };
  }, []);

  if (requestState.status === "loading") {
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (requestState.status === "error") {
    return <p className="route-error">{copy.loadFailed(requestState.error)}</p>;
  }

  return (
    <section className="memory-view">
      <div className="memory-tabs" role="tablist" aria-label="Memory">
        {[
          { id: "long_term" as const, label: copy.longTerm },
          { id: "daily" as const, label: copy.daily },
          { id: "root_instructions" as const, label: copy.rootInstructions },
          { id: "mandatory" as const, label: copy.mandatory },
          { id: "specification" as const, label: copy.specification },
        ].map((tab) => (
          <button
            key={tab.id}
            className={`memory-tab${activeTab === tab.id ? " active" : ""}`}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.id ? "true" : "false"}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === "long_term" ? (
        <section className="memory-panel active">
          <MemoryDocumentCard
            title={copy.longTerm}
            copy={copy}
            payload={memoryPayload?.long_term}
            empty={copy.noLongTerm}
          />
        </section>
      ) : null}

      {activeTab === "daily" ? (
        <section className="memory-panel active">
          <RouteCard
            title={copy.daily}
            type="memory"
            enabled={true}
            statusEnabledLabel={copy.readOnly}
            statusDisabledLabel={copy.readOnly}
          >
            <RouteFieldRow
              label={copy.sourceDirectory}
              value={memoryPayload?.daily?.directory}
              copyLabel="Copy value"
              copyable={true}
              mono={true}
              multiline={true}
            />
          </RouteCard>
          <div className="memory-daily-list">
            {renderDailyCards(memoryPayload?.daily?.items, copy)}
          </div>
        </section>
      ) : null}

      {activeTab === "mandatory" ? (
        <section className="memory-panel active">
          <MemoryDocumentCard
            title={copy.mandatory}
            copy={copy}
            payload={memoryPayload?.mandatory}
            empty={copy.noMandatory}
          />
        </section>
      ) : null}

      {activeTab === "root_instructions" ? (
        <section className="memory-panel active">
          <MemoryDocumentCard
            title={copy.rootInstructions}
            copy={copy}
            payload={memoryPayload?.root_instructions}
            empty={copy.noRootInstructions}
          />
        </section>
      ) : null}

      {activeTab === "specification" ? (
        <section className="memory-panel memory-panel-spec active">
          <MemorySpecificationCard copy={copy} payload={memoryPayload?.specification} />
        </section>
      ) : null}
    </section>
  );
}

function MemoryDocumentCard({
  title,
  copy,
  payload,
  empty,
}: {
  title: string;
  copy: Copy;
  payload?: MemoryDocument;
  empty: string;
}) {
  return (
    <RouteCard
      title={title}
      type="memory"
      enabled={true}
      statusEnabledLabel={copy.readOnly}
      statusDisabledLabel={copy.readOnly}
      body={
        payload?.error ? (
          <p className="route-error">{copy.loadFailed(payload.error)}</p>
        ) : payload?.exists && normalizeText(payload.content) ? (
          <RouteMarkdownContent className="memory-content" value={payload.content} />
        ) : (
          <p className="route-empty">{empty}</p>
        )
      }
    >
      <RouteFieldRow label={copy.path} value={payload?.path} copyLabel="Copy value" copyable={true} mono={true} multiline={true} />
      <RouteFieldRow label={copy.updated} value={formatDateTime(payload?.updated_at)} copyLabel="Copy value" />
      <RouteFieldRow label={copy.readOnly} value={copy.readOnly} copyLabel="Copy value" />
    </RouteCard>
  );
}

function MemorySpecificationCard({
  copy,
  payload,
}: {
  copy: Copy;
  payload?: MemoryDocument;
}) {
  const content = normalizeText(payload?.content);
  const sections = splitMarkdownSections(content);

  return (
    <RouteCard
      title={copy.specification}
      type="memory"
      enabled={true}
      statusEnabledLabel={copy.readOnly}
      statusDisabledLabel={copy.readOnly}
      body={
        payload?.error ? (
          <p className="route-error">{copy.loadFailed(payload.error)}</p>
        ) : payload?.exists && content ? (
          sections.length ? (
            <div className="memory-spec-sections">
              {sections.map((section) => (
                <section key={section.title} className="memory-spec-section">
                  <h5 className="memory-spec-title">{section.title}</h5>
                  <RouteMarkdownContent className="memory-content" value={section.content.trim()} />
                </section>
              ))}
            </div>
          ) : (
            <RouteMarkdownContent className="memory-content" value={content} />
          )
        ) : (
          <p className="route-empty">{copy.noSpecification}</p>
        )
      }
    >
      <RouteFieldRow label={copy.path} value={payload?.path} copyLabel="Copy value" copyable={true} mono={true} multiline={true} />
      <RouteFieldRow label={copy.updated} value={formatDateTime(payload?.updated_at)} copyLabel="Copy value" />
      <RouteFieldRow label={copy.readOnly} value={copy.readOnly} copyLabel="Copy value" />
    </RouteCard>
  );
}

function renderDailyCards(items: DailyMemoryItem[] | undefined, copy: Copy) {
  const dailyItems = Array.isArray(items) ? items : [];
  if (!dailyItems.length) {
    return <p className="route-empty">{copy.noDaily}</p>;
  }
  return dailyItems.map((item) => (
    <RouteCard
      key={normalizeText(item.date || item.path)}
      title={normalizeText(item.date)}
      type="memory"
      enabled={true}
      statusEnabledLabel={copy.readOnly}
      statusDisabledLabel={copy.readOnly}
      body={
        item.error ? (
          <p className="route-error">{copy.loadFailed(item.error)}</p>
        ) : normalizeText(item.content) ? (
          <>
            <p className="memory-summary"><span>{copy.summary}</span><strong>{summarizeMemoryContent(item.content)}</strong></p>
            <RouteMarkdownContent className="memory-content" value={item.content} />
          </>
        ) : (
          <p className="route-empty">{copy.noDaily}</p>
        )
      }
    >
      <RouteFieldRow label={copy.date} value={item.date} copyLabel="Copy value" />
      <RouteFieldRow label={copy.path} value={item.path} copyLabel="Copy value" copyable={true} mono={true} multiline={true} />
      <RouteFieldRow label={copy.updated} value={formatDateTime(item.updated_at)} copyLabel="Copy value" />
      <RouteFieldRow label={copy.readOnly} value={copy.readOnly} copyLabel="Copy value" />
    </RouteCard>
  ));
}

function splitMarkdownSections(content: string) {
  const text = normalizeText(content).replace(/\r\n/g, "\n");
  if (!text) {
    return [];
  }
  const rows = text.split("\n");
  const sections: Array<{ title: string; content: string }> = [];
  let current: { title: string; content: string } | null = null;

  for (const row of rows) {
    const heading = row.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      if (current) {
        sections.push(current);
      }
      current = { title: heading[2].trim(), content: "" };
      continue;
    }
    if (!current) {
      current = { title: "Overview", content: "" };
    }
    current.content += `${row}\n`;
  }

  if (current) {
    sections.push(current);
  }

  return sections;
}

function summarizeMemoryContent(content?: string) {
  const rows = normalizeText(content)
    .split("\n")
    .map((row) => row.trim())
    .filter(Boolean);
  return rows[0] || "-";
}

function normalizeText(value: unknown) {
  return String(value || "").trim();
}
