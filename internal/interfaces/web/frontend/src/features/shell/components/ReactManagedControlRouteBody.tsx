import { useEffect, useState } from "react";
import { createAPIClient } from "../../../shared/api/client";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import {
  normalizeText,
  RouteCard,
  RouteFieldRow,
} from "./RouteBodyPrimitives";

const REACT_MANAGED_CONTROL_ROUTES = ["channels", "skills", "mcp"] as const;

type ReactManagedControlRoute = (typeof REACT_MANAGED_CONTROL_ROUTES)[number];

type ControlRouteRecord = {
  id?: string;
  type?: string;
  name?: string;
  description?: string;
  scope?: string;
  version?: string;
  enabled?: boolean;
};

type ControlRouteResponse = {
  items?: ControlRouteRecord[];
};

type ControlRouteCopy = {
  loading: string;
  statusEnabled: string;
  statusDisabled: string;
  copyValue: string;
  fieldID: string;
  fieldType: string;
  fieldDescription: string;
  fieldName: string;
  fieldScope: string;
  fieldVersion: string;
  emptyChannels: string;
  emptySkills: string;
  emptyMCP: string;
  loadFailed: (message: string) => string;
};

const CONTROL_ROUTE_COPY: Record<LegacyShellLanguage, ControlRouteCopy> = {
  en: {
    loading: "Loading...",
    statusEnabled: "Enabled",
    statusDisabled: "Disabled",
    copyValue: "Copy value",
    fieldID: "ID",
    fieldType: "Type",
    fieldDescription: "Description",
    fieldName: "Name",
    fieldScope: "Scope",
    fieldVersion: "Version",
    emptyChannels: "No Channels available.",
    emptySkills: "No Skills available.",
    emptyMCP: "No MCP available.",
    loadFailed: (message) => `Load failed: ${message}`,
  },
  zh: {
    loading: "加载中...",
    statusEnabled: "启用",
    statusDisabled: "停用",
    copyValue: "复制内容",
    fieldID: "ID",
    fieldType: "类型",
    fieldDescription: "描述",
    fieldName: "名称",
    fieldScope: "范围",
    fieldVersion: "版本",
    emptyChannels: "暂无可用通道。",
    emptySkills: "暂无可用技能。",
    emptyMCP: "暂无 MCP 配置。",
    loadFailed: (message) => `加载失败：${message}`,
  },
};

type FieldSpec = {
  label: string;
  value: string | null | undefined;
  copyable?: boolean;
  mono?: boolean;
  multiline?: boolean;
  preview?: boolean;
  clampLines?: number;
};

type RequestState =
  | { status: "loading"; items: ControlRouteRecord[]; error: string }
  | { status: "ready"; items: ControlRouteRecord[]; error: string }
  | { status: "error"; items: ControlRouteRecord[]; error: string };

type ReactManagedControlRouteBodyProps = {
  route: ReactManagedControlRoute;
  language: LegacyShellLanguage;
};

type RouteConfig = {
  path: string;
  empty: (copy: ControlRouteCopy) => string;
  fields: (item: ControlRouteRecord, copy: ControlRouteCopy) => FieldSpec[];
};

const ROUTE_CONFIG: Record<ReactManagedControlRoute, RouteConfig> = {
  channels: {
    path: "/api/control/channels",
    empty: (copy) => copy.emptyChannels,
    fields: (item, copy) => [
      { label: copy.fieldID, value: item.id, copyable: true, mono: true },
      { label: copy.fieldType, value: item.type },
      {
        label: copy.fieldDescription,
        value: item.description,
        multiline: true,
        preview: true,
        clampLines: 3,
      },
    ],
  },
  skills: {
    path: "/api/control/skills",
    empty: (copy) => copy.emptySkills,
    fields: (item, copy) => [
      { label: copy.fieldID, value: item.id, copyable: true, mono: true },
      { label: copy.fieldType, value: item.type },
      { label: copy.fieldName, value: item.name },
      { label: copy.fieldScope, value: item.scope },
      { label: copy.fieldVersion, value: item.version },
    ],
  },
  mcp: {
    path: "/api/control/mcps",
    empty: (copy) => copy.emptyMCP,
    fields: (item, copy) => [
      { label: copy.fieldID, value: item.id, copyable: true, mono: true },
      { label: copy.fieldType, value: item.type },
      { label: copy.fieldName, value: item.name },
      { label: copy.fieldScope, value: item.scope },
      { label: copy.fieldVersion, value: item.version },
    ],
  },
};

export function isReactManagedControlRoute(route: string): route is ReactManagedControlRoute {
  return REACT_MANAGED_CONTROL_ROUTES.includes(route as ReactManagedControlRoute);
}

export function ReactManagedControlRouteBody({
  route,
  language,
}: ReactManagedControlRouteBodyProps) {
  const copy = CONTROL_ROUTE_COPY[language];
  const [state, setState] = useState<RequestState>({
    status: "loading",
    items: [],
    error: "",
  });

  useEffect(() => {
    let disposed = false;

    setState({
      status: "loading",
      items: [],
      error: "",
    });

    void createAPIClient()
      .get<ControlRouteResponse>(ROUTE_CONFIG[route].path)
      .then((payload) => {
        if (disposed) {
          return;
        }
        setState({
          status: "ready",
          items: Array.isArray(payload?.items) ? payload.items : [],
          error: "",
        });
      })
      .catch((error: unknown) => {
        if (disposed) {
          return;
        }
        setState({
          status: "error",
          items: [],
          error: error instanceof Error ? error.message : "unknown_error",
        });
      });

    return () => {
      disposed = true;
    };
  }, [route]);

  if (state.status === "loading") {
    return <p className="route-loading">{copy.loading}</p>;
  }

  if (state.status === "error") {
    return <p className="route-error">{copy.loadFailed(state.error)}</p>;
  }

  if (!state.items.length) {
    return <p className="route-empty">{ROUTE_CONFIG[route].empty(copy)}</p>;
  }

  return (
    <>
      {state.items.map((item) => (
        <RouteCard
          key={`${route}-${normalizeText(item.id)}`}
          title={item.id}
          type={item.type}
          enabled={Boolean(item.enabled)}
          statusEnabledLabel={copy.statusEnabled}
          statusDisabledLabel={copy.statusDisabled}
        >
          {ROUTE_CONFIG[route].fields(item, copy).map((field) => (
            <RouteFieldRow key={`${field.label}-${normalizeText(field.value)}`} copyLabel={copy.copyValue} {...field} />
          ))}
        </RouteCard>
      ))}
    </>
  );
}
