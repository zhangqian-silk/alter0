package application

import "testing"

func TestParseNativeMemoriesFeatureList(t *testing.T) {
	status := ParseNativeMemoriesFeatureList("shell_tool stable true\nmemories experimental false\n")
	if !status.Available || status.Diagnostic != "" {
		t.Fatalf("expected memories feature available regardless of base toggle, got %+v", status)
	}
}

func TestParseNativeMemoriesFeatureListReportsMissingFeature(t *testing.T) {
	status := ParseNativeMemoriesFeatureList("shell_tool stable true\n")
	if status.Available || status.Diagnostic == "" {
		t.Fatalf("expected missing memories diagnostic, got %+v", status)
	}
}
