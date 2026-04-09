package id

import (
	"regexp"
	"testing"
)

func TestRandomIDGeneratorNewIDUsesTimestampPrefixAndOptionalRandomSuffix(t *testing.T) {
	generator := NewRandomIDGenerator()

	id := generator.NewID()

	pattern := regexp.MustCompile(`^\d{8}T\d{6}\.\d{9}(-[0-9a-f]{16})?$`)
	if !pattern.MatchString(id) {
		t.Fatalf("NewID() = %q, want timestamp prefix with optional random suffix", id)
	}
}
