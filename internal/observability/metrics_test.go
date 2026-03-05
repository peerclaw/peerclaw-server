package observability

import (
	"testing"

	noopmetric "go.opentelemetry.io/otel/metric/noop"
)

func TestNewMetrics(t *testing.T) {
	meter := noopmetric.NewMeterProvider().Meter("test")
	m, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("NewMetrics() error = %v", err)
	}
	if m.HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal is nil")
	}
	if m.HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration is nil")
	}
	if m.HTTPActiveRequests == nil {
		t.Error("HTTPActiveRequests is nil")
	}
	if m.SignalingConnections == nil {
		t.Error("SignalingConnections is nil")
	}
	if m.SignalingMessagesTotal == nil {
		t.Error("SignalingMessagesTotal is nil")
	}
	if m.RegisteredAgents == nil {
		t.Error("RegisteredAgents is nil")
	}
	if m.BridgeMessagesTotal == nil {
		t.Error("BridgeMessagesTotal is nil")
	}
	if m.BridgeMessageDuration == nil {
		t.Error("BridgeMessageDuration is nil")
	}
}
