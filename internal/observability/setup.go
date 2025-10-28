package observability

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func SetupOTel(ctx context.Context) (func(context.Context) error, error) {
	metricExporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("creating metric exporter: %w", err)
	}

	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("creating trace exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName("acai-server"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	meterProvider := metricsdk.NewMeterProvider(
		metricsdk.WithReader(metric.NewPeriodicReader(metricExporter)),
		metricsdk.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	traceProvider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(traceExporter),
		tracesdk.WithResource(res),
	)
	otel.SetTracerProvider(traceProvider)

	slog.Info("OpenTelemetry initialized (stdout exporters active)")

	return func(ctx context.Context) error {
		err1 := meterProvider.Shutdown(ctx)
		err2 := traceProvider.Shutdown(ctx)
		if err1 != nil {
			return err1
		}
		return err2
	}, nil
}
