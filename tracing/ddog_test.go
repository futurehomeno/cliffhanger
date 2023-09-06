package tracing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/futurehomeno/cliffhanger/tracing"
)

func TestTracing_WrappingAndPassingThroughContext(t *testing.T) {
	t.Parallel()

	tracer.Start(tracer.WithServiceName("test-service"))
	t.Cleanup(tracer.Stop)

	tr := tracing.NewDatadogAdapter()

	ctx := context.Background()
	span, ctx := tr.StartSpanFromContext(ctx, "test-start", tracing.WithSpanType("mqtt"))

	span.SetTag("test-tag", "test")
	span.Finish()

	_, ok := tracer.SpanFromContext(ctx)

	assert.True(t, ok)
}
