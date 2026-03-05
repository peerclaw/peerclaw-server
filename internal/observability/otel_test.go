package observability

import (
	"context"
	"log/slog"
	"testing"

	"github.com/peerclaw/peerclaw-server/internal/config"
)

func TestInit_Disabled(t *testing.T) {
	cfg := config.ObservabilityConfig{
		Enabled: false,
	}
	provider, err := Init(context.Background(), cfg, slog.Default())
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer provider.Shutdown(context.Background())

	if provider.TracerProvider == nil {
		t.Error("TracerProvider is nil")
	}
	if provider.MeterProvider == nil {
		t.Error("MeterProvider is nil")
	}

	// Verify no-op tracer produces valid spans.
	tracer := provider.TracerProvider.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.End()
}

func TestTracer_And_Meter(t *testing.T) {
	cfg := config.ObservabilityConfig{Enabled: false}
	provider, err := Init(context.Background(), cfg, slog.Default())
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer provider.Shutdown(context.Background())

	tracer := Tracer("test-tracer")
	if tracer == nil {
		t.Error("Tracer() returned nil")
	}

	meter := Meter("test-meter")
	if meter == nil {
		t.Error("Meter() returned nil")
	}
}

func TestProvider_Shutdown_Nil(t *testing.T) {
	var p *Provider
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}
