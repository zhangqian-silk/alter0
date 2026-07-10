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
	if !CanSessionAcceptInput(SessionStatusFailed) {
		t.Fatalf("expected failed session to accept retry input")
	}
}
