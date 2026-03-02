package infrastructure

import (
	"context"
	"fmt"
)

type TemplateNLProcessor struct{}

func NewTemplateNLProcessor() *TemplateNLProcessor {
	return &TemplateNLProcessor{}
}

func (p *TemplateNLProcessor) Process(_ context.Context, content string, _ map[string]string) (string, error) {
	return fmt.Sprintf("nl_executor: %s", content), nil
}
