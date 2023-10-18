package otel

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/krzko/gcp-cloud-trace-otel/internal/types"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var tracer oteltrace.Tracer

func init() {
	ctx := context.Background()

	// Ensure the collector address is set
	collectorAddress := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if collectorAddress == "" {
		collectorAddress = "localhost:4317" // default address
	}

	// Determine if we should use insecure or secure communication
	var opts []grpc.DialOption
	if strings.HasSuffix(collectorAddress, ":443") {
		creds := credentials.NewClientTLSFromCert(nil, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	driver := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint(collectorAddress), otlptracegrpc.WithDialOption(opts...))
	exporter, err := otlptrace.New(ctx, driver)
	if err != nil {
		log.Fatalf("Failed to create OTLP exporter: %v", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		attribute.String("agent.name", "gcp-cloud-trace-otel"),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	tracer = otel.Tracer("gcp-cloud-trace-converter")
}

func ConvertToOTLPSpan(ctx context.Context, gctTrace *types.Trace, gctSpan *types.Span) {
	log.Printf("Converting: GCP Cloud Trace span ID: %d, Name: %s to OTLP format", gctSpan.SpanID, gctSpan.Name)

	start := time.Unix(gctSpan.StartTime.Seconds, int64(gctSpan.StartTime.Nanos))
	end := time.Unix(gctSpan.EndTime.Seconds, int64(gctSpan.EndTime.Nanos))

	// Convert TraceID from GCP Cloud Trace format to OpenTelemetry format
	traceID, err := oteltrace.TraceIDFromHex(gctTrace.TraceID)
	if err != nil {
		log.Printf("Error converting TraceID: %v", err)
		return
	}

	// Convert SpanID from GCP Cloud Trace format to OpenTelemetry format
	spanID, err := oteltrace.SpanIDFromHex(fmt.Sprintf("%016x", gctSpan.SpanID))
	if err != nil {
		log.Printf("Error converting SpanID: %v", err)
		return
	}

	// Create a new SpanContext with the original TraceID and SpanID
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
	})

	// Start the new span with the TraceContext set
	_, span := tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, sc),
		gctSpan.Name,
		oteltrace.WithTimestamp(start),
	)

	// End the span with the given end time
	span.End(oteltrace.WithTimestamp(end))

	// If there's a ParentSpanID, set it as an attribute
	if gctSpan.ParentSpanID != nil {
		span.SetAttributes(attribute.String("gct.parentSpanId", fmt.Sprintf("%d", *gctSpan.ParentSpanID)))
	}

	// Check for service.name attribute and set it if exists
	if serviceName, exists := gctSpan.Labels["service.name"]; exists {
		span.SetAttributes(attribute.String("service.name", serviceName))
	} else if serviceName, exists := gctSpan.Labels["g.co/gae/app/module"]; exists {
		span.SetAttributes(attribute.String("service.name", serviceName))
	} else if serviceName, exists := gctSpan.Labels["g.co/r/generic_task/job"]; exists {
		span.SetAttributes(attribute.String("service.name", serviceName))
	}

	log.Printf("Converted: GCP Cloud Trace span ID: %d, Name: %s to OTLP format successfully", gctSpan.SpanID, gctSpan.Name)
}
