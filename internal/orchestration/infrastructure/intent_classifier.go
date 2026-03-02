package infrastructure

import (
	"strings"

	orchdomain "alter0/internal/orchestration/domain"
)

type CommandLookup interface {
	Exists(name string) bool
}

type SimpleIntentClassifier struct {
	lookup CommandLookup
}

func NewSimpleIntentClassifier(lookup CommandLookup) *SimpleIntentClassifier {
	return &SimpleIntentClassifier{lookup: lookup}
}

func (c *SimpleIntentClassifier) Classify(content string) orchdomain.Intent {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return orchdomain.Intent{Type: orchdomain.IntentTypeNL}
	}

	tokens := strings.Fields(trimmed)
	commandToken := tokens[0]
	requiresSlash := strings.HasPrefix(commandToken, "/")
	if requiresSlash {
		commandToken = strings.TrimPrefix(commandToken, "/")
	}
	commandToken = strings.ToLower(commandToken)

	if requiresSlash || c.lookup.Exists(commandToken) {
		args := []string{}
		if len(tokens) > 1 {
			args = tokens[1:]
		}
		return orchdomain.Intent{
			Type:        orchdomain.IntentTypeCommand,
			CommandName: commandToken,
			Args:        args,
		}
	}

	return orchdomain.Intent{Type: orchdomain.IntentTypeNL}
}
