package workerqueue_test

import (
	"sync/atomic"
	"testing"

	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"

	"github.com/stretchr/testify/require"
)

func TestNewWorkQueue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		maxWorkers         int
		expectedMinWorkers int
	}{
		{
			name:               "explicit worker count",
			maxWorkers:         4,
			expectedMinWorkers: 4,
		},
		{
			name:               "zero defaults to CPU-based",
			maxWorkers:         0,
			expectedMinWorkers: 1, // At least 1
		},
		{
			name:               "negative defaults to CPU-based",
			maxWorkers:         -1,
			expectedMinWorkers: 1, // At least 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			workerFunc := func(task *workerqueue.TaskInfo) bool {
				return true
			}

			wq := workerqueue.NewWorkQueue(workerFunc, tt.maxWorkers)
			require.NotNil(t, wq)
		})
	}
}

func TestWorkQueueAddTaskAndExecute(t *testing.T) {
	t.Parallel()

	var processedCount atomic.Int32

	workerFunc := func(task *workerqueue.TaskInfo) bool {
		processedCount.Add(1)
		return true
	}

	wq := workerqueue.NewWorkQueue(workerFunc, 2)

	// Add tasks
	for i := 0; i < 5; i++ {
		task := workerqueue.NewTaskInfo(
			"echo hello",
			"",
			"svc",
			"root",
			nil,
			nil,
			60,
			false,
			1,
		)
		wq.AddTask(task)
	}

	// Execute
	results := wq.Execute()

	require.Len(t, results, 5)
	require.Equal(t, int32(5), processedCount.Load())

	// All should succeed
	for _, result := range results {
		require.True(t, result)
	}
}

func TestWorkQueueExecuteWithFailures(t *testing.T) {
	t.Parallel()

	callCount := 0

	workerFunc := func(task *workerqueue.TaskInfo) bool {
		callCount++
		// Fail every other task
		return callCount%2 == 0
	}

	wq := workerqueue.NewWorkQueue(workerFunc, 1) // Single worker for deterministic order

	for i := 0; i < 4; i++ {
		task := workerqueue.NewTaskInfo(
			"echo",
			"",
			"svc",
			"root",
			nil,
			nil,
			60,
			false,
			1,
		)
		wq.AddTask(task)
	}

	results := wq.Execute()

	require.Len(t, results, 4)

	// Count successes and failures
	successCount := 0
	failCount := 0
	for _, r := range results {
		if r {
			successCount++
		} else {
			failCount++
		}
	}

	require.Equal(t, 2, successCount)
	require.Equal(t, 2, failCount)
}

func TestWorkQueueExecuteEmpty(t *testing.T) {
	t.Parallel()

	workerFunc := func(task *workerqueue.TaskInfo) bool {
		return true
	}

	wq := workerqueue.NewWorkQueue(workerFunc, 2)

	// Execute without adding any tasks
	results := wq.Execute()

	require.Empty(t, results)
}

func TestWorkQueueConcurrency(t *testing.T) {
	t.Parallel()

	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	workerFunc := func(task *workerqueue.TaskInfo) bool {
		current := currentConcurrent.Add(1)

		// Track max concurrency
		for {
			max := maxConcurrent.Load()
			if current <= max {
				break
			}
			if maxConcurrent.CompareAndSwap(max, current) {
				break
			}
		}

		// Simulate work
		currentConcurrent.Add(-1)
		return true
	}

	wq := workerqueue.NewWorkQueue(workerFunc, 3)

	// Add more tasks than workers
	for i := 0; i < 10; i++ {
		task := workerqueue.NewTaskInfo(
			"echo",
			"",
			"svc",
			"root",
			nil,
			nil,
			60,
			false,
			1,
		)
		wq.AddTask(task)
	}

	results := wq.Execute()

	require.Len(t, results, 10)
	// Max concurrency should not exceed worker count
	require.LessOrEqual(t, maxConcurrent.Load(), int32(3))
}

func TestWorkQueueWithServiceNames(t *testing.T) {
	t.Parallel()

	processedServices := make([]string, 0)
	var mu atomic.Value
	mu.Store(processedServices)

	workerFunc := func(task *workerqueue.TaskInfo) bool {
		// Note: In real concurrent code, you'd need proper synchronization
		// This is simplified for testing
		return true
	}

	wq := workerqueue.NewWorkQueue(workerFunc, 2)

	services := []string{"svc-a", "svc-b", "svc-c"}
	for _, svc := range services {
		task := workerqueue.NewTaskInfo(
			"echo",
			"",
			svc,
			"root",
			nil,
			nil,
			60,
			false,
			1,
		)
		wq.AddTask(task)
	}

	results := wq.Execute()
	require.Len(t, results, 3)
}

func TestWorkQueueSingleWorker(t *testing.T) {
	t.Parallel()

	executionOrder := make([]int, 0)
	var mu atomic.Value
	mu.Store(executionOrder)

	counter := 0

	workerFunc := func(task *workerqueue.TaskInfo) bool {
		counter++
		return true
	}

	wq := workerqueue.NewWorkQueue(workerFunc, 1)

	for i := 0; i < 5; i++ {
		task := workerqueue.NewTaskInfo(
			"echo",
			"",
			"svc",
			"root",
			nil,
			nil,
			60,
			false,
			1,
		)
		wq.AddTask(task)
	}

	results := wq.Execute()

	require.Len(t, results, 5)
	require.Equal(t, 5, counter)
}
