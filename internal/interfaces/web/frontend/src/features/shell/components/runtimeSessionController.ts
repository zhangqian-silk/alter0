import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";
import { createAPIClient } from "../../../shared/api/client";
import { resolveSessionIDReference } from "../../../shared/session/sessionHash";
import {
  RUNTIME_SESSION_HISTORY_PAGE_TURN_LIMIT,
  runtimeSessionAttachmentsEndpoint,
  runtimeSessionCollectionEndpoint,
  runtimeSessionDetailEndpoint,
  runtimeSessionEventDetailEndpoint,
  runtimeSessionInputEndpoint,
  runtimeSessionPinEndpoint,
  type RuntimeSessionRoute,
} from "./runtimeSessionApi";

export type RuntimeSessionTurnPaging = {
  limit?: number;
  total?: number;
  byte_limit?: number;
  approx_bytes?: number;
  has_more_before?: boolean;
  has_more_after?: boolean;
  oldest_turn_id?: string;
  newest_turn_id?: string;
  next_before_turn_id?: string;
};

export type RuntimeSessionPayload = {
  id?: string;
  status?: string;
  title?: string;
  pinned?: boolean;
  created_at?: string | number;
  updated_at?: string | number;
  last_output_at?: string | number;
  activity_at?: string | number;
  revision?: string | number;
  model_provider_id?: string;
  model_id?: string;
  tool_ids?: string[];
  skill_ids?: string[];
  mcp_ids?: string[];
  turns?: unknown[];
  turns_paging?: RuntimeSessionTurnPaging;
  [key: string]: unknown;
};

type RuntimeSessionResponse = {
  session?: RuntimeSessionPayload;
};

type RuntimeSessionsResponse = {
  items?: RuntimeSessionPayload[];
};

export type RuntimeSessionPayloadSource = "summary" | "detail" | "event";

export type RuntimeSessionNormalizeContext = {
  source: RuntimeSessionPayloadSource;
};

export type RuntimeSessionEventDetail = {
  turn_id?: string;
  event?: unknown;
  blocks?: unknown[];
  searchable?: boolean;
};

type RuntimeSessionEventDetailResponse = {
  event?: RuntimeSessionEventDetail;
};

export type RuntimeSessionAttachmentUploadResponse = {
  items?: Array<{
    id?: string;
    name?: string;
    content_type?: string;
    size?: number;
    asset_url?: string;
    preview_url?: string;
  }>;
};

export type RuntimeSessionPollPlan = {
  enabled: boolean;
  interval: number;
  refreshActiveSession: boolean;
};

export type RuntimeSessionPollPlanOptions = {
  sessionCount?: number;
  status?: string;
  pageHidden: boolean;
  scrollingActive?: boolean;
  inputFocused?: boolean;
  pollWhenHidden?: boolean;
  pollInterval: number;
  interactionPollInterval?: number;
  hiddenPollInterval?: number;
};

export type RuntimeSessionControllerOptions<TSession extends { id: string }> = {
  route: RuntimeSessionRoute;
  initialSessions: TSession[];
  initialActiveSessionID: string;
  normalizeSession: (payload: RuntimeSessionPayload, previous: TSession | null, context: RuntimeSessionNormalizeContext) => TSession | null;
  mergeSession: (previous: TSession | undefined, incoming: TSession) => TSession;
  sortSessions: (sessions: TSession[]) => TSession[];
  getProgressiveHistoryTurnBefore?: (session: TSession) => string;
  getProgressiveHistoryPaging?: (session: TSession) => RuntimeSessionTurnPaging | undefined;
  canLoadProgressiveHistory?: (session: TSession) => boolean;
  enableProgressiveHistory?: boolean;
  manageState?: boolean;
  onSessionsChange?: (sessions: TSession[]) => void;
  onActiveSessionIDChange?: (sessionID: string) => void;
};

export type RuntimeSessionController<TSession extends { id: string }> = {
  apiClient: ReturnType<typeof createAPIClient>;
  route: RuntimeSessionRoute;
  sessions: TSession[];
  sessionsRef: MutableRefObject<TSession[]>;
  activeSessionID: string;
  activeSession: TSession | null;
  setSessions: Dispatch<SetStateAction<TSession[]>>;
  setActiveSessionID: Dispatch<SetStateAction<string>>;
  refreshList: () => Promise<TSession[]>;
  refreshActiveSession: (sessionID: string, options?: { turnBefore?: string; turnLimit?: number }) => Promise<TSession | null>;
  createSession: (body?: Record<string, unknown>) => Promise<TSession | null>;
  deleteSession: (sessionID: string) => Promise<void>;
  setSessionPinned: (sessionID: string, pinned: boolean) => Promise<TSession | null>;
  sendInput: (sessionID: string, body: Record<string, unknown>) => Promise<TSession | null>;
  uploadAttachments: (sessionID: string, body: Record<string, unknown>) => Promise<RuntimeSessionAttachmentUploadResponse>;
  loadEventDetail: (sessionID: string, turnID: string, eventID: string) => Promise<RuntimeSessionEventDetail | null>;
};

function normalizeRuntimeSessionText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeRuntimePollStatus(status: string): string {
  const normalized = normalizeRuntimeSessionText(status).toLowerCase();
  return normalized || "busy";
}

export function resolveRuntimeSessionPollPlan(options: RuntimeSessionPollPlanOptions): RuntimeSessionPollPlan {
  const sessionCount = options.sessionCount ?? 1;
  const normalized = normalizeRuntimePollStatus(options.status || "busy");
  const refreshActiveSession = normalized === "busy";

  if (sessionCount <= 0 || normalized !== "busy") {
    return {
      enabled: false,
      interval: 0,
      refreshActiveSession: false,
    };
  }

  if (options.scrollingActive) {
    return {
      enabled: false,
      interval: 0,
      refreshActiveSession,
    };
  }

  if (options.pageHidden && options.pollWhenHidden === false) {
    return {
      enabled: false,
      interval: 0,
      refreshActiveSession,
    };
  }

  return {
    enabled: true,
    interval: options.pageHidden
      ? options.hiddenPollInterval ?? options.pollInterval
      : options.inputFocused
        ? options.interactionPollInterval ?? options.pollInterval
        : options.pollInterval,
    refreshActiveSession,
  };
}

export function useRuntimeSessionController<TSession extends { id: string }>(
  options: RuntimeSessionControllerOptions<TSession>,
): RuntimeSessionController<TSession> {
  const apiClient = useMemo(() => createAPIClient(), []);
  const manageState = options.manageState !== false;
  const [sessions, setSessionsState] = useState<TSession[]>(options.initialSessions);
  const [activeSessionID, setActiveSessionIDState] = useState(options.initialActiveSessionID);
  const sessionsRef = useRef<TSession[]>(sessions);
  const deletedSessionIDsRef = useRef<Set<string>>(new Set());
  const progressiveHistoryLoadsRef = useRef<Set<string>>(new Set());
  const progressiveHistoryLoadedRef = useRef<Set<string>>(new Set());

  const setSessions = useCallback<Dispatch<SetStateAction<TSession[]>>>((updater) => {
    setSessionsState((current) => {
      const next = typeof updater === "function"
        ? (updater as (value: TSession[]) => TSession[])(current)
        : updater;
      sessionsRef.current = next;
      options.onSessionsChange?.(next);
      return next;
    });
  }, [options]);

  const setActiveSessionID = useCallback<Dispatch<SetStateAction<string>>>((updater) => {
    setActiveSessionIDState((current) => {
      const next = typeof updater === "function"
        ? (updater as (value: string) => string)(current)
        : updater;
      options.onActiveSessionIDChange?.(next);
      return next;
    });
  }, [options]);

  useEffect(() => {
    sessionsRef.current = sessions;
    options.onSessionsChange?.(sessions);
  }, [options, sessions]);

  useEffect(() => {
    options.onActiveSessionIDChange?.(activeSessionID);
  }, [activeSessionID, options]);

  const activeSession = useMemo(
    () => sessions.find((session) => session.id === activeSessionID) || null,
    [activeSessionID, sessions],
  );

  const mergeIncomingSessions = useCallback((incoming: TSession[]) => {
    setSessions(() => {
      const current = sessionsRef.current;
      const currentMap = new Map(current.map((session) => [session.id, session]));
      const merged = incoming.map((session) => options.mergeSession(currentMap.get(session.id), session));
      const next = options.sortSessions(merged);
      sessionsRef.current = next;
      return next;
    });
  }, [options, setSessions]);

  const refreshList = useCallback(async () => {
    const payload = await apiClient.get<RuntimeSessionsResponse>(runtimeSessionCollectionEndpoint(options.route));
    const currentMap = new Map(sessionsRef.current.map((session) => [session.id, session]));
    const nextSessions = (Array.isArray(payload.items) ? payload.items : [])
      .filter((item) => !deletedSessionIDsRef.current.has(normalizeRuntimeSessionText(item.id)))
      .map((item) => options.normalizeSession(item, currentMap.get(normalizeRuntimeSessionText(item.id)) || null, { source: "summary" }))
      .filter((session): session is TSession => session !== null);
    if (manageState) {
      mergeIncomingSessions(nextSessions);
      setActiveSessionID((current) => {
        const resolvedCurrent = resolveSessionIDReference(nextSessions, current);
        if (resolvedCurrent) {
          return resolvedCurrent;
        }
        return nextSessions[0]?.id || "";
      });
    }
    return nextSessions;
  }, [apiClient, manageState, mergeIncomingSessions, options, setActiveSessionID]);

  const refreshActiveSession = useCallback(async (
    sessionID: string,
    requestOptions: { turnBefore?: string; turnLimit?: number } = {},
  ) => {
    const normalizedSessionID = normalizeRuntimeSessionText(sessionID);
    if (!normalizedSessionID || deletedSessionIDsRef.current.has(normalizedSessionID)) {
      return null;
    }
    const payload = await apiClient.get<RuntimeSessionResponse>(
      runtimeSessionDetailEndpoint(options.route, normalizedSessionID, requestOptions),
    );
    if (!payload.session) {
      return null;
    }
    const current = sessionsRef.current.find((session) => session.id === normalizedSessionID) || null;
    const normalized = options.normalizeSession(payload.session, current, { source: "detail" });
    if (!normalized || deletedSessionIDsRef.current.has(normalized.id)) {
      return null;
    }
    if (manageState) {
      setSessions((items) => {
        const exists = items.some((session) => session.id === normalizedSessionID);
        const merged = exists
          ? items.map((session) => session.id === normalizedSessionID ? options.mergeSession(session, normalized) : session)
          : [normalized, ...items];
        return options.sortSessions(merged);
      });
    }
    return normalized;
  }, [apiClient, manageState, options, setSessions]);

  const createSession = useCallback(async (body: Record<string, unknown> = {}) => {
    const payload = await apiClient.post<RuntimeSessionResponse>(runtimeSessionCollectionEndpoint(options.route), body);
    if (!payload.session) {
      return null;
    }
    const normalized = options.normalizeSession(payload.session, null, { source: "detail" });
    if (!normalized) {
      return null;
    }
    if (manageState) {
      setSessions((items) =>
        options.sortSessions([normalized, ...items.filter((session) => session.id !== normalized.id)]),
      );
      setActiveSessionID(normalized.id);
    }
    return normalized;
  }, [apiClient, manageState, options, setActiveSessionID, setSessions]);

  const deleteSession = useCallback(async (sessionID: string) => {
    const normalizedSessionID = normalizeRuntimeSessionText(sessionID);
    if (!normalizedSessionID) {
      return;
    }
    await apiClient.delete(runtimeSessionDetailEndpoint(options.route, normalizedSessionID));
    if (manageState) {
      deletedSessionIDsRef.current.add(normalizedSessionID);
      setSessions((items) => {
        const next = items.filter((session) => session.id !== normalizedSessionID);
        setActiveSessionID((current) => current === normalizedSessionID ? next[0]?.id || "" : current);
        return next;
      });
    }
  }, [apiClient, manageState, options.route, setActiveSessionID, setSessions]);

  const setSessionPinned = useCallback(async (sessionID: string, pinned: boolean) => {
    const normalizedSessionID = normalizeRuntimeSessionText(sessionID);
    const payload = await apiClient.post<RuntimeSessionResponse>(
      runtimeSessionPinEndpoint(options.route, normalizedSessionID),
      { pinned },
    );
    const current = sessionsRef.current.find((session) => session.id === normalizedSessionID) || null;
    const normalized = payload.session
      ? options.normalizeSession(payload.session, current, { source: "detail" })
      : options.normalizeSession({ id: normalizedSessionID, pinned }, current, { source: "summary" });
    if (!normalized) {
      return null;
    }
    if (manageState) {
      setSessions((items) =>
        options.sortSessions(
          items.map((session) => session.id === normalizedSessionID ? options.mergeSession(session, normalized) : session),
        ),
      );
    }
    return normalized;
  }, [apiClient, manageState, options, setSessions]);

  const sendInput = useCallback(async (sessionID: string, body: Record<string, unknown>) => {
    const payload = await apiClient.post<RuntimeSessionResponse>(
      runtimeSessionInputEndpoint(options.route, sessionID),
      body,
    );
    if (!payload.session) {
      return null;
    }
    const current = sessionsRef.current.find((session) => session.id === sessionID) || null;
    const normalized = options.normalizeSession(payload.session, current, { source: "detail" });
    if (!normalized) {
      return null;
    }
    if (manageState) {
      setSessions((items) =>
        options.sortSessions(
          items.map((session) => session.id === sessionID ? options.mergeSession(session, normalized) : session),
        ),
      );
    }
    return normalized;
  }, [apiClient, manageState, options, setSessions]);

  const uploadAttachments = useCallback(async (sessionID: string, body: Record<string, unknown>) => {
    return apiClient.post<RuntimeSessionAttachmentUploadResponse>(
      runtimeSessionAttachmentsEndpoint(options.route, sessionID),
      body,
    );
  }, [apiClient, options.route]);

  const loadEventDetail = useCallback(async (sessionID: string, turnID: string, eventID: string) => {
    const payload = await apiClient.get<RuntimeSessionEventDetailResponse>(
      runtimeSessionEventDetailEndpoint(options.route, sessionID, turnID, eventID),
    );
    return payload.event || null;
  }, [apiClient, options.route]);

  useEffect(() => {
    const session = activeSession;
    if (options.enableProgressiveHistory === false) {
      return;
    }
    const paging = session && options.getProgressiveHistoryPaging?.(session);
    const beforeTurnID = session ? normalizeRuntimeSessionText(options.getProgressiveHistoryTurnBefore?.(session)) : "";
    if (
      !session
      || paging?.has_more_before !== true
      || !beforeTurnID
      || options.canLoadProgressiveHistory?.(session) === false
    ) {
      return;
    }
    const requestKey = `${session.id}:${beforeTurnID}`;
    if (
      progressiveHistoryLoadsRef.current.has(requestKey)
      || progressiveHistoryLoadedRef.current.has(requestKey)
    ) {
      return;
    }
    let cancelled = false;
    progressiveHistoryLoadsRef.current.add(requestKey);
    void (async () => {
      try {
        if (!cancelled) {
          await refreshActiveSession(session.id, {
            turnBefore: beforeTurnID,
            turnLimit: RUNTIME_SESSION_HISTORY_PAGE_TURN_LIMIT,
          });
          progressiveHistoryLoadedRef.current.add(requestKey);
        }
      } catch {
      } finally {
        progressiveHistoryLoadsRef.current.delete(requestKey);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [activeSession, options, refreshActiveSession]);

  return {
    apiClient,
    route: options.route,
    sessions,
    sessionsRef,
    activeSessionID,
    activeSession,
    setSessions,
    setActiveSessionID,
    refreshList,
    refreshActiveSession,
    createSession,
    deleteSession,
    setSessionPinned,
    sendInput,
    uploadAttachments,
    loadEventDetail,
  };
}
