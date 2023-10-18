package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/krzko/gcp-cloud-trace-otel/internal/types"
	"github.com/krzko/gcp-cloud-trace-otel/pkg/cloudtrace"
	"github.com/krzko/gcp-cloud-trace-otel/pkg/config"
	"github.com/krzko/gcp-cloud-trace-otel/pkg/otel"
)

const defaultPollInterval = 60 * time.Second

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("Failed to get configuration: %v", err)
	}

	err = cloudtrace.InitialiseCloudTraceClient()
	if err != nil {
		log.Fatalf("Failed to initialize Cloud Trace client: %v", err)
	}

	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		pollInterval = defaultPollInterval
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	// Trigger the fetch immediately
	go fetchAndProcessTraces(ctx, cfg, &wg)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				// Fetch traces
				go fetchAndProcessTraces(ctx, cfg, &wg)
			case <-stop:
				log.Println("Received stop signal. Exiting goroutine.")
				return
			}
		}
	}()

	log.Println("Service running. Press ctrl+c to stop...")
	// Wait for interrupt signal
	sig := <-signalCh
	log.Printf("Received signal: %s. Shutting down...", sig)
	cancel()    // Cancel the context
	close(stop) // Signal the goroutine to stop
	wg.Wait()   // Wait for all goroutines to finish
	log.Println("Service stopped.")
}

func fetchAndProcessTraces(ctx context.Context, cfg *config.Config, wg *sync.WaitGroup) {
	traces, err := cloudtrace.FetchTraces(ctx, cfg)
	if err != nil {
		log.Println("Error fetching traces:", err)
		return
	}

	for _, traceData := range traces {
		wg.Add(1)
		go func(td *types.Trace) {
			defer wg.Done()

			// Fetch root spans for the current trace
			rootSpans, err := cloudtrace.GetRootSpans(ctx, cfg, td.TraceID)
			if err != nil {
				log.Printf("Error getting root spans for Trace ID %s: %v", td.TraceID, err)
				return
			}

			// Process and export each span
			for _, span := range rootSpans {
				otel.ConvertToOTLPSpan(ctx, traceData, span) // Process and export the span
			}
		}(traceData)
	}
}
