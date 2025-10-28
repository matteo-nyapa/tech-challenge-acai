package observability

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func Setup(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	mexp, err := stdoutmetric.New(stdoutmetric.WithWriter(os.Stdout))
	if err != nil {
		return nil, err
	}
	meterProvider := metricsdk.NewMeterProvider(
		metricsdk.WithResource(res),
		metricsdk.WithReader(metric.NewPeriodicReader(mexp)),
	)
	otel.SetMeterProvider(meterProvider)

	texp, err := stdouttrace.New(stdouttrace.WithWriter(os.Stdout))
	if err != nil {
		return nil, err
	}
	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(texp),
		tracesdk.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	slog.Info("OpenTelemetry initialized (stdout exporters)")

	return func(ctx context.Context) error {
		var retErr error
		if err := tracerProvider.Shutdown(ctx); err != nil {
			retErr = err
		}
		if err := meterProvider.Shutdown(ctx); err != nil && retErr == nil {
			retErr = err
		}
		return retErr
	}, nil
}
