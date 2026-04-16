import { memo } from "react";
import { LEGACY_SHELL_IDS } from "../legacyDomContract";
import {
  LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT,
  requestLegacyChatRuntimeClose,
  requestLegacyChatRuntimeItem,
  requestLegacyChatRuntimeModel,
  requestLegacyChatRuntimePopover,
  requestLegacyChatRuntimeTarget,
  type LegacyShellChatRuntimeDetail,
  type LegacyShellChatRuntimeModelDetail,
  type LegacyShellChatRuntimeProviderDetail,
  type LegacyShellChatRuntimeSelectionDetail,
  type LegacyShellChatRuntimeTargetDetail,
  type LegacyShellChatRuntimeTargetOptionDetail,
} from "../legacyShellBridge";
import { getLegacyShellCopy, type LegacyShellLanguage } from "../legacyShellCopy";
import { useLegacyShellSnapshot } from "../legacyShellSnapshot";

type ChatRuntimeSnapshot = {
  route: string;
  compact: boolean;
  openPopover: string;
  note: string;
  agentRuntime: boolean;
  locked: boolean;
  target: ChatRuntimeTargetSnapshot;
  targetOptions: ChatRuntimeTargetOptionSnapshot[];
  selectedProviderId: string;
  selectedModelId: string;
  selectedModelLabel: string;
  toolCount: number;
  skillCount: number;
  providers: ChatRuntimeProviderSnapshot[];
  capabilities: ChatRuntimeSelectionSnapshot[];
  skills: ChatRuntimeSelectionSnapshot[];
};

type ChatRuntimeTargetSnapshot = {
  type: string;
  id: string;
  name: string;
};

type ChatRuntimeTargetOptionSnapshot = {
  type: string;
  id: string;
  name: string;
  subtitle: string;
  active: boolean;
};

type ChatRuntimeProviderSnapshot = {
  id: string;
  name: string;
  models: ChatRuntimeModelSnapshot[];
};

type ChatRuntimeModelSnapshot = {
  id: string;
  name: string;
  active: boolean;
};

type ChatRuntimeSelectionSnapshot = {
  id: string;
  name: string;
  description: string;
  kind: "tool" | "mcp" | "skill";
  active: boolean;
};

function useLegacyChatRuntimeSnapshot(currentRoute: string): ChatRuntimeSnapshot {
  return useLegacyShellSnapshot<LegacyShellChatRuntimeDetail, ChatRuntimeSnapshot>({
    currentRoute,
    eventName: LEGACY_SHELL_SYNC_CHAT_RUNTIME_EVENT,
    fallback: () => ({
      route: currentRoute,
      compact: false,
      openPopover: "",
      note: "",
      agentRuntime: false,
      locked: false,
      target: { type: "model", id: "raw-model", name: "Raw Model" },
      targetOptions: [],
      selectedProviderId: "",
      selectedModelId: "",
      selectedModelLabel: "",
      toolCount: 0,
      skillCount: 0,
      providers: [],
      capabilities: [],
      skills: [],
    }),
    normalizeDetail: normalizeChatRuntimeSnapshot,
  });
}

function normalizeChatRuntimeSnapshot(
  detail: LegacyShellChatRuntimeDetail,
): ChatRuntimeSnapshot | null {
  if (!detail || typeof detail.route !== "string") {
    return null;
  }

  return {
    route: detail.route,
    compact: Boolean(detail.compact),
    openPopover: typeof detail.openPopover === "string" ? detail.openPopover : "",
    note: typeof detail.note === "string" ? detail.note : "",
    agentRuntime: Boolean(detail.agentRuntime),
    locked: Boolean(detail.locked),
    target: normalizeRuntimeTarget(detail.target),
    targetOptions: Array.isArray(detail.targetOptions)
      ? detail.targetOptions.map(normalizeRuntimeTargetOption).filter((item) => item !== null)
      : [],
    selectedProviderId:
      typeof detail.selectedProviderId === "string" ? detail.selectedProviderId : "",
    selectedModelId: typeof detail.selectedModelId === "string" ? detail.selectedModelId : "",
    selectedModelLabel:
      typeof detail.selectedModelLabel === "string" ? detail.selectedModelLabel : "",
    toolCount: Number.isFinite(detail.toolCount) ? Math.max(Number(detail.toolCount), 0) : 0,
    skillCount: Number.isFinite(detail.skillCount) ? Math.max(Number(detail.skillCount), 0) : 0,
    providers: Array.isArray(detail.providers)
      ? detail.providers.map(normalizeRuntimeProvider).filter((item) => item !== null)
      : [],
    capabilities: Array.isArray(detail.capabilities)
      ? detail.capabilities.map(normalizeRuntimeSelection).filter((item) => item !== null)
      : [],
    skills: Array.isArray(detail.skills)
      ? detail.skills.map(normalizeRuntimeSelection).filter((item) => item !== null)
      : [],
  };
}

function normalizeRuntimeTarget(detail?: LegacyShellChatRuntimeTargetDetail): ChatRuntimeTargetSnapshot {
  const type = typeof detail?.type === "string" && detail.type === "agent" ? "agent" : "model";
  const id = typeof detail?.id === "string" && detail.id.trim() ? detail.id : type === "agent" ? "" : "raw-model";
  const name = typeof detail?.name === "string" && detail.name.trim() ? detail.name : id;
  return { type, id, name };
}

function normalizeRuntimeTargetOption(
  detail?: LegacyShellChatRuntimeTargetOptionDetail,
): ChatRuntimeTargetOptionSnapshot | null {
  const target = normalizeRuntimeTarget(detail);
  if (!target.id) {
    return null;
  }
  return {
    ...target,
    subtitle: typeof detail?.subtitle === "string" ? detail.subtitle : "",
    active: Boolean(detail?.active),
  };
}

function normalizeRuntimeProvider(
  detail?: LegacyShellChatRuntimeProviderDetail,
): ChatRuntimeProviderSnapshot | null {
  const id = typeof detail?.id === "string" ? detail.id.trim() : "";
  if (!id) {
    return null;
  }
  return {
    id,
    name: typeof detail?.name === "string" && detail.name.trim() ? detail.name : id,
    models: Array.isArray(detail?.models)
      ? detail.models.map(normalizeRuntimeModel).filter((item) => item !== null)
      : [],
  };
}

function normalizeRuntimeModel(
  detail?: LegacyShellChatRuntimeModelDetail,
): ChatRuntimeModelSnapshot | null {
  const id = typeof detail?.id === "string" ? detail.id.trim() : "";
  if (!id) {
    return null;
  }
  return {
    id,
    name: typeof detail?.name === "string" && detail.name.trim() ? detail.name : id,
    active: Boolean(detail?.active),
  };
}

function normalizeRuntimeSelection(
  detail?: LegacyShellChatRuntimeSelectionDetail,
): ChatRuntimeSelectionSnapshot | null {
  const id = typeof detail?.id === "string" ? detail.id.trim() : "";
  if (!id) {
    return null;
  }
  const kind =
    detail?.kind === "mcp" || detail?.kind === "skill" || detail?.kind === "tool"
      ? detail.kind
      : "tool";
  return {
    id,
    name: typeof detail?.name === "string" && detail.name.trim() ? detail.name : id,
    description: typeof detail?.description === "string" ? detail.description : "",
    kind,
    active: Boolean(detail?.active),
  };
}

function groupCapabilitySelections(items: ChatRuntimeSelectionSnapshot[]) {
  return {
    activeTools: items.filter((item) => item.active && item.kind === "tool"),
    activeMcps: items.filter((item) => item.active && item.kind === "mcp"),
    availableTools: items.filter((item) => !item.active && item.kind === "tool"),
    availableMcps: items.filter((item) => !item.active && item.kind === "mcp"),
  };
}

function RuntimeSectionTitle({
  title,
  count,
}: {
  title: string;
  count: number;
}) {
  return (
    <div className="composer-runtime-group-title">
      <strong>{title}</strong>
      <span>{count}</span>
    </div>
  );
}

function RuntimeEmpty({ text }: { text: string }) {
  return <p className="composer-runtime-empty">{text}</p>;
}

function RuntimeSelectionCheckbox({
  item,
  group,
}: {
  item: ChatRuntimeSelectionSnapshot;
  group: "capabilities" | "skills";
}) {
  return (
    <label className="composer-runtime-checkbox">
      <input
        type="checkbox"
        data-runtime-toggle-item={group}
        value={item.id}
        checked={item.active}
        onChange={(event) => {
          requestLegacyChatRuntimeItem({
            group,
            id: item.id,
            checked: event.currentTarget.checked,
            kind: item.kind === "skill" ? undefined : item.kind,
          });
        }}
      />
      <span className="composer-runtime-checkbox-copy">
        <strong>{item.name}</strong>
        <span>{item.description}</span>
      </span>
    </label>
  );
}

function RuntimeCapabilitySections({
  items,
  language,
}: {
  items: ChatRuntimeSelectionSnapshot[];
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  const grouped = groupCapabilitySelections(items);
  const sections = [
    { title: copy.runtimeActive, tools: grouped.activeTools, mcps: grouped.activeMcps },
    { title: copy.runtimeAvailable, tools: grouped.availableTools, mcps: grouped.availableMcps },
  ];

  return (
    <>
      {sections.map((section, index) => (
        <div key={section.title}>
          {index > 0 ? <div className="composer-runtime-separator"></div> : null}
          {!section.tools.length && !section.mcps.length ? (
            <RuntimeEmpty text={copy.runtimeNone} />
          ) : (
            <section className="composer-runtime-section">
              <RuntimeSectionTitle
                title={section.title}
                count={section.tools.length + section.mcps.length}
              />
              {section.tools.length ? (
                <section className="composer-runtime-section">
                  <RuntimeSectionTitle
                    title={copy.runtimeCategoryTools}
                    count={section.tools.length}
                  />
                  <div className="composer-runtime-checkbox-group">
                    {section.tools.map((item) => (
                      <RuntimeSelectionCheckbox key={item.id} item={item} group="capabilities" />
                    ))}
                  </div>
                </section>
              ) : null}
              {section.mcps.length ? (
                <section className="composer-runtime-section">
                  <RuntimeSectionTitle title={copy.runtimeCategoryMcps} count={section.mcps.length} />
                  <div className="composer-runtime-checkbox-group">
                    {section.mcps.map((item) => (
                      <RuntimeSelectionCheckbox key={item.id} item={item} group="capabilities" />
                    ))}
                  </div>
                </section>
              ) : null}
            </section>
          )}
        </div>
      ))}
    </>
  );
}

function RuntimeSkillSections({
  items,
  language,
}: {
  items: ChatRuntimeSelectionSnapshot[];
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  const active = items.filter((item) => item.active);
  const available = items.filter((item) => !item.active);

  return (
    <>
      {active.length ? (
        <section className="composer-runtime-section">
          <RuntimeSectionTitle title={copy.runtimeActive} count={active.length} />
          {active.map((item) => (
            <RuntimeSelectionCheckbox key={item.id} item={item} group="skills" />
          ))}
        </section>
      ) : (
        <RuntimeEmpty text={copy.runtimeNone} />
      )}
      <div className="composer-runtime-separator"></div>
      {available.length ? (
        <section className="composer-runtime-section">
          <RuntimeSectionTitle title={copy.runtimeAvailable} count={available.length} />
          {available.map((item) => (
            <RuntimeSelectionCheckbox key={item.id} item={item} group="skills" />
          ))}
        </section>
      ) : (
        <RuntimeEmpty text={copy.runtimeNone} />
      )}
    </>
  );
}

function RuntimeTargetPopover({
  snapshot,
  language,
}: {
  snapshot: ChatRuntimeSnapshot;
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  return (
    <div className="composer-runtime-popover" data-runtime-scroll-container="target">
      <div className="composer-runtime-popover-head">
        <strong>{copy.runtimeAgent}</strong>
        <p>{snapshot.locked ? copy.runtimeLocked : copy.runtimeAgentHint}</p>
      </div>
      <div className="composer-runtime-option-list">
        {snapshot.targetOptions.length ? (
          snapshot.targetOptions.map((item) => (
            <button
              key={item.id}
              className={item.active ? "composer-runtime-option is-active" : "composer-runtime-option"}
              type="button"
              data-runtime-target-type={item.type}
              data-runtime-target-id={item.id}
              data-runtime-target-name={item.name}
              disabled={snapshot.locked}
              onClick={() => {
                requestLegacyChatRuntimeTarget({
                  type: item.type,
                  id: item.id,
                  name: item.name,
                });
              }}
            >
              <strong>{item.name}</strong>
              <span className="composer-runtime-option-summary">{item.subtitle}</span>
            </button>
          ))
        ) : (
          <RuntimeEmpty text={copy.runtimeNone} />
        )}
      </div>
    </div>
  );
}

function RuntimeModelPopover({
  snapshot,
  language,
}: {
  snapshot: ChatRuntimeSnapshot;
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  return (
    <div className="composer-runtime-popover is-wide" data-runtime-scroll-container="model">
      <div className="composer-runtime-popover-head">
        <strong>{`${copy.runtimeProvider} / ${copy.runtimeModel}`}</strong>
        <p>{snapshot.note || copy.runtimeModelHint}</p>
      </div>
      {snapshot.providers.length ? (
        snapshot.providers.map((provider) => (
          <section key={provider.id} className="composer-runtime-provider-group">
            <RuntimeSectionTitle title={provider.name} count={provider.models.length} />
            <div className="composer-runtime-option-list">
              {provider.models.map((model) => (
                <button
                  key={`${provider.id}:${model.id}`}
                  className={
                    model.active ? "composer-runtime-model-option is-active" : "composer-runtime-model-option"
                  }
                  type="button"
                  data-runtime-provider-id={provider.id}
                  data-runtime-model-id={model.id}
                  onClick={() => {
                    requestLegacyChatRuntimeModel({
                      providerId: provider.id,
                      modelId: model.id,
                    });
                  }}
                >
                  <strong>{model.name}</strong>
                  <span>{provider.name}</span>
                </button>
              ))}
            </div>
          </section>
        ))
      ) : (
        <RuntimeEmpty text={copy.runtimeEmpty} />
      )}
    </div>
  );
}

function RuntimeCapabilitiesPopover({
  snapshot,
  language,
}: {
  snapshot: ChatRuntimeSnapshot;
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  return (
    <div className="composer-runtime-popover is-wide" data-runtime-scroll-container="capabilities">
      <div className="composer-runtime-popover-head">
        <strong>{copy.runtimeToolsMcp}</strong>
        <p>{snapshot.note || copy.runtimeToolsHint}</p>
      </div>
      <RuntimeCapabilitySections items={snapshot.capabilities} language={language} />
    </div>
  );
}

function RuntimeSkillsPopover({
  snapshot,
  language,
}: {
  snapshot: ChatRuntimeSnapshot;
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  return (
    <div className="composer-runtime-popover" data-runtime-scroll-container="skills">
      <div className="composer-runtime-popover-head">
        <strong>{copy.runtimeSkills}</strong>
        <p>{snapshot.note || copy.runtimeSkillsHint}</p>
      </div>
      <RuntimeSkillSections items={snapshot.skills} language={language} />
    </div>
  );
}

function RuntimeCompactMeta(snapshot: ChatRuntimeSnapshot, language: LegacyShellLanguage): string {
  const copy = getLegacyShellCopy(language);
  if (language === "zh") {
    return `模型 ${snapshot.selectedModelLabel || copy.runtimeServiceDefault} · 工具 ${snapshot.toolCount} · 技能 ${snapshot.skillCount}`;
  }
  return `Model ${snapshot.selectedModelLabel || copy.runtimeServiceDefault} · Tools ${snapshot.toolCount} · Skills ${snapshot.skillCount}`;
}

function RuntimeSummaryChips({
  snapshot,
  language,
}: {
  snapshot: ChatRuntimeSnapshot;
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  const chips = [];
  if (snapshot.agentRuntime && snapshot.target.id) {
    chips.push(`${copy.runtimeAgent} · ${snapshot.target.name}`);
  }
  chips.push(`${copy.runtimeModelShort} · ${snapshot.selectedModelLabel || copy.runtimeServiceDefault}`);
  chips.push(`${copy.runtimeToolsShort} ${snapshot.toolCount}`);
  chips.push(`${copy.runtimeSkillsShort} ${snapshot.skillCount}`);

  return (
    <div className="composer-runtime-summary">
      {chips.map((item) => (
        <span key={item} className="composer-runtime-chip">{item}</span>
      ))}
    </div>
  );
}

function RuntimeNote({
  note,
}: {
  note: string;
}) {
  return note ? <p className="chat-runtime-note chat-runtime-error">{note}</p> : null;
}

function RuntimeDesktopControls({
  snapshot,
  language,
}: {
  snapshot: ChatRuntimeSnapshot;
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);
  const targetLabel = snapshot.agentRuntime
    ? snapshot.target.id
      ? `${copy.runtimeAgent} · ${snapshot.target.name}`
      : copy.runtimeAgentPick
    : copy.runtimeRawModel;
  const modelLabel = `${copy.runtimeModelShort} · ${snapshot.selectedModelLabel || copy.runtimeServiceDefault}`;
  const toolLabel = `${copy.runtimeToolsShort} · ${snapshot.toolCount}`;
  const skillLabel = `${copy.runtimeSkillsShort} · ${snapshot.skillCount}`;

  return (
    <div className="composer-runtime-group">
      {snapshot.agentRuntime ? (
        <div className="composer-runtime-control">
          <button
            className={[
              "composer-runtime-trigger",
              snapshot.openPopover === "target" ? "is-open" : "",
              snapshot.locked ? "is-disabled" : "",
            ].filter(Boolean).join(" ")}
            type="button"
            data-runtime-toggle="target"
            aria-disabled={snapshot.locked ? "true" : "false"}
            title={snapshot.locked ? copy.runtimeLocked : copy.runtimeAgentHint}
            onClick={() => {
              if (snapshot.locked) {
                return;
              }
              requestLegacyChatRuntimePopover("target");
            }}
          >
            <span className="composer-runtime-trigger-icon">{"\uD83C\uDFAF"}</span>
            <span className="composer-runtime-trigger-label">{targetLabel}</span>
            <span className="composer-runtime-trigger-caret">▾</span>
          </button>
          {snapshot.openPopover === "target" ? (
            <RuntimeTargetPopover snapshot={snapshot} language={language} />
          ) : null}
        </div>
      ) : (
        <div className="composer-runtime-chip">{targetLabel}</div>
      )}
      <div className="composer-runtime-control">
        <button
          className={snapshot.openPopover === "model" ? "composer-runtime-trigger is-open" : "composer-runtime-trigger"}
          type="button"
          data-runtime-toggle="model"
          title={copy.runtimeModelHint}
          onClick={() => requestLegacyChatRuntimePopover("model")}
        >
          <span className="composer-runtime-trigger-icon">{"\u2728"}</span>
          <span className="composer-runtime-trigger-label">{modelLabel}</span>
          <span className="composer-runtime-trigger-caret">▾</span>
        </button>
        {snapshot.openPopover === "model" ? (
          <RuntimeModelPopover snapshot={snapshot} language={language} />
        ) : null}
      </div>
      <div className="composer-runtime-divider" aria-hidden="true"></div>
      <div className="composer-runtime-control">
        <button
          className={
            snapshot.openPopover === "capabilities"
              ? "composer-runtime-trigger is-open"
              : "composer-runtime-trigger"
          }
          type="button"
          data-runtime-toggle="capabilities"
          title={copy.runtimeToolsHint}
          onClick={() => requestLegacyChatRuntimePopover("capabilities")}
        >
          <span className="composer-runtime-trigger-icon">{"\uD83D\uDEE0\uFE0F"}</span>
          <span className="composer-runtime-trigger-label">{toolLabel}</span>
          <span className="composer-runtime-trigger-caret">▾</span>
        </button>
        {snapshot.openPopover === "capabilities" ? (
          <RuntimeCapabilitiesPopover snapshot={snapshot} language={language} />
        ) : null}
      </div>
      <div className="composer-runtime-control">
        <button
          className={
            snapshot.openPopover === "skills"
              ? "composer-runtime-trigger is-open"
              : "composer-runtime-trigger"
          }
          type="button"
          data-runtime-toggle="skills"
          title={copy.runtimeSkillsHint}
          onClick={() => requestLegacyChatRuntimePopover("skills")}
        >
          <span className="composer-runtime-trigger-icon">{"\u26A1"}</span>
          <span className="composer-runtime-trigger-label">{skillLabel}</span>
          <span className="composer-runtime-trigger-caret">▾</span>
        </button>
        {snapshot.openPopover === "skills" ? (
          <RuntimeSkillsPopover snapshot={snapshot} language={language} />
        ) : null}
      </div>
    </div>
  );
}

function RuntimeCompactSheet({
  snapshot,
  language,
}: {
  snapshot: ChatRuntimeSnapshot;
  language: LegacyShellLanguage;
}) {
  const copy = getLegacyShellCopy(language);

  if (!snapshot.compact || snapshot.openPopover !== "mobile") {
    return null;
  }

  return (
    <>
      <button
        className="composer-runtime-sheet-backdrop"
        type="button"
        data-runtime-close
        aria-label={copy.sessionClose}
        onClick={() => requestLegacyChatRuntimeClose()}
      ></button>
      <div
        className="composer-runtime-popover composer-runtime-popover-mobile is-wide"
        role="dialog"
        aria-modal="true"
        aria-label={copy.runtimeMobile}
      >
        <div className="composer-runtime-popover-mobile-topbar">
          <div className="composer-runtime-popover-head">
            <strong>{copy.runtimeMobile}</strong>
            <p>{snapshot.note || copy.runtimeMobileHint}</p>
          </div>
          <button
            className="composer-runtime-popover-mobile-close"
            type="button"
            data-runtime-close
            aria-label={copy.sessionClose}
            onClick={() => requestLegacyChatRuntimeClose()}
          >
            &times;
          </button>
        </div>
        <div className="composer-runtime-popover-mobile-body" data-runtime-scroll-container="mobile">
          <RuntimeSummaryChips snapshot={snapshot} language={language} />
          {snapshot.agentRuntime ? (
            <>
              <section className="composer-runtime-section">
                <div className="composer-runtime-popover-head">
                  <strong>{copy.runtimeAgent}</strong>
                  <p>{snapshot.locked ? copy.runtimeLocked : copy.runtimeAgentHint}</p>
                </div>
                <div className="composer-runtime-option-list">
                  {snapshot.targetOptions.length ? (
                    snapshot.targetOptions.map((item) => (
                      <button
                        key={item.id}
                        className={item.active ? "composer-runtime-option is-active" : "composer-runtime-option"}
                        type="button"
                        data-runtime-target-id={item.id}
                        disabled={snapshot.locked}
                        onClick={() => {
                          requestLegacyChatRuntimeTarget({
                            type: item.type,
                            id: item.id,
                            name: item.name,
                          });
                        }}
                      >
                        <strong>{item.name}</strong>
                        <span className="composer-runtime-option-summary">{item.subtitle}</span>
                      </button>
                    ))
                  ) : (
                    <RuntimeEmpty text={copy.runtimeNone} />
                  )}
                </div>
              </section>
              <div className="composer-runtime-separator"></div>
            </>
          ) : null}
          <section className="composer-runtime-section">
            <div className="composer-runtime-popover-head">
              <strong>{`${copy.runtimeProvider} / ${copy.runtimeModel}`}</strong>
              <p>{snapshot.note || copy.runtimeModelHint}</p>
            </div>
            {snapshot.providers.length ? (
              snapshot.providers.map((provider) => (
                <section key={provider.id} className="composer-runtime-provider-group">
                  <RuntimeSectionTitle title={provider.name} count={provider.models.length} />
                  <div className="composer-runtime-option-list">
                    {provider.models.map((model) => (
                      <button
                        key={`${provider.id}:${model.id}`}
                        className={
                          model.active
                            ? "composer-runtime-model-option is-active"
                            : "composer-runtime-model-option"
                        }
                        type="button"
                        onClick={() => {
                          requestLegacyChatRuntimeModel({
                            providerId: provider.id,
                            modelId: model.id,
                          });
                        }}
                      >
                        <strong>{model.name}</strong>
                        <span>{provider.name}</span>
                      </button>
                    ))}
                  </div>
                </section>
              ))
            ) : (
              <RuntimeEmpty text={copy.runtimeEmpty} />
            )}
          </section>
          <div className="composer-runtime-separator"></div>
          <section className="composer-runtime-section">
            <div className="composer-runtime-popover-head">
              <strong>{copy.runtimeToolsMcp}</strong>
              <p>{snapshot.note || copy.runtimeToolsHint}</p>
            </div>
            <RuntimeCapabilitySections items={snapshot.capabilities} language={language} />
          </section>
          <div className="composer-runtime-separator"></div>
          <section className="composer-runtime-section">
            <div className="composer-runtime-popover-head">
              <strong>{copy.runtimeSkills}</strong>
              <p>{snapshot.note || copy.runtimeSkillsHint}</p>
            </div>
            <RuntimeSkillSections items={snapshot.skills} language={language} />
          </section>
        </div>
      </div>
    </>
  );
}

export function ChatRuntimeHost({
  currentRoute,
  language,
}: {
  currentRoute: string;
  language: LegacyShellLanguage;
}) {
  const snapshot = useLegacyChatRuntimeSnapshot(currentRoute);
  const copy = getLegacyShellCopy(language);
  const compactMeta = RuntimeCompactMeta(snapshot, language);

  return (
    <div className="composer-runtime-bar" id={LEGACY_SHELL_IDS.chatRuntimePanel}>
      <div data-runtime-controls-root="">
        {snapshot.compact ? (
          <div className="composer-runtime-group composer-runtime-group-compact">
            <div className="composer-runtime-control composer-runtime-control-compact">
              <button
                className={[
                  "composer-runtime-trigger composer-runtime-trigger-compact",
                  snapshot.openPopover === "mobile" ? "is-open" : "",
                ].filter(Boolean).join(" ")}
                type="button"
                data-runtime-toggle="mobile"
                title={copy.runtimeMobileHint}
                onClick={() => requestLegacyChatRuntimePopover("mobile")}
              >
                <span className="composer-runtime-trigger-icon">{"\u2699\uFE0F"}</span>
                <span className="composer-runtime-trigger-copy">
                  <strong>{copy.runtimeMobile}</strong>
                  <span>{compactMeta}</span>
                </span>
                <span className="composer-runtime-trigger-caret">▾</span>
              </button>
            </div>
          </div>
        ) : (
          <RuntimeDesktopControls snapshot={snapshot} language={language} />
        )}
      </div>
      <div data-runtime-note-root="">
        <RuntimeNote note={snapshot.note} />
      </div>
    </div>
  );
}

export const ChatRuntimeSheetHost = memo(function ChatRuntimeSheetHost({
  currentRoute,
  language,
}: {
  currentRoute: string;
  language: LegacyShellLanguage;
}) {
  const snapshot = useLegacyChatRuntimeSnapshot(currentRoute);

  return (
    <div className="runtime-sheet-host" id={LEGACY_SHELL_IDS.chatRuntimeSheetHost}>
      <div data-runtime-sheet-root="">
        <RuntimeCompactSheet snapshot={snapshot} language={language} />
      </div>
    </div>
  );
});
