package tracing

import (
	"context"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

//nolint:godox
//TODO: move contents of this file outside of the project (to be decided where)

var _ Tracer = (*DatadogAdapter)(nil)

// DatadogAdapter represents a Datadog adapter for Tracer.
// The adapter assumes that Datadog tracer is already started. If not, call tracer.Start() before using it.
type DatadogAdapter struct{}

// NewDatadogAdapter returns a new instance of DatadogAdapter.
func NewDatadogAdapter() *DatadogAdapter {
	return &DatadogAdapter{}
}

func (a *DatadogAdapter) StartSpanFromContext(ctx context.Context, operationName string, opts ...SpanOption) (Span, context.Context) {
	options := make([]tracer.StartSpanOption, 0, len(opts))
	for _, o := range opts {
		options = append(options, tracer.Tag(o.Tag, o.Value))
	}

	ddSpan, ctx := tracer.StartSpanFromContext(ctx, operationName, options...)

	return &DatadogSpan{span: ddSpan}, ctx
}

var _ Span = (*DatadogSpan)(nil)

// DatadogSpan represents a wrapper around DataDog's span that is compatible with Span.
type DatadogSpan struct {
	span tracer.Span
}

func (d *DatadogSpan) SetTag(tag string, value any) {
	d.span.SetTag(tag, value)
}

func (d *DatadogSpan) Finish() {
	d.span.Finish()
}
