package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/twitchtv/twirp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey struct{}

var startKey ctxKey

type ServerMetrics struct {
	Requests metric.Int64Counter
	Duration metric.Float64Histogram
	Errors   metric.Int64Counter
	Tracer   trace.Tracer
}

func NewServerMetrics() (*twirp.ServerHooks, *ServerMetrics, error) {
	meter := otel.Meter("acai/server")

	reqs, err := meter.Int64Counter("http.server.requests",
		metric.WithDescription("Number of inbound HTTP/Twirp requests"),
	)
	if err != nil {
		return nil, nil, err
	}

	dur, err := meter.Float64Histogram("http.server.duration.seconds",
		metric.WithDescription("Duration of inbound HTTP/Twirp requests (s)"),
	)
	if err != nil {
		return nil, nil, err
	}

	errs, err := meter.Int64Counter("http.server.errors",
		metric.WithDescription("Number of server errors"),
	)
	if err != nil {
		return nil, nil, err
	}

	tracer := otel.Tracer("acai/server")

	sm := &ServerMetrics{
		Requests: reqs,
		Duration: dur,
		Errors:   errs,
		Tracer:   tracer,
	}

	h := &twirp.ServerHooks{
		RequestReceived: func(ctx context.Context) (context.Context, error) {
			method, _ := twirp.MethodName(ctx)
			service, _ := twirp.ServiceName(ctx)
			route := fmt.Sprintf("%s/%s", service, method)

			attrs := []attribute.KeyValue{
				attribute.String("rpc.system", "twirp"),
				attribute.String("rpc.service", service),
				attribute.String("rpc.method", method),
				attribute.String("http.route", route),
			}

			spanName := "twirp.request"
			if service != "" && method != "" {
				spanName = fmt.Sprintf("%s/%s", service, method)
			}
			ctx, span := sm.Tracer.Start(ctx, spanName)
			span.SetAttributes(attrs...)

			ctx = context.WithValue(ctx, startKey, time.Now())
			return ctx, nil
		},

		ResponseSent: func(ctx context.Context) {
			start, _ := ctx.Value(startKey).(time.Time)
			var elapsedSec float64
			if !start.IsZero() {
				elapsedSec = time.Since(start).Seconds()
			}

			method, _ := twirp.MethodName(ctx)
			service, _ := twirp.ServiceName(ctx)
			status, _ := twirp.StatusCode(ctx)
			route := fmt.Sprintf("%s/%s", service, method)

			attrs := []attribute.KeyValue{
				attribute.String("rpc.system", "twirp"),
				attribute.String("rpc.service", service),
				attribute.String("rpc.method", method),
				attribute.String("http.route", route),
				attribute.String("http.status_code", status),
			}

			sm.Requests.Add(ctx, 1, metric.WithAttributes(attrs...))
			sm.Duration.Record(ctx, elapsedSec, metric.WithAttributes(attrs...))

			if span := trace.SpanFromContext(ctx); span != nil {
				span.SetAttributes(attrs...)
				span.End()
			}
		},

		Error: func(ctx context.Context, twerr twirp.Error) context.Context {
			method, _ := twirp.MethodName(ctx)
			service, _ := twirp.ServiceName(ctx)
			status, _ := twirp.StatusCode(ctx)
			route := fmt.Sprintf("%s/%s", service, method)

			attrs := []attribute.KeyValue{
				attribute.String("rpc.system", "twirp"),
				attribute.String("rpc.service", service),
				attribute.String("rpc.method", method),
				attribute.String("http.route", route),
				attribute.String("http.status_code", status),
				attribute.String("twirp.error_code", string(twerr.Code())),
			}
			sm.Errors.Add(ctx, 1, metric.WithAttributes(attrs...))

			if span := trace.SpanFromContext(ctx); span != nil {
				span.RecordError(twerr)
			}
			return ctx
		},
	}

	return h, sm, nil
}
