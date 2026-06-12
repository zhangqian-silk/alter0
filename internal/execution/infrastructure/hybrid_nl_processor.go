package infrastructure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	execdomain "alter0/internal/execution/domain"
	llmdomain "alter0/internal/llm/domain"
	shareddomain "alter0/internal/shared/domain"
	taskapp "alter0/internal/task/application"
)

type reactRunnerFactory interface {
	GetReActRunner(ctx context.Context, providerID string, config llmdomain.ReActRunnerConfig) (*llmdomain.ReActRunner, error)
}

type HybridNLProcessor struct {
	codex           *CodexCLIProcessor
	react           reactRunnerFactory
	serviceDeployer workspaceServiceDeployer
	logger          *slog.Logger
}

type deliveryCompletionContext struct {
	SessionID        string
	RepoRoot         string
	SessionWorkspace string
	SessionAttrs     map[string]string
}

type deliveryCompletionFailure struct {
	Check   execdomain.CompletionCheck
	Message string
}

func NewHybridNLProcessor(
	codex *CodexCLIProcessor,
	react reactRunnerFactory,
	logger *slog.Logger,
) *HybridNLProcessor {
	if codex == nil {
		codex = NewCodexCLIProcessor()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &HybridNLProcessor{
		codex:           codex,
		react:           react,
		serviceDeployer: newWorkspaceServiceDeployer(),
		logger:          logger,
	}
}

func (p *HybridNLProcessor) Process(ctx context.Context, content string, metadata map[string]string) (string, error) {
	hasImages := len(execdomain.DecodeUserImageAttachments(metadata)) > 0
	engine := p.resolveEngine(metadata)
	if engine == execdomain.ExecutionEngineReact {
		output, err := p.processWithReact(ctx, content, metadata, nil)
		if err == nil {
			setExecutionSource(metadata, execdomain.ExecutionSourceModel)
			return output, nil
		}
		if hasImages {
			return "", err
		}
		p.logReactFallback(metadata, err)
	}
	output, err := p.codex.Process(ctx, content, metadata)
	if err == nil {
		output, err = p.finalizeValidatedOutput(ctx, content, output, metadata)
		setExecutionSource(metadata, execdomain.ExecutionSourceCodexCLI)
	}
	return output, err
}

func (p *HybridNLProcessor) ProcessStream(
	ctx context.Context,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	hasImages := len(execdomain.DecodeUserImageAttachments(metadata)) > 0
	engine := p.resolveEngine(metadata)
	if engine == execdomain.ExecutionEngineReact {
		output, err := p.processWithReact(ctx, content, metadata, emit)
		if err == nil {
			setExecutionSource(metadata, execdomain.ExecutionSourceModel)
			return output, nil
		}
		if hasImages {
			return "", err
		}
		p.logReactFallback(metadata, err)
	}
	output, err := p.codex.ProcessStream(ctx, content, metadata, emit)
	if err == nil {
		output, err = p.finalizeValidatedOutput(ctx, content, output, metadata)
		setExecutionSource(metadata, execdomain.ExecutionSourceCodexCLI)
	}
	return output, err
}

func (p *HybridNLProcessor) finalizeValidatedOutput(
	ctx context.Context,
	content string,
	output string,
	metadata map[string]string,
) (string, error) {
	finalized, err := p.finalizeDeliveryOutput(ctx, content, output, metadata)
	if err != nil {
		return "", err
	}
	failures, validateErr := validateDeliveryCompletion(metadata)
	if validateErr == nil {
		return finalized, nil
	}
	repairedOutput, repaired, repairErr := p.repairDeliveryCompletion(ctx, content, finalized, metadata, failures)
	if repairErr != nil {
		return "", repairErr
	}
	if !repaired {
		return "", validateErr
	}
	finalized, err = p.finalizeDeliveryOutput(ctx, content, repairedOutput, metadata)
	if err != nil {
		return "", err
	}
	if _, validateErr = validateDeliveryCompletion(metadata); validateErr != nil {
		return "", validateErr
	}
	return finalized, nil
}

func (p *HybridNLProcessor) resolveEngine(metadata map[string]string) string {
	override := strings.ToLower(strings.TrimSpace(metadataValue(metadata, execdomain.ExecutionEngineMetadataKey)))
	switch override {
	case execdomain.ExecutionEngineCodex:
		return execdomain.ExecutionEngineCodex
	case execdomain.ExecutionEngineReact:
		return execdomain.ExecutionEngineReact
	}
	if strings.EqualFold(strings.TrimSpace(metadataValue(metadata, taskapp.MetadataExecutionMode)), taskapp.ExecutionModeAsync) {
		return execdomain.ExecutionEngineCodex
	}
	return execdomain.ExecutionEngineReact
}

func (p *HybridNLProcessor) processWithReact(
	ctx context.Context,
	content string,
	metadata map[string]string,
	emit func(event execdomain.StreamEvent) error,
) (string, error) {
	if p.react == nil {
		return "", errors.New("react processor factory unavailable")
	}
	providerID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMProviderIDMetadataKey))
	modelID := strings.TrimSpace(metadataValue(metadata, execdomain.LLMModelMetadataKey))
	assistant, err := p.react.GetReActRunner(ctx, providerID, llmdomain.ReActRunnerConfig{
		Model:             modelID,
		SystemPrompt:      buildHybridReActSystemPrompt(metadata),
		MaxIterations:     1,
		UserMessagePuller: shareddomain.ConsumeLiveUserMessage,
	})
	if err != nil {
		return "", err
	}
	userMessage, err := buildUserInputMessage(content, metadata)
	if err != nil {
		return "", err
	}
	if emit == nil {
		return assistant.RunMessage(ctx, userMessage)
	}
	return assistant.RunMessageStream(ctx, userMessage, func(event llmdomain.ReActEvent) error {
		if event.Type != "answer" || strings.TrimSpace(event.Delta) == "" {
			return nil
		}
		return emit(execdomain.StreamEvent{
			Type: execdomain.StreamEventTypeOutput,
			Text: event.Delta,
		})
	})
}

func buildUserInputMessage(content string, metadata map[string]string) (llmdomain.Message, error) {
	parts := []llmdomain.MessagePart{{
		Type: llmdomain.MessagePartTypeText,
		Text: strings.TrimSpace(content),
	}}
	for _, attachment := range execdomain.DecodeUserImageAttachments(metadata) {
		imageURL, err := resolveUserAttachmentImageURL(attachment)
		if err != nil {
			return llmdomain.Message{}, err
		}
		if strings.TrimSpace(imageURL) == "" {
			continue
		}
		parts = append(parts, llmdomain.MessagePart{
			Type:     llmdomain.MessagePartTypeImage,
			ImageURL: imageURL,
			Name:     attachment.Name,
		})
	}
	return llmdomain.Message{
		Role:    "user",
		Content: strings.TrimSpace(content),
		Parts:   parts,
	}, nil
}

func resolveUserAttachmentImageURL(attachment execdomain.UserImageAttachment) (string, error) {
	if dataURL := strings.TrimSpace(attachment.DataURL); dataURL != "" {
		return dataURL, nil
	}
	path := strings.TrimSpace(attachment.WorkspacePath)
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read workspace attachment: %w", err)
	}
	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = "image/*"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (p *HybridNLProcessor) logReactFallback(metadata map[string]string, err error) {
	if p.logger == nil || err == nil {
		return
	}
	p.logger.Warn("react processor unavailable, falling back to codex",
		slog.String("session_id", strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeSessionIDMetadataKey))),
		slog.String("message_id", strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeMessageIDMetadataKey))),
		slog.String("error", strings.TrimSpace(err.Error())),
	)
}

func (p *HybridNLProcessor) repairDeliveryCompletion(
	ctx context.Context,
	content string,
	output string,
	metadata map[string]string,
	failures []deliveryCompletionFailure,
) (string, bool, error) {
	if p == nil || p.codex == nil || len(failures) == 0 {
		return output, false, nil
	}
	repairPrompt, ok, err := buildDeliveryCompletionRepairPrompt(content, output, metadata, failures)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return output, false, nil
	}
	repairOutput, err := p.codex.Process(ctx, repairPrompt, metadata)
	if err != nil {
		return "", false, fmt.Errorf("delivery completion repair failed after validation error: %w", err)
	}
	if strings.TrimSpace(output) != "" {
		return output, true, nil
	}
	return repairOutput, true, nil
}

func buildDeliveryCompletionRepairPrompt(
	content string,
	output string,
	metadata map[string]string,
	failures []deliveryCompletionFailure,
) (string, bool, error) {
	if len(failures) == 0 {
		return "", false, nil
	}
	completionContext, err := resolveDeliveryCompletionContext(metadata)
	if err != nil {
		return "", false, err
	}
	parts := []string{
		"Repair the missing required deliverables for the current skill run now.",
		"Only work on the current session workspace and the missing delivery artifacts. Do not edit unrelated repository files.",
	}
	if strings.TrimSpace(completionContext.SessionWorkspace) != "" {
		parts = append(parts, "Session workspace root: "+completionContext.SessionWorkspace)
	}
	repairLines := []string{}
	for _, failure := range failures {
		instruction := renderCompletionCheckTemplate(failure.Check.RepairInstruction, completionContext, failure.Check)
		if strings.TrimSpace(instruction) == "" {
			continue
		}
		repairLines = append(repairLines, "- "+strings.TrimSpace(failure.Check.Label)+": "+instruction)
	}
	if len(repairLines) == 0 {
		return "", false, nil
	}
	blockerLines := make([]string, 0, len(failures))
	for _, failure := range failures {
		blockerLines = append(blockerLines, "- "+failure.Message)
	}
	parts = append(parts, "Current validation blockers:\n"+strings.Join(blockerLines, "\n"))
	parts = append(parts, "Required repair actions:\n"+strings.Join(repairLines, "\n"))
	if trimmedContent := strings.TrimSpace(content); trimmedContent != "" {
		parts = append(parts, "Original user request:\n"+trimmedContent)
	}
	if trimmedOutput := strings.TrimSpace(output); trimmedOutput != "" {
		parts = append(parts, "Current conversational guide draft:\n"+trimmedOutput)
	}
	return strings.Join(parts, "\n\n"), true, nil
}

func buildHybridReActSystemPrompt(metadata map[string]string) string {
	parts := []string{
		"You are alter0's execution assistant.",
		"Follow the ReAct pattern internally, but do not expose Thought, Action, or Observation headings.",
		"Return only the final user-facing answer.",
	}
	if rawSkillContext := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey)); rawSkillContext != "" {
		parts = append(parts, "Skill context (JSON): "+rawSkillContext)
	}
	if rawMCPContext := strings.TrimSpace(metadataValue(metadata, execdomain.MCPContextMetadataKey)); rawMCPContext != "" {
		parts = append(parts, "MCP context (JSON): "+rawMCPContext)
	}
	if rawMemoryContext := strings.TrimSpace(metadataValue(metadata, execdomain.MemoryContextMetadataKey)); rawMemoryContext != "" {
		parts = append(parts, renderMemoryContextInstruction(rawMemoryContext))
	}
	return strings.Join(parts, "\n\n")
}

func validateDeliveryCompletion(metadata map[string]string) ([]deliveryCompletionFailure, error) {
	failures, err := collectDeliveryCompletionFailures(metadata)
	if err != nil {
		return nil, err
	}
	if len(failures) == 0 {
		return nil, nil
	}
	if len(failures) == 1 {
		return failures, errors.New(failures[0].Message)
	}
	lines := []string{"delivery deliverables are incomplete:"}
	for _, failure := range failures {
		lines = append(lines, "- "+failure.Message)
	}
	return failures, errors.New(strings.Join(lines, "\n"))
}

func collectDeliveryCompletionFailures(metadata map[string]string) ([]deliveryCompletionFailure, error) {
	checks, err := parseDeliveryCompletionChecks(metadata)
	if err != nil || len(checks) == 0 {
		return nil, err
	}
	completionContext, err := resolveDeliveryCompletionContext(metadata)
	if err != nil {
		return nil, err
	}
	failures := make([]deliveryCompletionFailure, 0, len(checks))
	for _, check := range checks {
		if !check.Required {
			continue
		}
		failed, message, err := evaluateCompletionCheck(check, completionContext)
		if err != nil {
			return nil, err
		}
		if failed {
			failures = append(failures, deliveryCompletionFailure{Check: check, Message: message})
		}
	}
	if len(failures) == 0 {
		return nil, nil
	}
	return failures, nil
}

func resolveDeliveryCompletionContext(metadata map[string]string) (deliveryCompletionContext, error) {
	sessionID := strings.TrimSpace(metadataValue(metadata, execdomain.RuntimeSessionIDMetadataKey))
	if sessionID == "" {
		return deliveryCompletionContext{}, errors.New("delivery completion check failed: runtime session id is missing")
	}
	repoRoot, err := resolveToolRepoRoot()
	if err != nil {
		return deliveryCompletionContext{}, err
	}
	sessionWorkspace := filepath.FromSlash(buildSessionWorkspacePath(repoRoot, sessionID))
	if strings.TrimSpace(sessionWorkspace) == "" {
		return deliveryCompletionContext{}, errors.New("delivery completion check failed: session workspace is unavailable")
	}
	return deliveryCompletionContext{
		SessionID:        sessionID,
		RepoRoot:         repoRoot,
		SessionWorkspace: sessionWorkspace,
		SessionAttrs:     parseDeliveryAttributes(metadata),
	}, nil
}

func evaluateCompletionCheck(check execdomain.CompletionCheck, completionContext deliveryCompletionContext) (bool, string, error) {
	switch check.Type {
	case execdomain.CompletionCheckTypeSessionFileExists:
		target := filepath.Join(completionContext.SessionWorkspace, filepath.FromSlash(strings.Trim(strings.TrimSpace(check.SessionPath), "/")))
		if _, err := os.Stat(target); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return true, renderCompletionCheckFailureMessage(check, completionContext, filepath.ToSlash(target), ""), nil
			}
			return false, "", err
		}
		return false, "", nil
	case execdomain.CompletionCheckTypeWorkspaceServicePublished:
		entry, ok, err := resolvePublishedWorkspaceService(completionContext.RepoRoot, completionContext.SessionID, check.ServiceID)
		if err != nil {
			return false, "", err
		}
		if !ok {
			return true, renderCompletionCheckFailureMessage(check, completionContext, "", ""), nil
		}
		if check.RequirePublicReadOnly && !entry.PublicReadOnly {
			return true, renderCompletionCheckFailureMessage(check, completionContext, "", strings.TrimSpace(entry.URL)), nil
		}
		if check.RequireServiceURL && strings.TrimSpace(entry.URL) == "" {
			return true, renderCompletionCheckFailureMessage(check, completionContext, "", ""), nil
		}
		return false, "", nil
	case execdomain.CompletionCheckTypeSessionAttributeNonEmpty:
		value := strings.TrimSpace(completionContext.SessionAttrs[strings.TrimSpace(check.SessionAttributeKey)])
		if value == "" {
			return true, renderCompletionCheckFailureMessage(check, completionContext, "", ""), nil
		}
		return false, "", nil
	default:
		return false, "", nil
	}
}

func parseDeliveryCompletionChecks(metadata map[string]string) ([]execdomain.CompletionCheck, error) {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.CompletionChecksMetadataKey))
	if raw == "" {
		return nil, nil
	}
	var checks []execdomain.CompletionCheck
	if err := json.Unmarshal([]byte(raw), &checks); err != nil {
		return nil, fmt.Errorf("invalid delivery completion checks metadata: %w", err)
	}
	return checks, nil
}

func parseDeliveryAttributes(metadata map[string]string) map[string]string {
	attributes := map[string]string{}
	if len(metadata) == 0 {
		return attributes
	}
	if raw := strings.TrimSpace(metadataValue(metadata, execdomain.DeliveryAttributesMetadataKey)); raw != "" {
		var payload map[string]string
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			for key, value := range payload {
				normalizedKey := strings.TrimSpace(key)
				normalizedValue := strings.TrimSpace(value)
				if normalizedKey == "" || normalizedValue == "" {
					continue
				}
				attributes[normalizedKey] = normalizedValue
			}
		}
	}
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if !strings.HasPrefix(trimmedKey, execdomain.DeliveryAttributeMetadataPrefix) {
			continue
		}
		attrKey := strings.TrimPrefix(trimmedKey, execdomain.DeliveryAttributeMetadataPrefix)
		attrValue := strings.TrimSpace(value)
		if strings.TrimSpace(attrKey) == "" || attrValue == "" {
			continue
		}
		attributes[attrKey] = attrValue
	}
	return attributes
}

func renderCompletionCheckFailureMessage(
	check execdomain.CompletionCheck,
	completionContext deliveryCompletionContext,
	sessionFile string,
	serviceURL string,
) string {
	if custom := renderCompletionCheckTemplate(check.FailureMessage, completionContext, check, sessionFile, serviceURL); strings.TrimSpace(custom) != "" {
		return custom
	}
	switch check.Type {
	case execdomain.CompletionCheckTypeSessionFileExists:
		return "required session file is missing: " + sessionFile
	case execdomain.CompletionCheckTypeWorkspaceServicePublished:
		return "required workspace service is not published: " + strings.TrimSpace(check.ServiceID)
	case execdomain.CompletionCheckTypeSessionAttributeNonEmpty:
		return "required session attribute is missing: " + strings.TrimSpace(check.SessionAttributeKey)
	default:
		return "required deliverable is incomplete: " + strings.TrimSpace(check.Label)
	}
}

func renderCompletionCheckTemplate(
	template string,
	completionContext deliveryCompletionContext,
	check execdomain.CompletionCheck,
	extraValues ...string,
) string {
	trimmed := strings.TrimSpace(template)
	if trimmed == "" {
		return ""
	}
	sessionFile := ""
	serviceURL := ""
	if len(extraValues) > 0 {
		sessionFile = strings.TrimSpace(extraValues[0])
	}
	if len(extraValues) > 1 {
		serviceURL = strings.TrimSpace(extraValues[1])
	}
	replacer := strings.NewReplacer(
		"{{session_id}}", completionContext.SessionID,
		"{{session_workspace}}", filepath.ToSlash(completionContext.SessionWorkspace),
		"{{session_file}}", sessionFile,
		"{{service_id}}", strings.TrimSpace(check.ServiceID),
		"{{service_url}}", serviceURL,
		"{{session_attribute_key}}", strings.TrimSpace(check.SessionAttributeKey),
	)
	return strings.TrimSpace(replacer.Replace(trimmed))
}

type publishedWorkspaceService struct {
	URL            string
	PublicReadOnly bool
}

func resolvePublishedWorkspaceService(repoRoot string, sessionID string, serviceID string) (publishedWorkspaceService, bool, error) {
	registryPath := filepath.Join(repoRoot, ".alter0", "workspace-services.json")
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return publishedWorkspaceService{}, false, nil
		}
		return publishedWorkspaceService{}, false, err
	}
	var payload struct {
		Items []struct {
			SessionID      string `json:"session_id"`
			ServiceID      string `json:"service_id"`
			ServiceType    string `json:"service_type"`
			URL            string `json:"url"`
			PublicReadOnly bool   `json:"public_read_only"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return publishedWorkspaceService{}, false, err
	}
	for _, item := range payload.Items {
		if !strings.EqualFold(strings.TrimSpace(item.SessionID), sessionID) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(item.ServiceID), serviceID) {
			continue
		}
		return publishedWorkspaceService{
			URL:            strings.TrimSpace(item.URL),
			PublicReadOnly: item.PublicReadOnly,
		}, true, nil
	}
	return publishedWorkspaceService{}, false, nil
}

func resolvePublishedWorkspaceServiceURL(repoRoot string, sessionID string, serviceID string) (string, bool, error) {
	service, ok, err := resolvePublishedWorkspaceService(repoRoot, sessionID, serviceID)
	if err != nil || !ok {
		return "", ok, err
	}
	return strings.TrimSpace(service.URL), true, nil
}

func isTravelSkillRun(metadata map[string]string) bool {
	return hasSelectedSkill(metadata, "travel")
}

func hasSelectedSkill(metadata map[string]string, skillID string) bool {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.SkillContextMetadataKey))
	skillID = strings.TrimSpace(skillID)
	if raw == "" || skillID == "" {
		return false
	}
	var context execdomain.SkillContext
	if err := json.Unmarshal([]byte(raw), &context); err != nil {
		return false
	}
	for _, skill := range context.Skills {
		if strings.EqualFold(strings.TrimSpace(skill.ID), skillID) {
			return true
		}
	}
	return false
}

func renderDeliverablesInstruction(metadata map[string]string) string {
	raw := strings.TrimSpace(metadataValue(metadata, execdomain.DeliverablesMetadataKey))
	if raw == "" {
		return ""
	}
	var deliverables []execdomain.Deliverable
	if err := json.Unmarshal([]byte(raw), &deliverables); err != nil || len(deliverables) == 0 {
		return ""
	}
	lines := []string{
		"Current delivery contract:",
		"Do not finish with only a conversational answer when explicit deliverables are declared. Drive execution until the required deliverables are produced or you can clearly explain the blocker.",
	}
	for _, item := range deliverables {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}
		parts := make([]string, 0, 4)
		if item.Required {
			parts = append(parts, "required")
		} else {
			parts = append(parts, "optional")
		}
		if format := strings.TrimSpace(item.Format); format != "" {
			parts = append(parts, format)
		}
		if field := strings.TrimSpace(item.SessionAttributeKey); field != "" {
			parts = append(parts, "session attribute "+field)
		}
		line := "- " + label
		if description := strings.TrimSpace(item.Description); description != "" {
			line += ": " + description
		}
		if len(parts) > 0 {
			line += " (" + strings.Join(parts, ", ") + ")"
		}
		lines = append(lines, line)
	}
	if len(lines) == 2 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func renderMemoryContextInstruction(raw string) string {
	context := execdomain.MemoryContext{}
	if err := json.Unmarshal([]byte(raw), &context); err != nil || len(context.Files) == 0 {
		return "Resolved memory context (JSON): " + raw
	}

	var builder strings.Builder
	builder.WriteString("Resolved memory files:\n")
	for _, file := range context.Files {
		builder.WriteString("- ")
		builder.WriteString(strings.TrimSpace(file.Title))
		builder.WriteString("\n  path: ")
		builder.WriteString(strings.TrimSpace(file.Path))
		builder.WriteString("\n  exists: ")
		if file.Exists {
			builder.WriteString("true")
		} else {
			builder.WriteString("false")
		}
		if updatedAt := strings.TrimSpace(file.UpdatedAt); updatedAt != "" {
			builder.WriteString("\n  updated_at: ")
			builder.WriteString(updatedAt)
		}
		if content := strings.TrimSpace(file.Content); content != "" {
			builder.WriteString("\n  content:\n")
			builder.WriteString(content)
		}
		builder.WriteString("\n")
	}
	if len(context.Recall) > 0 {
		builder.WriteString("\nAuto-recalled memory snippets:\n")
		for _, hit := range context.Recall {
			builder.WriteString("- ")
			if title := strings.TrimSpace(hit.Title); title != "" {
				builder.WriteString(title)
			} else {
				builder.WriteString(strings.TrimSpace(hit.MemoryID))
			}
			if hit.Line > 0 {
				builder.WriteString(fmt.Sprintf(":%d", hit.Line))
			}
			if path := strings.TrimSpace(hit.Path); path != "" {
				builder.WriteString("\n  path: ")
				builder.WriteString(path)
			}
			if snippet := strings.TrimSpace(hit.Snippet); snippet != "" {
				builder.WriteString("\n  snippet:\n")
				builder.WriteString(snippet)
			}
			builder.WriteString("\n")
		}
	}
	return strings.TrimSpace(builder.String())
}

func setExecutionSource(metadata map[string]string, source string) {
	if len(metadata) == 0 {
		return
	}
	metadata[execdomain.ExecutionSourceMetadataKey] = strings.TrimSpace(source)
}

var _ interface {
	Process(ctx context.Context, content string, metadata map[string]string) (string, error)
	ProcessStream(ctx context.Context, content string, metadata map[string]string, emit func(event execdomain.StreamEvent) error) (string, error)
} = (*HybridNLProcessor)(nil)
