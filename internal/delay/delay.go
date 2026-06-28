package delay

import (
	"context"
	"math"
	"math/rand"
	"time"

	"vimock/internal/mapping"
)

type Sleeper func(context.Context, time.Duration) error

func InitialDuration(definition mapping.ResponseDefinition, random *rand.Rand) time.Duration {
	if definition.FixedDelayMilliseconds > 0 {
		return milliseconds(definition.FixedDelayMilliseconds)
	}
	if definition.DelayDistribution == nil {
		return 0
	}

	switch definition.DelayDistribution.Type {
	case "uniform":
		lower := definition.DelayDistribution.Lower
		upper := definition.DelayDistribution.Upper
		if upper <= lower {
			return milliseconds(lower)
		}
		if random == nil {
			random = rand.New(rand.NewSource(time.Now().UnixNano()))
		}
		return milliseconds(lower + random.Intn(upper-lower+1))
	case "lognormal":
		if random == nil {
			random = rand.New(rand.NewSource(time.Now().UnixNano()))
		}
		median := float64(definition.DelayDistribution.Median)
		sigma := definition.DelayDistribution.Sigma
		return time.Duration(math.Exp(math.Log(median)+sigma*random.NormFloat64())) * time.Millisecond
	default:
		return 0
	}
}

func ChunkedInterval(definition mapping.ResponseDefinition) (int, time.Duration) {
	if definition.ChunkedDribbleDelay == nil {
		return 0, 0
	}
	chunks := definition.ChunkedDribbleDelay.NumberOfChunks
	if chunks <= 1 {
		return chunks, 0
	}

	total := milliseconds(definition.ChunkedDribbleDelay.TotalDurationMilliseconds)
	return chunks, total / time.Duration(chunks-1)
}

func Sleep(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func milliseconds(value int) time.Duration {
	return time.Duration(value) * time.Millisecond
}
