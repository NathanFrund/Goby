package pubsub

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracingConfig holds configuration for OpenTelemetry tracing
type TracingConfig struct {
	Enabled     bool   // Whether tracing is enabled
	ServiceName string // Service name for traces
	ZipkinURL   string // Zipkin exporter URL
}

// DefaultTracingConfig returns a default tracing configuration
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:     false, // Disabled by default
		ServiceName: "goby-service",
		ZipkinURL:   "http://localhost:9411/api/v2/spans",
	}
}

// SetupOTel initializes OpenTelemetry with Zipkin exporter for pubsub observability.
// This provides tracing capabilities to monitor message flows through the pubsub system.
// If config.Enabled is false, returns a no-op tracer.
func SetupOTel(ctx context.Context, config TracingConfig) (trace.Tracer, func(), error) {
	if !config.Enabled {
		// Return no-op tracer when disabled
		tracer := noop.NewTracerProvider().Tracer("goby-pubsub")
		cleanup := func() {} // No-op cleanup
		return tracer, cleanup, nil
	}

	// Create Zipkin exporter
	exporter, err := zipkin.New(config.ZipkinURL)
	if err != nil {
		return nil, nil, err
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Create tracer
	tracer := tp.Tracer("goby-pubsub")

	// Return cleanup function
	cleanup := func() {
		if err := tp.Shutdown(ctx); err != nil {
			panic(err)
		}
	}

	return tracer, cleanup, nil
}

// SetupOTelSimple is a convenience function for simple setup with default config.
// Deprecated: Use SetupOTel with TracingConfig instead.
func SetupOTelSimple(ctx context.Context, serviceName, zipkinURL string) (trace.Tracer, func(), error) {
	config := TracingConfig{
		Enabled:     true,
		ServiceName: serviceName,
		ZipkinURL:   zipkinURL,
	}
	return SetupOTel(ctx, config)
}
