package delay

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"vimock/internal/mapping"
)

func TestInitialDurationFixedDelay(t *testing.T) {
	got := InitialDuration(mapping.ResponseDefinition{
		FixedDelayMilliseconds: 25,
		DelayDistribution: &mapping.DelayDistribution{
			Type:  "uniform",
			Lower: 1,
			Upper: 1,
		},
	}, rand.New(rand.NewSource(1)))
	if got != 25*time.Millisecond {
		t.Fatalf("duration = %s, want 25ms", got)
	}
}

func TestInitialDurationUniformDistribution(t *testing.T) {
	random := rand.New(rand.NewSource(1))
	for range 20 {
		got := InitialDuration(mapping.ResponseDefinition{
			DelayDistribution: &mapping.DelayDistribution{
				Type:  "uniform",
				Lower: 10,
				Upper: 20,
			},
		}, random)
		if got < 10*time.Millisecond || got > 20*time.Millisecond {
			t.Fatalf("duration = %s, want within [10ms,20ms]", got)
		}
	}
}

func TestInitialDurationLognormalDistribution(t *testing.T) {
	got := InitialDuration(mapping.ResponseDefinition{
		DelayDistribution: &mapping.DelayDistribution{
			Type:   "lognormal",
			Median: 50,
			Sigma:  0.2,
		},
	}, rand.New(rand.NewSource(1)))
	if got <= 0 {
		t.Fatalf("duration = %s, want positive", got)
	}
}

func TestChunkedInterval(t *testing.T) {
	chunks, interval := ChunkedInterval(mapping.ResponseDefinition{
		ChunkedDribbleDelay: &mapping.ChunkedDribbleDelay{
			NumberOfChunks:            3,
			TotalDurationMilliseconds: 30,
		},
	})
	if chunks != 3 {
		t.Fatalf("chunks = %d, want 3", chunks)
	}
	if interval != 15*time.Millisecond {
		t.Fatalf("interval = %s, want 15ms", interval)
	}
}

func TestSleepHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Sleep(ctx, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Sleep() error = %v, want context.Canceled", err)
	}
}
