// Copyright 2022 Criticality Score Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
