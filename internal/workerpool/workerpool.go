package workerpool

import (
	"sync"
)

// WorkerFunc implements a function that can be used in a WorkerPool.
//
// worker is a unique integer identifying the worker, starting at 0.
type WorkerFunc func(worker int)

// WorkerPool starts a pool of n workers each running the WorkerFunc w.
//
// Returns a function waitFunc, that when called waits for all workers to
// finish.
func WorkerPool(n int, w WorkerFunc) (waitFunc func()) {
	wg := &sync.WaitGroup{}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(worker int) {
			defer wg.Done()
			w(worker)
		}(i)
	}
	return wg.Wait
}
