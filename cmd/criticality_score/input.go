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

package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/url"
	"os"

	"github.com/ossf/criticality_score/internal/infile"
)

// iter is a simple interface for iterating across a list of items.
//
// This interface is modeled on the bufio.Scanner behavior.
type iter[T any] interface {
	// Item returns the current item in the iterator
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

// iterCloser is an iter, but also embeds the io.Closer interface, so it can be
// used to wrap a file for iterating through.
type iterCloser[T any] interface {
	iter[T]
	io.Closer
}

// scannerIter implements iter using a bufio.Scanner to iterate through lines in
// a file.
type scannerIter struct {
	r       io.ReadCloser
	scanner *bufio.Scanner
}

func (i *scannerIter) Item() string {
	return i.scanner.Text()
}

func (i *scannerIter) Next() bool {
	return i.scanner.Scan()
}

func (i *scannerIter) Err() error {
	return i.scanner.Err()
}

func (i *scannerIter) Close() error {
	return i.r.Close()
}

// sliceIter implements iter using a slice for iterating.
type sliceIter[T any] struct {
	values []T
	next   int
	size   int
}

func (i *sliceIter[T]) Item() T {
	return i.values[i.next-1]
}

func (i *sliceIter[T]) Next() bool {
	if i.next <= i.size {
		i.next++
	}
	return i.next <= i.size
}

func (i *sliceIter[T]) Err() error {
	return nil
}

func (i *sliceIter[T]) Close() error {
	return nil
}

// initInput returns an iterCloser for iterating across repositories for
// collecting signals.
//
// If only one arg is specified, the code will treat it as a file and attempt to
// open it. If the file doesn't exist, and is parseable as a URL the arg will be
// treated as a repo.
//
// If more than one arg is specified they are all considered to be repos.
//
// TODO: support the ability to force args to be interpreted as either a file,
// or a list of repos.
func initInput(args []string) (iterCloser[string], error) {
	if len(args) == 1 {
		// If there is 1 arg, attempt to open it as a file.
		fileOrRepo := args[0]
		_, err := url.Parse(fileOrRepo)
		notAUrl := err != nil

		// Open the in-file for reading
		r, err := infile.Open(context.Background(), fileOrRepo)
		if err == nil {
			return &scannerIter{
				r:       r,
				scanner: bufio.NewScanner(r),
			}, nil
		} else if err != nil && (notAUrl || !errors.Is(err, os.ErrNotExist)) {
			// Only report errors if the file doesn't appear to be a URL, or if
			// it doesn't exist.
			return nil, err
		}
	}
	// If file loading failed, or there are 2 or more args, treat args as a list
	// of repos.
	return &sliceIter[string]{
		size:   len(args),
		values: args,
	}, nil
}
