package domain

type IntentType string

const (
	IntentTypeCommand IntentType = "command"
	IntentTypeNL      IntentType = "nl"
)

type Intent struct {
	Type        IntentType
	CommandName string
	Args        []string
}

type IntentClassifier interface {
	Classify(content string) Intent
}
