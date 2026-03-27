package application

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

type Service struct {
	processor      execdomain.NLProcessor
	skillResolver  *skillContextResolver
	mcpResolver    *mcpContextResolver
	memoryResolver *memoryContextResolver
	logger         *slog.Logger
}

type streamProcessor interface {
	ProcessStream(
		ctx context.Context,
		content string,
		metadata map[string]string,
		emit func(event execdomain.StreamEvent) error,
	) (string, error)
}

const (
	resultSkillInjectedIDsKey = "skills.injected_ids"
	resultSkillInjectedKey    = "skills.injected_count"
	resultSkillProtocolKey    = "skills.protocol"
	resultSkillConflictKey    = "skills.conflict_count"
	resultSkillConflictTypes  = "skills.conflict_types"
	resultSkillConflictDetail = "skills.conflicts"

	resultMCPInjectedIDsKey = "mcp.injected_ids"
	resultMCPInjectedKey    = "mcp.injected_count"
	resultMCPProtocolKey    = "mcp.protocol"
	resultMCPAuditCountKey  = "mcp.audit_count"
	resultMCPAuditDetailKey = "mcp.audit"

	resultMemoryInjectedIDsKey = "memory.injected_ids"
	resultMemoryInjectedKey    = "memory.injected_count"
	resultMemoryProtocolKey    = "memory.protocol"
)

func NewService(processor execdomain.NLProcessor) *Service {
	return &Service{processor: processor}
}

func NewServiceWithSkills(
	processor execdomain.NLProcessor,
	skillSource SkillCapabilitySource,
	logger *slog.Logger,
) *Service {
	return &Service{
		processor:      processor,
		skillResolver:  newSkillContextResolver(skillSource),
		mcpResolver:    newMCPContextResolver(skillSource),
		memoryResolver: newMemoryContextResolver(),
		logger:         logger,
	}
}

func (s *Service) ExecuteNaturalLanguage(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.ExecutionResult, error) {
	output, resultMetadata, err := s.executeNaturalLanguage(ctx, msg, nil)
	if err != nil {
		return shareddomain.ExecutionResult{}, err
	}
	return shareddomain.ExecutionResult{
		Output:   output,
		Metadata: normalizeResultMetadata(resultMetadata),
	}, nil
}

func (s *Service) ExecuteNaturalLanguageStream(
	ctx context.Context,
	msg shareddomain.UnifiedMessage,
	onDelta func(string) error,
) (shareddomain.ExecutionResult, error) {
	streamHandler := func(event execdomain.StreamEvent) error {
		if onDelta == nil {
			return nil
		}
		if event.Type != execdomain.StreamEventTypeOutput {
			return nil
		}
		if strings.TrimSpace(event.Text) == "" {
			return nil
		}
		return onDelta(event.Text)
	}

	output, resultMetadata, err := s.executeNaturalLanguage(ctx, msg, streamHandler)
	if err != nil {
		return shareddomain.ExecutionResult{}, err
	}
	return shareddomain.ExecutionResult{
		Output:   output,
		Metadata: normalizeResultMetadata(resultMetadata),
	}, nil
}

func (s *Service) executeNaturalLanguage(
	ctx context.Context,
	msg shareddomain.UnifiedMessage,
	streamHandler func(event execdomain.StreamEvent) error,
) (string, map[string]string, error) {
	content := strings.TrimSpace(msg.Content)
	metadata := cloneMetadata(msg.Metadata)
	attachRuntimeMetadata(msg, metadata)
	resultMetadata := map[string]string{}

	if err := s.resolveSkillContext(msg, metadata, resultMetadata); err != nil {
		return "", nil, err
	}
	if err := s.resolveMCPContext(msg, metadata, resultMetadata); err != nil {
		return "", nil, err
	}
	if err := s.resolveMemoryContext(msg, metadata, resultMetadata); err != nil {
		return "", nil, err
	}

	if streamHandler != nil {
		if processor, ok := s.processor.(streamProcessor); ok {
			output, err := processor.ProcessStream(ctx, content, metadata, streamHandler)
			if err != nil {
				return "", nil, err
			}
			attachExecutionSourceMetadata(metadata, resultMetadata)
			return output, resultMetadata, nil
		}
	}

	output, err := s.processor.Process(ctx, content, metadata)
	if err != nil {
		return "", nil, err
	}
	attachExecutionSourceMetadata(metadata, resultMetadata)
	if streamHandler != nil && strings.TrimSpace(output) != "" {
		if err := streamHandler(execdomain.StreamEvent{
			Type: execdomain.StreamEventTypeOutput,
			Text: output,
		}); err != nil {
			return "", nil, err
		}
	}
	return output, resultMetadata, nil
}

func attachExecutionSourceMetadata(metadata map[string]string, resultMetadata map[string]string) {
	source := strings.TrimSpace(metadata[execdomain.ExecutionSourceMetadataKey])
	if source == "" {
		return
	}
	resultMetadata[execdomain.ExecutionSourceMetadataKey] = source
}

func (s *Service) resolveSkillContext(
	msg shareddomain.UnifiedMessage,
	metadata map[string]string,
	resultMetadata map[string]string,
) error {
	if s.skillResolver == nil {
		return nil
	}
	resolution := s.skillResolver.Resolve(msg)
	skillContext := resolution.Context
	if len(skillContext.Skills) > 0 {
		rawSkillContext, err := json.Marshal(skillContext)
		if err != nil {
			return err
		}
		metadata[execdomain.SkillContextMetadataKey] = string(rawSkillContext)
		resultMetadata[resultSkillProtocolKey] = skillContext.Protocol
		resultMetadata[resultSkillInjectedIDsKey] = strings.Join(resolution.InjectedIDs, ",")
	}
	resultMetadata[resultSkillInjectedKey] = strconv.Itoa(len(skillContext.Skills))
	resultMetadata[resultSkillConflictKey] = strconv.Itoa(len(skillContext.Conflicts))
	if len(resolution.ConflictTypes) > 0 {
		resultMetadata[resultSkillConflictTypes] = strings.Join(resolution.ConflictTypes, ",")
	}
	if len(skillContext.Conflicts) > 0 {
		rawConflicts, err := json.Marshal(skillContext.Conflicts)
		if err != nil {
			return err
		}
		resultMetadata[resultSkillConflictDetail] = string(rawConflicts)
	}
	if s.logger != nil {
		logHandler := s.logger.Info
		if len(skillContext.Conflicts) > 0 {
			logHandler = s.logger.Warn
		}
		logHandler("skills injected",
			slog.String("session_id", msg.SessionID),
			slog.String("message_id", msg.MessageID),
			slog.Int("skills_injected", len(skillContext.Skills)),
			slog.Int("skills_conflicts", len(skillContext.Conflicts)),
			slog.String("skills_injected_ids", strings.Join(resolution.InjectedIDs, ",")),
			slog.String("skills_conflict_types", strings.Join(resolution.ConflictTypes, ",")),
		)
	}
	return nil
}

func (s *Service) resolveMCPContext(
	msg shareddomain.UnifiedMessage,
	metadata map[string]string,
	resultMetadata map[string]string,
) error {
	if s.mcpResolver == nil {
		return nil
	}
	resolution := s.mcpResolver.Resolve(msg)
	mcpContext := resolution.Context
	if len(mcpContext.Servers) > 0 {
		rawMCPContext, err := json.Marshal(mcpContext)
		if err != nil {
			return err
		}
		metadata[execdomain.MCPContextMetadataKey] = string(rawMCPContext)
		resultMetadata[resultMCPProtocolKey] = mcpContext.Protocol
		resultMetadata[resultMCPInjectedIDsKey] = strings.Join(resolution.InjectedIDs, ",")
	}
	resultMetadata[resultMCPInjectedKey] = strconv.Itoa(len(mcpContext.Servers))
	resultMetadata[resultMCPAuditCountKey] = strconv.Itoa(len(mcpContext.Audit))
	if len(mcpContext.Audit) > 0 {
		rawAudit, err := json.Marshal(mcpContext.Audit)
		if err != nil {
			return err
		}
		resultMetadata[resultMCPAuditDetailKey] = string(rawAudit)
	}
	if s.logger != nil {
		logHandler := s.logger.Info
		blocked := 0
		for _, item := range mcpContext.Audit {
			if item.Decision != "enabled" {
				blocked++
			}
		}
		if blocked > 0 {
			logHandler = s.logger.Warn
		}
		logHandler("mcp resolved",
			slog.String("session_id", msg.SessionID),
			slog.String("message_id", msg.MessageID),
			slog.Int("mcp_injected", len(mcpContext.Servers)),
			slog.Int("mcp_blocked", blocked),
			slog.String("mcp_injected_ids", strings.Join(resolution.InjectedIDs, ",")),
		)
	}
	return nil
}

func (s *Service) resolveMemoryContext(
	msg shareddomain.UnifiedMessage,
	metadata map[string]string,
	resultMetadata map[string]string,
) error {
	if s.memoryResolver == nil {
		return nil
	}
	resolution := s.memoryResolver.Resolve(msg)
	memoryContext := resolution.Context
	if len(memoryContext.Files) > 0 {
		rawMemoryContext, err := json.Marshal(memoryContext)
		if err != nil {
			return err
		}
		metadata[execdomain.MemoryContextMetadataKey] = string(rawMemoryContext)
		resultMetadata[resultMemoryProtocolKey] = memoryContext.Protocol
		resultMetadata[resultMemoryInjectedIDsKey] = strings.Join(resolution.InjectedIDs, ",")
	}
	resultMetadata[resultMemoryInjectedKey] = strconv.Itoa(len(memoryContext.Files))
	if s.logger != nil {
		s.logger.Info("memory context resolved",
			slog.String("session_id", msg.SessionID),
			slog.String("message_id", msg.MessageID),
			slog.Int("memory_files_injected", len(memoryContext.Files)),
			slog.String("memory_injected_ids", strings.Join(resolution.InjectedIDs, ",")),
		)
	}
	return nil
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func attachRuntimeMetadata(msg shareddomain.UnifiedMessage, metadata map[string]string) {
	if metadata == nil {
		return
	}
	if sessionID := strings.TrimSpace(msg.SessionID); sessionID != "" {
		metadata[execdomain.RuntimeSessionIDMetadataKey] = sessionID
	}
	if messageID := strings.TrimSpace(msg.MessageID); messageID != "" {
		metadata[execdomain.RuntimeMessageIDMetadataKey] = messageID
	}
	if traceID := strings.TrimSpace(msg.TraceID); traceID != "" {
		metadata[execdomain.RuntimeTraceIDMetadataKey] = traceID
	}
}

func normalizeResultMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range metadata {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
