package domain

import "testing"

func TestNormalizeSessionStatus(t *testing.T) {
	tests := []struct {
		name   string
		input  SessionStatus
		expect SessionStatus
	}{
		{name: "empty defaults to ready", input: "", expect: SessionStatusReady},
		{name: "ready stays ready", input: SessionStatusReady, expect: SessionStatusReady},
		{name: "busy stays busy", input: SessionStatusBusy, expect: SessionStatusBusy},
		{name: "exited stays exited", input: SessionStatusExited, expect: SessionStatusExited},
		{name: "failed stays failed", input: SessionStatusFailed, expect: SessionStatusFailed},
		{name: "interrupted stays interrupted", input: SessionStatusInterrupted, expect: SessionStatusInterrupted},
		{name: "unknown falls back to ready", input: SessionStatus("unknown"), expect: SessionStatusReady},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeSessionStatus(tc.input); got != tc.expect {
				t.Fatalf("NormalizeSessionStatus(%q) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}

func TestSessionOpenAndInputAvailability(t *testing.T) {
	if !IsSessionOpenStatus(SessionStatusReady) {
		t.Fatalf("expected ready session to be open")
	}
	if !IsSessionOpenStatus(SessionStatusBusy) {
		t.Fatalf("expected busy session to be open")
	}
	if IsSessionOpenStatus(SessionStatusExited) {
		t.Fatalf("expected exited session to be closed")
	}
	if !CanSessionAcceptInput(SessionStatusReady) {
		t.Fatalf("expected ready session to accept input")
	}
	if CanSessionAcceptInput(SessionStatusBusy) {
		t.Fatalf("expected busy session to reject input")
	}
}

func TestNewRepositoryBindingStartsPreparingInFixedWorkspacePath(t *testing.T) {
	repository := Repository{
		Provider:      RepositoryProviderGitHub,
		ID:            "123456789",
		FullName:      " owner/repository ",
		Private:       true,
		DefaultBranch: "main",
	}

	binding := NewRepositoryBinding(repository)

	if binding.Provider != RepositoryProviderGitHub {
		t.Fatalf("expected github provider, got %q", binding.Provider)
	}
	if binding.ID != "123456789" || binding.FullName != "owner/repository" {
		t.Fatalf("expected normalized repository identity, got %+v", binding)
	}
	if binding.Status != RepositoryPreparationStatusPreparing {
		t.Fatalf("expected preparing status, got %q", binding.Status)
	}
	if binding.WorkspacePath != RepositoryWorkspacePath {
		t.Fatalf("expected fixed workspace path %q, got %q", RepositoryWorkspacePath, binding.WorkspacePath)
	}
	if !binding.Matches(RepositoryRef{Provider: RepositoryProviderGitHub, ID: "123456789"}) {
		t.Fatalf("expected binding to match the same stable repository id")
	}
	if binding.Matches(RepositoryRef{Provider: RepositoryProviderGitHub, ID: "987654321"}) {
		t.Fatalf("expected binding to reject a different repository id")
	}
}

func TestNormalizeRepositoryPreparationStatusRejectsUnknownValues(t *testing.T) {
	for _, test := range []struct {
		input RepositoryPreparationStatus
		want  RepositoryPreparationStatus
	}{
		{input: RepositoryPreparationStatusPreparing, want: RepositoryPreparationStatusPreparing},
		{input: RepositoryPreparationStatusReady, want: RepositoryPreparationStatusReady},
		{input: RepositoryPreparationStatusFailed, want: RepositoryPreparationStatusFailed},
		{input: "unknown", want: RepositoryPreparationStatusFailed},
		{input: "", want: RepositoryPreparationStatusFailed},
	} {
		if got := NormalizeRepositoryPreparationStatus(test.input); got != test.want {
			t.Fatalf("NormalizeRepositoryPreparationStatus(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
