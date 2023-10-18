package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

const (
	DefaultPollInterval        = 10 * time.Second
	DefaultGetTraceRateLimit   = 1 // 50 https://cloud.google.com/trace/docs/quotas
	DefaultListTracesRateLimit = 1 // 12 https://cloud.google.com/trace/docs/quotas
	DefaultTracePageSize       = 1
)

type Config struct {
	PollInterval        time.Duration
	ProjectID           string
	GetTraceRateLimit   int
	ListTracesRateLimit int
	TracePageSize       int32
}

func GetConfig() (*Config, error) {
	getTraceRateLimitStr := os.Getenv("GET_TRACE_RATE_LIMIT")
	if getTraceRateLimitStr == "" {
		getTraceRateLimitStr = strconv.Itoa(DefaultGetTraceRateLimit)
	}
	getTraceRateLimit, _ := strconv.Atoi(getTraceRateLimitStr)

	listTracesRateLimitStr := os.Getenv("LIST_TRACES_RATE_LIMIT")
	if listTracesRateLimitStr == "" {
		listTracesRateLimitStr = strconv.Itoa(DefaultListTracesRateLimit)
	}
	listTracesRateLimit, _ := strconv.Atoi(listTracesRateLimitStr)

	tracePageSizeStr := os.Getenv("TRACE_PAGE_SIZE")
	if tracePageSizeStr == "" {
		tracePageSizeStr = strconv.Itoa(DefaultListTracesRateLimit)
	}
	tracePageSizeLimit, _ := strconv.Atoi(tracePageSizeStr)

	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		return nil, errors.New("variable PROJECT_ID is not set. It is a mandatory configuration")
	}

	return &Config{
		ProjectID:           projectID,
		PollInterval:        DefaultPollInterval,
		ListTracesRateLimit: listTracesRateLimit,
		GetTraceRateLimit:   getTraceRateLimit,
		TracePageSize:       int32(tracePageSizeLimit),
	}, nil
}
