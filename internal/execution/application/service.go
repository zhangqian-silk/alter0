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
	processor     execdomain.NLProcessor
	skillResolver *skillContextResolver
	logger        *slog.Logger
}

const (
	resultSkillInjectedIDsKey = "skills.injected_ids"
	resultSkillInjectedKey    = "skills.injected_count"
	resultSkillProtocolKey    = "skills.protocol"
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
		processor:     processor,
		skillResolver: newSkillContextResolver(skillSource),
		logger:        logger,
	}
}

func (s *Service) ExecuteNaturalLanguage(ctx context.Context, msg shareddomain.UnifiedMessage) (shareddomain.ExecutionResult, error) {
	content := strings.TrimSpace(msg.Content)
	metadata := cloneMetadata(msg.Metadata)
	resultMetadata := map[string]string{}
	if s.skillResolver != nil {
		skillContext, injectedIDs := s.skillResolver.Resolve(msg)
		if len(skillContext.Skills) > 0 {
			rawSkillContext, err := json.Marshal(skillContext)
			if err != nil {
				return shareddomain.ExecutionResult{}, err
			}
			metadata[execdomain.SkillContextMetadataKey] = string(rawSkillContext)
			resultMetadata[resultSkillProtocolKey] = skillContext.Protocol
			resultMetadata[resultSkillInjectedIDsKey] = strings.Join(injectedIDs, ",")
		}
		resultMetadata[resultSkillInjectedKey] = strconv.Itoa(len(skillContext.Skills))
		if s.logger != nil {
			s.logger.Info("skills injected",
				slog.String("session_id", msg.SessionID),
				slog.String("message_id", msg.MessageID),
				slog.Int("skills_injected", len(skillContext.Skills)),
			)
		}
	}

	output, err := s.processor.Process(ctx, content, metadata)
	if err != nil {
		return shareddomain.ExecutionResult{}, err
	}
	return shareddomain.ExecutionResult{
		Output:   output,
		Metadata: normalizeResultMetadata(resultMetadata),
	}, nil
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
