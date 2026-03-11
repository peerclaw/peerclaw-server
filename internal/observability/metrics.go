package observability

import (
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all OpenTelemetry metric instruments for PeerClaw.
type Metrics struct {
	HTTPRequestsTotal     metric.Int64Counter
	HTTPRequestDuration   metric.Float64Histogram
	HTTPActiveRequests    metric.Int64UpDownCounter
	SignalingConnections  metric.Int64UpDownCounter
	SignalingMessagesTotal metric.Int64Counter
	RegisteredAgents     metric.Int64UpDownCounter
	BridgeMessagesTotal  metric.Int64Counter
	BridgeMessageDuration  metric.Float64Histogram
	GatewayRequestsTotal   metric.Int64Counter
}

// NewMetrics creates all metric instruments from the given meter.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}
	var err error

	m.HTTPRequestsTotal, err = meter.Int64Counter("peerclaw.http.requests.total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	m.HTTPRequestDuration, err = meter.Float64Histogram("peerclaw.http.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.HTTPActiveRequests, err = meter.Int64UpDownCounter("peerclaw.http.requests.active",
		metric.WithDescription("Number of active HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	m.SignalingConnections, err = meter.Int64UpDownCounter("peerclaw.signaling.connections",
		metric.WithDescription("Number of active WebSocket signaling connections"),
	)
	if err != nil {
		return nil, err
	}

	m.SignalingMessagesTotal, err = meter.Int64Counter("peerclaw.signaling.messages.total",
		metric.WithDescription("Total number of signaling messages forwarded"),
	)
	if err != nil {
		return nil, err
	}

	m.RegisteredAgents, err = meter.Int64UpDownCounter("peerclaw.registry.agents",
		metric.WithDescription("Number of registered agents"),
	)
	if err != nil {
		return nil, err
	}

	m.BridgeMessagesTotal, err = meter.Int64Counter("peerclaw.bridge.messages.total",
		metric.WithDescription("Total number of bridge messages sent"),
	)
	if err != nil {
		return nil, err
	}

	m.BridgeMessageDuration, err = meter.Float64Histogram("peerclaw.bridge.message.duration",
		metric.WithDescription("Bridge message send duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.GatewayRequestsTotal, err = meter.Int64Counter("peerclaw.gateway.requests.total",
		metric.WithDescription("Total number of gateway requests by detected protocol"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}
