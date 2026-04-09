package observability

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTelemetryMetricsHandlerExportsCountersAndAverages(t *testing.T) {
	telemetry := NewTelemetry()
	telemetry.CountGateway("web")
	telemetry.CountGateway("web")
	telemetry.CountGateway("cli")
	telemetry.CountRoute("nl")
	telemetry.CountCommand("help")
	telemetry.CountError("nl")
	telemetry.CountMemoryEvent("recall")
	telemetry.ObserveDuration("nl", 2*time.Second)
	telemetry.ObserveDuration("nl", 4*time.Second)
	telemetry.CountQueueEvent("accepted")
	telemetry.ObserveQueueWait(3 * time.Second)
	telemetry.SetQueueDepth(5)
	telemetry.SetWorkerInFlight(2)

	response := httptest.NewRecorder()
	telemetry.MetricsHandler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))

	if got := response.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want Prometheus text content type", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		`alter0_gateway_messages_total{channel_type="cli"} 1`,
		`alter0_gateway_messages_total{channel_type="web"} 2`,
		`alter0_route_requests_total{route="nl"} 1`,
		`alter0_command_requests_total{command="help"} 1`,
		`alter0_route_errors_total{route="nl"} 1`,
		`alter0_memory_events_total{event="recall"} 1`,
		`alter0_route_duration_seconds_avg{route="nl"} 3.000000`,
		`alter0_queue_events_total{event="accepted"} 1`,
		`alter0_queue_wait_seconds_avg 3.000000`,
		`alter0_queue_depth 5`,
		`alter0_worker_inflight 2`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q:\n%s", want, body)
		}
	}
	if strings.Index(body, `channel_type="cli"`) > strings.Index(body, `channel_type="web"`) {
		t.Fatalf("gateway labels are not sorted:\n%s", body)
	}
}

func TestTelemetryMetricsHandlerExportsZeroQueueMetricsWithoutSamples(t *testing.T) {
	response := httptest.NewRecorder()
	NewTelemetry().MetricsHandler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))

	body := response.Body.String()
	for _, want := range []string{
		`alter0_queue_wait_seconds_avg 0.000000`,
		`alter0_queue_depth 0`,
		`alter0_worker_inflight 0`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q:\n%s", want, body)
		}
	}
}
