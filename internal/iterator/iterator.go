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

package iterator

import (
	"io"
)

// Iter is a simple interface for iterating across a list of items.
//
// This interface is modeled on the bufio.Scanner behavior.
type Iter[T any] interface {
	// Item returns the current item in the iterator.
	//
	// Next() must be called before calling Item().
	Item() T

	// Next advances the iterator to the next item and returns true if there is
	// an item to consume, and false if the end of the input has been reached,
	// or there has been an error.
	//
	// Next must be called before each call to Item.
	Next() bool

	// Err returns any error produced while iterating.
	Err() error
}

// IterCloser is an iter, but also embeds the io.Closer interface, so it can be
// used to wrap a file for iterating through.
type IterCloser[T any] interface {
	Iter[T]
	io.Closer
}
