package localfile

import (
	"fmt"
	"strings"
)

type Format string

const (
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

func ParseFormat(raw string) (Format, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "", string(FormatJSON):
		return FormatJSON, nil
	case "md", "markdown":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("unsupported persistence format %q", raw)
	}
}

func extension(format Format) string {
	switch format {
	case FormatMarkdown:
		return "md"
	case FormatJSON:
		return "json"
	default:
		return "data"
	}
}
