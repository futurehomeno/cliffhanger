package tracing

const (
	// SpanType represents a span option tag for span type (compatible with DataDog).
	SpanType = "span.type"
)

// SpanOption represents a span option.
type SpanOption struct {
	Tag   string
	Value any
}

// WithSpanType returns a span option for a span type.
func WithSpanType(t string) SpanOption {
	return SpanOption{
		Tag:   SpanType,
		Value: t,
	}
}
