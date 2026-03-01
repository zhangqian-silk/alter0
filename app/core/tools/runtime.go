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
	ErrorCodeUnsupportedTool = "TOOL_UNSUPPORTED"
	ErrorCodePolicyDenied    = "TOOL_POLICY_DENIED"
	ErrorCodeConfirmRequired = "TOOL_CONFIRM_REQUIRED"
	ErrorCodeExecutionFailed = "TOOL_EXECUTION_FAILED"
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
}

type AgentPolicy struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

type Runtime struct {
	policy   *PolicyGate
	auditor  *Auditor
	specs    []Spec
	specByID map[string]Spec
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
	}
}

func (r *Runtime) Evaluate(req Request) Decision {
	if r == nil || r.policy == nil {
		return Decision{Allowed: false, Code: ErrorCodePolicyDenied, Reason: "tool runtime not configured"}
	}
	return r.policy.Evaluate(req)
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
		},
		"policy": r.policy.Snapshot(),
	}
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
	agent          map[string]agentRule
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
		agent:          map[string]agentRule{},
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
		"agent":           agentPolicy,
	}
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

var supportedTools = []Spec{
	{Name: "web_search", Description: "Search the web and return ranked results.", RiskLevel: "low"},
	{Name: "web_fetch", Description: "Fetch and extract readable content from a URL.", RiskLevel: "low"},
	{Name: "browser", Description: "Automate browser interactions and snapshots.", RiskLevel: "high"},
	{Name: "canvas", Description: "Operate node canvas UI and run JavaScript snippets.", RiskLevel: "high"},
	{Name: "nodes", Description: "Control paired devices (camera, screen, location, run).", RiskLevel: "high"},
	{Name: "message", Description: "Send outbound messages and channel actions.", RiskLevel: "high"},
	{Name: "tts", Description: "Convert text to speech output.", RiskLevel: "medium"},
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
