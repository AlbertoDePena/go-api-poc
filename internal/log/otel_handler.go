package log

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type otelHandler struct {
	slog.Handler
}

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func New(base slog.Handler) *slog.Logger {
	return slog.New(&otelHandler{Handler: base})
}
