package application

import (
	"errors"
	"strings"
)

func extractMarkdownJSONPayload(content string) (string, error) {
	start := strings.Index(content, "```json")
	if start < 0 {
		return "", errors.New("markdown memory payload missing json code block")
	}
	body := content[start+len("```json"):]
	if strings.HasPrefix(body, "\r\n") {
		body = body[2:]
	} else if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}
	end := strings.Index(body, "```")
	if end < 0 {
		return "", errors.New("markdown memory payload has unclosed json code block")
	}
	return strings.TrimSpace(body[:end]), nil
}
