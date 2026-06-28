import { useEffect, useState } from "react";
import type { createAPIClient } from "../../../shared/api/client";

export type RuntimeSessionProviderModel = {
  id: string;
  name: string;
  is_enabled?: boolean;
  supports_vision?: boolean;
};

export type RuntimeSessionProvider = {
  id: string;
  name: string;
  is_enabled?: boolean;
  is_default?: boolean;
  default_model?: string;
  models?: RuntimeSessionProviderModel[];
};

export type RuntimeSessionCapability = {
  id?: string;
  name?: string;
  description?: string;
  scope?: string;
  enabled?: boolean;
  metadata?: Record<string, string>;
};

export type RuntimeSessionCatalogs = {
  providers: RuntimeSessionProvider[];
  skills: RuntimeSessionCapability[];
  mcps: RuntimeSessionCapability[];
  skillsLoaded: boolean;
};

export function useRuntimeSessionCatalogs(
  apiClient: ReturnType<typeof createAPIClient>,
): RuntimeSessionCatalogs {
  const [providers, setProviders] = useState<RuntimeSessionProvider[]>([]);
  const [skills, setSkills] = useState<RuntimeSessionCapability[]>([]);
  const [mcps, setMcps] = useState<RuntimeSessionCapability[]>([]);
  const [skillsLoaded, setSkillsLoaded] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const loadCatalogs = async () => {
      try {
        const providerPayload = await apiClient.get<{ items?: RuntimeSessionProvider[] }>("/api/control/llm/providers");
        if (!cancelled) {
          setProviders(Array.isArray(providerPayload.items) ? providerPayload.items : []);
        }
      } catch {
      }
      try {
        const skillPayload = await apiClient.get<{ items?: RuntimeSessionCapability[] }>("/api/control/skills");
        if (!cancelled) {
          setSkills(Array.isArray(skillPayload.items) ? skillPayload.items : []);
        }
      } catch {
        if (!cancelled) {
          setSkills([]);
        }
      }
      try {
        const mcpPayload = await apiClient.get<{ items?: RuntimeSessionCapability[] }>("/api/control/mcps");
        if (!cancelled) {
          setMcps(Array.isArray(mcpPayload.items) ? mcpPayload.items : []);
        }
      } catch {
        if (!cancelled) {
          setMcps([]);
        }
      } finally {
        if (!cancelled) {
          setSkillsLoaded(true);
        }
      }
    };
    void loadCatalogs();
    return () => {
      cancelled = true;
    };
  }, [apiClient]);

  return { providers, skills, mcps, skillsLoaded };
}
