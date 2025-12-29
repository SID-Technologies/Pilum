package workerqueue_test

import (
	"testing"

	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"
	"github.com/stretchr/testify/require"
)

func TestExponentialBackoffWithJitter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		attempt   int
		baseDelay float64
		maxDelay  float64
		minResult float64
		maxResult float64
	}{
		{
			name:      "first attempt (0)",
			attempt:   0,
			baseDelay: 1.0,
			maxDelay:  60.0,
			minResult: 0.5,  // 1 * 2^0 * 0.5 = 0.5
			maxResult: 1.5,  // 1 * 2^0 * 1.5 = 1.5
		},
		{
			name:      "second attempt (1)",
			attempt:   1,
			baseDelay: 1.0,
			maxDelay:  60.0,
			minResult: 1.0,  // 1 * 2^1 * 0.5 = 1.0
			maxResult: 3.0,  // 1 * 2^1 * 1.5 = 3.0
		},
		{
			name:      "third attempt (2)",
			attempt:   2,
			baseDelay: 1.0,
			maxDelay:  60.0,
			minResult: 2.0,  // 1 * 2^2 * 0.5 = 2.0
			maxResult: 6.0,  // 1 * 2^2 * 1.5 = 6.0
		},
		{
			name:      "capped by maxDelay",
			attempt:   10,
			baseDelay: 1.0,
			maxDelay:  10.0,
			minResult: 5.0,  // 10 * 0.5 = 5.0
			maxResult: 15.0, // 10 * 1.5 = 15.0
		},
		{
			name:      "different base delay",
			attempt:   0,
			baseDelay: 2.0,
			maxDelay:  60.0,
			minResult: 1.0, // 2 * 2^0 * 0.5 = 1.0
			maxResult: 3.0, // 2 * 2^0 * 1.5 = 3.0
		},
		{
			name:      "zero attempt zero base",
			attempt:   0,
			baseDelay: 0.0,
			maxDelay:  60.0,
			minResult: 0.0,
			maxResult: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Run multiple times to account for jitter randomness
			for i := 0; i < 100; i++ {
				result := workerqueue.ExponentialBackoffWithJitter(tt.attempt, tt.baseDelay, tt.maxDelay)
				require.GreaterOrEqual(t, result, tt.minResult, "result should be >= minResult")
				require.LessOrEqual(t, result, tt.maxResult, "result should be <= maxResult")
			}
		})
	}
}

func TestExponentialBackoffProgression(t *testing.T) {
	t.Parallel()

	baseDelay := 1.0
	maxDelay := 1000.0

	// Test that average delay increases with attempts
	// We calculate average of many runs to account for jitter
	getAverageDelay := func(attempt int) float64 {
		sum := 0.0
		runs := 1000
		for i := 0; i < runs; i++ {
			sum += workerqueue.ExponentialBackoffWithJitter(attempt, baseDelay, maxDelay)
		}
		return sum / float64(runs)
	}

	avg0 := getAverageDelay(0)
	avg1 := getAverageDelay(1)
	avg2 := getAverageDelay(2)
	avg3 := getAverageDelay(3)

	// Each subsequent attempt should have roughly double the average delay
	require.Greater(t, avg1, avg0, "attempt 1 should have higher average than attempt 0")
	require.Greater(t, avg2, avg1, "attempt 2 should have higher average than attempt 1")
	require.Greater(t, avg3, avg2, "attempt 3 should have higher average than attempt 2")

	// Check approximate doubling (with some tolerance for randomness)
	require.InDelta(t, avg0*2, avg1, avg0*0.5, "attempt 1 should be ~2x attempt 0")
	require.InDelta(t, avg1*2, avg2, avg1*0.5, "attempt 2 should be ~2x attempt 1")
}

func TestExponentialBackoffMaxDelayCap(t *testing.T) {
	t.Parallel()

	baseDelay := 1.0
	maxDelay := 10.0

	// With high attempt number, the exponential would exceed maxDelay
	// but should be capped. With jitter, result can be up to 1.5x maxDelay
	for i := 0; i < 100; i++ {
		result := workerqueue.ExponentialBackoffWithJitter(20, baseDelay, maxDelay)
		// Max possible is maxDelay * 1.5 (due to jitter)
		require.LessOrEqual(t, result, maxDelay*1.5, "result should not exceed maxDelay * 1.5")
		// Min possible is maxDelay * 0.5 (due to jitter)
		require.GreaterOrEqual(t, result, maxDelay*0.5, "result should not be below maxDelay * 0.5")
	}
}
