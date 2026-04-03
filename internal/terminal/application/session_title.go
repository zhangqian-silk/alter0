package application

import (
	"strings"
	"unicode/utf8"
)

const autoSessionTitleStableScore = 5

var sessionTitlePolitePrefixes = []string{
	"请帮我",
	"麻烦帮我",
	"麻烦先帮我",
	"先帮我",
	"帮我先",
	"帮我",
	"麻烦先",
	"麻烦",
	"请先",
	"请",
	"please",
	"can you",
	"could you",
	"help me",
}

var sessionTitleBootstrapVerbs = []string{
	"拉取",
	"同步",
	"clone",
	"pull",
	"checkout",
	"fetch",
	"查看",
	"看下",
	"看看",
	"分析",
	"熟悉",
	"inspect",
	"analyze",
	"analyse",
	"read",
	"open",
	"explore",
}

var sessionTitleRepoWords = []string{
	"仓库",
	"代码库",
	"repo",
	"repository",
	"codebase",
	"branch",
}

var sessionTitleConnectors = []string{
	"然后",
	"之后",
	"后再",
	"再",
	"then",
	"and then",
	"after that",
}

var sessionTitleActionWords = []string{
	"修复",
	"修改",
	"实现",
	"新增",
	"添加",
	"删除",
	"优化",
	"重构",
	"调整",
	"排查",
	"补齐",
	"支持",
	"更新",
	"rename",
	"fix",
	"change",
	"update",
	"implement",
	"add",
	"remove",
	"optimize",
	"optimise",
	"refactor",
	"debug",
	"support",
}

var sessionTitleTechnicalWords = []string{
	"terminal",
	"agent",
	"chat",
	"session",
	"title",
	"name",
	"logic",
	"api",
	"route",
	"ui",
	"ux",
	"test",
	"readme",
	"docs",
	"会话",
	"标题",
	"命名",
	"逻辑",
	"接口",
	"路由",
	"页面",
	"前端",
	"后端",
	"文档",
	"测试",
	"工作区",
}

var sessionTitleFollowUps = []string{
	"继续",
	"继续处理",
	"接着",
	"然后呢",
	"next",
	"continue",
}

func buildAutoSessionTitle(input string, fallback string, maxLen int) (string, int, bool) {
	normalized := normalizeSessionTitleText(input)
	if normalized == "" {
		return strings.TrimSpace(fallback), 0, true
	}
	trimmed := trimSessionBootstrapPrefix(normalized)
	score := scoreSessionTitle(trimmed)
	if score <= 0 {
		if maxLen > 0 {
			return truncateSessionTitle(normalized, maxLen), 0, true
		}
		return normalized, 0, true
	}
	if maxLen > 0 {
		trimmed = truncateSessionTitle(trimmed, maxLen)
	}
	return trimmed, score, score < autoSessionTitleStableScore
}

func inferAutoSessionTitleState(title string, fallback string) (bool, int) {
	normalizedTitle := normalizeSessionTitleText(title)
	normalizedFallback := normalizeSessionTitleText(fallback)
	if normalizedTitle == "" || strings.EqualFold(normalizedTitle, normalizedFallback) {
		return true, 0
	}
	trimmed := trimSessionBootstrapPrefix(normalizedTitle)
	score := scoreSessionTitle(trimmed)
	if score < autoSessionTitleStableScore && (trimmed != normalizedTitle || isBootstrapClause(normalizedTitle) || isFollowUpTitle(normalizedTitle)) {
		return true, score
	}
	return false, score
}

func nextAutoSessionTitle(currentTitle string, currentAuto bool, currentScore int, input string, fallback string, maxLen int) (string, bool, int, bool) {
	if !currentAuto {
		return currentTitle, currentAuto, currentScore, false
	}
	title, score, nextAuto := buildAutoSessionTitle(input, fallback, maxLen)
	if strings.TrimSpace(title) == "" {
		return currentTitle, currentAuto, currentScore, false
	}
	if strings.EqualFold(normalizeSessionTitleText(currentTitle), normalizeSessionTitleText(fallback)) && title != currentTitle {
		return title, nextAuto, score, true
	}
	if score <= currentScore {
		return currentTitle, currentAuto, currentScore, false
	}
	return title, nextAuto, score, true
}

func normalizeSessionTitleText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func trimSessionBootstrapPrefix(value string) string {
	normalized := normalizeSessionTitleText(value)
	if normalized == "" {
		return ""
	}
	segments := splitSessionTitleSegments(normalized)
	if len(segments) == 0 {
		return normalized
	}
	first := stripSessionTitlePrefix(segments[0])
	if remainder := extractBootstrapRemainder(first); remainder != "" {
		rest := append([]string{remainder}, segments[1:]...)
		return normalizeSessionTitleText(strings.Join(rest, " "))
	}
	if isBootstrapClause(first) && len(segments) > 1 {
		return normalizeSessionTitleText(strings.Join(segments[1:], " "))
	}
	return normalized
}

func splitSessionTitleSegments(value string) []string {
	replacer := strings.NewReplacer(
		"，", "\n",
		",", "\n",
		"。", "\n",
		".", "\n",
		"；", "\n",
		";", "\n",
		"：", "\n",
		":", "\n",
		"\r", "\n",
	)
	parts := strings.Split(replacer.Replace(value), "\n")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := normalizeSessionTitleText(part)
		if trimmed != "" {
			segments = append(segments, trimmed)
		}
	}
	return segments
}

func stripSessionTitlePrefix(value string) string {
	trimmed := normalizeSessionTitleText(value)
	lower := strings.ToLower(trimmed)
	updated := true
	for updated {
		updated = false
		for _, prefix := range sessionTitlePolitePrefixes {
			if strings.HasPrefix(lower, prefix) {
				trimmed = normalizeSessionTitleText(trimmed[len(prefix):])
				lower = strings.ToLower(trimmed)
				updated = true
			}
		}
	}
	return strings.TrimLeft(trimmed, "- ")
}

func extractBootstrapRemainder(value string) string {
	trimmed := stripSessionTitlePrefix(value)
	if !isBootstrapClause(trimmed) {
		return ""
	}
	lower := strings.ToLower(trimmed)
	for _, connector := range sessionTitleConnectors {
		idx := strings.Index(lower, connector)
		if idx < 0 {
			continue
		}
		remainder := normalizeSessionTitleText(trimmed[idx+len(connector):])
		if remainder != "" {
			return stripSessionTitlePrefix(remainder)
		}
	}
	return ""
}

func isBootstrapClause(value string) bool {
	lower := strings.ToLower(stripSessionTitlePrefix(value))
	if lower == "" {
		return false
	}
	return containsAnySessionTitleWord(lower, sessionTitleBootstrapVerbs) && containsAnySessionTitleWord(lower, sessionTitleRepoWords)
}

func isFollowUpTitle(value string) bool {
	normalized := strings.ToLower(normalizeSessionTitleText(value))
	for _, item := range sessionTitleFollowUps {
		if normalized == item {
			return true
		}
	}
	return false
}

func scoreSessionTitle(value string) int {
	normalized := normalizeSessionTitleText(value)
	if normalized == "" {
		return 0
	}
	score := 1
	length := utf8.RuneCountInString(normalized)
	switch {
	case length >= 10:
		score += 2
	case length >= 6:
		score++
	}
	lower := strings.ToLower(normalized)
	if containsAnySessionTitleWord(lower, sessionTitleActionWords) {
		score += 2
	}
	if containsAnySessionTitleWord(lower, sessionTitleTechnicalWords) {
		score += 2
	}
	if strings.ContainsAny(normalized, "/_.#") {
		score++
	}
	if isBootstrapClause(normalized) {
		score -= 4
	}
	if isFollowUpTitle(normalized) {
		score -= 3
	}
	if length <= 4 {
		score -= 2
	}
	if score < 0 {
		return 0
	}
	return score
}

func containsAnySessionTitleWord(value string, words []string) bool {
	for _, item := range words {
		if strings.Contains(value, item) {
			return true
		}
	}
	return false
}

func truncateSessionTitle(value string, maxLen int) string {
	if maxLen <= 0 {
		return value
	}
	if utf8.RuneCountInString(value) <= maxLen {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxLen-3]) + "..."
}
