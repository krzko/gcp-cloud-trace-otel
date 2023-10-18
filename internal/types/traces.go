package types

type TimeStamp struct {
	Seconds int64
	Nanos   int64
}

type Span struct {
	SpanID       uint64
	Name         string
	StartTime    TimeStamp
	EndTime      TimeStamp
	ParentSpanID *uint64
	Labels       map[string]string
}

type Trace struct {
	ProjectID string
	TraceID   string
	Spans     []Span
}
