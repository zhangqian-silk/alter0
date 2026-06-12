package infrastructure

import "testing"

type fakeLookup struct {
	commands map[string]struct{}
}

func (f *fakeLookup) Exists(name string) bool {
	_, ok := f.commands[name]
	return ok
}

func TestClassifySlashCommand(t *testing.T) {
	classifier := NewSimpleIntentClassifier(&fakeLookup{
		commands: map[string]struct{}{
			"echo": {},
		},
	})

	intent := classifier.Classify("/echo hello world")
	if intent.Type != "command" {
		t.Fatalf("expected command intent, got %s", intent.Type)
	}
	if intent.CommandName != "echo" {
		t.Fatalf("expected command echo, got %s", intent.CommandName)
	}
	if len(intent.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(intent.Args))
	}
}

func TestClassifyRegisteredCommandWithoutSlash(t *testing.T) {
	classifier := NewSimpleIntentClassifier(&fakeLookup{
		commands: map[string]struct{}{
			"time": {},
		},
	})

	intent := classifier.Classify("time")
	if intent.Type != "command" {
		t.Fatalf("expected command intent, got %s", intent.Type)
	}
	if intent.CommandName != "time" {
		t.Fatalf("expected command time, got %s", intent.CommandName)
	}
}

func TestClassifyUnknownSlashInputAsAgent(t *testing.T) {
	classifier := NewSimpleIntentClassifier(&fakeLookup{
		commands: map[string]struct{}{
			"echo": {},
		},
	})

	intent := classifier.Classify("/goal ship native slash support")
	if intent.Type != "agent" {
		t.Fatalf("expected agent intent, got %s", intent.Type)
	}
	if intent.CommandName != "" {
		t.Fatalf("expected no command name for agent intent, got %q", intent.CommandName)
	}
}

func TestClassifyAgent(t *testing.T) {
	classifier := NewSimpleIntentClassifier(&fakeLookup{
		commands: map[string]struct{}{
			"echo": {},
		},
	})

	intent := classifier.Classify("帮我总结这段文本")
	if intent.Type != "agent" {
		t.Fatalf("expected agent intent, got %s", intent.Type)
	}
}
