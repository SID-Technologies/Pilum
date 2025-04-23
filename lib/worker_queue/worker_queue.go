package worker_queue

import (
	"log"
	"runtime"
	"sync"
)

// WorkQueue manages parallel execution of tasks.
type WorkQueue struct {
	taskQueue    chan *TaskInfo
	workerFunc   func(*TaskInfo) bool
	maxWorkers   int
	results      []bool
	resultsMutex sync.Mutex
}

// NewWorkQueue creates a new work queue with the given worker function.
func NewWorkQueue(workerFunc func(*TaskInfo) bool, maxWorkers int) *WorkQueue {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU() / 2
		if maxWorkers < 1 {
			maxWorkers = 1
		}
	}

	log.Println("Created Worker Queue")
	log.Printf("Using %d workers\n", maxWorkers)
	log.Println()

	return &WorkQueue{
		taskQueue:  make(chan *TaskInfo, 100),
		workerFunc: workerFunc,
		maxWorkers: maxWorkers,
		results:    make([]bool, 0),
	}
}

// AddTask adds a task to the queue.
func (wq *WorkQueue) AddTask(task *TaskInfo) {
	wq.taskQueue <- task
}

// worker processes tasks from the queue.
func (wq *WorkQueue) worker(wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range wq.taskQueue {
		result := wq.workerFunc(task)
		wq.resultsMutex.Lock()
		wq.results = append(wq.results, result)
		wq.resultsMutex.Unlock()
	}
}

// Execute runs all tasks in the queue and returns the results.
func (wq *WorkQueue) Execute() []bool {
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < wq.maxWorkers; i++ {
		wg.Add(1)
		go wq.worker(&wg)
	}

	// Close the channel when all tasks are added
	close(wq.taskQueue)

	// Wait for all workers to complete
	wg.Wait()

	return wq.results
}
