import { useEffect, useMemo, useState, type Dispatch, type FormEvent, type ReactNode, type SetStateAction } from "react";
import { createAPIClient } from "../../../shared/api/client";
import { formatDateTime } from "../../../shared/time/format";
import type { LegacyShellLanguage } from "../legacyShellCopy";
import { normalizeText } from "./RouteBodyPrimitives";

const PRODUCT_ROUTE_STORAGE_KEY = "alter0.web.products.route.v1";

type ProductRouteState = {
  selectedProductID: string;
  selectedDraftID: string;
  activePanel: "workspace" | "studio";
  selectedSpaceID: string;
  workspaceSessionByProduct: Record<string, string>;
};

type ProductWorker = {
  agent_id: string;
  role: string;
  responsibility: string;
  capabilities: string[];
  enabled: boolean;
};

type ProductDraft = {
  id: string;
  name: string;
  slug: string;
  summary: string;
  status: string;
  visibility: string;
  owner_type: string;
  version: string;
  master_agent_id: string;
  entry_route: string;
  tags: string[];
  artifact_types: string[];
  knowledge_sources: string[];
  worker_agents: ProductWorker[];
};

type ProductDraftAgent = {
  agent_id: string;
  name: string;
  description: string;
  system_prompt: string;
  max_iterations: number;
  tools: string[];
  skills: string[];
  mcps: string[];
  memory_files: string[];
  capabilities: string[];
  allowed_delegate_targets: string[];
  enabled: boolean;
  delegatable: boolean;
};

type ProductDraftWorker = {
  agent_id: string;
  name: string;
  role: string;
  responsibility: string;
  description: string;
  system_prompt: string;
  input_contract: string;
  output_contract: string;
  allowed_tools: string[];
  allowed_delegate_targets: string[];
  dependencies: string[];
  skills: string[];
  mcps: string[];
  memory_files: string[];
  capabilities: string[];
  priority: number;
  max_iterations: number;
  enabled: boolean;
};

type ProductStudioDraft = {
  draft_id: string;
  mode: string;
  review_status: string;
  generated_by: string;
  generated_at: string;
  updated_at: string;
  goal: string;
  target_users: string[];
  core_capabilities: string[];
  constraints: string[];
  expected_artifacts: string[];
  integration_requirements: string[];
  conflict_suggestions: string[];
  published_product_id: string;
  product: ProductDraft;
  master_agent: ProductDraftAgent;
  worker_matrix: ProductDraftWorker[];
};

type WorkspaceMasterAgent = {
  agent_id: string;
  name: string;
  description: string;
  capabilities: string[];
  tools: string[];
  skills: string[];
  mcps: string[];
  memory_files: string[];
};

type WorkspaceSpaceSummary = {
  space_id: string;
  title: string;
  slug: string;
  html_path: string;
  summary: string;
  type: string;
  status: string;
  revision: number;
  updated_at: string;
  tags: string[];
};

type WorkspaceGuide = {
  id: string;
  city: string;
  days: number;
  travel_style: string;
  budget: string;
  companions: string[];
  must_visit: string[];
  avoid: string[];
  additional_requirements: string[];
  keep_conditions: string[];
  replace_conditions: string[];
  notes: string[];
  daily_routes: Array<{
    day: number;
    theme: string;
    stops: string[];
    transit: string[];
    dining_plan: string[];
  }>;
  map_layers: Array<{
    id: string;
    label: string;
    description: string;
  }>;
  content: string;
  revision: number;
  updated_at: string;
};

type ProductWorkspace = {
  product: ProductDraft;
  master_agent: WorkspaceMasterAgent | null;
  space_type: string;
  space_label: string;
  workspace_hint: string;
  spaces: WorkspaceSpaceSummary[];
};

type ProductWorkspaceDetail = {
  space: WorkspaceSpaceSummary;
  guide: WorkspaceGuide | null;
};

type ProductWorkspaceMessage = {
  id: string;
  role: "user" | "assistant";
  text: string;
  status: string;
  error: boolean;
  at: number;
};

type DraftRequest = {
  name: string;
  goal: string;
  target_users: string;
  core_capabilities: string;
  constraints: string;
  expected_artifacts: string;
  integration_requirements: string;
  mode: string;
};

type ProductRouteCopy = {
  loading: string;
  title: string;
  subtitle: string;
  empty: string;
  create: string;
  edit: string;
  panelWorkspace: string;
  panelStudio: string;
  workspaceTitle: string;
  workspaceSubtitle: string;
  workspaceEmpty: string;
  workspaceOverview: string;
  workspaceMaster: string;
  workspaceOutput: string;
  workspaceSources: string;
  workspaceTags: string;
  workspaceWorkers: string;
  workspaceChatTitle: string;
  workspaceChatHint: string;
  workspaceChatPlaceholder: string;
  workspaceChatAriaLabel: string;
  workspaceChatSend: string;
  workspaceChatFailed: (message: string) => string;
  workspaceSynced: string;
  workspaceSpaces: string;
  workspaceSpaceEmpty: string;
  workspaceOpenNew: string;
  workspaceDetail: string;
  workspaceDetailEmpty: string;
  workspaceSpaceRevision: string;
  workspaceUpdated: string;
  workspaceDays: string;
  workspaceContent: string;
  workspaceNotes: string;
  workspaceRoutes: string;
  workspaceLayers: string;
  workspaceOpenPage: string;
  workspaceCompanions: string;
  workspaceAvoid: string;
  formNew: string;
  formID: string;
  formVersion: string;
  formOwner: string;
  formName: string;
  formSlug: string;
  formSummary: string;
  formStatus: string;
  formVisibility: string;
  formMaster: string;
  formEntryRoute: string;
  formTags: string;
  formArtifacts: string;
  formKnowledge: string;
  formWorkerAgents: string;
  formManaged: string;
  formSave: string;
  formDelete: string;
  formCancel: string;
  formBuiltinNotice: string;
  saved: string;
  deleted: string;
  saveFailed: (message: string) => string;
  deleteFailed: (message: string) => string;
  draftsTitle: string;
  draftsSubtitle: string;
  draftsEmpty: string;
  draftsGenerated: string;
  draftsPublished: string;
  draftsReviewSaved: string;
  draftsGenerateFailed: (message: string) => string;
  draftsPublishFailed: (message: string) => string;
  draftsReviewFailed: (message: string) => string;
  draftsFormName: string;
  draftsFormGoal: string;
  draftsFormTargetUsers: string;
  draftsFormCore: string;
  draftsFormConstraints: string;
  draftsFormArtifacts: string;
  draftsFormIntegrations: string;
  draftsFormMode: string;
  draftsFormGenerate: string;
  draftsFormSave: string;
  draftsFormPublish: string;
  draftsFormEditor: string;
  draftsFormEditorHint: string;
  draftsFormExpandDisabled: string;
  draftsFormModeBootstrap: string;
  draftsFormModeExpand: string;
  draftsDetailMaster: string;
  draftsDetailWorkers: string;
  draftsDetailConflicts: string;
  draftsDetailReview: string;
  draftsDetailEmpty: string;
  draftsDetailGenerated: string;
  draftsDetailUpdated: string;
  draftsDetailMode: string;
  draftsDetailProduct: string;
  draftsDetailPublished: string;
  statusEnabled: string;
  statusDisabled: string;
  processing: string;
  receivedEmpty: string;
};

const PRODUCT_ROUTE_COPY: Record<LegacyShellLanguage, ProductRouteCopy> = {
  en: {
    loading: "Loading...",
    title: "Products",
    subtitle: "Manage product workspaces, master agents, and reusable product context",
    empty: "No Products available.",
    create: "Create Product",
    edit: "Edit Product",
    panelWorkspace: "Workspace",
    panelStudio: "Studio",
    workspaceTitle: "Product Workspace",
    workspaceSubtitle: "Talk to the product master agent and maintain product detail pages.",
    workspaceEmpty: "Select a Product to open its workspace.",
    workspaceOverview: "Overview",
    workspaceMaster: "Master Agent",
    workspaceOutput: "Artifacts",
    workspaceSources: "Knowledge Sources",
    workspaceTags: "Tags",
    workspaceWorkers: "Supporting Agents",
    workspaceChatTitle: "Master Agent Conversation",
    workspaceChatHint: "Use the product master agent to create or revise detail pages.",
    workspaceChatPlaceholder: "Example: create a 3-day Wuhan page with metro-first travel and local food.",
    workspaceChatAriaLabel: "Workspace message",
    workspaceChatSend: "Send to Master",
    workspaceChatFailed: (message) => `Workspace request failed: ${message}`,
    workspaceSynced: "Workspace synced.",
    workspaceSpaces: "Detail Pages",
    workspaceSpaceEmpty: "No detail pages yet. Ask the master agent to create one.",
    workspaceOpenNew: "New Detail Page",
    workspaceDetail: "Page Detail",
    workspaceDetailEmpty: "Select a detail page to view its content.",
    workspaceSpaceRevision: "Revision",
    workspaceUpdated: "Updated",
    workspaceDays: "Days",
    workspaceContent: "Page Content",
    workspaceNotes: "Notes",
    workspaceRoutes: "Daily Routes",
    workspaceLayers: "Map Layers",
    workspaceOpenPage: "Open HTML Page",
    workspaceCompanions: "Companions / Must Visit",
    workspaceAvoid: "Avoid / Additional",
    formNew: "Unsaved Product",
    formID: "Product ID",
    formVersion: "Version",
    formOwner: "Owner Type",
    formName: "Product Name",
    formSlug: "Slug",
    formSummary: "Summary",
    formStatus: "Status",
    formVisibility: "Visibility",
    formMaster: "Master Agent",
    formEntryRoute: "Entry Route",
    formTags: "Tags",
    formArtifacts: "Artifact Types",
    formKnowledge: "Knowledge Sources",
    formWorkerAgents: "Worker Agents",
    formManaged: "Service-managed Fields",
    formSave: "Save Product",
    formDelete: "Delete Product",
    formCancel: "Reset",
    formBuiltinNotice: "Built-in Products are managed by the service and are read-only here.",
    saved: "Product saved.",
    deleted: "Product deleted.",
    saveFailed: (message) => `Save Product failed: ${message}`,
    deleteFailed: (message) => `Delete Product failed: ${message}`,
    draftsTitle: "Draft Studio",
    draftsSubtitle: "Generate product drafts and publish reviewed matrices",
    draftsEmpty: "No product drafts yet.",
    draftsGenerated: "Product draft generated.",
    draftsPublished: "Product draft published.",
    draftsReviewSaved: "Product draft saved.",
    draftsGenerateFailed: (message) => `Generate draft failed: ${message}`,
    draftsPublishFailed: (message) => `Publish draft failed: ${message}`,
    draftsReviewFailed: (message) => `Save draft failed: ${message}`,
    draftsFormName: "Draft Name",
    draftsFormGoal: "Goal",
    draftsFormTargetUsers: "Target Users",
    draftsFormCore: "Core Capabilities",
    draftsFormConstraints: "Constraints",
    draftsFormArtifacts: "Expected Artifacts",
    draftsFormIntegrations: "Integrations",
    draftsFormMode: "Mode",
    draftsFormGenerate: "Generate Draft",
    draftsFormSave: "Save Draft",
    draftsFormPublish: "Publish Draft",
    draftsFormEditor: "Draft JSON",
    draftsFormEditorHint: "Review and adjust the draft JSON before publish.",
    draftsFormExpandDisabled: "Expand mode requires a selected managed Product.",
    draftsFormModeBootstrap: "Bootstrap",
    draftsFormModeExpand: "Expand Selected Product",
    draftsDetailMaster: "Master Agent",
    draftsDetailWorkers: "Supporting Agents",
    draftsDetailConflicts: "Conflict Suggestions",
    draftsDetailReview: "Review Status",
    draftsDetailEmpty: "Select a draft to review, edit, and publish.",
    draftsDetailGenerated: "Generated At",
    draftsDetailUpdated: "Updated At",
    draftsDetailMode: "Generation Mode",
    draftsDetailProduct: "Product Draft",
    draftsDetailPublished: "Published Product",
    statusEnabled: "Enabled",
    statusDisabled: "Disabled",
    processing: "Processing...",
    receivedEmpty: "Received empty response.",
  },
  zh: {
    loading: "加载中...",
    title: "Products",
    subtitle: "管理 Product Workspace、主 Agent 与可复用产品上下文",
    empty: "暂无 Product。",
    create: "创建 Product",
    edit: "编辑 Product",
    panelWorkspace: "Workspace",
    panelStudio: "Studio",
    workspaceTitle: "Product Workspace",
    workspaceSubtitle: "通过主 Agent 对话维护产品详情页与具体空间页面。",
    workspaceEmpty: "选择一个 Product 后进入对应 Workspace。",
    workspaceOverview: "概览",
    workspaceMaster: "主 Agent",
    workspaceOutput: "产物",
    workspaceSources: "知识源",
    workspaceTags: "标签",
    workspaceWorkers: "辅助 Agent",
    workspaceChatTitle: "主 Agent 对话",
    workspaceChatHint: "通过主 Agent 创建或修改具体详情页。",
    workspaceChatPlaceholder: "例如：给武汉创建一个三天地铁优先、带夜宵推荐的城市页。",
    workspaceChatAriaLabel: "Workspace 消息",
    workspaceChatSend: "发送给主 Agent",
    workspaceChatFailed: (message) => `Workspace 请求失败：${message}`,
    workspaceSynced: "Workspace 已同步。",
    workspaceSpaces: "详情页空间",
    workspaceSpaceEmpty: "还没有详情页，可直接让主 Agent 创建。",
    workspaceOpenNew: "新建详情页",
    workspaceDetail: "页面详情",
    workspaceDetailEmpty: "选择一个详情页后查看页面内容。",
    workspaceSpaceRevision: "版本修订",
    workspaceUpdated: "更新时间",
    workspaceDays: "天数",
    workspaceContent: "页面正文",
    workspaceNotes: "补充说明",
    workspaceRoutes: "每日路线",
    workspaceLayers: "地图图层",
    workspaceOpenPage: "打开 HTML 页面",
    workspaceCompanions: "同行人 / 必去点",
    workspaceAvoid: "避坑 / 额外要求",
    formNew: "未保存 Product",
    formID: "Product ID",
    formVersion: "版本",
    formOwner: "归属类型",
    formName: "Product 名称",
    formSlug: "Slug",
    formSummary: "摘要",
    formStatus: "状态",
    formVisibility: "可见性",
    formMaster: "总 Agent",
    formEntryRoute: "入口路由",
    formTags: "标签",
    formArtifacts: "产物类型",
    formKnowledge: "知识源",
    formWorkerAgents: "子 Agent",
    formManaged: "服务端维护字段",
    formSave: "保存 Product",
    formDelete: "删除 Product",
    formCancel: "重置",
    formBuiltinNotice: "内置 Product 由服务维护，此处只读。",
    saved: "Product 已保存。",
    deleted: "Product 已删除。",
    saveFailed: (message) => `保存 Product 失败：${message}`,
    deleteFailed: (message) => `删除 Product 失败：${message}`,
    draftsTitle: "Draft Studio",
    draftsSubtitle: "生成 Product 草稿并发布审核后的矩阵",
    draftsEmpty: "暂无 Product 草稿。",
    draftsGenerated: "Product 草稿已生成。",
    draftsPublished: "Product 草稿已发布。",
    draftsReviewSaved: "Product 草稿已保存。",
    draftsGenerateFailed: (message) => `生成 Product 草稿失败：${message}`,
    draftsPublishFailed: (message) => `发布 Product 草稿失败：${message}`,
    draftsReviewFailed: (message) => `保存 Product 草稿失败：${message}`,
    draftsFormName: "草稿名称",
    draftsFormGoal: "目标",
    draftsFormTargetUsers: "目标用户",
    draftsFormCore: "核心能力",
    draftsFormConstraints: "约束",
    draftsFormArtifacts: "预期产物",
    draftsFormIntegrations: "集成要求",
    draftsFormMode: "模式",
    draftsFormGenerate: "生成草稿",
    draftsFormSave: "保存草稿",
    draftsFormPublish: "发布草稿",
    draftsFormEditor: "草稿 JSON",
    draftsFormEditorHint: "发布前先审核并调整草稿 JSON。",
    draftsFormExpandDisabled: "Expand 模式需要先选中一个托管 Product。",
    draftsFormModeBootstrap: "Bootstrap",
    draftsFormModeExpand: "扩展当前 Product",
    draftsDetailMaster: "主 Agent",
    draftsDetailWorkers: "辅助 Agent",
    draftsDetailConflicts: "冲突建议",
    draftsDetailReview: "审核状态",
    draftsDetailEmpty: "选择一个草稿后可查看、编辑并发布。",
    draftsDetailGenerated: "生成时间",
    draftsDetailUpdated: "更新时间",
    draftsDetailMode: "生成模式",
    draftsDetailProduct: "Product 草稿",
    draftsDetailPublished: "已发布 Product",
    statusEnabled: "启用",
    statusDisabled: "停用",
    processing: "处理中...",
    receivedEmpty: "收到空响应。",
  },
};

export function ReactManagedProductsRouteBody({
  language,
}: {
  language: LegacyShellLanguage;
}) {
  const copy = PRODUCT_ROUTE_COPY[language];
  const apiClient = createAPIClient();
  const [routeState, setRouteState] = useState<ProductRouteState>(() => loadProductRouteState());
  const [products, setProducts] = useState<ProductDraft[]>([]);
  const [drafts, setDrafts] = useState<ProductStudioDraft[]>([]);
  const [draft, setDraft] = useState<ProductDraft>(() => normalizeProductBuilderDraft());
  const [draftRequest, setDraftRequest] = useState<DraftRequest>(() => createProductDraftRequest());
  const [selectedDraft, setSelectedDraft] = useState<ProductStudioDraft>(() => normalizeProductStudioDraft());
  const [workspace, setWorkspace] = useState<ProductWorkspace>(() => normalizeProductWorkspace());
  const [workspaceDetail, setWorkspaceDetail] = useState<ProductWorkspaceDetail>(() =>
    normalizeProductWorkspaceSpaceDetail(),
  );
  const [workspaceMessagesByProduct, setWorkspaceMessagesByProduct] = useState<Record<string, ProductWorkspaceMessage[]>>(
    {},
  );
  const [workspaceLoading, setWorkspaceLoading] = useState(false);
  const [workspacePending, setWorkspacePending] = useState(false);
  const [workspaceError, setWorkspaceError] = useState("");
  const [draftEditorText, setDraftEditorText] = useState("");
  const [statusMessage, setStatusMessage] = useState("");
  const [statusKind, setStatusKind] = useState<"success" | "error" | "">("");
  const [loading, setLoading] = useState(true);

  const selectedProduct = useMemo(
    () => products.find((item) => item.id === routeState.selectedProductID) ?? null,
    [products, routeState.selectedProductID],
  );

  const selectedSpace = useMemo(
    () => workspace.spaces.find((item) => item.space_id === routeState.selectedSpaceID) ?? null,
    [workspace.spaces, routeState.selectedSpaceID],
  );

  const currentWorkspaceMessages = workspaceMessagesByProduct[routeState.selectedProductID] ?? [];

  useEffect(() => {
    persistProductRouteState(routeState);
  }, [routeState]);

  useEffect(() => {
    void reload();
  }, []);

  useEffect(() => {
    if (!selectedProduct) {
      setDraft(normalizeProductBuilderDraft());
      return;
    }
    setDraft(normalizeProductBuilderDraft(selectedProduct));
  }, [selectedProduct]);

  useEffect(() => {
    const currentDraft =
      drafts.find((item) => item.draft_id === routeState.selectedDraftID) ?? normalizeProductStudioDraft();
    setSelectedDraft(currentDraft);
    setDraftEditorText(currentDraft.draft_id ? JSON.stringify(currentDraft, null, 2) : "");
  }, [drafts, routeState.selectedDraftID]);

  async function reload(nextStatusMessage = "", nextStatusKind: "success" | "error" | "" = "") {
    setLoading(true);
    setStatusMessage(nextStatusMessage);
    setStatusKind(nextStatusKind);
    try {
      const [productsPayload, draftsPayload] = await Promise.all([
        apiClient.get<{ items?: unknown[] }>("/api/control/products"),
        apiClient.get<{ items?: unknown[] }>("/api/control/products/drafts"),
      ]);
      const nextProducts = Array.isArray(productsPayload?.items)
        ? productsPayload.items.map((item) => normalizeProductBuilderDraft(item)).filter((item) => item.id)
        : [];
      const nextDrafts = Array.isArray(draftsPayload?.items)
        ? draftsPayload.items.map((item) => normalizeProductStudioDraft(item)).filter((item) => item.draft_id)
        : [];
      const selectedProductID =
        nextProducts.some((item) => item.id === routeState.selectedProductID)
          ? routeState.selectedProductID
          : nextProducts[0]?.id ?? "";
      const selectedDraftID =
        nextDrafts.some((item) => item.draft_id === routeState.selectedDraftID)
          ? routeState.selectedDraftID
          : nextDrafts[0]?.draft_id ?? "";

      setProducts(nextProducts);
      setDrafts(nextDrafts);
      setRouteState((current) => ({
        ...current,
        selectedProductID,
        selectedDraftID,
      }));
      await loadWorkspace(selectedProductID);
    } catch (error) {
      setProducts([]);
      setDrafts([]);
      setWorkspace(normalizeProductWorkspace());
      setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail());
      setWorkspaceError(error instanceof Error ? error.message : "unknown_error");
      setStatusMessage(nextStatusMessage);
      setStatusKind(nextStatusKind);
    } finally {
      setLoading(false);
    }
  }

  async function loadWorkspace(productID: string, preferredSpaceID = "") {
    const selectedID = normalizeTextValue(productID);
    if (!selectedID) {
      setWorkspace(normalizeProductWorkspace());
      setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail());
      setWorkspaceError("");
      setWorkspaceLoading(false);
      return;
    }

    setWorkspaceLoading(true);
    setWorkspaceError("");
    try {
      const payload = await apiClient.get(`/api/products/${encodeURIComponent(selectedID)}/workspace`);
      const nextWorkspace = normalizeProductWorkspace(payload);
      let selectedSpaceID = normalizeTextValue(preferredSpaceID) || routeState.selectedSpaceID;
      const validSpaceIDs = new Set(nextWorkspace.spaces.map((item) => item.space_id));
      if (!validSpaceIDs.has(selectedSpaceID)) {
        selectedSpaceID = nextWorkspace.spaces[0]?.space_id ?? "";
      }
      setWorkspace(nextWorkspace);
      setRouteState((current) => ({
        ...current,
        selectedProductID: selectedID,
        selectedSpaceID,
      }));
      if (selectedSpaceID) {
        const detailPayload = await apiClient.get(
          `/api/products/${encodeURIComponent(selectedID)}/workspace/spaces/${encodeURIComponent(selectedSpaceID)}`,
        );
        setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail(detailPayload));
      } else {
        setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail());
      }
    } catch (error) {
      setWorkspace(normalizeProductWorkspace());
      setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail());
      setWorkspaceError(error instanceof Error ? error.message : "unknown_error");
    } finally {
      setWorkspaceLoading(false);
    }
  }

  async function saveProduct(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (selectedProduct?.owner_type === "builtin" && draft.id) {
      return;
    }
    const payload = {
      name: draft.name,
      slug: draft.slug,
      summary: draft.summary,
      status: draft.status,
      visibility: draft.visibility,
      master_agent_id: draft.master_agent_id,
      entry_route: draft.entry_route,
      tags: draft.tags,
      artifact_types: draft.artifact_types,
      knowledge_sources: draft.knowledge_sources,
      worker_agents: draft.worker_agents,
    };
    try {
      const saved = await apiClient.request<{ id?: string }>(
        draft.id ? `/api/control/products/${encodeURIComponent(draft.id)}` : "/api/control/products",
        {
          method: draft.id ? "PUT" : "POST",
          body: payload,
        },
      );
      const nextSelectedProductID = normalizeTextValue(saved?.id) || draft.id;
      setRouteState((current) => ({
        ...current,
        selectedProductID: nextSelectedProductID,
      }));
      await reload(copy.saved, "success");
    } catch (error) {
      setStatusMessage(copy.saveFailed(error instanceof Error ? error.message : "unknown_error"));
      setStatusKind("error");
    }
  }

  async function deleteProduct() {
    if (!selectedProduct?.id || selectedProduct.owner_type === "builtin") {
      return;
    }
    try {
      await apiClient.delete(`/api/control/products/${encodeURIComponent(selectedProduct.id)}`);
      setRouteState((current) => ({
        ...current,
        selectedProductID: "",
      }));
      await reload(copy.deleted, "success");
    } catch (error) {
      setStatusMessage(copy.deleteFailed(error instanceof Error ? error.message : "unknown_error"));
      setStatusKind("error");
    }
  }

  async function generateDraft(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const mode = normalizeTextValue(draftRequest.mode) || "bootstrap";
    if (mode === "expand" && (!selectedProduct?.id || selectedProduct.owner_type === "builtin")) {
      setStatusMessage(copy.draftsFormExpandDisabled);
      setStatusKind("error");
      return;
    }
    const payload = {
      name: draftRequest.name,
      goal: draftRequest.goal,
      target_users: parseCommaSeparatedList(draftRequest.target_users),
      core_capabilities: parseCommaSeparatedList(draftRequest.core_capabilities),
      constraints: parseCommaSeparatedList(draftRequest.constraints),
      expected_artifacts: parseCommaSeparatedList(draftRequest.expected_artifacts),
      integration_requirements: parseCommaSeparatedList(draftRequest.integration_requirements),
      mode,
    };
    const path =
      mode === "expand" && selectedProduct?.id
        ? `/api/control/products/${encodeURIComponent(selectedProduct.id)}/matrix/generate`
        : "/api/control/products/generate";
    try {
      const created = await apiClient.post<{ draft_id?: string }>(path, payload);
      setRouteState((current) => ({
        ...current,
        selectedDraftID: normalizeTextValue(created?.draft_id),
      }));
      setDraftRequest(createProductDraftRequest(mode));
      await reload(copy.draftsGenerated, "success");
    } catch (error) {
      setStatusMessage(copy.draftsGenerateFailed(error instanceof Error ? error.message : "unknown_error"));
      setStatusKind("error");
    }
  }

  async function saveDraftReview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedDraft.draft_id) {
      return;
    }
    try {
      const payload = JSON.parse(draftEditorText || "{}");
      await apiClient.put(`/api/control/products/drafts/${encodeURIComponent(selectedDraft.draft_id)}`, payload);
      await reload(copy.draftsReviewSaved, "success");
    } catch (error) {
      setStatusMessage(copy.draftsReviewFailed(error instanceof Error ? error.message : "unknown_error"));
      setStatusKind("error");
    }
  }

  async function publishDraft() {
    if (!selectedDraft.draft_id) {
      return;
    }
    try {
      const payload = await apiClient.post<{
        draft?: { draft_id?: string };
        product?: { id?: string };
      }>(`/api/control/products/drafts/${encodeURIComponent(selectedDraft.draft_id)}/publish`);
      setRouteState((current) => ({
        ...current,
        selectedDraftID: normalizeTextValue(payload?.draft?.draft_id) || selectedDraft.draft_id,
        selectedProductID: normalizeTextValue(payload?.product?.id) || current.selectedProductID,
      }));
      await reload(copy.draftsPublished, "success");
    } catch (error) {
      setStatusMessage(copy.draftsPublishFailed(error instanceof Error ? error.message : "unknown_error"));
      setStatusKind("error");
    }
  }

  async function selectSpace(spaceID: string) {
    const selectedID = normalizeTextValue(spaceID);
    if (!selectedProduct?.id || !selectedID || selectedID === routeState.selectedSpaceID) {
      return;
    }
    setRouteState((current) => ({
      ...current,
      selectedSpaceID: selectedID,
    }));
    try {
      const detailPayload = await apiClient.get(
        `/api/products/${encodeURIComponent(selectedProduct.id)}/workspace/spaces/${encodeURIComponent(selectedID)}`,
      );
      setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail(detailPayload));
    } catch (error) {
      setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail());
      setWorkspaceError(error instanceof Error ? error.message : "unknown_error");
    }
  }

  async function sendWorkspaceMessage(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProduct?.id || workspacePending) {
      return;
    }
    const form = event.currentTarget;
    const formData = new FormData(form);
    const content = normalizeTextValue(formData.get("content"));
    if (!content) {
      return;
    }
    const assistantMessage = createProductWorkspaceMessage("assistant", copy.processing, {
      status: "streaming",
    });
    setWorkspaceMessagesByProduct((current) => ({
      ...current,
      [selectedProduct.id]: [
        ...(current[selectedProduct.id] ?? []),
        createProductWorkspaceMessage("user", content),
        assistantMessage,
      ],
    }));
    setWorkspacePending(true);
    form.reset();
    try {
      if (selectedProduct.id === "travel") {
        const payload = await apiClient.post<{
          reply?: string;
          guide?: { id?: string };
          space_id?: string;
        }>(`/api/products/${encodeURIComponent(selectedProduct.id)}/workspace/chat`, {
          session_id: routeState.workspaceSessionByProduct[selectedProduct.id] ?? "",
          space_id: routeState.selectedSpaceID,
          content,
        });
        updateWorkspaceAssistantMessage(selectedProduct.id, assistantMessage.id, {
          text: normalizeTextValue(payload?.reply) || copy.receivedEmpty,
          status: "done",
          error: false,
        });
        const nextSpaceID = normalizeTextValue(payload?.guide?.id) || normalizeTextValue(payload?.space_id);
        if (nextSpaceID) {
          await loadWorkspaceWithSpace(selectedProduct.id, nextSpaceID);
        }
      } else {
        const payload = await apiClient.post<{
          result?: {
            output?: string;
            session_id?: string;
          };
        }>(`/api/products/${encodeURIComponent(selectedProduct.id)}/messages`, {
          session_id: routeState.workspaceSessionByProduct[selectedProduct.id] ?? "",
          content,
        });
        updateWorkspaceAssistantMessage(selectedProduct.id, assistantMessage.id, {
          text: normalizeTextValue(payload?.result?.output) || copy.receivedEmpty,
          status: "done",
          error: false,
        });
        const nextSessionID = normalizeTextValue(payload?.result?.session_id);
        if (nextSessionID) {
          setRouteState((current) => ({
            ...current,
            workspaceSessionByProduct: {
              ...current.workspaceSessionByProduct,
              [selectedProduct.id]: nextSessionID,
            },
          }));
        }
      }
      setStatusMessage(copy.workspaceSynced);
      setStatusKind("success");
    } catch (error) {
      const message = copy.workspaceChatFailed(error instanceof Error ? error.message : "unknown_error");
      updateWorkspaceAssistantMessage(selectedProduct.id, assistantMessage.id, {
        text: message,
        status: "error",
        error: true,
      });
      setStatusMessage(message);
      setStatusKind("error");
    } finally {
      setWorkspacePending(false);
    }
  }

  async function loadWorkspaceWithSpace(productID: string, spaceID: string) {
    await loadWorkspace(productID, spaceID);
  }

  function updateWorkspaceAssistantMessage(
    productID: string,
    messageID: string,
    update: Pick<ProductWorkspaceMessage, "text" | "status" | "error">,
  ) {
    setWorkspaceMessagesByProduct((current) => ({
      ...current,
      [productID]: (current[productID] ?? []).map((item) =>
        item.id === messageID
          ? {
              ...item,
              ...update,
            }
          : item,
      ),
    }));
  }

  function startCreateProduct() {
    setRouteState((current) => ({
      ...current,
      selectedProductID: "",
      selectedSpaceID: "",
      activePanel: "studio",
    }));
    setDraft(normalizeProductBuilderDraft());
    setWorkspace(normalizeProductWorkspace());
    setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail());
    setStatusMessage("");
    setStatusKind("");
  }

  function resetDraft() {
    setDraft(selectedProduct ? normalizeProductBuilderDraft(selectedProduct) : normalizeProductBuilderDraft());
    setStatusMessage("");
    setStatusKind("");
  }

  if (loading) {
    return <p className="route-loading">{copy.loading}</p>;
  }

  return (
    <section className="agent-studio-view">
      <aside className="route-surface agent-studio-list-pane">
        <div className="agent-route-pane-head">
          <div className="agent-route-pane-copy">
            <h4>{copy.title}</h4>
          </div>
          <button className="route-primary-button" type="button" onClick={startCreateProduct}>
            {copy.create}
          </button>
        </div>
        <div className="agent-route-list">
          {products.length ? (
            products.map((item) => (
              <button
                key={item.id}
                className={`agent-route-card${item.id === routeState.selectedProductID ? " is-active" : ""}`}
                type="button"
                onClick={() => {
                  if (item.id === routeState.selectedProductID) {
                    return;
                  }
                  setRouteState((current) => ({
                    ...current,
                    selectedProductID: item.id,
                    selectedSpaceID: "",
                  }));
                  void loadWorkspace(item.id);
                }}
              >
                <div className="agent-route-card-head">
                  <div className="agent-route-card-copy">
                    <h4>{item.name || item.id}</h4>
                    <span title={item.id}>{item.id}</span>
                  </div>
                  <span className={`agent-route-state ${item.status === "active" ? "is-enabled" : "is-disabled"}`}>
                    {normalizeText(item.status || "draft")}
                  </span>
                </div>
                <p className="agent-route-card-prompt">{item.summary || copy.empty}</p>
                <div className="agent-route-card-tags">
                  {[normalizeText(item.owner_type), normalizeText(item.status), `${item.worker_agents.length} agents`]
                    .filter((tag) => tag !== "-")
                    .map((tag) => (
                      <span key={`${item.id}-${tag}`}>{tag}</span>
                    ))}
                </div>
              </button>
            ))
          ) : (
            <p className="route-empty">{copy.empty}</p>
          )}
        </div>
      </aside>
      <section className="route-surface agent-studio-form-pane">
        {statusMessage ? (
          <p className={`agent-builder-status ${statusKind === "error" ? "is-error" : "is-success"}`}>
            {statusMessage}
          </p>
        ) : null}
        <div className="product-workspace-tabs">
          <button
            className={`memory-tab ${routeState.activePanel === "workspace" ? "active" : ""}`}
            type="button"
            onClick={() => setRouteState((current) => ({ ...current, activePanel: "workspace" }))}
          >
            {copy.panelWorkspace}
          </button>
          <button
            className={`memory-tab ${routeState.activePanel === "studio" ? "active" : ""}`}
            type="button"
            onClick={() => setRouteState((current) => ({ ...current, activePanel: "studio" }))}
          >
            {copy.panelStudio}
          </button>
        </div>
        {routeState.activePanel === "workspace"
          ? renderWorkspacePanel({
              copy,
              selectedProduct,
              selectedSpace,
              workspace,
              workspaceDetail,
              workspaceLoading,
              workspaceError,
              workspacePending,
              currentWorkspaceMessages,
              routeState,
              onSelectSpace: selectSpace,
              onClearSpace: () => {
                setRouteState((current) => ({
                  ...current,
                  selectedSpaceID: "",
                }));
                setWorkspaceDetail(normalizeProductWorkspaceSpaceDetail());
              },
              onSendWorkspaceMessage: sendWorkspaceMessage,
            })
          : renderStudioPanel({
              copy,
              selectedProduct,
              draft,
              setDraft,
              draftRequest,
              setDraftRequest,
              drafts,
              selectedDraft,
              draftEditorText,
              setDraftEditorText,
              onReset: resetDraft,
              onSaveProduct: saveProduct,
              onDeleteProduct: deleteProduct,
              onGenerateDraft: generateDraft,
              onPublishDraft: publishDraft,
              onSaveDraftReview: saveDraftReview,
              routeState,
              setRouteState,
            })}
      </section>
    </section>
  );
}

function renderWorkspacePanel({
  copy,
  selectedProduct,
  selectedSpace,
  workspace,
  workspaceDetail,
  workspaceLoading,
  workspaceError,
  workspacePending,
  currentWorkspaceMessages,
  routeState,
  onSelectSpace,
  onClearSpace,
  onSendWorkspaceMessage,
}: {
  copy: ProductRouteCopy;
  selectedProduct: ProductDraft | null;
  selectedSpace: WorkspaceSpaceSummary | null;
  workspace: ProductWorkspace;
  workspaceDetail: ProductWorkspaceDetail;
  workspaceLoading: boolean;
  workspaceError: string;
  workspacePending: boolean;
  currentWorkspaceMessages: ProductWorkspaceMessage[];
  routeState: ProductRouteState;
  onSelectSpace: (spaceID: string) => void;
  onClearSpace: () => void;
  onSendWorkspaceMessage: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (!selectedProduct?.id) {
    return <p className="route-empty">{copy.workspaceEmpty}</p>;
  }
  if (workspaceLoading) {
    return <p className="route-loading">{copy.loading}</p>;
  }
  if (workspaceError) {
    return <p className="route-error">{workspaceError}</p>;
  }
  const guide = workspaceDetail.guide;
  const chatPlaceholder = selectedSpace
    ? `${copy.workspaceChatPlaceholder} (${selectedSpace.title})`
    : copy.workspaceChatPlaceholder;

  return (
    <section className="product-workspace-grid">
      <section className="route-surface product-workspace-panel">
        <div className="agent-route-pane-head">
          <div className="agent-route-pane-copy">
            <h4>{copy.workspaceTitle}</h4>
          </div>
        </div>
        <div className="agent-builder-managed">
          <ManagedItem label={copy.formID} value={selectedProduct.id || "-"} />
          <ManagedItem label={copy.formStatus} value={selectedProduct.status || "-"} />
          <ManagedItem label={copy.formVisibility} value={selectedProduct.visibility || "-"} />
          <ManagedItem label={copy.formVersion} value={selectedProduct.version || "-"} />
        </div>
        <div className="product-workspace-overview-grid">
          <DetailSection title={copy.workspaceOverview}>
            <p className="product-workspace-summary">{selectedProduct.summary || "-"}</p>
          </DetailSection>
          <DetailSection title={copy.workspaceMaster}>
            {workspace.master_agent ? (
              <>
                <div className="product-workspace-master">
                  <strong>{workspace.master_agent.name || workspace.master_agent.agent_id || "-"}</strong>
                  <span>{workspace.master_agent.agent_id || "-"}</span>
                  <p>{workspace.master_agent.description || "-"}</p>
                </div>
                <ChipList items={[...workspace.master_agent.capabilities, ...workspace.master_agent.tools]} />
              </>
            ) : (
              <p className="route-empty">-</p>
            )}
          </DetailSection>
          <DetailSection title={copy.workspaceOutput}>
            <ChipList items={selectedProduct.artifact_types} />
          </DetailSection>
          <DetailSection title={copy.workspaceSources}>
            <ChipList items={selectedProduct.knowledge_sources} />
          </DetailSection>
          <DetailSection title={copy.workspaceTags}>
            <ChipList items={selectedProduct.tags} />
          </DetailSection>
          {selectedProduct.worker_agents.length ? (
            <DetailSection title={copy.workspaceWorkers}>
              <ChipList items={selectedProduct.worker_agents.map((item) => item.role || item.agent_id)} />
            </DetailSection>
          ) : null}
        </div>
      </section>

      <section className="route-surface product-workspace-panel">
        <div className="agent-route-pane-head">
          <div className="agent-route-pane-copy">
            <h4>{copy.workspaceChatTitle}</h4>
          </div>
        </div>
        <div className="product-workspace-chat-log">
          {currentWorkspaceMessages.length ? (
            currentWorkspaceMessages.map((item) => (
              <article
                key={item.id}
                className={`product-workspace-message is-${item.role}${item.error ? " is-error" : ""}`}
              >
                <div className="product-workspace-message-meta">
                  <strong>{item.role === "user" ? "You" : "Agent"}</strong>
                  <span>{item.status === "streaming" ? copy.processing : formatDateTime(item.at) || ""}</span>
                </div>
                <p>{item.text || "-"}</p>
              </article>
            ))
          ) : (
            <p className="route-empty">{copy.workspaceChatHint}</p>
          )}
        </div>
        <form className="product-workspace-chat-form" onSubmit={onSendWorkspaceMessage}>
          <label>
            <span className="sr-only">{copy.workspaceChatAriaLabel}</span>
            <textarea
              aria-label={copy.workspaceChatAriaLabel}
              name="content"
              rows={4}
              placeholder={chatPlaceholder}
              disabled={workspacePending}
            ></textarea>
          </label>
          <div className="task-filter-actions">
            <button
              className="task-filter-reset"
              type="button"
              onClick={onClearSpace}
              disabled={!selectedSpace}
            >
              {copy.workspaceOpenNew}
            </button>
            <button className="task-filter-apply" type="submit" disabled={workspacePending}>
              {copy.workspaceChatSend}
            </button>
          </div>
        </form>
      </section>

      <section className="route-surface product-workspace-panel">
        <div className="agent-route-pane-head">
          <div className="agent-route-pane-copy">
            <h4>{workspace.space_label || copy.workspaceSpaces}</h4>
          </div>
        </div>
        <div className="agent-route-list">
          {workspace.spaces.length ? (
            workspace.spaces.map((item) => (
              <article key={item.space_id} className={`agent-route-card${item.space_id === routeState.selectedSpaceID ? " is-active" : ""}`}>
                <button className="agent-route-card-button" type="button" onClick={() => onSelectSpace(item.space_id)}>
                  <div className="agent-route-card-head">
                    <div className="agent-route-card-copy">
                      <strong title={item.title || item.space_id}>{item.title || item.space_id}</strong>
                      <span title={item.slug || item.space_id}>{item.slug || item.space_id}</span>
                    </div>
                    <span className={`agent-route-state ${item.status === "active" ? "is-enabled" : "is-disabled"}`}>
                      {normalizeText(item.status || "active")}
                    </span>
                  </div>
                  <p className="agent-route-card-prompt">{item.summary || "-"}</p>
                  <div className="agent-route-card-tags">
                    {item.tags.map((tag) => (
                      <span key={`${item.space_id}-${tag}`}>{tag}</span>
                    ))}
                  </div>
                </button>
                {item.html_path ? (
                  <div className="task-filter-actions">
                    <a className="task-filter-reset" href={item.html_path} target="_blank" rel="noreferrer">
                      {copy.workspaceOpenPage}
                    </a>
                  </div>
                ) : null}
              </article>
            ))
          ) : (
            <p className="route-empty">{copy.workspaceSpaceEmpty}</p>
          )}
        </div>
      </section>

      <section className="route-surface product-workspace-panel product-workspace-panel-wide">
        <div className="agent-route-pane-head">
          <div className="agent-route-pane-copy">
            <h4>{selectedSpace?.title || copy.workspaceDetail}</h4>
            <p>{selectedSpace?.summary || copy.workspaceDetailEmpty}</p>
          </div>
        </div>
        {selectedSpace && guide ? (
          <>
            {selectedSpace.html_path ? (
              <div className="task-filter-actions">
                <a className="task-filter-apply" href={selectedSpace.html_path} target="_blank" rel="noreferrer">
                  {copy.workspaceOpenPage}
                </a>
              </div>
            ) : null}
            <div className="agent-builder-managed">
              <ManagedItem label={copy.workspaceDays} value={String(guide.days || 0)} />
              <ManagedItem label={copy.workspaceSpaceRevision} value={String(guide.revision || 0)} />
              <ManagedItem label={copy.workspaceUpdated} value={formatDateTime(guide.updated_at) || "-"} />
            </div>
            <div className="product-workspace-detail-grid">
              <DetailSection title={copy.workspaceTags}>
                <ChipList items={[guide.city, guide.travel_style, guide.budget].filter(Boolean)} />
              </DetailSection>
              <DetailSection title={copy.workspaceCompanions}>
                <ChipList items={[...guide.companions, ...guide.must_visit]} />
              </DetailSection>
              <DetailSection title={copy.workspaceAvoid}>
                <ChipList
                  items={[
                    ...guide.avoid,
                    ...guide.additional_requirements,
                    ...guide.keep_conditions,
                    ...guide.replace_conditions,
                  ]}
                />
              </DetailSection>
            </div>
            <section className="product-workspace-detail-section product-workspace-detail-content">
              <h5>{copy.workspaceContent}</h5>
              <pre>{guide.content || "-"}</pre>
            </section>
            <DetailSection title={copy.workspaceNotes}>
              <ChipList items={guide.notes} />
            </DetailSection>
            <DetailSection title={copy.workspaceRoutes}>
              <div className="product-workspace-detail-stack">
                {guide.daily_routes.length ? (
                  guide.daily_routes.map((item) => (
                    <article key={`${guide.id}-day-${item.day}`} className="product-workspace-detail-card">
                      <strong>{`Day ${item.day}`}</strong>
                      <p>{item.theme || item.stops.join(" -> ") || "-"}</p>
                      {item.stops.length ? <span>{item.stops.join(" -> ")}</span> : null}
                      {item.transit.length ? <span>{item.transit.join(" / ")}</span> : null}
                    </article>
                  ))
                ) : (
                  <p className="route-empty">-</p>
                )}
              </div>
            </DetailSection>
            <DetailSection title={copy.workspaceLayers}>
              <div className="product-workspace-detail-stack">
                {guide.map_layers.length ? (
                  guide.map_layers.map((item) => (
                    <article key={`${guide.id}-${item.id}`} className="product-workspace-detail-card">
                      <strong>{item.label || item.id || "-"}</strong>
                      <p>{item.description || "-"}</p>
                    </article>
                  ))
                ) : (
                  <p className="route-empty">-</p>
                )}
              </div>
            </DetailSection>
          </>
        ) : (
          <p className="route-empty">{copy.workspaceDetailEmpty}</p>
        )}
      </section>
    </section>
  );
}

function renderStudioPanel({
  copy,
  selectedProduct,
  draft,
  setDraft,
  draftRequest,
  setDraftRequest,
  drafts,
  selectedDraft,
  draftEditorText,
  setDraftEditorText,
  onReset,
  onSaveProduct,
  onDeleteProduct,
  onGenerateDraft,
  onPublishDraft,
  onSaveDraftReview,
  routeState,
  setRouteState,
}: {
  copy: ProductRouteCopy;
  selectedProduct: ProductDraft | null;
  draft: ProductDraft;
  setDraft: Dispatch<SetStateAction<ProductDraft>>;
  draftRequest: DraftRequest;
  setDraftRequest: Dispatch<SetStateAction<DraftRequest>>;
  drafts: ProductStudioDraft[];
  selectedDraft: ProductStudioDraft;
  draftEditorText: string;
  setDraftEditorText: Dispatch<SetStateAction<string>>;
  onReset: () => void;
  onSaveProduct: (event: FormEvent<HTMLFormElement>) => void;
  onDeleteProduct: () => void;
  onGenerateDraft: (event: FormEvent<HTMLFormElement>) => void;
  onPublishDraft: () => void;
  onSaveDraftReview: (event: FormEvent<HTMLFormElement>) => void;
  routeState: ProductRouteState;
  setRouteState: Dispatch<SetStateAction<ProductRouteState>>;
}) {
  const isBuiltin = selectedProduct?.owner_type === "builtin";
  const canDelete = Boolean(selectedProduct?.id) && !isBuiltin;
  const canSave = !isBuiltin;
  const currentDraftMode = draftRequest.mode || "bootstrap";
  const canExpandDraft = Boolean(selectedProduct?.id) && !isBuiltin;
  const canPublishDraft = Boolean(selectedDraft.draft_id) && !selectedDraft.published_product_id && selectedDraft.product.owner_type !== "builtin";
  const masterTags = [
    `${selectedDraft.master_agent.tools.length} tools`,
    `${selectedDraft.master_agent.skills.length} skills`,
    `${selectedDraft.master_agent.allowed_delegate_targets.length} delegates`,
  ].filter((tag) => !tag.startsWith("0 "));

  return (
    <>
      <div className="agent-route-pane-head">
        <div className="agent-route-pane-copy">
          <h4>{draft.id ? copy.edit : copy.formNew}</h4>
        </div>
        <div className="agent-builder-actions">
          <button type="button" onClick={onReset}>
            {copy.formCancel}
          </button>
        </div>
      </div>
      {isBuiltin ? <p className="agent-builder-status">{copy.formBuiltinNotice}</p> : null}
      <div className="agent-builder-managed">
        <ManagedItem label={copy.formID} value={draft.id || "-"} />
        <ManagedItem label={copy.formVersion} value={draft.version || "-"} />
        <ManagedItem label={copy.formOwner} value={draft.owner_type || "managed"} />
      </div>
      <form className="agent-builder-form" onSubmit={onSaveProduct}>
        <FormInput label={copy.formName} value={draft.name} onChange={(value) => setDraft((current) => ({ ...current, name: value }))} />
        <FormInput label={copy.formSlug} value={draft.slug} onChange={(value) => setDraft((current) => ({ ...current, slug: value }))} />
        <FormInput
          label={copy.formMaster}
          value={draft.master_agent_id}
          onChange={(value) => setDraft((current) => ({ ...current, master_agent_id: value }))}
        />
        <FormSelect
          label={copy.formStatus}
          value={draft.status}
          options={["draft", "active", "disabled", "archived"]}
          onChange={(value) => setDraft((current) => ({ ...current, status: value }))}
        />
        <FormSelect
          label={copy.formVisibility}
          value={draft.visibility}
          options={["private", "public"]}
          onChange={(value) => setDraft((current) => ({ ...current, visibility: value }))}
        />
        <FormInput
          label={copy.formEntryRoute}
          value={draft.entry_route}
          onChange={(value) => setDraft((current) => ({ ...current, entry_route: value }))}
        />
        <FormTextArea
          label={copy.formSummary}
          rows={4}
          wide
          value={draft.summary}
          onChange={(value) => setDraft((current) => ({ ...current, summary: value }))}
        />
        <FormInput
          label={copy.formTags}
          wide
          value={draft.tags.join(", ")}
          onChange={(value) => setDraft((current) => ({ ...current, tags: parseCommaSeparatedList(value) }))}
        />
        <FormInput
          label={copy.formArtifacts}
          wide
          value={draft.artifact_types.join(", ")}
          onChange={(value) =>
            setDraft((current) => ({ ...current, artifact_types: parseCommaSeparatedList(value) }))
          }
        />
        <FormInput
          label={copy.formKnowledge}
          wide
          value={draft.knowledge_sources.join(", ")}
          onChange={(value) =>
            setDraft((current) => ({ ...current, knowledge_sources: parseCommaSeparatedList(value) }))
          }
        />
        <FormTextArea
          label={copy.formWorkerAgents}
          rows={6}
          wide
          value={serializeProductWorkerAgents(draft.worker_agents)}
          onChange={(value) => setDraft((current) => ({ ...current, worker_agents: parseProductWorkerAgentInput(value) }))}
        />
        <div className="task-filter-actions">
          <button className="task-filter-apply" type="submit" disabled={!canSave}>
            {copy.formSave}
          </button>
          <button className="task-filter-reset" type="button" onClick={onDeleteProduct} disabled={!canDelete}>
            {copy.formDelete}
          </button>
        </div>
      </form>

      <div className="agent-builder-section">
        <div className="agent-route-pane-head">
          <div className="agent-route-pane-copy">
            <h4>{copy.draftsTitle}</h4>
          </div>
        </div>
        {currentDraftMode === "expand" && !canExpandDraft ? (
          <p className="agent-builder-status">{copy.draftsFormExpandDisabled}</p>
        ) : null}
        {currentDraftMode === "expand" && canExpandDraft && selectedProduct?.id ? (
          <p className="agent-builder-status">{`${copy.formID}: ${selectedProduct.id}`}</p>
        ) : null}
        <form className="agent-builder-form" onSubmit={onGenerateDraft}>
          <FormInput
            label={copy.draftsFormName}
            value={draftRequest.name}
            onChange={(value) => setDraftRequest((current) => ({ ...current, name: value }))}
          />
          <FormSelect
            label={copy.draftsFormMode}
            value={currentDraftMode}
            options={[
              { value: "bootstrap", label: copy.draftsFormModeBootstrap },
              { value: "expand", label: copy.draftsFormModeExpand },
            ]}
            onChange={(value) => setDraftRequest((current) => ({ ...current, mode: value }))}
          />
          <FormTextArea
            label={copy.draftsFormGoal}
            rows={3}
            wide
            value={draftRequest.goal}
            onChange={(value) => setDraftRequest((current) => ({ ...current, goal: value }))}
          />
          <FormInput
            label={copy.draftsFormTargetUsers}
            wide
            value={draftRequest.target_users}
            onChange={(value) => setDraftRequest((current) => ({ ...current, target_users: value }))}
          />
          <FormInput
            label={copy.draftsFormCore}
            wide
            value={draftRequest.core_capabilities}
            onChange={(value) => setDraftRequest((current) => ({ ...current, core_capabilities: value }))}
          />
          <FormInput
            label={copy.draftsFormConstraints}
            wide
            value={draftRequest.constraints}
            onChange={(value) => setDraftRequest((current) => ({ ...current, constraints: value }))}
          />
          <FormInput
            label={copy.draftsFormArtifacts}
            wide
            value={draftRequest.expected_artifacts}
            onChange={(value) => setDraftRequest((current) => ({ ...current, expected_artifacts: value }))}
          />
          <FormInput
            label={copy.draftsFormIntegrations}
            wide
            value={draftRequest.integration_requirements}
            onChange={(value) =>
              setDraftRequest((current) => ({ ...current, integration_requirements: value }))
            }
          />
          <div className="task-filter-actions">
            <button
              className="task-filter-apply"
              type="submit"
              disabled={currentDraftMode === "expand" && !canExpandDraft}
            >
              {copy.draftsFormGenerate}
            </button>
          </div>
        </form>

        <div className="agent-route-list">
          {drafts.length ? (
            drafts.map((item) => {
              const workerCount = item.worker_matrix.length;
              const topologyTag = workerCount > 0 ? `${workerCount} workers` : "single agent";
              return (
                <button
                  key={item.draft_id}
                  className={`agent-route-card${item.draft_id === routeState.selectedDraftID ? " is-active" : ""}`}
                  type="button"
                  onClick={() =>
                    setRouteState((current) => ({
                      ...current,
                      selectedDraftID: item.draft_id,
                    }))
                  }
                >
                  <div className="agent-route-card-head">
                    <div className="agent-route-card-copy">
                      <strong title={item.product.name || item.product.id || item.draft_id}>
                        {item.product.name || item.product.id || item.draft_id}
                      </strong>
                      <span title={item.draft_id}>{item.draft_id}</span>
                    </div>
                    <span
                      className={`agent-route-state ${
                        item.review_status === "published" || item.review_status === "reviewed"
                          ? "is-enabled"
                          : "is-disabled"
                      }`}
                    >
                      {normalizeText(item.review_status)}
                    </span>
                  </div>
                  <p className="agent-route-card-prompt">{item.goal || item.product.summary || copy.draftsEmpty}</p>
                  <div className="agent-route-card-tags">
                    {[normalizeText(item.mode), normalizeText(item.review_status), normalizeText(item.product.id), topologyTag]
                      .filter((tag) => tag !== "-")
                      .map((tag) => (
                        <span key={`${item.draft_id}-${tag}`}>{tag}</span>
                      ))}
                  </div>
                </button>
              );
            })
          ) : (
            <p className="route-empty">{copy.draftsEmpty}</p>
          )}
        </div>

        {selectedDraft.draft_id ? (
          <>
            <div className="agent-builder-managed">
              <ManagedItem label={copy.formID} value={selectedDraft.draft_id} />
              <ManagedItem label={copy.draftsDetailReview} value={selectedDraft.review_status} />
              <ManagedItem label={copy.draftsDetailMode} value={selectedDraft.mode} />
              <ManagedItem label={copy.draftsDetailProduct} value={selectedDraft.product.id || "-"} />
              <ManagedItem label={copy.draftsDetailGenerated} value={formatDateTime(selectedDraft.generated_at) || "-"} />
              <ManagedItem label={copy.draftsDetailUpdated} value={formatDateTime(selectedDraft.updated_at) || "-"} />
              <ManagedItem label={copy.draftsDetailPublished} value={selectedDraft.published_product_id || "-"} />
            </div>
            <div className="product-workspace-detail-grid">
              <DetailSection title={copy.draftsDetailMaster}>
                <div className="product-workspace-master">
                  <strong>{selectedDraft.master_agent.name || selectedDraft.master_agent.agent_id || "-"}</strong>
                  <span>{selectedDraft.master_agent.agent_id || "-"}</span>
                  <p>{selectedDraft.master_agent.description || "-"}</p>
                </div>
                <ChipList items={masterTags} />
              </DetailSection>
              <DetailSection title={copy.draftsDetailWorkers}>
                {selectedDraft.worker_matrix.length ? (
                  <div className="agent-route-list">{selectedDraft.worker_matrix.map((item) => renderDraftWorkerCard(item, copy))}</div>
                ) : (
                  <p className="route-empty">{copy.draftsEmpty}</p>
                )}
              </DetailSection>
              <DetailSection title={copy.draftsDetailConflicts}>
                <ChipList items={selectedDraft.conflict_suggestions} />
              </DetailSection>
            </div>
            <form className="agent-builder-form" onSubmit={onSaveDraftReview}>
              <FormTextArea
                label={copy.draftsFormEditor}
                rows={18}
                wide
                value={draftEditorText}
                onChange={(value) => setDraftEditorText(value)}
              />
              <div className="agent-builder-wide">
                <p className="agent-route-pane-copy">
                  <span>{copy.draftsFormEditorHint}</span>
                </p>
              </div>
              <div className="task-filter-actions">
                <button className="task-filter-apply" type="submit">
                  {copy.draftsFormSave}
                </button>
                <button className="task-filter-reset" type="button" onClick={onPublishDraft} disabled={!canPublishDraft}>
                  {copy.draftsFormPublish}
                </button>
              </div>
            </form>
          </>
        ) : (
          <p className="route-empty">{copy.draftsDetailEmpty}</p>
        )}
      </div>
    </>
  );
}

function renderDraftWorkerCard(item: ProductDraftWorker, copy: ProductRouteCopy) {
  const workerName = item.name || item.agent_id;
  const tags = [normalizeText(item.role), `${item.allowed_tools.length} tools`, `${item.dependencies.length} deps`].filter(
    (tag) => tag !== "-",
  );
  const prompt = [item.responsibility, item.input_contract, item.output_contract].filter(Boolean).join(" | ");
  return (
    <article key={item.agent_id} className="agent-route-card">
      <div className="agent-route-card-head">
        <div className="agent-route-card-copy">
          <strong title={workerName}>{workerName}</strong>
          <span title={item.agent_id}>{item.agent_id}</span>
        </div>
        <span className={`agent-route-state ${item.enabled ? "is-enabled" : "is-disabled"}`}>
          {item.enabled ? copy.statusEnabled : copy.statusDisabled}
        </span>
      </div>
      <p className="agent-route-card-prompt">{prompt || item.description || "-"}</p>
      <div className="agent-route-card-tags">
        {tags.map((tag) => (
          <span key={`${item.agent_id}-${tag}`}>{tag}</span>
        ))}
      </div>
    </article>
  );
}

function ManagedItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="agent-builder-managed-item">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function DetailSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="product-workspace-detail-section">
      <h5>{title}</h5>
      {children}
    </section>
  );
}

function ChipList({ items }: { items: string[] }) {
  const values = items.filter(Boolean);
  if (!values.length) {
    return <p className="route-empty">-</p>;
  }
  return (
    <div className="product-workspace-chip-list">
      {values.map((item) => (
        <span key={item}>{item}</span>
      ))}
    </div>
  );
}

function FormInput({
  label,
  value,
  onChange,
  wide = false,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  wide?: boolean;
}) {
  return (
    <label className={wide ? "agent-builder-wide" : undefined}>
      <span>{label}</span>
      <input type="text" value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function FormTextArea({
  label,
  value,
  onChange,
  rows,
  wide = false,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  rows: number;
  wide?: boolean;
}) {
  return (
    <label className={wide ? "agent-builder-wide" : undefined}>
      <span>{label}</span>
      <textarea rows={rows} value={value} onChange={(event) => onChange(event.target.value)}></textarea>
    </label>
  );
}

function FormSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: string[] | Array<{ value: string; label: string }>;
  onChange: (value: string) => void;
}) {
  return (
    <label>
      <span>{label}</span>
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        {options.map((option) =>
          typeof option === "string" ? (
            <option key={option} value={option}>
              {option}
            </option>
          ) : (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ),
        )}
      </select>
    </label>
  );
}

function loadProductRouteState(): ProductRouteState {
  try {
    const raw = window.localStorage.getItem(PRODUCT_ROUTE_STORAGE_KEY);
    if (!raw) {
      return normalizeProductRouteState();
    }
    return normalizeProductRouteState(JSON.parse(raw));
  } catch {
    return normalizeProductRouteState();
  }
}

function persistProductRouteState(routeState: ProductRouteState) {
  try {
    window.localStorage.setItem(PRODUCT_ROUTE_STORAGE_KEY, JSON.stringify(routeState));
  } catch {
  }
}

function normalizeProductRouteState(routeState: Partial<ProductRouteState> = {}): ProductRouteState {
  return {
    selectedProductID: normalizeTextValue(routeState.selectedProductID),
    selectedDraftID: normalizeTextValue(routeState.selectedDraftID),
    activePanel: routeState.activePanel === "studio" ? "studio" : "workspace",
    selectedSpaceID: normalizeTextValue(routeState.selectedSpaceID),
    workspaceSessionByProduct:
      routeState.workspaceSessionByProduct && typeof routeState.workspaceSessionByProduct === "object"
        ? { ...routeState.workspaceSessionByProduct }
        : {},
  };
}

function normalizeProductBuilderDraft(product: unknown = {}): ProductDraft {
  const value = asRecord(product);
  return {
    id: normalizeTextValue(value.id),
    name: normalizeTextValue(value.name),
    slug: normalizeTextValue(value.slug),
    summary: normalizeTextValue(value.summary),
    status: normalizeTextValue(value.status) || "draft",
    visibility: normalizeTextValue(value.visibility) || "private",
    owner_type: normalizeTextValue(value.owner_type) || "managed",
    version: normalizeTextValue(value.version),
    master_agent_id: normalizeTextValue(value.master_agent_id),
    entry_route: normalizeTextValue(value.entry_route) || "products",
    tags: normalizeStringArray(value.tags),
    artifact_types: normalizeStringArray(value.artifact_types),
    knowledge_sources: normalizeStringArray(value.knowledge_sources),
    worker_agents: Array.isArray(value.worker_agents)
      ? value.worker_agents
          .map((item) => {
            const worker = asRecord(item);
            return {
              agent_id: normalizeTextValue(worker.agent_id),
              role: normalizeTextValue(worker.role),
              responsibility: normalizeTextValue(worker.responsibility),
              capabilities: normalizeStringArray(worker.capabilities),
              enabled: worker.enabled !== false,
            };
          })
          .filter((item) => item.agent_id)
      : [],
  };
}

function normalizeProductDraftAgent(agent: unknown = {}): ProductDraftAgent {
  const value = asRecord(agent);
  return {
    agent_id: normalizeTextValue(value.agent_id),
    name: normalizeTextValue(value.name),
    description: normalizeTextValue(value.description),
    system_prompt: normalizeTextValue(value.system_prompt),
    max_iterations: toNonNegativeNumber(value.max_iterations),
    tools: normalizeStringArray(value.tools),
    skills: normalizeStringArray(value.skills),
    mcps: normalizeStringArray(value.mcps),
    memory_files: normalizeStringArray(value.memory_files),
    capabilities: normalizeStringArray(value.capabilities),
    allowed_delegate_targets: normalizeStringArray(value.allowed_delegate_targets),
    enabled: value.enabled !== false,
    delegatable: value.delegatable !== false,
  };
}

function normalizeProductDraftWorker(worker: unknown = {}): ProductDraftWorker {
  const value = asRecord(worker);
  return {
    agent_id: normalizeTextValue(value.agent_id),
    name: normalizeTextValue(value.name),
    role: normalizeTextValue(value.role),
    responsibility: normalizeTextValue(value.responsibility),
    description: normalizeTextValue(value.description),
    system_prompt: normalizeTextValue(value.system_prompt),
    input_contract: normalizeTextValue(value.input_contract),
    output_contract: normalizeTextValue(value.output_contract),
    allowed_tools: normalizeStringArray(value.allowed_tools),
    allowed_delegate_targets: normalizeStringArray(value.allowed_delegate_targets),
    dependencies: normalizeStringArray(value.dependencies),
    skills: normalizeStringArray(value.skills),
    mcps: normalizeStringArray(value.mcps),
    memory_files: normalizeStringArray(value.memory_files),
    capabilities: normalizeStringArray(value.capabilities),
    priority: toNonNegativeNumber(value.priority),
    max_iterations: toNonNegativeNumber(value.max_iterations),
    enabled: value.enabled !== false,
  };
}

function normalizeProductStudioDraft(draft: unknown = {}): ProductStudioDraft {
  const value = asRecord(draft);
  return {
    draft_id: normalizeTextValue(value.draft_id),
    mode: normalizeTextValue(value.mode) || "bootstrap",
    review_status: normalizeTextValue(value.review_status) || "draft",
    generated_by: normalizeTextValue(value.generated_by),
    generated_at: normalizeTextValue(value.generated_at),
    updated_at: normalizeTextValue(value.updated_at),
    goal: normalizeTextValue(value.goal),
    target_users: normalizeStringArray(value.target_users),
    core_capabilities: normalizeStringArray(value.core_capabilities),
    constraints: normalizeStringArray(value.constraints),
    expected_artifacts: normalizeStringArray(value.expected_artifacts),
    integration_requirements: normalizeStringArray(value.integration_requirements),
    conflict_suggestions: normalizeStringArray(value.conflict_suggestions),
    published_product_id: normalizeTextValue(value.published_product_id),
    product: normalizeProductBuilderDraft(value.product),
    master_agent: normalizeProductDraftAgent(value.master_agent),
    worker_matrix: Array.isArray(value.worker_matrix)
      ? value.worker_matrix.map((item) => normalizeProductDraftWorker(item)).filter((item) => item.agent_id)
      : [],
  };
}

function normalizeProductWorkspaceMasterAgent(agent: unknown = {}): WorkspaceMasterAgent {
  const value = asRecord(agent);
  return {
    agent_id: normalizeTextValue(value.agent_id),
    name: normalizeTextValue(value.name),
    description: normalizeTextValue(value.description),
    capabilities: normalizeStringArray(value.capabilities),
    tools: normalizeStringArray(value.tools),
    skills: normalizeStringArray(value.skills),
    mcps: normalizeStringArray(value.mcps),
    memory_files: normalizeStringArray(value.memory_files),
  };
}

function normalizeProductWorkspaceSpaceSummary(space: unknown = {}): WorkspaceSpaceSummary {
  const value = asRecord(space);
  return {
    space_id: normalizeTextValue(value.space_id),
    title: normalizeTextValue(value.title),
    slug: normalizeTextValue(value.slug),
    html_path: normalizeTextValue(value.html_path),
    summary: normalizeTextValue(value.summary),
    type: normalizeTextValue(value.type),
    status: normalizeTextValue(value.status) || "active",
    revision: toNonNegativeNumber(value.revision),
    updated_at: normalizeTextValue(value.updated_at),
    tags: normalizeStringArray(value.tags),
  };
}

function normalizeProductWorkspace(payload: unknown = {}): ProductWorkspace {
  const value = asRecord(payload);
  return {
    product: normalizeProductBuilderDraft(value.product),
    master_agent: value.master_agent ? normalizeProductWorkspaceMasterAgent(value.master_agent) : null,
    space_type: normalizeTextValue(value.space_type),
    space_label: normalizeTextValue(value.space_label),
    workspace_hint: normalizeTextValue(value.workspace_hint),
    spaces: Array.isArray(value.spaces)
      ? value.spaces.map((item) => normalizeProductWorkspaceSpaceSummary(item)).filter((item) => item.space_id)
      : [],
  };
}

function normalizeProductWorkspaceSpaceDetail(payload: unknown = {}): ProductWorkspaceDetail {
  const value = asRecord(payload);
  const guideRecord = asRecord(value.guide);
  return {
    space: normalizeProductWorkspaceSpaceSummary(value.space),
    guide: value.guide && typeof value.guide === "object"
      ? {
          id: normalizeTextValue(guideRecord.id),
          city: normalizeTextValue(guideRecord.city),
          days: toNonNegativeNumber(guideRecord.days),
          travel_style: normalizeTextValue(guideRecord.travel_style),
          budget: normalizeTextValue(guideRecord.budget),
          companions: normalizeStringArray(guideRecord.companions),
          must_visit: normalizeStringArray(guideRecord.must_visit),
          avoid: normalizeStringArray(guideRecord.avoid),
          additional_requirements: normalizeStringArray(guideRecord.additional_requirements),
          keep_conditions: normalizeStringArray(guideRecord.keep_conditions),
          replace_conditions: normalizeStringArray(guideRecord.replace_conditions),
          notes: normalizeStringArray(guideRecord.notes),
          daily_routes: Array.isArray(guideRecord.daily_routes)
            ? guideRecord.daily_routes.map((item) => {
                const route = asRecord(item);
                return {
                  day: toNonNegativeNumber(route.day),
                  theme: normalizeTextValue(route.theme),
                  stops: normalizeStringArray(route.stops),
                  transit: normalizeStringArray(route.transit),
                  dining_plan: normalizeStringArray(route.dining_plan),
                };
              })
            : [],
          map_layers: Array.isArray(guideRecord.map_layers)
            ? guideRecord.map_layers.map((item) => {
                const layer = asRecord(item);
                return {
                  id: normalizeTextValue(layer.id),
                  label: normalizeTextValue(layer.label),
                  description: normalizeTextValue(layer.description),
                };
              })
            : [],
          content: normalizeTextValue(guideRecord.content),
          revision: toNonNegativeNumber(guideRecord.revision),
          updated_at: normalizeTextValue(guideRecord.updated_at),
        }
      : null,
  };
}

function createProductWorkspaceMessage(
  role: "user" | "assistant",
  text: string,
  options: Partial<Pick<ProductWorkspaceMessage, "status" | "error" | "at">> = {},
): ProductWorkspaceMessage {
  return {
    id: `product-workspace-${Math.random().toString(16).slice(2)}`,
    role,
    text: normalizeTextValue(text),
    status: normalizeTextValue(options.status) || "done",
    error: Boolean(options.error),
    at: typeof options.at === "number" ? options.at : Date.now(),
  };
}

function createProductDraftRequest(mode = "bootstrap"): DraftRequest {
  return {
    name: "",
    goal: "",
    target_users: "",
    core_capabilities: "",
    constraints: "",
    expected_artifacts: "",
    integration_requirements: "",
    mode: normalizeTextValue(mode) || "bootstrap",
  };
}

function serializeProductWorkerAgents(items: ProductWorker[]) {
  if (!items.length) {
    return "";
  }
  return items
    .map((item) => [item.agent_id, item.role, item.responsibility].filter(Boolean).join(" | "))
    .filter(Boolean)
    .join("\n");
}

function parseProductWorkerAgentInput(value: string): ProductWorker[] {
  return value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const parts = line.split("|").map((item) => item.trim()).filter(Boolean);
      return {
        agent_id: normalizeTextValue(parts[0]),
        role: normalizeTextValue(parts[1]),
        responsibility: normalizeTextValue(parts.slice(2).join(" | ")),
        capabilities: [],
        enabled: true,
      };
    })
    .filter((item) => item.agent_id);
}

function parseCommaSeparatedList(value: string) {
  return Array.from(new Set(value.split(",").map((item) => item.trim()).filter(Boolean)));
}

function normalizeStringArray(value: unknown): string[] {
  return Array.isArray(value)
    ? value.map((item) => normalizeTextValue(item)).filter(Boolean)
    : [];
}

function normalizeTextValue(value: unknown) {
  return String(value || "").trim();
}

function toNonNegativeNumber(value: unknown) {
  const numeric = Number(value);
  return Number.isFinite(numeric) ? Math.max(0, numeric) : 0;
}

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : {};
}
