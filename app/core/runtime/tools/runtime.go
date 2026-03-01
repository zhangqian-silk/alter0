package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ResultStatusSuccess   = "success"
	ResultStatusFailed    = "failed"
	ResultStatusRetryable = "retryable"
	ResultStatusBlocked   = "blocked"
)

const (
	ErrorCodeUnsupportedTool   = "TOOL_UNSUPPORTED"
	ErrorCodePolicyDenied      = "TOOL_POLICY_DENIED"
	ErrorCodeConfirmRequired   = "TOOL_CONFIRM_REQUIRED"
	ErrorCodeInvalidArgs       = "TOOL_INVALID_ARGS"
	ErrorCodeExecutionFailed   = "TOOL_EXECUTION_FAILED"
	ErrorCodeContextRestricted = "TOOL_CONTEXT_RESTRICTED"
)

type Spec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RiskLevel   string `json:"risk_level"`
}

type Request struct {
	Tool      string                 `json:"tool"`
	Args      map[string]interface{} `json:"args,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	ChannelID string                 `json:"channel_id,omitempty"`
	Surface   string                 `json:"surface,omitempty"`
	Confirmed bool                   `json:"confirmed,omitempty"`
}

type Result struct {
	Tool       string      `json:"tool"`
	Status     string      `json:"status"`
	Code       string      `json:"code,omitempty"`
	Message    string      `json:"message,omitempty"`
	Retryable  bool        `json:"retryable,omitempty"`
	DurationMS int64       `json:"duration_ms"`
	Data       interface{} `json:"data,omitempty"`
}

type ExecutionFunc func(context.Context, Request) (interface{}, error)

type ToolchainAdapterMap map[string]ExecutionFunc

type ToolError struct {
	Code      string
	Message   string
	Retryable bool
}

func (e *ToolError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Code) != "" {
		return e.Code
	}
	return "tool execution error"
}

type Decision struct {
	Allowed              bool   `json:"allowed"`
	Code                 string `json:"code,omitempty"`
	Reason               string `json:"reason,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
}

type Config struct {
	GlobalAllow    []string               `json:"global_allow,omitempty"`
	GlobalDeny     []string               `json:"global_deny,omitempty"`
	RequireConfirm []string               `json:"require_confirm,omitempty"`
	Agent          map[string]AgentPolicy `json:"agent,omitempty"`
	Memory         MemoryPolicy           `json:"memory,omitempty"`
}

type AgentPolicy struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

type MemoryPolicy struct {
	TrustedChannels []string `json:"trusted_channels,omitempty"`
	RestrictedPaths []string `json:"restricted_paths,omitempty"`
}

type Runtime struct {
	policy   *PolicyGate
	auditor  *Auditor
	specs    []Spec
	specByID map[string]Spec

	mu       sync.RWMutex
	adapters ToolchainAdapterMap
}

func NewRuntime(cfg Config, auditBasePath string) *Runtime {
	specs := SupportedSpecs()
	specByID := make(map[string]Spec, len(specs))
	for _, spec := range specs {
		specByID[spec.Name] = spec
	}
	return &Runtime{
		policy:   NewPolicyGate(cfg),
		auditor:  NewAuditor(auditBasePath),
		specs:    specs,
		specByID: specByID,
		adapters: ToolchainAdapterMap{},
	}
}

func (r *Runtime) Evaluate(req Request) Decision {
	if r == nil || r.policy == nil {
		return Decision{Allowed: false, Code: ErrorCodePolicyDenied, Reason: "tool runtime not configured"}
	}
	return r.policy.Evaluate(req)
}

func (r *Runtime) RegisterAdapter(toolName string, adapter ExecutionFunc) {
	if r == nil {
		return
	}
	name := NormalizeToolName(toolName)
	if name == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.adapters == nil {
		r.adapters = ToolchainAdapterMap{}
	}
	if adapter == nil {
		delete(r.adapters, name)
		return
	}
	r.adapters[name] = adapter
}

func (r *Runtime) RegisterAdapters(adapters ToolchainAdapterMap) {
	if r == nil {
		return
	}
	for toolName, adapter := range adapters {
		r.RegisterAdapter(toolName, adapter)
	}
}

func (r *Runtime) adapterFor(toolName string) ExecutionFunc {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.adapters[NormalizeToolName(toolName)]
}

func (r *Runtime) Invoke(ctx context.Context, req Request, execute ExecutionFunc) Result {
	started := time.Now()
	toolName := NormalizeToolName(req.Tool)
	decision := r.Evaluate(req)
	if !decision.Allowed {
		result := Result{
			Tool:       toolName,
			Status:     ResultStatusBlocked,
			Code:       decision.Code,
			Message:    decision.Reason,
			Retryable:  decision.Code == ErrorCodeConfirmRequired,
			DurationMS: time.Since(started).Milliseconds(),
		}
		r.appendAudit(started, req, decision, result)
		return result
	}

	if execute == nil {
		execute = r.adapterFor(toolName)
	}
	if execute == nil {
		result := Result{
			Tool:       toolName,
			Status:     ResultStatusFailed,
			Code:       ErrorCodeExecutionFailed,
			Message:    "tool execution callback is nil",
			DurationMS: time.Since(started).Milliseconds(),
		}
		r.appendAudit(started, req, decision, result)
		return result
	}

	if err := validateStructuredToolArgs(toolName, req.Args); err != nil {
		result := fromExecutionError(toolName, err, time.Since(started).Milliseconds())
		r.appendAudit(started, req, decision, result)
		return result
	}

	payload, err := execute(ctx, req)
	if err != nil {
		result := fromExecutionError(toolName, err, time.Since(started).Milliseconds())
		r.appendAudit(started, req, decision, result)
		return result
	}

	result := Result{
		Tool:       toolName,
		Status:     ResultStatusSuccess,
		DurationMS: time.Since(started).Milliseconds(),
		Data:       payload,
	}
	r.appendAudit(started, req, decision, result)
	return result
}

func (r *Runtime) StatusSnapshot() map[string]interface{} {
	if r == nil {
		return map[string]interface{}{}
	}
	specs := make([]Spec, 0, len(r.specs))
	specs = append(specs, r.specs...)
	return map[string]interface{}{
		"protocol": map[string]interface{}{
			"version": 1,
			"tools":   specs,
			"toolchain": map[string]interface{}{
				"browser":     browserActionSchema,
				"canvas":      canvasActionSchema,
				"nodes":       nodesActionSchema,
				"live_wiring": r.liveWiringSnapshot(),
			},
		},
		"policy":           r.policy.Snapshot(),
		"security_posture": r.policy.PostureSnapshot(),
	}
}

func (r *Runtime) liveWiringSnapshot() map[string]bool {
	status := map[string]bool{
		"browser": r.adapterFor("browser") != nil,
		"canvas":  r.adapterFor("canvas") != nil,
		"nodes":   r.adapterFor("nodes") != nil,
	}
	return status
}

func validateStructuredToolArgs(toolName string, args map[string]interface{}) error {
	schema, ok := actionSchemas[toolName]
	if !ok {
		return nil
	}
	action := strings.ToLower(strings.TrimSpace(getStringArg(args, "action")))
	if action == "" {
		return &ToolError{Code: ErrorCodeInvalidArgs, Message: fmt.Sprintf("%s requires args.action", toolName)}
	}
	if _, exists := schema[action]; !exists {
		return &ToolError{Code: ErrorCodeInvalidArgs, Message: fmt.Sprintf("unsupported %s action: %s", toolName, action)}
	}
	if toolName == "browser" && action == "act" {
		request, _ := args["request"].(map[string]interface{})
		kind := strings.ToLower(strings.TrimSpace(getStringArg(request, "kind")))
		if kind == "" {
			return &ToolError{Code: ErrorCodeInvalidArgs, Message: "browser action act requires args.request.kind"}
		}
		if _, ok := browserActKinds[kind]; !ok {
			return &ToolError{Code: ErrorCodeInvalidArgs, Message: fmt.Sprintf("unsupported browser act kind: %s", kind)}
		}
	}
	if toolName == "nodes" {
		switch action {
		case "run":
			if len(getStringSliceArg(args, "command")) == 0 {
				return &ToolError{Code: ErrorCodeInvalidArgs, Message: "nodes action run requires args.command"}
			}
		case "invoke":
			if strings.TrimSpace(getStringArg(args, "invokeCommand")) == "" {
				return &ToolError{Code: ErrorCodeInvalidArgs, Message: "nodes action invoke requires args.invokeCommand"}
			}
		}
	}
	return nil
}

func getStringArg(payload map[string]interface{}, key string) string {
	if len(payload) == 0 {
		return ""
	}
	raw, _ := payload[key]
	if text, ok := raw.(string); ok {
		return text
	}
	return ""
}

func getStringSliceArg(payload map[string]interface{}, key string) []string {
	if len(payload) == 0 {
		return nil
	}
	raw, exists := payload[key]
	if !exists || raw == nil {
		return nil
	}
	if items, ok := raw.([]string); ok {
		clean := make([]string, 0, len(items))
		for _, item := range items {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			clean = append(clean, trimmed)
		}
		return clean
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	clean := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	return clean
}

func fromExecutionError(toolName string, err error, durationMS int64) Result {
	if err == nil {
		return Result{Tool: toolName, Status: ResultStatusSuccess, DurationMS: durationMS}
	}
	if toolErr, ok := err.(*ToolError); ok {
		status := ResultStatusFailed
		if toolErr.Retryable {
			status = ResultStatusRetryable
		}
		code := strings.TrimSpace(toolErr.Code)
		if code == "" {
			code = ErrorCodeExecutionFailed
		}
		return Result{
			Tool:       toolName,
			Status:     status,
			Code:       code,
			Message:    strings.TrimSpace(toolErr.Message),
			Retryable:  toolErr.Retryable,
			DurationMS: durationMS,
		}
	}
	return Result{
		Tool:       toolName,
		Status:     ResultStatusFailed,
		Code:       ErrorCodeExecutionFailed,
		Message:    strings.TrimSpace(err.Error()),
		DurationMS: durationMS,
	}
}

func (r *Runtime) appendAudit(ts time.Time, req Request, decision Decision, result Result) {
	if r == nil || r.auditor == nil {
		return
	}
	req.Tool = NormalizeToolName(req.Tool)
	entry := AuditEntry{
		Timestamp:            ts.UTC().Format(time.RFC3339Nano),
		Tool:                 req.Tool,
		AgentID:              fallbackText(req.AgentID, "default"),
		SessionID:            fallbackText(req.SessionID, "n/a"),
		RequestID:            fallbackText(req.RequestID, "n/a"),
		UserID:               fallbackText(req.UserID, "anonymous"),
		ChannelID:            fallbackText(req.ChannelID, "unknown"),
		Surface:              fallbackText(req.Surface, "default"),
		Decision:             boolToDecision(decision.Allowed),
		DecisionCode:         strings.TrimSpace(decision.Code),
		DecisionReason:       strings.TrimSpace(decision.Reason),
		Status:               strings.TrimSpace(result.Status),
		ErrorCode:            strings.TrimSpace(result.Code),
		Retryable:            result.Retryable,
		RequiresConfirmation: decision.RequiresConfirmation,
		Confirmed:            req.Confirmed,
		DurationMS:           result.DurationMS,
		ArgsSummary:          summarizeMap(req.Args),
		ResultSummary:        summarizeResultData(result.Data, result.Message),
	}
	_ = r.auditor.Append(ts, entry)
}

func boolToDecision(allowed bool) string {
	if allowed {
		return "allow"
	}
	return "deny"
}

func fallbackText(raw string, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func summarizeMap(payload map[string]interface{}) map[string]string {
	if len(payload) == 0 {
		return map[string]string{}
	}
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	summary := make(map[string]string, len(keys))
	for _, key := range keys {
		summary[key] = summarizeValue(payload[key])
	}
	return summary
}

func summarizeValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return "string(0)"
		}
		runes := []rune(text)
		if len(runes) > 60 {
			return fmt.Sprintf("string(%d):%s...", len(runes), string(runes[:60]))
		}
		return fmt.Sprintf("string(%d):%s", len(runes), text)
	case bool:
		if v {
			return "bool:true"
		}
		return "bool:false"
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("int:%v", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("uint:%v", v)
	case float32, float64:
		return fmt.Sprintf("float:%v", v)
	case []string:
		return fmt.Sprintf("[]string(len=%d)", len(v))
	case []interface{}:
		return fmt.Sprintf("[]any(len=%d)", len(v))
	case map[string]interface{}:
		return fmt.Sprintf("map(len=%d)", len(v))
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func summarizeResultData(data interface{}, message string) string {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage != "" {
		return summarizeValue(trimmedMessage)
	}
	if data == nil {
		return ""
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return summarizeValue(data)
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) > 120 {
		return string(runes[:120]) + "..."
	}
	return text
}

type PolicyGate struct {
	globalAllow    map[string]struct{}
	globalDeny     map[string]struct{}
	requireConfirm map[string]struct{}
	highRisk       map[string]struct{}
	agent          map[string]agentRule
	memory         memoryRule
}

type memoryRule struct {
	trustedChannels map[string]struct{}
	restrictedPaths []string
}

type agentRule struct {
	allow map[string]struct{}
	deny  map[string]struct{}
}

func NewPolicyGate(cfg Config) *PolicyGate {
	gate := &PolicyGate{
		globalAllow:    toSet(cfg.GlobalAllow),
		globalDeny:     toSet(cfg.GlobalDeny),
		requireConfirm: toSet(cfg.RequireConfirm),
		highRisk:       highRiskToolSet(),
		agent:          map[string]agentRule{},
		memory:         newMemoryRule(cfg.Memory),
	}
	for agentID, policy := range cfg.Agent {
		id := strings.TrimSpace(agentID)
		if id == "" {
			continue
		}
		gate.agent[id] = agentRule{allow: toSet(policy.Allow), deny: toSet(policy.Deny)}
	}
	return gate
}

func (p *PolicyGate) Evaluate(req Request) Decision {
	toolName := NormalizeToolName(req.Tool)
	if toolName == "" {
		return Decision{Allowed: false, Code: ErrorCodeUnsupportedTool, Reason: "tool is required"}
	}
	if _, ok := supportedToolMap[toolName]; !ok {
		return Decision{Allowed: false, Code: ErrorCodeUnsupportedTool, Reason: "unsupported tool: " + toolName}
	}

	agentID := strings.TrimSpace(req.AgentID)
	rule, hasAgentRule := p.agent[agentID]

	if inSet(p.globalDeny, toolName) || (hasAgentRule && inSet(rule.deny, toolName)) {
		return Decision{Allowed: false, Code: ErrorCodePolicyDenied, Reason: "tool denied by policy"}
	}
	if hasAgentRule && len(rule.allow) > 0 && !inSet(rule.allow, toolName) {
		return Decision{Allowed: false, Code: ErrorCodePolicyDenied, Reason: "tool not in agent allowlist"}
	}
	if len(p.globalAllow) > 0 && !inSet(p.globalAllow, toolName) {
		return Decision{Allowed: false, Code: ErrorCodePolicyDenied, Reason: "tool not in global allowlist"}
	}
	if decision := p.evaluateMemoryIsolation(req, toolName); !decision.Allowed {
		return decision
	}
	if inSet(p.highRisk, toolName) && !req.Confirmed {
		return Decision{
			Allowed:              false,
			Code:                 ErrorCodeConfirmRequired,
			Reason:               "high-risk tool requires confirmation",
			RequiresConfirmation: true,
		}
	}
	if inSet(p.requireConfirm, toolName) && !req.Confirmed {
		return Decision{
			Allowed:              false,
			Code:                 ErrorCodeConfirmRequired,
			Reason:               "tool requires confirmation",
			RequiresConfirmation: true,
		}
	}
	return Decision{Allowed: true}
}

func (p *PolicyGate) Snapshot() map[string]interface{} {
	if p == nil {
		return map[string]interface{}{}
	}
	agentPolicy := map[string]map[string][]string{}
	agentIDs := make([]string, 0, len(p.agent))
	for id := range p.agent {
		agentIDs = append(agentIDs, id)
	}
	sort.Strings(agentIDs)
	for _, id := range agentIDs {
		rule := p.agent[id]
		agentPolicy[id] = map[string][]string{
			"allow": setToSlice(rule.allow),
			"deny":  setToSlice(rule.deny),
		}
	}
	return map[string]interface{}{
		"global_allow":    setToSlice(p.globalAllow),
		"global_deny":     setToSlice(p.globalDeny),
		"require_confirm": setToSlice(p.requireConfirm),
		"high_risk":       setToSlice(p.highRisk),
		"agent":           agentPolicy,
		"memory": map[string]interface{}{
			"trusted_channels": setToSlice(p.memory.trustedChannels),
			"restricted_paths": append([]string(nil), p.memory.restrictedPaths...),
		},
	}
}

func (p *PolicyGate) PostureSnapshot() map[string]interface{} {
	if p == nil {
		return map[string]interface{}{}
	}

	issues := make([]map[string]interface{}, 0, 4)
	if len(p.globalAllow) == 0 {
		issues = append(issues, map[string]interface{}{
			"id":             "global_allow_not_set",
			"severity":       "medium",
			"message":        "global tool allowlist is not configured",
			"recommendation": "set security.tools.global_allow to reduce exposed tool surface",
		})
	}
	conflicts := intersectSets(p.globalAllow, p.globalDeny)
	if len(conflicts) > 0 {
		issues = append(issues, map[string]interface{}{
			"id":             "allow_deny_conflict",
			"severity":       "high",
			"message":        "some tools exist in both global_allow and global_deny",
			"conflict_tools": conflicts,
			"recommendation": "remove conflicting tools from one side to avoid policy ambiguity",
		})
	}
	if len(p.memory.trustedChannels) == 0 {
		issues = append(issues, map[string]interface{}{
			"id":             "memory_trusted_channels_empty",
			"severity":       "high",
			"message":        "memory trusted channels is empty",
			"recommendation": "set security.memory.trusted_channels to at least cli/http",
		})
	}
	if len(p.memory.restrictedPaths) == 0 {
		issues = append(issues, map[string]interface{}{
			"id":             "memory_restricted_paths_empty",
			"severity":       "high",
			"message":        "memory restricted paths is empty",
			"recommendation": "set security.memory.restricted_paths to include MEMORY.md",
		})
	}

	return map[string]interface{}{
		"ok":                     len(issues) == 0,
		"issues":                 issues,
		"baseline_high_risk":     setToSlice(p.highRisk),
		"config_require_confirm": setToSlice(p.requireConfirm),
		"guardrails": map[string]interface{}{
			"high_risk_confirmation": true,
			"memory_isolation":       true,
		},
	}
}

func highRiskToolSet() map[string]struct{} {
	set := map[string]struct{}{}
	for _, spec := range supportedTools {
		if strings.EqualFold(strings.TrimSpace(spec.RiskLevel), "high") {
			set[spec.Name] = struct{}{}
		}
	}
	return set
}

func intersectSets(a map[string]struct{}, b map[string]struct{}) []string {
	if len(a) == 0 || len(b) == 0 {
		return []string{}
	}
	items := make([]string, 0)
	for key := range a {
		if _, ok := b[key]; ok {
			items = append(items, key)
		}
	}
	sort.Strings(items)
	return items
}

func toSet(items []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, item := range items {
		name := NormalizeToolName(item)
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}

func setToSlice(set map[string]struct{}) []string {
	if len(set) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(set))
	for item := range set {
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

func inSet(set map[string]struct{}, name string) bool {
	_, ok := set[name]
	return ok
}

func newMemoryRule(cfg MemoryPolicy) memoryRule {
	trustedChannels := toSet(cfg.TrustedChannels)
	if len(trustedChannels) == 0 {
		trustedChannels = toSet([]string{"cli", "http"})
	}
	restrictedPaths := normalizeRestrictedPaths(cfg.RestrictedPaths)
	if len(restrictedPaths) == 0 {
		restrictedPaths = []string{"memory.md"}
	}
	return memoryRule{trustedChannels: trustedChannels, restrictedPaths: restrictedPaths}
}

func normalizeRestrictedPaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{}
	}
	set := map[string]struct{}{}
	items := make([]string, 0, len(paths))
	for _, raw := range paths {
		normalized := normalizeMemoryPath(raw)
		if normalized == "" {
			continue
		}
		if _, exists := set[normalized]; exists {
			continue
		}
		set[normalized] = struct{}{}
		items = append(items, normalized)
	}
	sort.Strings(items)
	return items
}

func normalizeMemoryPath(raw string) string {
	cleaned := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(raw, "\\", "/")))
	cleaned = strings.TrimPrefix(cleaned, "./")
	for strings.Contains(cleaned, "//") {
		cleaned = strings.ReplaceAll(cleaned, "//", "/")
	}
	return cleaned
}

func (p *PolicyGate) evaluateMemoryIsolation(req Request, toolName string) Decision {
	if toolName != "memory_get" && toolName != "memory_search" {
		return Decision{Allowed: true}
	}
	if p.isMainSurface(req) {
		return Decision{Allowed: true}
	}
	if toolName == "memory_search" {
		includeLongTerm, _ := req.Args["include_long_term"].(bool)
		if includeLongTerm {
			return Decision{Allowed: false, Code: ErrorCodeContextRestricted, Reason: "shared surface cannot access long-term memory"}
		}
		return Decision{Allowed: true}
	}
	path, _ := req.Args["path"].(string)
	normalizedPath := normalizeMemoryPath(path)
	if normalizedPath == "" {
		return Decision{Allowed: false, Code: ErrorCodeContextRestricted, Reason: "memory_get path is required on shared surface"}
	}
	for _, restricted := range p.memory.restrictedPaths {
		if normalizedPath == restricted || strings.HasPrefix(normalizedPath, restricted+"/") {
			return Decision{Allowed: false, Code: ErrorCodeContextRestricted, Reason: "memory path restricted on shared surface"}
		}
	}
	return Decision{Allowed: true}
}

func (p *PolicyGate) isMainSurface(req Request) bool {
	surface := strings.ToLower(strings.TrimSpace(req.Surface))
	switch surface {
	case "main", "private", "direct":
		return true
	case "shared", "group", "public":
		return false
	}
	channel := NormalizeToolName(req.ChannelID)
	if channel == "" {
		return false
	}
	return inSet(p.memory.trustedChannels, channel)
}

var browserActionSchema = map[string]interface{}{
	"required_args": []string{"action"},
	"actions":       sortedSchemaActions(browserActions),
	"act_kinds":     sortedSchemaActions(browserActKinds),
}

var canvasActionSchema = map[string]interface{}{
	"required_args": []string{"action"},
	"actions":       sortedSchemaActions(canvasActions),
}

var nodesActionSchema = map[string]interface{}{
	"required_args": []string{"action"},
	"actions":       sortedSchemaActions(nodesActions),
	"required_by_action": map[string][]string{
		"invoke": {"invokeCommand"},
		"run":    {"command"},
	},
}

var actionSchemas = map[string]map[string]struct{}{
	"browser": browserActions,
	"canvas":  canvasActions,
	"nodes":   nodesActions,
}

var browserActions = makeActionSet([]string{"status", "start", "stop", "profiles", "tabs", "open", "focus", "close", "snapshot", "screenshot", "navigate", "console", "pdf", "upload", "dialog", "act"})

var browserActKinds = makeActionSet([]string{"click", "type", "press", "hover", "drag", "select", "fill", "resize", "wait", "evaluate", "close"})

var canvasActions = makeActionSet([]string{"present", "hide", "navigate", "eval", "snapshot", "a2ui_push", "a2ui_reset"})

var nodesActions = makeActionSet([]string{"status", "describe", "pending", "approve", "reject", "notify", "camera_snap", "camera_list", "camera_clip", "screen_record", "location_get", "run", "invoke"})

func makeActionSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		name := strings.ToLower(strings.TrimSpace(item))
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}

func sortedSchemaActions(set map[string]struct{}) []string {
	items := make([]string, 0, len(set))
	for item := range set {
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

var supportedTools = []Spec{
	{Name: "web_search", Description: "Search the web and return ranked results.", RiskLevel: "low"},
	{Name: "web_fetch", Description: "Fetch and extract readable content from a URL.", RiskLevel: "low"},
	{Name: "browser", Description: "Automate browser interactions and snapshots.", RiskLevel: "high"},
	{Name: "canvas", Description: "Operate node canvas UI and run JavaScript snippets.", RiskLevel: "high"},
	{Name: "nodes", Description: "Control paired devices (camera, screen, location, run).", RiskLevel: "high"},
	{Name: "message", Description: "Send outbound messages and channel actions.", RiskLevel: "high"},
	{Name: "tts", Description: "Convert text to speech output.", RiskLevel: "medium"},
	{Name: "memory_search", Description: "Search MEMORY.md and memory/*.md for relevant snippets.", RiskLevel: "medium"},
	{Name: "memory_get", Description: "Read specific line ranges from MEMORY.md or memory/*.md.", RiskLevel: "medium"},
}

var supportedToolMap = buildSupportedToolMap()

func buildSupportedToolMap() map[string]Spec {
	out := make(map[string]Spec, len(supportedTools))
	for _, spec := range supportedTools {
		out[spec.Name] = spec
	}
	return out
}

func SupportedSpecs() []Spec {
	items := make([]Spec, 0, len(supportedTools))
	items = append(items, supportedTools...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func NormalizeToolName(name string) string {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	return trimmed
}

type AuditEntry struct {
	Timestamp            string            `json:"timestamp"`
	Tool                 string            `json:"tool"`
	AgentID              string            `json:"agent_id"`
	SessionID            string            `json:"session_id"`
	RequestID            string            `json:"request_id"`
	UserID               string            `json:"user_id"`
	ChannelID            string            `json:"channel_id"`
	Surface              string            `json:"surface"`
	Decision             string            `json:"decision"`
	DecisionCode         string            `json:"decision_code,omitempty"`
	DecisionReason       string            `json:"decision_reason,omitempty"`
	Status               string            `json:"status"`
	ErrorCode            string            `json:"error_code,omitempty"`
	Retryable            bool              `json:"retryable,omitempty"`
	RequiresConfirmation bool              `json:"requires_confirmation,omitempty"`
	Confirmed            bool              `json:"confirmed,omitempty"`
	DurationMS           int64             `json:"duration_ms"`
	ArgsSummary          map[string]string `json:"args_summary,omitempty"`
	ResultSummary        string            `json:"result_summary,omitempty"`
}

type Auditor struct {
	basePath string
	mu       sync.Mutex
}

func NewAuditor(basePath string) *Auditor {
	trimmed := strings.TrimSpace(basePath)
	if trimmed == "" {
		trimmed = filepath.Join("output", "audit")
	}
	return &Auditor{basePath: trimmed}
}

func (a *Auditor) Append(ts time.Time, entry AuditEntry) error {
	if a == nil {
		return nil
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	dayDir := filepath.Join(a.basePath, ts.UTC().Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dayDir, "tool_execution.jsonl")
	a.mu.Lock()
	defer a.mu.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(payload, '\n'))
	return err
}
