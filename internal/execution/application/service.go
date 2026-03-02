package application

import (
	"context"
	"strings"

	execdomain "alter0/internal/execution/domain"
	shareddomain "alter0/internal/shared/domain"
)

type Service struct {
	processor execdomain.NLProcessor
}

func NewService(processor execdomain.NLProcessor) *Service {
	return &Service{processor: processor}
}

func (s *Service) ExecuteNaturalLanguage(ctx context.Context, msg shareddomain.UnifiedMessage) (string, error) {
	content := strings.TrimSpace(msg.Content)
	return s.processor.Process(ctx, content, msg.Metadata)
}
