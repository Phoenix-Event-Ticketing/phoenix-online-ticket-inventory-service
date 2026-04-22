package observability

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracing configures global tracing and returns a shutdown function.
func InitTracing(ctx context.Context, serviceName, jaegerEndpoint, samplerRaw string) (func(context.Context) error, error) {
	if strings.TrimSpace(jaegerEndpoint) == "" {
		return func(context.Context) error { return nil }, nil
	}

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("jaeger exporter: %w", err)
	}

	sampler := tracesdk.ParentBased(tracesdk.TraceIDRatioBased(1.0))
	if strings.TrimSpace(samplerRaw) != "" {
		if v, parseErr := strconv.ParseFloat(strings.TrimSpace(samplerRaw), 64); parseErr == nil && v >= 0 && v <= 1 {
			sampler = tracesdk.ParentBased(tracesdk.TraceIDRatioBased(v))
		}
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(strings.TrimSpace(serviceName)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("resource: %w", err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithSampler(sampler),
		tracesdk.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
