package localfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func readIfExists(path string) ([]byte, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

func writeFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func marshalPayload(format Format, title string, payload any) ([]byte, error) {
	jsonRaw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}

	switch format {
	case FormatJSON:
		return append(jsonRaw, '\n'), nil
	case FormatMarkdown:
		builder := &strings.Builder{}
		builder.WriteString("# ")
		builder.WriteString(title)
		builder.WriteString("\n\n")
		builder.WriteString("GeneratedAt: ")
		builder.WriteString(time.Now().UTC().Format(time.RFC3339))
		builder.WriteString("\n\n```json\n")
		builder.Write(jsonRaw)
		builder.WriteString("\n```\n")
		return []byte(builder.String()), nil
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

func unmarshalPayload(format Format, raw []byte, out any) error {
	switch format {
	case FormatJSON:
		return json.Unmarshal(raw, out)
	case FormatMarkdown:
		payload, err := extractMarkdownJSON(string(raw))
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(payload), out)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func extractMarkdownJSON(content string) (string, error) {
	start := strings.Index(content, "```json")
	if start < 0 {
		return "", errors.New("markdown storage missing json code block")
	}
	body := content[start+len("```json"):]
	if strings.HasPrefix(body, "\r\n") {
		body = body[2:]
	} else if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}

	end := strings.Index(body, "```")
	if end < 0 {
		return "", errors.New("markdown storage has unclosed json code block")
	}
	return strings.TrimSpace(body[:end]), nil
}
