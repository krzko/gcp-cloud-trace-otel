package cloudtrace

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/protobuf/types/known/timestamppb"

	trace "cloud.google.com/go/trace/apiv1"
	"cloud.google.com/go/trace/apiv1/tracepb"
	"github.com/krzko/gcp-cloud-trace-otel/internal/types"
	"github.com/krzko/gcp-cloud-trace-otel/pkg/config"
)

var (
	client            *trace.Client
	listTracesLimiter *rate.Limiter
	getTraceLimiter   *rate.Limiter
)

func InitialiseCloudTraceClient() error {
	var err error
	client, err = trace.NewClient(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create trace client: %v", err)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	listTracesLimiter = rate.NewLimiter(rate.Limit(cfg.ListTracesRateLimit), cfg.ListTracesRateLimit)
	getTraceLimiter = rate.NewLimiter(rate.Limit(cfg.GetTraceRateLimit), cfg.GetTraceRateLimit)

	log.Println("Initialised Cloud Trace client successfully.")
	return nil
}

func FetchTraces(ctx context.Context, cfg *config.Config) ([]*types.Trace, error) {
	if err := listTracesLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	log.Printf("Fetching: traces for Project ID: %s", cfg.ProjectID)
	endTime := time.Now()
	startTime := endTime.Add(-3 * time.Second)

	listTracesReq := &tracepb.ListTracesRequest{
		ProjectId: cfg.ProjectID,
		View:      tracepb.ListTracesRequest_ROOTSPAN,
		PageSize:  cfg.TracePageSize,
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
		// Filter:    "label:service.name=YOUR_SERVICE_NAME",
	}
	tracesIterator := client.ListTraces(ctx, listTracesReq)

	var fetchedTraces []*types.Trace
	for {
		traceProto, err := tracesIterator.Next()
		if err != nil {
			break
		}

		traceData := &types.Trace{
			ProjectID: traceProto.ProjectId,
			TraceID:   traceProto.TraceId,
			Spans:     convertSpans(traceProto.Spans),
		}

		fetchedTraces = append(fetchedTraces, traceData)
	}

	log.Printf("Fetched: %d traces for Project ID: %s in %v", len(fetchedTraces), cfg.ProjectID, time.Since(startTime))
	return fetchedTraces, nil
}

func convertSpans(spanProtos []*tracepb.TraceSpan) []types.Span {
	spans := make([]types.Span, len(spanProtos))
	for i, spanProto := range spanProtos {
		parentSpanID := spanProto.ParentSpanId
		var parentSpanIDPtr *uint64
		if parentSpanID != 0 {
			parentSpanIDPtr = &parentSpanID
		}
		spans[i] = types.Span{
			SpanID:       spanProto.SpanId,
			Name:         spanProto.Name,
			StartTime:    types.TimeStamp{Seconds: spanProto.StartTime.Seconds, Nanos: int64(spanProto.StartTime.Nanos)},
			EndTime:      types.TimeStamp{Seconds: spanProto.EndTime.Seconds, Nanos: int64(spanProto.EndTime.Nanos)},
			ParentSpanID: parentSpanIDPtr,
			Labels:       spanProto.Labels,
		}
	}
	return spans
}

func GetRootSpans(ctx context.Context, cfg *config.Config, traceId string) ([]*types.Span, error) {
	if err := getTraceLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	log.Printf("Fetching: root spans for Trace ID: %s", traceId)
	req := &tracepb.GetTraceRequest{
		ProjectId: cfg.ProjectID,
		TraceId:   traceId,
	}

	traceResp, err := client.GetTrace(ctx, req)
	if err != nil {
		return nil, err
	}

	traceData := &types.Trace{
		ProjectID: traceResp.ProjectId,
		TraceID:   traceResp.TraceId,
		Spans:     convertSpans(traceResp.Spans),
	}

	var rootSpans []*types.Span
	for _, span := range traceData.Spans {
		if span.ParentSpanID == nil {
			rootSpans = append(rootSpans, &span)
		}
	}

	log.Printf("Fetched: %d root spans for Trace ID: %s", len(rootSpans), traceId)
	return rootSpans, nil
}

func ProcessTraces(ctx context.Context, cfg *config.Config) ([]*types.Span, error) {
	log.Printf("Processing: traces for Project ID: %s", cfg.ProjectID)

	traces, err := FetchTraces(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("error fetching traces for Project ID %s: %w", cfg.ProjectID, err)
	}

	var allRootSpans []*types.Span
	for _, traceData := range traces {
		rootSpans, err := GetRootSpans(ctx, cfg, traceData.TraceID)
		if err != nil {
			log.Printf("Error: fetching root spans for Trace ID %s: %v", traceData.TraceID, err)
			continue
		}
		allRootSpans = append(allRootSpans, rootSpans...)
	}

	log.Printf("Processed: %d traces for Project ID: %s", len(traces), cfg.ProjectID)
	return allRootSpans, nil
}
