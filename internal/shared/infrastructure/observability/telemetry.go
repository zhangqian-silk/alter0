package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type routeLatency struct {
	Count int64
	SumNs int64
}

type waitLatency struct {
	Count int64
	SumNs int64
}

type Telemetry struct {
	mu sync.Mutex

	gatewayCount map[string]int64
	routeCount   map[string]int64
	commandCount map[string]int64
	errorCount   map[string]int64
	memoryEvents map[string]int64
	routeLatency map[string]routeLatency
	queueEvents  map[string]int64
	queueWait    waitLatency
	queueDepth   int64
	workerFlight int64
}

func NewTelemetry() *Telemetry {
	return &Telemetry{
		gatewayCount: map[string]int64{},
		routeCount:   map[string]int64{},
		commandCount: map[string]int64{},
		errorCount:   map[string]int64{},
		memoryEvents: map[string]int64{},
		routeLatency: map[string]routeLatency{},
		queueEvents:  map[string]int64{},
	}
}

func (t *Telemetry) CountGateway(channelType string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gatewayCount[channelType]++
}

func (t *Telemetry) CountRoute(route string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.routeCount[route]++
}

func (t *Telemetry) CountCommand(command string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.commandCount[command]++
}

func (t *Telemetry) CountError(route string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.errorCount[route]++
}

func (t *Telemetry) CountMemoryEvent(event string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.memoryEvents[event]++
}

func (t *Telemetry) ObserveDuration(route string, d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	latency := t.routeLatency[route]
	latency.Count++
	latency.SumNs += d.Nanoseconds()
	t.routeLatency[route] = latency
}

func (t *Telemetry) CountQueueEvent(event string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.queueEvents[event]++
}

func (t *Telemetry) ObserveQueueWait(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.queueWait.Count++
	t.queueWait.SumNs += d.Nanoseconds()
}

func (t *Telemetry) SetQueueDepth(depth int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.queueDepth = int64(depth)
}

func (t *Telemetry) SetWorkerInFlight(inFlight int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.workerFlight = int64(inFlight)
}

func (t *Telemetry) MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.mu.Lock()
		gatewayCount := cloneMap(t.gatewayCount)
		routeCount := cloneMap(t.routeCount)
		commandCount := cloneMap(t.commandCount)
		errorCount := cloneMap(t.errorCount)
		memoryEvents := cloneMap(t.memoryEvents)
		routeLatency := cloneLatencyMap(t.routeLatency)
		queueEvents := cloneMap(t.queueEvents)
		queueWait := t.queueWait
		queueDepth := t.queueDepth
		workerFlight := t.workerFlight
		t.mu.Unlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		builder := &strings.Builder{}

		writeCounter(builder, "alter0_gateway_messages_total", "channel_type", gatewayCount)
		writeCounter(builder, "alter0_route_requests_total", "route", routeCount)
		writeCounter(builder, "alter0_command_requests_total", "command", commandCount)
		writeCounter(builder, "alter0_route_errors_total", "route", errorCount)
		writeCounter(builder, "alter0_memory_events_total", "event", memoryEvents)
		writeLatency(builder, routeLatency)
		writeCounter(builder, "alter0_queue_events_total", "event", queueEvents)
		writeWait(builder, queueWait)
		fmt.Fprintf(builder, "alter0_queue_depth %d\n", queueDepth)
		fmt.Fprintf(builder, "alter0_worker_inflight %d\n", workerFlight)

		_, _ = w.Write([]byte(builder.String()))
	})
}

func cloneMap(src map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloneLatencyMap(src map[string]routeLatency) map[string]routeLatency {
	out := make(map[string]routeLatency, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func writeCounter(builder *strings.Builder, name, label string, values map[string]int64) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(builder, "%s{%s=%q} %d\n", name, label, key, values[key])
	}
}

func writeLatency(builder *strings.Builder, values map[string]routeLatency) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		metric := values[key]
		avg := 0.0
		if metric.Count > 0 {
			avg = float64(metric.SumNs) / float64(metric.Count) / float64(time.Second)
		}
		fmt.Fprintf(builder, "alter0_route_duration_seconds_avg{route=%q} %f\n", key, avg)
	}
}

func writeWait(builder *strings.Builder, metric waitLatency) {
	avg := 0.0
	if metric.Count > 0 {
		avg = float64(metric.SumNs) / float64(metric.Count) / float64(time.Second)
	}
	fmt.Fprintf(builder, "alter0_queue_wait_seconds_avg %f\n", avg)
}
