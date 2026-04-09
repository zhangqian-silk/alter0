package application

import (
	"testing"
	"time"
)

type contractIDGenerator struct{}

func (contractIDGenerator) NewID() string { return "id-1" }

type contractTelemetry struct {
	gatewayCount int
	routeCount   int
	commandCount int
	errorCount   int
	memoryCount  int
	duration     time.Duration
}

func (t *contractTelemetry) CountGateway(string) { t.gatewayCount++ }

func (t *contractTelemetry) CountRoute(string) { t.routeCount++ }

func (t *contractTelemetry) CountCommand(string) { t.commandCount++ }

func (t *contractTelemetry) CountError(string) { t.errorCount++ }

func (t *contractTelemetry) CountMemoryEvent(string) { t.memoryCount++ }

func (t *contractTelemetry) ObserveDuration(_ string, duration time.Duration) {
	t.duration = duration
}

func TestApplicationPortsKeepStableMethodContracts(t *testing.T) {
	var generator IDGenerator = contractIDGenerator{}
	if got := generator.NewID(); got != "id-1" {
		t.Fatalf("NewID() = %q, want id-1", got)
	}

	telemetry := &contractTelemetry{}
	var port Telemetry = telemetry
	port.CountGateway("web")
	port.CountRoute("nl")
	port.CountCommand("help")
	port.CountError("nl")
	port.CountMemoryEvent("recall")
	port.ObserveDuration("nl", 2*time.Second)

	if telemetry.gatewayCount != 1 || telemetry.routeCount != 1 || telemetry.commandCount != 1 || telemetry.errorCount != 1 || telemetry.memoryCount != 1 {
		t.Fatalf("telemetry counts = %+v, want one call per counter", telemetry)
	}
	if telemetry.duration != 2*time.Second {
		t.Fatalf("duration = %v, want 2s", telemetry.duration)
	}
}
