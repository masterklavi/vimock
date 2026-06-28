package delay

import (
	"context"
	"testing"
	"time"

	"vimock/internal/mapping"
)

func TestInitialDurationAdditionalBranches(t *testing.T) {
	if got := InitialDuration(mapping.ResponseDefinition{}, nil); got != 0 {
		t.Fatalf("no delay = %s", got)
	}
	if got := InitialDuration(mapping.ResponseDefinition{DelayDistribution: &mapping.DelayDistribution{Type: "uniform", Lower: 5, Upper: 5}}, nil); got != 5*time.Millisecond {
		t.Fatalf("uniform equal bounds = %s", got)
	}
	if got := InitialDuration(mapping.ResponseDefinition{DelayDistribution: &mapping.DelayDistribution{Type: "unknown"}}, nil); got != 0 {
		t.Fatalf("unknown delay = %s", got)
	}
}

func TestChunkedIntervalAdditionalBranches(t *testing.T) {
	chunks, interval := ChunkedInterval(mapping.ResponseDefinition{})
	if chunks != 0 || interval != 0 {
		t.Fatalf("no chunk delay = %d %s", chunks, interval)
	}
	chunks, interval = ChunkedInterval(mapping.ResponseDefinition{ChunkedDribbleDelay: &mapping.ChunkedDribbleDelay{NumberOfChunks: 1, TotalDurationMilliseconds: 10}})
	if chunks != 1 || interval != 0 {
		t.Fatalf("single chunk = %d %s", chunks, interval)
	}
}

func TestSleepZeroAndTimer(t *testing.T) {
	if err := Sleep(context.Background(), 0); err != nil {
		t.Fatalf("Sleep(0): %v", err)
	}
	if err := Sleep(context.Background(), time.Nanosecond); err != nil {
		t.Fatalf("Sleep(nanosecond): %v", err)
	}
}
