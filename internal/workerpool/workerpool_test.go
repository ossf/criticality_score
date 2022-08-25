package workerpool_test

import (
	"sync/atomic"
	"testing"

	"github.com/ossf/criticality_score/internal/workerpool"
)

func TestOneWorker(t *testing.T) {
	var counter int32
	wait := workerpool.WorkerPool(1, func(worker int) {
		atomic.AddInt32(&counter, 1)
	})
	wait()
	if counter != 1 {
		t.Fatalf("counter = %d; want 1", counter)
	}
}

func TestManyWorkers(t *testing.T) {
	var counter int32
	wait := workerpool.WorkerPool(10, func(worker int) {
		atomic.AddInt32(&counter, 1)
	})
	wait()
	if counter != 10 {
		t.Fatalf("counter = %d; want 10", counter)
	}
}

func TestUniqueWorkerId(t *testing.T) {
	var counters [10]int32
	wait := workerpool.WorkerPool(10, func(worker int) {
		atomic.AddInt32(&(counters[worker]), 1)
	})
	wait()
	for worker, counter := range counters {
		if counter != 1 {
			t.Fatalf("counters[%d] = %d; want 1", worker, counter)
		}
	}
}

func TestExampleWorkload(t *testing.T) {
	nums := make(chan int)
	doubles := make(chan int)
	done := make(chan bool)
	var results []int

	// Consume the doubles channel
	go func() {
		for i := range doubles {
			results = append(results, i)
		}
		done <- true
	}()

	// Start a pool of workers
	wait := workerpool.WorkerPool(5, func(worker int) {
		for n := range nums {
			doubles <- n * 2
		}
	})

	// Send 0-9 into the nums channel
	for i := 0; i < 10; i++ {
		nums <- i
	}

	// Close nums which causes the workers to quit
	close(nums)

	// Wait for all the workers to be finished
	wait()

	// Close the doubles channels to terminate the consumer.
	close(doubles)

	// Wait for the consumer to be finished.
	<-done

	// Make sure all were generated
	if l := len(results); l != 10 {
		t.Fatalf("len(results) = %d; want 10", l)
	}
	// Make sure the results are actually doubles
	for _, r := range results {
		if r%2 != 0 || r < 0 || r/2 >= 10 {
			t.Fatalf("result = %d, want result to be divisible by 2, and less than 20", r)
		}
	}
}
