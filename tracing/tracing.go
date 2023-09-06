package tracing

import (
	"context"
)

// Tracer represents a tracer.
type Tracer interface {
	StartSpanFromContext(ctx context.Context, operationName string, opts ...SpanOption) (Span, context.Context)
}

// Span represents a span.
type Span interface {
	SetTag(tag string, value any)
	Finish()
}

var _ Tracer = (*NoOpTracer)(nil)

// NoOpTracer represents a no-op tracer. Used by Cliffhanger router by default.
type NoOpTracer struct{}

// NewNoOpTracer returns a new instance of NoOpTracer.
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

func (n NoOpTracer) StartSpanFromContext(ctx context.Context, _ string, _ ...SpanOption) (Span, context.Context) {
	return NoOpSpan{}, ctx
}

var _ Span = (*NoOpSpan)(nil)

// NoOpSpan represents a no-op span.
type NoOpSpan struct{}

func (n NoOpSpan) SetTag(_ string, _ any) {}

func (n NoOpSpan) Finish() {}
