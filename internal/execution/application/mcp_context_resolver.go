package application

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	controldomain "alter0/internal/control/domain"
	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

const (
	defaultMCPTimeoutMS = 15000
	maxMCPTimeoutMS     = 120000

	mcpTransportMetadataKey = "mcp.transport"
	mcpCommandMetadataKey   = "mcp.command"
	mcpArgsMetadataKey      = "mcp.args"
	mcpEnvMetadataKey       = "mcp.env"
	mcpURLMetadataKey       = "mcp.url"
	mcpHeadersMetadataKey   = "mcp.headers"

	mcpToolWhitelistMetadataKey    = "mcp.tools.whitelist"
	mcpTimeoutMetadataKey          = "mcp.timeout_ms"
	mcpFailureIsolationMetadataKey = "mcp.failure_isolation"

	mcpSessionEnableKey = "alter0.mcp.session.enable"
	mcpRequestEnableKey = "alter0.mcp.request.enable"
	mcpExcludeKey       = "alter0.mcp.exclude"

	mcpTransportStdio = "stdio"
	mcpTransportHTTP  = "http"
)

type mcpResolution struct {
	Context     execdomain.MCPContext
	InjectedIDs []string
}

type mcpContextResolver struct {
	source            SkillCapabilitySource
	mu                sync.RWMutex
	sessionSelections map[string]map[string]struct{}
}

func newMCPContextResolver(source SkillCapabilitySource) *mcpContextResolver {
	if source == nil {
		return nil
	}
	return &mcpContextResolver{
		source:            source,
		sessionSelections: map[string]map[string]struct{}{},
	}
}

func (r *mcpContextResolver) Resolve(msg shareddomain.UnifiedMessage) mcpResolution {
	resolution := mcpResolution{
		Context: execdomain.MCPContext{
			Protocol: execdomain.MCPContextProtocolVersion,
		},
	}
	if r == nil || r.source == nil {
		return resolution
	}

	sessionEnabled := r.resolveSessionSelection(msg)
	requestEnabled := parseLookupSet(metadataValue(msg.Metadata, mcpRequestEnableKey))
	excluded := parseLookupSet(metadataValue(msg.Metadata, mcpExcludeKey))

	items := r.source.ListCapabilitiesByType(controldomain.CapabilityTypeMCP)
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].ID) < strings.ToLower(items[j].ID)
	})

	now := time.Now().UTC()
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		normalized := item.Normalized()
		if matchCapabilityFilter(normalized.ID, normalized.Name, excluded) {
			resolution.Context.Audit = append(resolution.Context.Audit, newMCPAuditItem(msg, normalized, "blocked", "excluded_by_request", now))
			continue
		}
		if !scopeEnabled(normalized, sessionEnabled, requestEnabled) {
			reason := "scope_not_enabled"
			switch normalized.Scope {
			case controldomain.CapabilityScopeSession:
				reason = "session_scope_not_enabled"
			case controldomain.CapabilityScopeRequest:
				reason = "request_scope_not_enabled"
			}
			resolution.Context.Audit = append(resolution.Context.Audit, newMCPAuditItem(msg, normalized, "blocked", reason, now))
			continue
		}

		server, err := buildMCPServer(normalized)
		if err != nil {
			resolution.Context.Audit = append(resolution.Context.Audit, newMCPAuditItem(msg, normalized, "blocked", err.Error(), now))
			continue
		}

		resolution.Context.Servers = append(resolution.Context.Servers, server)
		resolution.Context.Audit = append(resolution.Context.Audit, newMCPAuditItem(msg, normalized, "enabled", "", now))
		resolution.InjectedIDs = append(resolution.InjectedIDs, server.ID)
	}
	return resolution
}

func (r *mcpContextResolver) resolveSessionSelection(msg shareddomain.UnifiedMessage) map[string]struct{} {
	sessionID := strings.TrimSpace(msg.SessionID)
	if sessionID == "" {
		return nil
	}
	raw, hasSessionUpdate := msg.Metadata[mcpSessionEnableKey]
	if hasSessionUpdate {
		selection := parseLookupSet(raw)
		r.mu.Lock()
		if len(selection) == 0 {
			delete(r.sessionSelections, sessionID)
		} else {
			r.sessionSelections[sessionID] = selection
		}
		r.mu.Unlock()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	selection := r.sessionSelections[sessionID]
	if len(selection) == 0 {
		return nil
	}
	copied := make(map[string]struct{}, len(selection))
	for key := range selection {
		copied[key] = struct{}{}
	}
	return copied
}

func scopeEnabled(
	capability controldomain.Capability,
	sessionEnabled map[string]struct{},
	requestEnabled map[string]struct{},
) bool {
	switch capability.Scope {
	case controldomain.CapabilityScopeGlobal:
		return true
	case controldomain.CapabilityScopeSession:
		return matchCapabilityFilter(capability.ID, capability.Name, sessionEnabled)
	case controldomain.CapabilityScopeRequest:
		return matchCapabilityFilter(capability.ID, capability.Name, requestEnabled)
	default:
		return false
	}
}

func matchCapabilityFilter(id, name string, filter map[string]struct{}) bool {
	if len(filter) == 0 {
		return false
	}
	if _, ok := filter[strings.ToLower(strings.TrimSpace(id))]; ok {
		return true
	}
	if _, ok := filter[strings.ToLower(strings.TrimSpace(name))]; ok {
		return true
	}
	return false
}

func buildMCPServer(capability controldomain.Capability) (execdomain.MCPServer, error) {
	server := execdomain.MCPServer{
		ID:               capability.ID,
		Name:             capability.Name,
		Scope:            string(capability.Scope),
		ToolWhitelist:    parseList(metadataValue(capability.Metadata, mcpToolWhitelistMetadataKey)),
		TimeoutMS:        defaultMCPTimeoutMS,
		FailureIsolation: true,
	}

	if timeoutMS, err := parseMCPTimeoutMS(capability.Metadata); err != nil {
		return execdomain.MCPServer{}, err
	} else if timeoutMS > 0 {
		server.TimeoutMS = timeoutMS
	}

	if failureIsolation, ok := parseOptionalBool(capability.Metadata, mcpFailureIsolationMetadataKey); ok {
		server.FailureIsolation = failureIsolation
	}

	transport := strings.ToLower(strings.TrimSpace(metadataValue(capability.Metadata, mcpTransportMetadataKey)))
	switch transport {
	case mcpTransportStdio:
		server.Transport = mcpTransportStdio
		command := strings.TrimSpace(metadataValue(capability.Metadata, mcpCommandMetadataKey))
		if command == "" {
			return execdomain.MCPServer{}, errorsForCapability(capability.ID, "missing mcp.command for stdio transport")
		}
		server.Command = command
		server.Args = parseArgumentList(metadataValue(capability.Metadata, mcpArgsMetadataKey))
		server.Env = parseStringMap(metadataValue(capability.Metadata, mcpEnvMetadataKey))
	case mcpTransportHTTP:
		server.Transport = mcpTransportHTTP
		url := strings.TrimSpace(metadataValue(capability.Metadata, mcpURLMetadataKey))
		if url == "" {
			return execdomain.MCPServer{}, errorsForCapability(capability.ID, "missing mcp.url for http transport")
		}
		server.URL = url
		server.Headers = parseStringMap(metadataValue(capability.Metadata, mcpHeadersMetadataKey))
	default:
		return execdomain.MCPServer{}, errorsForCapability(capability.ID, "mcp.transport must be stdio or http")
	}
	return server, nil
}

func parseMCPTimeoutMS(metadata map[string]string) (int, error) {
	raw := strings.TrimSpace(metadataValue(metadata, mcpTimeoutMetadataKey))
	if raw == "" {
		return defaultMCPTimeoutMS, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errorsForCapability("", "mcp.timeout_ms must be integer")
	}
	if value <= 0 {
		return 0, errorsForCapability("", "mcp.timeout_ms must be positive")
	}
	if value > maxMCPTimeoutMS {
		return maxMCPTimeoutMS, nil
	}
	return value, nil
}

func parseArgumentList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var items []string
		if err := json.Unmarshal([]byte(trimmed), &items); err == nil {
			return normalizeList(items)
		}
	}
	return normalizeList(strings.Split(trimmed, ","))
}

func parseStringMap(raw string) map[string]string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	normalized := map[string]string{}
	for key, value := range out {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		normalized[trimmedKey] = trimmedValue
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func parseOptionalBool(metadata map[string]string, key string) (bool, bool) {
	raw := strings.TrimSpace(metadataValue(metadata, key))
	if raw == "" {
		return false, false
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return value, true
}

func errorsForCapability(id, message string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid mcp configuration: %s", message)
	}
	return fmt.Errorf("invalid mcp configuration for %s: %s", id, message)
}

func newMCPAuditItem(
	msg shareddomain.UnifiedMessage,
	capability controldomain.Capability,
	decision string,
	reason string,
	occurredAt time.Time,
) execdomain.MCPAuditItem {
	return execdomain.MCPAuditItem{
		ServerID:   capability.ID,
		ServerName: capability.Name,
		Scope:      string(capability.Scope),
		Decision:   decision,
		Reason:     reason,
		TraceID:    msg.TraceID,
		SessionID:  msg.SessionID,
		MessageID:  msg.MessageID,
		OccurredAt: occurredAt,
	}
}
